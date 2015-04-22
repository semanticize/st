package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/semanticize/st/storage"
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

var dohttp = flag.String("http", "",
	"Serve HTTP requests, e.g., -http=localhost:8080")

func main() {
	log.SetPrefix("semanticizest ")

	flag.Parse()
	fmt.Printf("%q\n", *dohttp)

	var err error
	check := func() {
		if err != nil {
			log.Fatal(err)
		}
	}

	db, settings, err := storage.LoadModel(os.Args[1])
	check()
	ngramcount, err := storage.LoadCM(db)
	check()

	sem := semanticizer{db, ngramcount, settings.MaxNGram}

	if *dohttp == "" {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Split(splitPara)
		for scanner.Scan() {
			var candidates []candidate
			candidates, err = sem.allCandidates(scanner.Text())
			check()
			for _, c := range candidates {
				fmt.Printf("%f %f %q\n", c.commonness, c.senseprob, c.target)
			}
		}
		err = scanner.Err()
		check()
	} else {
		log.Fatal(restServer(*dohttp, settings))
	}
}
