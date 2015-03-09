package main

import (
    "compress/bzip2"
    "fmt"
    "github.com/semanticize/dumpparser/wikidump"
    "io"
    "log"
    "os"
    "strings"
    "sync"
)

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
    if len(os.Args) != 2 {
        fmt.Fprintf(os.Stderr, "usage: %s wikidump\n", os.Args[0])
        os.Exit(1)
    }

    f, err := open(os.Args[1])
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()

    articles := make(chan *wikidump.Page)
    redirects := make(chan *wikidump.Redirect)
    go wikidump.GetPages(f, articles, redirects)

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
