package wikidump

import (
	"bytes"
	"errors"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type mockTransport struct{}

func (t mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	path := "/scowiki/latest/scowiki-latest-pages-articles.xml.bz2"

	var msg string
	switch {
	case req.Method != "GET":
		msg = "not a GET request"
	case req.URL.Host != "dumps.wikimedia.org":
		msg = "wrong host"
	case req.URL.Path != path:
		msg = "wrong path"
	case req.Body != nil:
		msg = "non-nil Body"
	}
	if msg != "" {
		return nil, errors.New(msg)
	}

	content := []byte("all went well")
	resp := http.Response{
		Status:        "200 OK",
		StatusCode:    200,
		Body:          ioutil.NopCloser(bytes.NewBuffer(content)),
		ContentLength: int64(len(content)),
		Request:       req,
	}
	return &resp, nil
}

var (
	mockClient = http.Client{Transport: mockTransport{}}
)

func TestDownload(t *testing.T) {
	d, err := ioutil.TempDir("", "dumpparser-test")
	if err != nil {
		panic(err)
	}

	path := filepath.Join(d, "sco.bz2")
	path, err = download("scowiki", path, false, &mockClient)
	if err != nil {
		t.Error(err)
	} else {
		if base := filepath.Base(path); base != "sco.bz2" {
			t.Errorf("unexpected filename: %s", base)
		}
		if dir := filepath.Dir(path); dir != d {
			t.Errorf("downloaded to wrong directory %q (wanted %q)", dir, d)
		}
	}

	content, err := ioutil.ReadFile(path)
	if err != nil {
		t.Error(err)
	}
	if string(content) != "all went well" {
		t.Errorf("expected %q, got %q", "all went well", string(content))
	}

	err = os.Remove(path)
	if err != nil {
		panic(err)
	}
	err = os.Remove(d)
	if err != nil {
		panic(err)
	}
}

func TestParseDumpIndex(t *testing.T) {
	var err error
	check := func() {
		if err != nil {
			t.Fatal(err)
		}
	}

	f, err := os.Open("enwiki-20150304.html")
	check()
	defer f.Close()

	doc, err := goquery.NewDocumentFromReader(f)
	check()

	expected := []string {
		"/enwiki/20150304/enwiki-20150304-pages-articles1.xml-p000000010p000010000.bz2",
		"/enwiki/20150304/enwiki-20150304-pages-articles2.xml-p000010002p000024999.bz2",
		"/enwiki/20150304/enwiki-20150304-pages-articles3.xml-p000025001p000055000.bz2",
		"/enwiki/20150304/enwiki-20150304-pages-articles4.xml-p000055002p000104998.bz2",
		"/enwiki/20150304/enwiki-20150304-pages-articles5.xml-p000105002p000184999.bz2",
		"/enwiki/20150304/enwiki-20150304-pages-articles6.xml-p000185003p000305000.bz2",
		"/enwiki/20150304/enwiki-20150304-pages-articles7.xml-p000305002p000464996.bz2",
		"/enwiki/20150304/enwiki-20150304-pages-articles8.xml-p000465001p000665000.bz2",
		"/enwiki/20150304/enwiki-20150304-pages-articles9.xml-p000665001p000925000.bz2",
		"/enwiki/20150304/enwiki-20150304-pages-articles10.xml-p000925001p001325000.bz2",
		"/enwiki/20150304/enwiki-20150304-pages-articles11.xml-p001325001p001825000.bz2",
		"/enwiki/20150304/enwiki-20150304-pages-articles12.xml-p001825001p002425000.bz2",
		"/enwiki/20150304/enwiki-20150304-pages-articles13.xml-p002425002p003124997.bz2",
		"/enwiki/20150304/enwiki-20150304-pages-articles14.xml-p003125001p003924999.bz2",
		"/enwiki/20150304/enwiki-20150304-pages-articles15.xml-p003925001p004824998.bz2",
		"/enwiki/20150304/enwiki-20150304-pages-articles16.xml-p004825005p006024996.bz2",
		"/enwiki/20150304/enwiki-20150304-pages-articles17.xml-p006025001p007524997.bz2",
		"/enwiki/20150304/enwiki-20150304-pages-articles18.xml-p007525004p009225000.bz2",
		"/enwiki/20150304/enwiki-20150304-pages-articles19.xml-p009225002p011124997.bz2",
		"/enwiki/20150304/enwiki-20150304-pages-articles20.xml-p011125004p013324998.bz2",
		"/enwiki/20150304/enwiki-20150304-pages-articles21.xml-p013325003p015724999.bz2",
		"/enwiki/20150304/enwiki-20150304-pages-articles22.xml-p015725013p018225000.bz2",
		"/enwiki/20150304/enwiki-20150304-pages-articles23.xml-p018225004p020925000.bz2",
		"/enwiki/20150304/enwiki-20150304-pages-articles24.xml-p020925002p023724999.bz2",
		"/enwiki/20150304/enwiki-20150304-pages-articles25.xml-p023725001p026624997.bz2",
		"/enwiki/20150304/enwiki-20150304-pages-articles26.xml-p026625004p029624976.bz2",
		"/enwiki/20150304/enwiki-20150304-pages-articles27.xml-p029625017p045581259.bz2",

	}

	paths, err := parseDumpIndex(doc)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(paths, expected) {
		t.Errorf("expected %v, got %v", expected, paths)
	}
}
