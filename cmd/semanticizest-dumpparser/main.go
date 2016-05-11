// Semanticizer, STandalone: parser for Wikipedia database dumps.
//
// Takes a Wikipedia database dump (or downloads one automatically) and
// produces a model for use by the semanticizest program/web server.
//
// Run with --help for command-line usage.
package main

import (
	"log"
	"os"
	"runtime"
	"strconv"

	"gopkg.in/alecthomas/kingpin.v1"

	"github.com/semanticize/st/internal/dumpparser"
	"github.com/semanticize/st/internal/storage"
)

func init() {
	if os.Getenv("GOMAXPROCS") == "" {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}
}

var (
	dbpath   = kingpin.Arg("model", "path to model").Required().String()
	dumppath = kingpin.Arg("dump", "path to Wikipedia dump").String()
	download = kingpin.Flag("download",
		"download Wikipedia dump (e.g., enwiki)").String()
	nrows = kingpin.Flag("nrows",
		"number of rows in count-min sketch").Default("16").Int()
	ncols = kingpin.Flag("ncols",
		"number of columns in count-min sketch").Default("16777216").Int()
	maxNGram = kingpin.Flag("ngram",
		"max. length of n-grams").Default(strconv.Itoa(storage.DefaultMaxNGram)).Int()
)

func main() {
	kingpin.Parse()

	l := log.New(os.Stderr, "dumpparser ", log.Ldate|log.Ltime)
	err := dumpparser.Main(*dbpath, *dumppath, *download, *nrows, *ncols,
		*maxNGram, l)
	if err != nil {
		l.Fatal(err)
	}
}
