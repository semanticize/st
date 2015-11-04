package nlp

import (
	"hash"
	"io"
	"sync"
)

func HashNGram(h hash.Hash32, tokens []string) uint32 {
	for _, token := range tokens {
		io.WriteString(h, token)
		h.Write([]byte("\x00"))
	}
	return h.Sum32()
}

// Generate n-grams of length in [minN, maxN].
func NGrams(tokens []string, minN, maxN int) [][]string {
	var positions [][2]int
	if pooled := posnPool.Get(); pooled != nil {
		positions = pooled.([][2]int)[:0]
	} else {
		positions = makePositions(tokens, minN, maxN)
	}

	positions = ngramsPos(tokens, minN, maxN, positions)

	out := make([][]string, 0, len(positions))
	for _, pos := range positions {
		out = append(out, tokens[pos[0]:pos[1]])
	}
	posnPool.Put(positions)
	return out
}

// Generate start/end positions of n-grams of length in [minN, maxN].
func NGramsPos(tokens []string, minN, maxN int) (positions [][2]int) {
	positions = makePositions(tokens, minN, maxN)
	return ngramsPos(tokens, minN, maxN, positions)
}

func makePositions(tokens []string, minN, maxN int) [][2]int {
	return make([][2]int, 0, (maxN-minN+1)*len(tokens))
}

func ngramsPos(tokens []string, minN, maxN int, positions [][2]int) [][2]int {
	positions = positions[:0]
	for i := range tokens {
		for n := minN; n <= min(maxN, len(tokens)-i); n++ {
			positions = append(positions, [2]int{i, i + n})
		}
	}
	return positions
}

var posnPool = sync.Pool{}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
