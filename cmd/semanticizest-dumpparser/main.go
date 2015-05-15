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
)

func init() {
	if os.Getenv("GOMAXPROCS") == "" {
		// Four is about the number of cores that we can put to useful work
		// when the disk is fast.
		runtime.GOMAXPROCS(min(runtime.NumCPU(), 4))
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
	dbpath   = kingpin.Arg("model", "path to model").Required().String()
	dumppath = kingpin.Arg("dump", "path to Wikipedia dump").String()
	download = kingpin.Flag("download",
		"download Wikipedia dump (e.g., enwiki)").String()
	nrows = kingpin.Flag("nrows",
		"number of rows in count-min sketch").Default("16").Int()
	ncols = kingpin.Flag("ncols",
		"number of columns in count-min sketch").Default("65536").Int()
	maxNGram = kingpin.Flag("ngram",
		"max. length of n-grams").Default(strconv.Itoa(storage.DefaultMaxNGram)).Int()
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
		*dumppath, err = wikidump.Download(*download, *dumppath, true)
		check()
	} else if *dumppath == "" {
		log.Fatal("no --download and no dumppath specified (try --help)")
	}

	f, err := open(*dumppath)
	check()
	defer f.Close()

	log.Printf("Creating database at %s", *dbpath)
	db, err := storage.MakeDB(*dbpath, true,
		&storage.Settings{*dumppath, uint(*maxNGram)})
	check()

	// The numbers here are completely arbitrary.
	nworkers := runtime.GOMAXPROCS(0)
	articles := make(chan *wikidump.Page, 10*nworkers)
	linkch := make(chan *processedLink, 10*nworkers)
	redirects := make(chan *wikidump.Redirect, 100)

	go wikidump.GetPages(f, articles, redirects)

	// Clean up and tokenize articles, extract links, count n-grams.
	maxN := int(*maxNGram)
	counters := make(chan *countmin.Sketch, nworkers)

	// If this countmin.New succeeds, then so will the rest, so we don't need
	// to check their return values.
	counterTotal, err := countmin.New(int(*nrows), int(*ncols))
	check()

	log.Printf("processing dump with %d workers", nworkers)
	for i := 0; i < nworkers; i++ {
		go func() {
			ngramcount, _ := countmin.New(int(*nrows), int(*ncols))
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
			}
			counters <- ngramcount
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

	// Collect redirects.
	wg.Add(1)
	redirmap := make(map[string]string)
	go func() {
		for r := range redirects {
			redirmap[r.Title] = r.Target
		}
		wg.Done()
	}()

	err = storeLinks(db, linkch)
	check()

	wg.Wait()

	log.Printf("Processing %d redirects", len(redirmap))
	storage.StoreRedirects(db, redirmap, true)

	err = storage.StoreCM(db, counterTotal)
	check()

	log.Println("Finalizing database")
	err = storage.Finalize(db)
	check()
	err = db.Close()
	check()
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
	insTitle, err := db.Prepare(`insert or ignore into titles values (NULL, ?)`)
	if err != nil {
		return
	}
	insLink, err := db.Prepare(
		`insert or ignore into linkstats values
		 (?, (select id from titles where title = ?), 0)`)
	if err != nil {
		return
	}
	update, err := db.Prepare(
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
	return
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
