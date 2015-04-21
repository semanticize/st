package main

import (
	"database/sql"
	"github.com/semanticize/dumpparser/hash"
	"github.com/semanticize/dumpparser/hash/countmin"
	"github.com/semanticize/dumpparser/nlp"
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
	q := `select (select title from titles where id = targetid), count
	      from linkstats where ngramhash = ?`
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
