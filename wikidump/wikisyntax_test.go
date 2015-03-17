package wikidump

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func assertStringEq(t *testing.T, a, b string) {
	if a != b {
		t.Errorf("%q != %q", a, b)
	}
}

func TestCleanup(t *testing.T) {
	in := "|}Hello,<ref group=\"note\">1</rf> world{{math|bla{{?}}}}!{{bla"
	assertStringEq(t, Cleanup(in), "Hello, world!")
}

var ws = regexp.MustCompile(`\s+`)

func checkLink(t *testing.T, got Link, target, anchor string) {
	// Don't care about whitespace in the anchor...
	gotAnchor := ws.ReplaceAllString(strings.TrimSpace(got.Anchor), " ")
	if gotAnchor != anchor {
		t.Errorf("expected anchor %q, got %q", anchor, gotAnchor)
	}

	// ... but the target should be normalized.
	if got.Target != target {
		t.Errorf("expected target %q, got %q", target, got.Target)
	}
}

func TestExtractLinks_single(t *testing.T) {
	onlyLink := func(text string) Link {
		links := ExtractLinks(text)
		if len(links) != 1 {
			t.Errorf("expected one link, got at least %d", len(links))
		}
		for link, count := range links {
			if count != 1 {
				t.Errorf("expected one link, got %d", count)
			}
			return link
		}
		panic("no links")
	}

	cases := []struct {
		text, target, anchor string
	}{
		{"[[foo|bar]]", "Foo", "bar"},
		{"[[foo]]", "Foo", "foo"},
		{"[[File:picture!]] [[foo]]", "Foo", "foo"},
		{"[[foo]]bar.", "Foo", "foobar"},
		{"[[baz|foobar]];", "Baz", "foobar"},
		{"[[baz#quux]]", "Baz", "baz#quux"},
		{"[[FOO_BAR|foo bar]]", "FOO BAR", "foo bar"},

		{"[[C. Stephen Evans | Evans, C. Stephen]]",
			"C. Stephen Evans", "Evans, C. Stephen"},

		// Links like these commonly occur in nlwiki (and presumably dewiki
		// and other compounding languages):
		{"foo[[baz|bar]]", "Baz", "foobar"},
		{"before[[_target _page_ #\nsection|inside]]after",
			"Target page", "beforeinsideafter"},

		// MediaWiki only considers alphabetic characters outside [[]] part
		// of the anchor.
		{"foo-[[bar]]", "Bar", "bar"},
		{"[[bar]]/baz", "Bar", "bar"},

		// XXX The following are broken. They do occur in the wild, e.g.,
		// -18[[Celsius|°C]] and 700[[Megabyte|MB]]-cd (found in nlwiki dump).
		//{"[[bar]]0", "Bar", "bar"},
		//{"[[bar]]_", "Bar", "bar"},

		// We're not interested in section links
		{"[[#Some section|elsewhere]] [[other_article]]",
			"Other article", "other_article"},
	}

	for _, c := range cases {
		checkLink(t, onlyLink(c.text), c.target, c.anchor)
	}
}

// Simulate the old API.
func extractLinks(s string) []Link {
	links := make([]Link, 0)
	for k, v := range ExtractLinks(s) {
		for i := 0; i < v; i++ {
			links = append(links, k)
		}
	}
	return links
}

func TestExtractLinks_multiple(t *testing.T) {
	cases := [][]string{
		// This construct appears in enwiki for chemical formulae etc.,
		// but also in nlwiki (and dewiki?) for more general compound nouns.
		{"[[Lithium|Li]][[Fluorine|F]]", "Lithium", "Li", "Fluorine", "F"},

		{"[[tera-|tera]][[becquerel]]s",
			"Tera-", "tera", "Becquerel", "becquerels"},

		// Newlines in links.
		{`[[Lord's
          prayer]]
          [[Dismissal
          (cricket)|dismissal]] [[Badass|Chuck
          Norris]]`,
			"Lord's prayer", "Lord's prayer",
			"Dismissal (cricket)", "dismissal",
			"Badass", "Chuck Norris"},
	}

	for _, c := range cases {
		links := extractLinks(c[0])
		if len(links) != (len(c)-1)/2 {
			t.Errorf("Wrong number of links %d in %q", len(links), c[0])
		}
		for i := range links {
			checkLink(t, links[i], c[i*2+1], c[i*2+2])
		}
	}
}

func getPages() []string {
	f, err := os.Open("nlwiki-20140927-sample.xml")
	if err != nil {
		panic(err)
	}
	pc, rc := make(chan *Page), make(chan *Redirect)
	go GetPages(f, pc, rc)
	go func() {
		for _ = range rc {
		}
	}()

	pages := make([]string, 0)
	for p := range pc {
		pages = append(pages, p.Text)
	}
	return pages
}

var pages = getPages()

func BenchmarkCleanup(b *testing.B) {
	for i := 0; i < 5; i++ {
		for _, p := range pages {
			Cleanup(p)
		}
	}
}

func BenchmarkExtractLinks(b *testing.B) {
	for i := 0; i < 5; i++ {
		for _, p := range pages {
			ExtractLinks(p)
		}
	}
}
