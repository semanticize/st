package wikidump

import (
	"os"
	"regexp"
	"sort"
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

		// Nor file and category links
		{"[[File:foo.png]] [[foo|see picture]]",
			"Foo", "see picture"},
		{"[[Category:Foos of the world]] [[foo]]", "Foo", "foo"},
	}

	for _, c := range cases {
		checkLink(t, onlyLink(c.text), c.target, c.anchor)
	}
}

type sortByAnchor []Link

func (s sortByAnchor) Len() int { return len(s) }

func (s sortByAnchor) Less(i, j int) bool {
	l := ([]Link)(s)
	return l[i].Anchor < l[j].Anchor
}

func (s sortByAnchor) Swap(i, j int) {
	l := ([]Link)(s)
	l[i], l[j] = l[j], l[i]
}

// Simulate the old API, except for the ordering.
func extractLinks(s string) []Link {
	links := make(sortByAnchor, 0)
	for k, v := range ExtractLinks(s) {
		for i := 0; i < v; i++ {
			links = append(links, k)
		}
	}
	sort.Sort(links)
	return ([]Link)(links)
}

func TestExtractLinks_multiple(t *testing.T) {
	// Expected links have to be sorted by anchor, UTF8-betically.
	cases := [][]string{
		// This construct appears in enwiki for chemical formulae etc.,
		// but also in nlwiki (and dewiki?) for more general compound nouns.
		{"[[Lithium|Li]][[Fluorine|F]]", "Fluorine", "F", "Lithium", "Li"},

		{"[[tera-|tera]][[becquerel]]s",
			"Becquerel", "becquerels", "Tera-", "tera"},

		// Newlines in links.
		{`[[Lord's
          prayer]]
          [[Dismissal
          (cricket)|dismissal]] [[Badass|Chuck
          Norris]]`,
			"Badass", "Chuck Norris",
			"Lord's prayer", "Lord's prayer",
			"Dismissal (cricket)", "dismissal"},
	}

	for _, c := range cases {
		links := extractLinks(c[0])
		if len(links) != (len(c)-1)/2 {
			t.Errorf("Wrong number of links %d in %q", len(links), c[0])
		}
		for i, l := range links {
			checkLink(t, l, c[i*2+1], c[i*2+2])
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
	for i := 0; i < b.N; i++ {
		for _, p := range pages {
			Cleanup(p)
		}
	}
}

func BenchmarkExtractLinks(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, p := range pages {
			ExtractLinks(p)
		}
	}
}
