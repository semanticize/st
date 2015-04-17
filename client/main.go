package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"github.com/semanticize/dumpparser/hash"
	"github.com/semanticize/dumpparser/hash/countmin"
	"github.com/semanticize/dumpparser/nlp"
	"github.com/semanticize/dumpparser/storage"
	"log"
	"os"
	"regexp"
)

type semanticizer struct {
	db         *sql.DB
	ngramcount *countmin.Sketch
	maxNGram   uint
}

type candidate struct {
	target                string
	commonness, senseprob float64
}

// Get candidates for hash value h from the database.
func (sem semanticizer) candidates(h uint32) (cands []candidate, err error) {
	q := `select target, count from linkstats where ngramhash = ?`
	rows, err := sem.db.Query(q, h)
	if err != nil {
		return
	}

	var count, total float64
	var target string
	for rows.Next() {
		rows.Scan(&target, &count)
		total += count
		// Initially use the commonness field to store the count.
		cands = append(cands, candidate{target, count, 0})
	}
	rows.Close()
	err = rows.Err()
	if err != nil {
		return
	}

	for i := range cands {
		c := &cands[i]
		c.senseprob = c.commonness / float64(sem.ngramcount.Get(h))
		c.commonness /= total
	}
	return
}

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

func (sem semanticizer) allCandidates(s string) (cands []candidate, err error) {
	tokens := nlp.Tokenize(s)
	for _, h := range hash.NGrams(tokens, 1, int(sem.maxNGram)) {
		add, err := sem.candidates(h)
		if err != nil {
			break
		}
		cands = append(cands, add...)
	}
	return
}

func main() {
	log.SetPrefix("semanticizest ")

	var err error
	check := func() {
		if err != nil {
			log.Fatal(err)
		}
	}

	db, maxNGram, err := storage.LoadModel(os.Args[1])
	check()
	ngramcount, err := storage.LoadCM(db)
	check()

	sem := semanticizer{db, ngramcount, maxNGram}

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
}
