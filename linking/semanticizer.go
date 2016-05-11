// Entity linking package.
package linking

import (
	"database/sql"

	"github.com/semanticize/st/hash"
	"github.com/semanticize/st/hash/countmin"
	"github.com/semanticize/st/internal/storage"
	"github.com/semanticize/st/nlp"
)

type Semanticizer struct {
	db         *sql.DB
	ngramcount *countmin.Sketch
	maxNGram   uint
	allQuery   *sql.Stmt
}

// Load a semanticizer (entity linker) from modelpath.
//
// Also returns a settings object that represents the dumpparser settings used
// to generate the model.
func Load(modelpath string) (sem *Semanticizer,
	settings *storage.Settings, err error) {

	var db *sql.DB
	defer func() {
		if db != nil && err != nil {
			db.Close()
		}
	}()

	db, settings, err = storage.LoadModel(modelpath)
	if err != nil {
		return
	}
	ngramcount, err := storage.LoadCM(db)
	if err != nil {
		return
	}
	allq, err := prepareAllQuery(db)
	if err != nil {
		return
	}

	sem = &Semanticizer{db: db, ngramcount: ngramcount,
		maxNGram: settings.MaxNGram, allQuery: allq}
	return
}

// Represents a mention of an entity.
type Entity struct {
	// Title of target Wikipedia article.
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

func prepareAllQuery(db *sql.DB) (*sql.Stmt, error) {
	return db.Prepare(
		`select (select title from titles where id = targetid), count
		 from linkstats where ngramhash = ?`)
}

// Get candidates for hash value h from the database. offset and end index
// into the original string and are stored on the return values.
func (sem Semanticizer) candidates(h uint32, offset, end int) (cands []Entity, err error) {
	rows, err := sem.allQuery.Query(h)
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
		cands = append(cands, Entity{
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
		c.LinkCount = totalLinkCount
	}
	return
}

// Get all candidate entity mentions in the string s.
func (sem Semanticizer) All(s string) (cands []Entity, err error) {
	tokens, tokpos := nlp.TokenizePos(s)
	return sem.allFromTokens(tokens, tokpos)
}

// Get all candidate entity mentions for the string s.
//
// A candidate entity's anchor text must be exactly s.
func (sem Semanticizer) ExactMatch(s string) (cands []Entity, err error) {
	tokens := nlp.Tokenize(s)
	h := hash.NGrams(tokens, len(tokens), len(tokens))[0]
	return sem.candidates(h, 0, len(tokens))
}

// Returns candidates in sorted order.
func (sem Semanticizer) allFromTokens(tokens []string,
	tokpos [][]int) (cands []Entity, err error) {

	for _, hpos := range hash.NGramsPos(tokens, int(sem.maxNGram)) {
		start, end := hpos.Start, hpos.End-1
		start, end = tokpos[start][0], tokpos[end][1]

		add, err := sem.candidates(hpos.Hash, start, end)
		if err != nil {
			break
		}
		cands = append(cands, add...)
	}
	return
}
