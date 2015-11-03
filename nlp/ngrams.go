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

// Generate n-grams.
func NGrams(tokens []string, minN, maxN int) [][]string {
	out := make([][]string, 0)
	for i := range tokens {
		for n := minN; n <= min(maxN, len(tokens)-i); n++ {
			out = append(out, tokens[i:i+n])
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
