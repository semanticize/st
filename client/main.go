package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"github.com/semanticize/dumpparser/hash"
	"github.com/semanticize/dumpparser/nlp"
	_ "github.com/semanticize/dumpparser/storage"
	"log"
	"os"
	"regexp"
)

const maxNGram = 7  // TODO read from database

type semanticizer struct {
	db *sql.DB
}

type candidate struct {
	target     string
	commonness float64
}

// Commonness (prior probability of being a link) for a hash value.
func (sem semanticizer) commonness(h uint32) (cands []candidate, err error) {
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
		cands = append(cands, candidate{target, count})
	}
	rows.Close()
	err = rows.Err()
	if err != nil {
		return
	}

	for i := range cands {
		cands[i].commonness /= total
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
	for _, h := range hash.NGrams(tokens, 1, maxNGram) {
		add, err := sem.commonness(h)
		if err != nil {
			break
		}
		cands = append(cands, add...)
	}
	return
}

func main() {
	log.SetPrefix("semanticizest")

	var err error
	check := func() {
		if err != nil {
			log.Fatal(err)
		}
	}

	db, err := sql.Open("sqlite3", os.Args[1])
	check()

	sem := semanticizer{db}

	//ngramcount, err := storage.LoadCM(db)
	//check()

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(splitPara)
	for scanner.Scan() {
		var candidates []candidate
		candidates, err = sem.allCandidates(scanner.Text())
		check()
		for _, c := range candidates {
			fmt.Printf("%f %q\n", c.commonness, c.target)
		}
	}
	err = scanner.Err()
	check()
}
