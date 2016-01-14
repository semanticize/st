// Entity linking package.
package linking

import (
	"database/sql"
	"hash/fnv"
	"math"

	"github.com/semanticize/st/countmin"
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

// Get candidates for n-gram tokens from the database. offset and end index
// into the original string and are stored on the return values.
func (sem Semanticizer) candidates(tokens []string, offset, end int) (cands []Entity, err error) {
	h := nlp.HashNGram(fnv.New32(), tokens)
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
		c.NGramCount = sem.ngramcount.GetMeanNGram(tokens)
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
	return sem.candidates(tokens, 0, len(tokens))
}

// Returns candidates in sorted order.
func (sem Semanticizer) allFromTokens(tokens []string,
	tokpos [][]int) (cands []Entity, err error) {

	for _, hpos := range nlp.NGramsPos(tokens, 1, int(sem.maxNGram)) {
		start, end := hpos[0], hpos[1]
		ngram := tokens[start:end]
		start, end = tokpos[start][0], tokpos[end-1][1]

		add, err := sem.candidates(ngram, start, end)
		if err != nil {
			break
		}
		cands = append(cands, add...)
	}
	return
}

// Reports entity mentions in s according to a best path (Viterbi) algorithm.
//
// This gets rid of overlapping candidates.
func (sem Semanticizer) BestPath(s string) ([]Entity, error) {
	tokens, tokpos := nlp.TokenizePos(s)
	if len(tokens) == 0 {
		return nil, nil
	}
	all, err := sem.allFromTokens(tokens, tokpos)
	if err != nil {
		return nil, err
	}
	return bestPath(all), nil
}

func bestPath(cands []Entity) []Entity {
	// TODO sink state
	h := hmm{cands, make(map[int]int), make(map[int]map[int]float64)}
	var endall int
	for i := range cands {
		start := cands[i].Offset
		end := start + cands[i].Length
		if end > endall {
			endall = end
		}
		h.nStart[start]++
		for j := start; j < end; j++ {
			if h.obsProb[j] == nil {
				h.obsProb[j] = make(map[int]float64)
			}
			h.obsProb[j][i]++
		}
	}
	// Normalize observation probabilities.
	for _, probs := range h.obsProb {
		var sum float64
		for _, count := range probs {
			sum += count
		}
		sum = math.Log(sum)
		for c, count := range probs {
			probs[c] = math.Log(count) - sum
		}
	}

	path := h.viterbi(endall)
	keep := make(map[*Entity]bool)
	for _, i := range path {
		if i != h.sinkState() {
			keep[&cands[i]] = true
		}
	}
	cands = make([]Entity, 0)
	for k := range keep {
		cands = append(cands, *k)
	}

	return cands
}

type hmm struct {
	cands []Entity

	// Maps position to number of candidates that start there.
	nStart map[int]int

	// Maps (position, candidate index) to log-probability
	obsProb map[int]map[int]float64
}

func (h hmm) sinkState() int {
	return len(h.cands)
}

const eps = math.SmallestNonzeroFloat64

// Log-probability of transition from state i to j at position p.
func (h hmm) transProb(i, j, pos int) (logP float64) {
	start, end := -1, -1
	sink := h.sinkState()
	if i < sink {
		start = h.cands[i].Offset
		end = start + h.cands[i].Length
	}

	// Probability of an entity to start at pos.
	// TODO use Commonness here.
	startProb := func() float64 {
		if count := h.nStart[pos]; count != 0 {
			return -math.Log(float64(count))
		}
		return math.Inf(-1)
	}

	logP = eps
	switch {
	case i == sink && j == sink:
		// Only stay in the sink state if we can't enter a mention.
		if h.nStart[pos] == 0 {
			logP = 0
		}
	case i == sink:
		logP = startProb()
	case j == sink:
		// Only enter sink state when we're at the end of entity i.
		if pos == end {
			logP = startProb()
		}
	default:
		if pos == end {
			logP = startProb()
		} else if i == j {
			logP = 0
		}
	}

	return
}

func (h hmm) viterbi(obsLength int) (path []int) {
	t := newTable(obsLength+1, h.sinkState()+1)

	// Straight from Jurafsky and Martin, p. 181.
	for j := 0; j < t.ncols-1; j++ {
		t.at(0, j).v = eps
	}
	sink := h.sinkState()
	t.at(0, sink).v = 0

	var argmax int
	for pos := 0; pos < obsLength; pos++ {
		for k := 0; k < t.ncols; k++ {
			argmax = -1
			max := math.Inf(-1)
			obsProb, ok := h.obsProb[pos][k]
			if !ok && k != sink {
				obsProb = math.Inf(-1)
			}
			for j, e := range t.row(pos) {
				if v := e.v + obsProb + h.transProb(j, k, pos); v > max {
					argmax = j
					max = v
				}
			}
			e := t.at(pos+1, k)
			e.prev = argmax
			e.v = max
		}
	}

	path = make([]int, obsLength)
	path[obsLength-1] = argmax
	for i := len(path) - 2; i >= 0; i-- {
		path[i] = t.at(i+1, path[i+1]).prev
		if path[i] == -1 {
			panic("-1 in path")
		}
	}
	return
}

// Dynamic programming table: row-major matrix with backpointers.
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
