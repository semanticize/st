package main

import (
	"bufio"
	"compress/bzip2"
	"database/sql"
	"flag"
	"fmt"
	"github.com/semanticize/st/hash"
	"github.com/semanticize/st/hash/countmin"
	"github.com/semanticize/st/nlp"
	"github.com/semanticize/st/storage"
	"github.com/semanticize/st/dumpparser/wikidump"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
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

var download = flag.String("download", "",
	"download Wikipedia dump (e.g., 'enwiki')")
var nrows = flag.Uint("nrows", 16, "number of rows in count-min sketch")
var ncols = flag.Uint("ncols", 262144, "number of columns in count-min sketch")
var maxNGram = flag.Uint("ngram", storage.DefaultMaxNGram,
	"max. length of n-grams")

func main() {
	log.SetPrefix("dumpparser ")

	var err error
	check := func() {
		if err != nil {
			log.Fatal(err)
		}
	}

	flag.Parse()
	args := flag.Args()

	var dbpath, inputpath string
	if *download != "" {
		if len(args) != 1 {
			fmt.Fprintf(os.Stderr,
				"usage: %s -download=wikiname model.db\n", os.Args[0])
			os.Exit(1)
		}
		inputpath, err = wikidump.Download(*download, true)
		check()
		dbpath = args[0]
	} else {
		if len(args) != 2 {
			fmt.Fprintf(os.Stderr, "usage: %s wikidump model.db\n", os.Args[0])
			os.Exit(1)
		}
		inputpath, dbpath = args[0], args[1]
	}

	f, err := open(inputpath)
	check()
	defer f.Close()

	log.Printf("Creating database at %s", dbpath)
	db, err := storage.MakeDB(dbpath, true,
		&storage.Settings{inputpath, *maxNGram})
	check()

	// The numbers here are completely arbitrary.
	nworkers := runtime.GOMAXPROCS(0)
	articles := make(chan *wikidump.Page, 10*nworkers)
	links := make(chan map[wikidump.Link]int, 10*nworkers)
	redirects := make(chan *wikidump.Redirect, 100)

	var wg sync.WaitGroup

	// Collect redirects.
	wg.Add(1)
	redirmap := make(map[string]string)
	go func() {
		for r := range redirects {
			redirmap[r.Title] = r.Target
		}
		wg.Done()
	}()

	// Clean up and tokenize articles, extract links, count n-grams.
	maxN := int(*maxNGram)
	counters := make([]*countmin.Sketch, nworkers)

	var worker sync.WaitGroup
	worker.Add(nworkers)
	log.Printf("%d workers", nworkers)
	for i := 0; i < nworkers; i++ {
		counters[i], err = countmin.New(int(*nrows), int(*ncols))
		check()

		go func(ngramcount *countmin.Sketch) {
			for a := range articles {
				text := wikidump.Cleanup(a.Text)
				links <- wikidump.ExtractLinks(text)

				tokens := nlp.Tokenize(text)
				for _, h := range hash.NGrams(tokens, 1, maxN) {
					ngramcount.Add1(h)
				}
			}
			worker.Done()
		}(counters[i])
	}

	wg.Add(1)
	go func() {
		worker.Wait()
		close(links)

		for i := 1; i < nworkers; i++ {
			counters[0].Sum(counters[i])
		}
		counters = counters[:1]

		wg.Done()
	}()

	// Collect links and store them in the database.
	wg.Add(1)
	done := make(chan struct{})
	go func() {
		if slerr := storeLinks(db, links, maxN); slerr != nil {
			panic(slerr)
		}
		wg.Done()
	}()

	go wikidump.GetPages(f, articles, redirects)

	wg.Wait()
	close(done)

	log.Printf("Processing %d redirects", len(redirmap))
	storage.ProcessRedirects(db, redirmap)

	err = storage.StoreCM(db, counters[0])
	check()

	log.Println("Finalizing database")
	err = storage.Finalize(db)
	check()
	err = db.Close()
	check()
}

func storeLinks(db *sql.DB, links <-chan map[wikidump.Link]int,
	maxN int) (err error) {

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

	for linkFreq := range links {
		for link, freq := range linkFreq {
			tokens := nlp.Tokenize(link.Anchor)
			n := min(maxN, len(tokens))
			hashes := hash.NGrams(tokens, n, n)
			count := float64(freq)
			if len(hashes) > 1 {
				count = 1 / float64(len(hashes))
			}
			for _, h := range hashes {
				_, err = insTitle.Exec(link.Target)
				if err != nil {
					return
				}
				_, err = insLink.Exec(h, link.Target)
				if err != nil {
					return
				}
				_, err = update.Exec(count, h, link.Target)
				if err != nil {
					return
				}
			}
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
