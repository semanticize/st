// Simplistic tokenizer for English/similar languages.

package nlp

import "regexp"

var (
	// Four-digit strings are typically years, and are often linked.
	numericRE = regexp.MustCompile(`^\d([\d\.\,]{4,})?$`)
	tokenRE   = regexp.MustCompile(`(\w|\b['\.,]\b)+`)
)

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
