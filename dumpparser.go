package main

import (
    "compress/bzip2"
    "encoding/xml"
    "io"
    "log"
    "os"
    "strings"
    "sync"
)

type page struct {
    title, text string
}

type redirect struct {
    title, target string
}

// Parse out a single page or redirect. Assumes a <page> start tag has just
// been consumed.
func parsePage(d *xml.Decoder, pages chan<- *page, redirs chan<- *redirect) {
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
                            redirs <- &redirect{title, attr.Value}
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
                    pages <- &page{title, text}
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
func getText(d *xml.Decoder) string {
    tok, _ := d.Token()
    text := string(tok.(xml.CharData))
    tok, _ = d.Token()
    _ = tok.(xml.EndElement)
    return text
}

func getPages(r io.Reader, pages chan<- *page, redirs chan<- *redirect) {
    d := xml.NewDecoder(r)

    defer close(pages)
    defer close(redirs)

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

func open(path string) (r io.ReadCloser, err error) {
    r, err = os.Open(path)
    if err == nil && strings.HasSuffix(path, ".bz2") {
        r = struct {
            io.Reader
            io.Closer
        }{bzip2.NewReader(r), r}
    }
    return
}

func main() {
    f, err := open(os.Args[1])
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()

    articles := make(chan *page)
    redirects := make(chan *redirect)
    go getPages(f, articles, redirects)

    var wg sync.WaitGroup
    wg.Add(2)
    done := make(chan struct{})
    go func() {
        wg.Wait()
        close(done)
    }()

    var narticles, nredirects int
loop:
    for {
        select {
        case _, ok := <-articles:
            if ok {
                narticles++
                if narticles % 10000 == 0 {
                    log.Printf("%d articles", narticles)
                }
            } else {
                articles = nil
                wg.Done()
            }
        case _, ok := <-redirects:
            if ok {
                nredirects++
                if nredirects % 10000 == 0 {
                    log.Printf("%d redirects", nredirects)
                }
            } else {
                redirects = nil
                wg.Done()
            }
        case <-done:
            break loop
        }
    }
    log.Printf("%d articles, %d redirects", narticles, nredirects)
}
