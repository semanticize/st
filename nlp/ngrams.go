package nlp

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
