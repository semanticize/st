// Semanticizer, STandalone: parser for Wikipedia database dumps.
//
// Takes a Wikipedia database dump (or downloads one automatically) and
// produces a model for use by the semanticizest program/web server.
//
// Run with --help for command-line usage.
package main

import (
	"bufio"
	"compress/bzip2"
	"database/sql"
	"github.com/cheggaaa/pb"
	"github.com/semanticize/st/hash"
	"github.com/semanticize/st/hash/countmin"
	"github.com/semanticize/st/nlp"
	"github.com/semanticize/st/storage"
	"github.com/semanticize/st/wikidump"
	"gopkg.in/alecthomas/kingpin.v1"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

func init() {
	if os.Getenv("GOMAXPROCS") == "" {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}
}

func open(path string) (r io.ReadCloser, err error) {
	rf, err := os.Open(path)
	if err != nil {
		return
	}
	r = struct {
		*bufio.Reader
		io.Closer
	}{bufio.NewReader(rf), rf}
	if filepath.Ext(path) == ".bz2" {
		r = struct {
			io.Reader
			io.Closer
		}{bzip2.NewReader(r), rf}
	}
	return
}

var (
	dbpath    = kingpin.Arg("model", "path to model").Required().String()
	download  = kingpin.Flag("download",
		"download Wikipedia dump (e.g., enwiki/latest)").String()
	nrows = kingpin.Flag("nrows",
		"number of rows in count-min sketch").Default("16").Int()
	ncols = kingpin.Flag("ncols",
		"number of columns in count-min sketch").Default("65536").Int()
	maxNGram = kingpin.Flag("ngram",
		"max. length of n-grams").Default(strconv.Itoa(storage.DefaultMaxNGram)).Int()
	dumppaths = kingpin.Arg("dump", "path to Wikipedia dump").Strings()
)

func main() {
	kingpin.Parse()

	log.SetPrefix("dumpparser ")

	var err error
	check := func() {
		if err != nil {
			log.Fatal(err)
		}
	}

	if *download != "" {
		*dumppaths, err = wikidump.DownloadParts(*download)
		check()
	}
	if len(*dumppaths) == 0 {
		log.Fatal("no --download and no dumppath specified (try --help)")
	}

	log.Printf("Creating database at %s", *dbpath)
	db, err := storage.MakeDB(*dbpath, true,
		&storage.Settings{"", uint(*maxNGram)})
	check()

	// The numbers here are completely arbitrary.
	nworkers := runtime.GOMAXPROCS(0)
	articles := make(chan *wikidump.Page, 10*nworkers)
	linkch := make(chan *processedLink, 10*nworkers)
	redirch := make(chan *wikidump.Redirect, 10*nworkers)

	var inputWg sync.WaitGroup
	for _, path := range *dumppaths {
		inputWg.Add(1)
		go func() {
			var f *os.File
			f, err = open(path)
			check()
			defer f.Close()
			wikidump.GetPages(f, articles, redirch)
			inputWg.Done()
		}()
	}
	go func() {
		inputWg.Wait()
		close(articles)
		close(redirch)
	}()

	// Clean up and tokenize articles, extract links, count n-grams.
	counters := make(chan *countmin.Sketch, nworkers)
	counterTotal, err := countmin.New(int(*nrows), int(*ncols))
	check()

	log.Printf("processing dump with %d workers", nworkers)
	var narticles uint32
	for i := 0; i < nworkers; i++ {
		go func() {
			counters <- processPages(articles, linkch, &narticles)
		}()
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		for i := 1; i < nworkers; i++ {
			counterTotal.Sum(<-counters)
		}
		close(counters) // Force panic for programmer error.
		close(linkch)   // We know the workers are done now.
		wg.Done()
	}()

	// Collect redirects. We store these in nworkers slices to avoid having
	// to copy them into a single structure.
	// The allRedirects channel MUST be buffered.
	wg.Add(nworkers)
	allRedirects := make(chan []wikidump.Redirect, nworkers)
	var nredirs uint32
	for i := 0; i < nworkers; i++ {
		go func() {
			slice := collectRedirects(redirch)
			atomic.AddUint32(&nredirs, uint32(len(slice)))
			allRedirects <- slice
			wg.Done()
		}()
	}

	go pageProgress(&narticles, &wg)

	err = storeLinks(db, linkch)
	check()

	wg.Wait()
	close(allRedirects)

	log.Printf("Processing redirects")
	bar := pb.StartNew(int(nredirs))
	for slice := range allRedirects {
		err = storage.StoreRedirects(db, slice, bar)
		check()
	}
	bar.Finish()

	err = storage.StoreCM(db, counterTotal)
	check()

	log.Println("Finalizing database")
	err = storage.Finalize(db)
	check()
	err = db.Close()
	check()
}

// Collect redirects from redirch into a slice.
//
// We have to collect these in memory because we process them only after all
// link statistics have been dumped into the database.
func collectRedirects(redirch <-chan *wikidump.Redirect) []wikidump.Redirect {

	redirects := make([]wikidump.Redirect, 0, 1024) // The 1024 is arbitrary.
	for r := range redirch {
		// XXX *r copies the struct.
		// Maybe copying the pointer is cheaper; should profile.
		redirects = append(redirects, *r)
	}
	return redirects
}

func processPages(articles <-chan *wikidump.Page,
	linkch chan<- *processedLink, narticles *uint32) *countmin.Sketch {

	ngramcount, err := countmin.New(int(*nrows), int(*ncols))
	if err != nil {
		// Shouldn't happen; we already constructed a count-min sketch
		// with the exact same size in main.
		panic(err)
	}

	maxN := int(*maxNGram)
	for a := range articles {
		text := wikidump.Cleanup(a.Text)
		links := wikidump.ExtractLinks(text)
		for link, freq := range links {
			linkch <- processLink(&link, freq, maxN)
		}

		tokens := nlp.Tokenize(text)
		for _, h := range hash.NGrams(tokens, 1, maxN) {
			ngramcount.Add1(h)
		}
		atomic.AddUint32(narticles, 1)
	}
	return ngramcount
}

// Regularly report the number of pages processed so far.
func pageProgress(narticles *uint32, wg *sync.WaitGroup) {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		done <- struct{}{}
	}()

	timeout := time.Tick(15 * time.Second)
	for {
		select {
		case <-done:
			log.Printf("processed all %d articles",
				atomic.LoadUint32(narticles))
			return
		case <-timeout:
			log.Printf("processed %d articles", atomic.LoadUint32(narticles))
		}
	}
}

type processedLink struct {
	target       string
	anchorHashes []uint32
	freq         float64
}

func processLink(link *wikidump.Link, freq, maxN int) *processedLink {
	tokens := nlp.Tokenize(link.Anchor)
	n := min(maxN, len(tokens))
	hashes := hash.NGrams(tokens, n, n)
	count := float64(freq)
	if len(hashes) > 1 {
		count = 1 / float64(len(hashes))
	}
	return &processedLink{link.Target, hashes, count}
}

// Collect links and store them in the database.
func storeLinks(db *sql.DB, links <-chan *processedLink) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return
	}
	insTitle, err := tx.Prepare(`insert or ignore into titles values (NULL, ?)`)
	if err != nil {
		return
	}
	insLink, err := tx.Prepare(
		`insert or ignore into linkstats values
		 (?, (select id from titles where title = ?), 0)`)
	if err != nil {
		return
	}
	update, err := tx.Prepare(
		`update linkstats set count = count + ?
		 where ngramhash = ?
		 and targetid = (select id from titles where title =?)`)
	if err != nil {
		return
	}

	exec := func(stmt *sql.Stmt, args ...interface{}) {
		if err == nil {
			_, err = stmt.Exec(args...)
		}
	}

	for link := range links {
		count := link.freq
		for _, h := range link.anchorHashes {
			exec(insTitle, link.target)
			exec(insLink, h, link.target)
			exec(update, count, h, link.target)
		}
		if err != nil {
			break
		}
	}
	err = tx.Commit()
	return
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
