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

	// The number 10 is completely arbitrary, but seems to speed things up.
	articles := make(chan *wikidump.Page, 10)
	links := make(chan *wikidump.Link, 10)
	redirects := make(chan *wikidump.Redirect, 10)
	tocounter := make(chan string, 10)

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

	// Clean up and tokenize articles, extract links.
	wg.Add(1)
	go func() {
		for a := range articles {
			log.Printf("processing %q\n", a.Title)
			text := wikidump.Cleanup(a.Text)
			tocounter <- text
			for _, link := range wikidump.ExtractLinks(text) {
				links <- &link
			}
		}
		close(links)
		close(tocounter)
		wg.Done()
	}()

	// Collect n-gram (hash) counts.
	wg.Add(1)
	ngramcount := countmin.New(int(*nrows), int(*ncols))
	maxN := int(*maxNGram)
	go func() {
		for text := range tocounter {
			tokens := nlp.Tokenize(text)
			for _, h := range hash.NGrams(tokens, 1, maxN) {
				ngramcount.Add1(h)
			}
		}
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
