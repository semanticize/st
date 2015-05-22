package wikidump

import (
	"encoding/xml"
	"io"
)

// A Wikipedia page.
type Page struct {
	Title, Text string
}

// A Wikipedia redirect to Target.
type Redirect struct {
	Title, Target string
}

// Parse out a single page or redirect. Assumes a <page> start tag has just
// been consumed.
func parsePage(d *xml.Decoder, pages chan<- *Page, redirs chan<- *Redirect) {
	var mainNS bool
	var text, title string

	for {
		t, err := d.Token()
		if err != nil {
			panic(err)
		}

		switch tok := t.(type) {
		case xml.StartElement:
			switch tok.Name.Local {
			case "ns":
				ns := getText(d)
				mainNS = string(ns) == "0"
			case "redirect":
				if mainNS {
					for _, attr := range tok.Attr {
						if attr.Name.Local == "title" {
							redirs <- &Redirect{title, attr.Value}
							return
						}
					}
				}
			case "text":
				if mainNS {
					text = getText(d)
				}
			case "title":
				// No check for mainNS because the <ns> comes *after* the
				// title. Let's hope titles are short.
				title = getText(d)
			}

		case xml.EndElement:
			if tok.Name.Local == "page" {
				if mainNS {
					pages <- &Page{title, text}
				}
				return
			}
		}
	}
	panic("not reached")
}

// Parse text out of an element. Assumes element has the form
// <foo>some text</foo> and the start tag has already been consumed.
// Consumes the whole element.
func getText(d *xml.Decoder) (text string) {
	tok, _ := d.Token()
	switch tok := tok.(type) {
	case xml.CharData:
		text = string(tok)
		nexttok, _ := d.Token()
		_ = nexttok.(xml.EndElement)
	case xml.EndElement:
		text = ""
	}
	return text
}

// Get pages and redirects from wikidump r. Only retrieves the pages in the
// main namespace.
//
// Doesn't close either of the channels passed to it to support dumps
// consisting of multiple parts.
//
// XXX needs cleaner error handling. Currently panics.
func GetPages(r io.Reader, pages chan<- *Page, redirs chan<- *Redirect) {
	d := xml.NewDecoder(r)

	for {
		t, err := d.Token()
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}

		tok, ok := t.(xml.StartElement)
		if ok {
			if tok.Name.Local == "page" {
				parsePage(d, pages, redirs)
			}
		}
	}
}
