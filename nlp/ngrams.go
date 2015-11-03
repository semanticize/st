package nlp

import (
	"hash"
	"io"
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
	positions := NGramsPos(tokens, minN, maxN)
	out := make([][]string, 0, len(positions))
	for _, pos := range positions {
		out = append(out, tokens[pos[0]:pos[1]])
	}
	return out
}

// Generate start/end positions of n-grams of length in [minN, maxN].
func NGramsPos(tokens []string, minN, maxN int) [][2]int {
	out := make([][2]int, 0, (maxN-minN+1)*len(tokens))
	for i := range tokens {
		for n := minN; n <= min(maxN, len(tokens)-i); n++ {
			out = append(out, [2]int{i, i + n})
		}
	}
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
