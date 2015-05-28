// Semanticizer, STandalone: parser for Wikipedia database dumps.
//
// Takes a Wikipedia database dump (or downloads one automatically) and
// produces a model for use by the semanticizest program/web server.
//
// Run with --help for command-line usage.
package internal

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
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

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

func Main(dbpath, dumppath, download string, nrows, ncols, maxNGram int,
	logger *log.Logger) (err error) {

	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()
	realMain(dbpath, dumppath, download, nrows, ncols, maxNGram, logger)
	return
}

func realMain(dbpath, dumppath, download string, nrows, ncols, maxNGram int,
	logger *log.Logger) {

	var err error
	check := func() {
		if err != nil {
			panic(err)
		}
	}

	if download != "" {
		dumppath, err = wikidump.Download(download, dumppath, true)
		check()
	} else if dumppath == "" {
		panic("no --download and no dumppath specified (try --help)")
	}

	f, err := open(dumppath)
	check()
	defer f.Close()

	logger.Printf("Creating database at %s", dbpath)
	db, err := storage.MakeDB(dbpath, true,
		&storage.Settings{dumppath, uint(maxNGram)})
	check()

	// The numbers here are completely arbitrary.
	nworkers := runtime.GOMAXPROCS(0)
	articles := make(chan *wikidump.Page, 10*nworkers)
	linkch := make(chan *processedLink, 10*nworkers)
	redirch := make(chan *wikidump.Redirect, 10*nworkers)

	// Clean up and tokenize articles, extract links, count n-grams.
	counters := make(chan *countmin.Sketch, nworkers)
	counterTotal, err := countmin.New(nrows, ncols)
	check()

	go wikidump.GetPages(f, articles, redirch)

	logger.Printf("processing dump with %d workers", nworkers)
	var narticles uint32
	for i := 0; i < nworkers; i++ {
		// These signal completion by sending on counters.
		go func() {
			counters <- processPages(articles, linkch, &narticles,
				nrows, ncols, maxNGram)
		}()
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		for i := 0; i < nworkers; i++ {
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

	go pageProgress(&narticles, logger, &wg)

	err = storeLinks(db, linkch)

	wg.Wait()
	close(allRedirects)
	// Check error from storeLinks now, after goroutines have stopped.
	check()

	logger.Printf("Processing redirects")
	bar := pb.StartNew(int(nredirs))
	for slice := range allRedirects {
		err = storage.StoreRedirects(db, slice, bar)
		check()
	}
	bar.Finish()

	err = storage.StoreCM(db, counterTotal)
	check()

	logger.Println("Finalizing database")
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
	linkch chan<- *processedLink, narticles *uint32,
	nrows, ncols, maxN int) *countmin.Sketch {

	ngramcount, err := countmin.New(nrows, ncols)
	if err != nil {
		// Shouldn't happen; we already constructed a count-min sketch
		// with the exact same size in main.
		panic(err)
	}

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
func pageProgress(narticles *uint32, logger *log.Logger, wg *sync.WaitGroup) {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		done <- struct{}{}
	}()

	timeout := time.Tick(15 * time.Second)
	for {
		select {
		case <-done:
			logger.Printf("processed all %d articles",
				atomic.LoadUint32(narticles))
			return
		case <-timeout:
			logger.Printf("processed %d articles", atomic.LoadUint32(narticles))
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
