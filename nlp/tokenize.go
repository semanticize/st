package nlp

import "regexp"

var (
	// Four-digit strings are typically years, and are often linked.
	numericRE = regexp.MustCompile(`^\d([\d\.\,]{4,})?$`)
	tokenRE   = regexp.MustCompile(`([A-Za-z]*\p{Sc}|(\w|\b['\.,]\b)+)`)
)

// Simple tokenizer for English/similar languages.
//
// Does some token normalization.
func Tokenize(s string) (tokens []string) {
	matches := tokenRE.FindAllString(s, -1)
	tokens = make([]string, 0, len(matches))
	for _, token := range matches {
		if numericRE.MatchString(token) {
			token = "<NUM>"
		}
		tokens = append(tokens, token)
	}
	return
}

// Equivalent to Tokenize, but also returns offsets into the input string.
//
// Token positions are returned as a slice of two-element slices
// []int{start, end}, where end is exclusive.
//
// Because tokens are normalized, s[pos[i][0]:pos[i][1]] need not match
// tokens[i].
func TokenizePos(s string) (tokens []string, pos [][]int) {
	pos = tokenRE.FindAllStringIndex(s, -1)
	tokens = make([]string, 0, len(pos))
	for _, p := range pos {
		start, end := p[0], p[1]
		text := s[start:end]
		if numericRE.MatchString(text) {
			text = "<NUM>"
		}
		tokens = append(tokens, text)
	}
	return
}
