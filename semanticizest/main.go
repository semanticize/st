// Semanticizer, STandalone: command-line program and REST API server.
//
// Takes a semanticizer model and some text (from stdin or an HTTP connection),
// produces entity links in a simple JSON format.
//
// Run with --help to see command-line usage.
package main

import (
	"bufio"
	"encoding/json"
	"github.com/semanticize/st/storage"
	"gopkg.in/alecthomas/kingpin.v1"
	"log"
	"os"
	"regexp"
)

var paraEnd = regexp.MustCompile(`\n\s*\n`)

// Paragraph splitter for bufio.Scanner.
// XXX If we use regexp's RuneReader support, we can do this without buffering.
func splitPara(data []byte, atEOF bool) (advance int, token []byte, err error) {
	loc := paraEnd.FindIndex(data)
	if loc != nil {
		advance = loc[1]
		token = data[:loc[0]]
	} else if atEOF {
		advance = len(data)
		token = data
	}
	return
}

var (
	dbpath = kingpin.Arg("model", "path to model file").Required().String()
	dohttp = kingpin.Flag("http", "HTTP server address").Default("").String()
)

func main() {
	kingpin.Parse()

	log.SetPrefix("semanticizest ")

	var err error
	check := func() {
		if err != nil {
			log.Fatal(err)
		}
	}

	db, settings, err := storage.LoadModel(*dbpath)
	check()
	ngramcount, err := storage.LoadCM(db)
	check()

	sem := semanticizer{db, ngramcount, settings.MaxNGram}

	if *dohttp == "" {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Split(splitPara)

		out := json.NewEncoder(os.Stdout)

		for scanner.Scan() {
			var candidates []candidate
			candidates, err = sem.allCandidates(scanner.Text())
			check()

			err = out.Encode(candidates)
			check()
		}
		err = scanner.Err()
		check()
	} else {
		log.Fatal(restServer(*dohttp, &sem, settings))
	}
}
