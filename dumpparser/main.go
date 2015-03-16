package main

import (
	"compress/bzip2"
	"flag"
	"fmt"
	"github.com/semanticize/dumpparser/hash"
	"github.com/semanticize/dumpparser/hash/countmin"
	"github.com/semanticize/dumpparser/nlp"
	"github.com/semanticize/dumpparser/storage"
	"github.com/semanticize/dumpparser/wikidump"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

func open(path string) (r io.ReadCloser, err error) {
	r, err = os.Open(path)
	if err == nil && filepath.Ext(path) == ".bz2" {
		r = struct {
			io.Reader
			io.Closer
		}{bzip2.NewReader(r), r}
	}
	return
}

var download = flag.String("download", "",
	"download Wikipedia dump (e.g., 'enwiki')")
var nrows = flag.Uint("nrows", 16, "number of rows in count-min sketch")
var ncols = flag.Uint("ncols", 262144, "number of columns in count-min sketch")
var maxNGram = flag.Uint("ngram", 7, "max. length of n-grams")

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
	db, err := storage.MakeDB(dbpath, true)
	check()

	// The numbers here are completely arbitrary.
	nworkers := runtime.GOMAXPROCS(0)
	articles := make(chan *wikidump.Page, 10*nworkers)
	links := make(chan *wikidump.Link, 100*nworkers)
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
		counters[i] = countmin.New(int(*nrows), int(*ncols))

		go func(ngramcount *countmin.Sketch) {
			for a := range articles {
				text := wikidump.Cleanup(a.Text)
				for _, link := range wikidump.ExtractLinks(text) {
					links <- &link
				}

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
		ins := storage.MustPrepare(db,
			`insert or ignore into linkstats values (?,?,0)`)
		update := storage.MustPrepare(db,
			`update linkstats set count = count + ? where ngramhash = ? and target = ?`)

		for link := range links {
			tokens := nlp.Tokenize(link.Anchor)
			n := min(maxN, len(tokens))
			hashes := hash.NGrams(tokens, n, n)
			count := 1.
			if len(hashes) > 1 {
				count = 1 / float64(len(hashes))
			}
			for _, h := range hashes {
				_, err = ins.Exec(h, link.Target)
				check()
				_, err = update.Exec(count, h, link.Target)
				check()
			}
		}
		wg.Done()
	}()

	go wikidump.GetPages(f, articles, redirects)

	wg.Wait()
	close(done)

	log.Printf("Processing %d redirects", len(redirmap))
	storage.ProcessRedirects(db, redirmap)

	log.Println("Finalizing database")
	err = storage.Finalize(db)
	check()
	db.Close()
	check()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
