package main

import (
    "bytes"
    "fmt"
    "html"
    "regexp"
)

var (
    special = regexp.MustCompile(`{{|{\||\|}|}}|<[a-z][a-z0-9 "=]*/?>|</[a-z]+>`)
    starttag = regexp.MustCompile("<[a-z].*>")
    endtag = regexp.MustCompile("</[a-z]+>")
)

// Get rid of tables, template calls, quasi-XML. Throws away their content.
//
// Assumes tables, templates and tags are properly nested, except for spurious
// end-of-{table,template,element} tags, which are ignored.
func cleanup(s string) string {
    var depth int
    output := bytes.NewBuffer(make([]byte, 0, len(s)))

    for {
        next := special.FindStringIndex(s)
        if next == nil {
            if depth == 0 {
                output.WriteString(s)
            }
            break
        }
        i, j := next[0], next[1]

        if depth == 0 {
            output.WriteString(s[:i])
        }

        tag := s[i:j]
        switch {
        case tag == "{{":
            depth++
        case tag == "{|":
            depth++
        case starttag.MatchString(tag):
            depth++
        case tag == "}}":
            fallthrough
        case tag == "|}":
            fallthrough
        case endtag.MatchString(tag):
            if depth > 0 {
                depth--
            }
        }

        s = s[j:]
    }
    return html.UnescapeString(output.String())
}

func main() {
    fmt.Println(cleanup("|}Hello,<ref group=\"note\">1</rf> world{{math|bla{{?}}}}!{{bla"))
}
