package wikidump

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
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

	path, err := download("scowiki", d, false, &mockClient)
	if err != nil {
		t.Error(err)
	} else {
		base := filepath.Base(path)
		if base != "scowiki-latest-pages-articles.xml.bz2" {
			t.Errorf("unexpected filename: %s", base)
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
