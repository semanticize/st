package main

import (
	"database/sql"
	"github.com/semanticize/st/hash"
	"github.com/semanticize/st/hash/countmin"
	"github.com/semanticize/st/nlp"
)

type semanticizer struct {
	db         *sql.DB
	ngramcount *countmin.Sketch
	maxNGram   uint
}

type candidate struct {
	Target string `json:"target"`

	// Raw n-gram count estimate.
	NGramCount float64 `json:"ngramcount"`

	// Total number of links to Target in Wikipedia.
	LinkCount float64 `json:"linkcount"`

	Commonness float64 `json:"commonness"`
	Senseprob  float64 `json:"senseprob"`

	// Offset of anchor in input string.
	Offset int `json:"offset"`

	// Length of anchor in input string.
	Length int `json:"length"`
}

// Get candidates for hash value h from the database.
func (sem semanticizer) candidates(h uint32, offset, end int) (cands []candidate, err error) {
	q := `select (select title from titles where id = targetid), count
	      from linkstats where ngramhash = ?`
	rows, err := sem.db.Query(q, h)
	if err != nil {
		return
	}

	var count, totalLinkCount float64
	var target string
	for rows.Next() {
		rows.Scan(&target, &count)
		totalLinkCount += count
		// Initially use the Commonness field to store the number of links
		// to the target with the given hash.
		cands = append(cands, candidate{
			Target:     target,
			Commonness: count,
			Senseprob:  0,
			Offset:     offset,
			Length:     end - offset,
		})
	}
	rows.Close()
	err = rows.Err()
	if err != nil {
		return
	}

	for i := range cands {
		c := &cands[i]
		c.NGramCount = float64(sem.ngramcount.Get(h))
		c.Senseprob = c.Commonness / c.NGramCount
		c.Commonness /= totalLinkCount
	}
	return
}

func (sem semanticizer) allCandidates(s string) (cands []candidate, err error) {
	tokens, tokpos := nlp.TokenizePos(s)
	for _, hpos := range hash.NGramsPos(tokens, int(sem.maxNGram)) {
		start, end := hpos.Start, hpos.End
		start, end = tokpos[start][0], tokpos[end][1]

		add, err := sem.candidates(hpos.Hash, start, end)
		if err != nil {
			break
		}
		cands = append(cands, add...)
	}
	return
}
