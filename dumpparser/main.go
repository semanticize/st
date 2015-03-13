package main

import (
    "compress/bzip2"
    "flag"
    "fmt"
    "github.com/semanticize/dumpparser/wikidump"
    "io"
    "log"
    "os"
    "path/filepath"
    "sync"
)

func open(path string) (r io.ReadCloser, err error) {
    r, err = os.Open(path)
    if err == nil && filepath.Ext(path) == ".bz2" {
        r = struct {
            io.Reader
            io.Closer
        }{bzip2.NewReader(r), r}
    }
    return
}

var download = flag.String("download", "",
                           "download Wikipedia dump (e.g., 'enwiki')")

func main() {
    var err error
    check := func() {
        if err != nil {
            log.Fatal(err)
        }
    }

    flag.Parse()
    args := flag.Args()

    var filepath string
    if *download != "" {
        if len(args) != 1 {
            fmt.Fprintf(os.Stderr,
                        "usage: %s -download=wikiname model.db\n", os.Args[0])
            os.Exit(1)
        }
        inputpath, err = wikidump.Download(*download, true)
        check()
    } else {
        if len(args) != 2 {
            fmt.Fprintf(os.Stderr, "usage: %s wikidump model.db\n", os.Args[0])
            os.Exit(1)
        }
        filepath = args[0]
    }

    f, err := open(filepath)
    check()
    defer f.Close()

    articles := make(chan *wikidump.Page)
    redirects := make(chan *wikidump.Redirect)
    go wikidump.GetPages(f, articles, redirects)

    links := make(chan *wikidump.Link)
    go func() {
        for a := range articles {
            text := wikidump.Cleanup(a.Text)
            for _, lnk := range wikidump.ExtractLinks(text) {
                links <- &lnk
            }
        }
        close(links)
    }()

    var wg sync.WaitGroup
    wg.Add(2)

    done := make(chan struct{})
    go func() {
        wg.Wait()
        close(done)
    }()

    for {
        select {
        case lnk, ok := <-links:
            if ok {
                fmt.Printf("link: %q → %q\n", lnk.Anchor, lnk.Target)
            } else {
                links = nil
                wg.Done()
            }
        case redir, ok := <-redirects:
            if ok {
                fmt.Printf("redirect: %q → %q\n", redir.Title, redir.Target)
            } else {
                redirects = nil
                wg.Done()
            }
        case <-done:
            return
        }
    }
}
