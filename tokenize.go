// Simplistic tokenizer for English/similar languages.

package dumpparser

import (
    "bufio"
    "fmt"
    "os"
    "regexp"
)

var (
    numericRE = regexp.MustCompile(`\d[\d\.]+`)
    tokenRE = regexp.MustCompile(`(\w|\b['\.]\b)+`)
)

func tokenize(s string) []string {
    out := make([]string, 0)
    for _, token := range tokenRE.FindAllString(s, -1) {
        if numericRE.MatchString(token) {
            token = "<NUM>"
        }
        out = append(out, token)
    }
    return out
}
