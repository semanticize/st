package wikidump

import (
	"os"
	"sync"
	"testing"
)

func assertIntEq(t *testing.T, a, b int) {
	if a != b {
		t.Errorf("%d != %d", a, b)
	}
}

func TestGetPages(t *testing.T) {
	input, err := os.Open("nlwiki-20140927-sample.xml")
	if err != nil {
		panic(err)
	}
	pages := make(chan *Page)
	redirs := make(chan *Redirect)
	go GetPages(input, pages, redirs)

	var nredirs, npages int
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		for _ = range pages {
			npages++
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

	assertIntEq(t, npages, 19)
	assertIntEq(t, nredirs, 1)
}
