package wikidump

import (
	"bytes"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"testing"
)

func TestGetPages(t *testing.T) {
	input, err := os.Open("nlwiki-20140927-sample.xml")
	if err != nil {
		panic(err)
	}
	pages, redirs := make(chan *Page), make(chan *Redirect)
	go GetPages(input, pages, redirs)

	var titles []string
	var nredirs int
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		for p := range pages {
			titles = append(titles, p.Title)
			if strings.HasPrefix(p.Title, "Empty text") && p.Text != "" {
				t.Errorf("empty text not handled correctly, got %q", p.Text)
			}
		}
		wg.Done()
	}()
	go func() {
		for _ = range redirs {
			nredirs++
		}
		wg.Done()
	}()
	wg.Wait()

	if len(titles) != 22 {
		t.Errorf("expected 22 titles, got %d: %v", len(titles), titles)
	}
	if nredirs != 1 {
		t.Errorf("expected one redirect, got %d", nredirs)
	}
}

func BenchmarkGetPages(b *testing.B) {
	b.StopTimer()
	f, err := os.Open("nlwiki-20140927-sample.xml")
	if err != nil {
		panic(err)
	}
	content, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}
	f.Close()

	for i := 0; i < b.N; i++ {
		r := bytes.NewBuffer(content)
		pages, redirs := make(chan *Page), make(chan *Redirect)

		b.StartTimer()
		go GetPages(r, pages, redirs)
		go func() {
			for _ = range pages {
			}
		}()
		for _ = range redirs {
		}
		b.StopTimer()
	}
}
