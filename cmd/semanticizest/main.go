// Semanticizer, STandalone: command-line program and REST API server.
//
// Takes a semanticizer model and some text (from stdin or an HTTP connection),
// produces entity links in a simple JSON format.
//
// Run with --help to see command-line usage.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log"
	"os"
	"regexp"

	"gopkg.in/alecthomas/kingpin.v1"

	"github.com/semanticize/st/linking"
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
	token = bytes.TrimSpace(token)
	return
}

var (
	dbpath = kingpin.Arg("model", "path to model file").Required().String()
	dohttp = kingpin.Flag("http",
		"HTTP server address; use :0 for a random port").Default("").String()
	portfile = kingpin.Flag("portfile",
		"write server port to this file (useful with :0)").Default("").String()
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

	log.Printf("loading database from %s", *dbpath)
	sem, settings, err := linking.Load(*dbpath)
	check()
	log.Print("database loaded")

	if *dohttp == "" {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Split(splitPara)

		out := json.NewEncoder(os.Stdout)

		for scanner.Scan() {
			var candidates []linking.Entity
			candidates, err = sem.All(scanner.Text())
			check()

			err = out.Encode(candidates)
			check()
		}
		err = scanner.Err()
		check()
	} else {
		log.Fatal(restServer(*dohttp, *portfile, sem, settings))
	}
}
