// Simplistic tokenizer for English/similar languages.

package nlp

import "regexp"

var (
	// Four-digit strings are typically years, and are often linked.
	numericRE = regexp.MustCompile(`^\d([\d\.\,]{4,})?$`)
	tokenRE   = regexp.MustCompile(`(\w|\b['\.,]\b)+`)
)

func Tokenize(s string) []string {
	out := make([]string, 0)
	for _, token := range tokenRE.FindAllString(s, -1) {
		if numericRE.MatchString(token) {
			token = "<NUM>"
		}
		out = append(out, token)
	}
	return out
}
