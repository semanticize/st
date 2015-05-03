package main

import (
	"database/sql"
	"github.com/semanticize/st/hash"
	"github.com/semanticize/st/hash/countmin"
	"github.com/semanticize/st/nlp"
	"math"
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

// Dynamic programming table: row-matrix matrix with backpointers.
type dpTable struct {
	a     []entry
	ncols int
}

type entry struct {
	v    float64
	prev int
}

func newTable(n, m int) dpTable {
	return dpTable{make([]entry, n*m), m}
}

func (m *dpTable) at(i, j int) *entry {
	return &m.a[i*m.ncols+j]
}

func (m *dpTable) nrows() int {
	return len(m.a) / m.ncols
}

func (m *dpTable) row(i int) []entry {
	return m.a[i*m.ncols : (i+1)*m.ncols]
}

// Computes the best path through the DP table, which must have been
// pre-populated with values.
func (m *dpTable) viterbi() (path []int) {
	for i := 0; i < m.nrows()-1; i++ {
		for k := 0; k < m.ncols; k++ {
			var argmax int
			max := -math.Inf(-1)
			for j, e := range m.row(i) {
				if e.v > max {
					argmax = j
					max = e.v
				}
			}
			e := m.at(i+1, k)
			e.prev = argmax
			e.v = max
		}
	}

	path = make([]int, m.nrows())
	var argmax int
	for j := 0; j < m.ncols; j++ {
		var max float64
		if v := m.at(m.nrows()-1, j).v; v > max {
			argmax = j
			max = v
		}
	}
	path[len(path)-1] = argmax
	for i := len(path) - 2; i >= 0; i-- {
		path[i] = m.at(i+1, path[i+1]).prev
	}
	return
}
