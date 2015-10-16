package wikidump

import (
	"bytes"
	"golang.org/x/text/unicode/norm"
	"html"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	starttag = regexp.MustCompile(`^<[a-z][^>]*>`)
	endtag   = regexp.MustCompile(`^</[a-z]+>`)
)

// Get rid of tables, template calls, quasi-XML. Throws away their content.
//
// Assumes tables, templates and tags are properly nested, except for spurious
// end-of-{table,template,element} tags, which are ignored.
func Cleanup(s string) string {
	var depth int
	output := bytes.NewBuffer(make([]byte, 0, len(s)))

	for {
		next := strings.IndexAny(s, "{}|<")
		if next == -1 {
			if depth == 0 {
				output.WriteString(s)
			}
			break
		}

		if depth == 0 {
			output.WriteString(s[:next])
		}
		s = s[next:]

		var skip int
		if strings.HasPrefix(s, "{{") || strings.HasPrefix(s, "{|") {
			depth++
			skip = 2
		} else if strings.HasPrefix(s, "|}") || strings.HasPrefix(s, "}}") {
			if depth > 0 {
				depth--
			}
			skip = 2
		} else if s[0] != '<' {
			// This case prevents regexp matching for a 20% speedup.
			skip = 1
		} else if span := starttag.FindStringIndex(s); span != nil {
			depth++
			skip = span[1]
		} else if span := endtag.FindStringIndex(s); span != nil {
			if depth > 0 {
				depth--
			}
			skip = span[1]
		} else {
			skip = 1
		}

		// If skip == 1, we didn't find a tag/table marker.
		if skip == 1 && depth == 0 && len(s) > 0 {
			output.WriteByte(s[0])
		}
		s = s[skip:]
	}
	return norm.NFC.String(html.UnescapeString(output.String()))
}

// A link to the article Target with anchor text Anchor.
type Link struct {
	Anchor, Target string
}

func normSpace(s string) string {
	parts := strings.FieldsFunc(s, func(c rune) bool {
		return c == '_' || unicode.IsSpace(c)
	})
	return strings.Join(parts, " ")
}

var linkRE = regexp.MustCompile(`\w*\[\[[^]]+\]\]\w*`)

// Extract all the wikilinks from s. Returns a frequency table.
func ExtractLinks(s string) map[Link]int {
	freq := make(map[Link]int)

	for _, candidate := range linkRE.FindAllStringSubmatch(s, -1) {
		// Parse complex links like "foo[[bar|baz]]quux". We used to do this
		// with capturing groups in linkRE, but those are *slow*.
		text := candidate[0]
		openbrack := strings.IndexByte(text, '[')
		closebrack := strings.LastIndex(text, "]")
		before := text[:openbrack]
		after := text[closebrack+1:]
		mid := text[openbrack+2:closebrack-1]

		var target, anchor string
		if pipe := strings.IndexByte(mid, '|'); pipe != -1 {
			target, anchor = mid[:pipe], mid[pipe+1:]
		} else {
			target = mid
			anchor = mid
		}

		// If the anchor contains a colon, assume it's a file or category link.
		// XXX Maybe skip matches for `:\s`? Proper solution would parse the
		// dump to find non-main namespace prefixes.
		if strings.IndexByte(target, ':') != -1 {
			continue
		}

		// Remove section links.
		if hash := strings.IndexByte(target, '#'); hash == 0 {
			continue
		} else if hash != -1 {
			target = target[:hash]
		}

		// Normalize to the format used in <redirect> elements:
		// uppercase first character, spaces instead of underscores.
		target = normSpace(target)
		first, size := utf8.DecodeRuneInString(target)
		// XXX Upper case or title case? Should look up the difference...
		if unicode.IsLower(first) {
			target = string(unicode.ToUpper(first)) + target[size:]
		}

		anchor = before + anchor + after
		freq[Link{anchor, target}]++
	}
	return freq
}
