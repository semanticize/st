package wikidump

import (
	"errors"
	"fmt"
	"github.com/cheggaaa/pb"
	"github.com/PuerkitoBio/goquery"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
)

func nullLogger(string, ...interface{}) {
}

// Writer with progressbar.
type pbWriter struct {
	w   io.WriteCloser
	bar *pb.ProgressBar
}

func newPbWriter(w io.WriteCloser, total int64) *pbWriter {
	pbw := &pbWriter{w, pb.New64(total).SetUnits(pb.U_BYTES)}
	pbw.bar.Start()
	return pbw
}

func (w *pbWriter) Close() error {
	w.bar.Finish()
	return w.w.Close()
}

func (w *pbWriter) Write(p []byte) (n int, err error) {
	n, err = w.w.Write(p)
	w.bar.Add(n)
	return
}

// Download database dump for wikiname (e.g., "en", "sco", "nds_nl") from
// WikiMedia.
//
// If path is not nil, writes the dump to path. Else, derives an appropriate
// path from the URL and returns that.
//
// Logs its progress on the standard log if logProgress is true.
func Download(wikiname string, logProgress bool) (string, error) {
	return download(wikiname, logProgress, http.DefaultClient)
}

func download(wikiname string, logProgress bool,
	client *http.Client) (string, error) {

	var err error

	logprint := nullLogger
	if logProgress {
		logprint = log.Printf
	}

	urlstr := fmt.Sprintf(
		"https://dumps.wikimedia.org/%s/latest/%s-latest-pages-articles.xml.bz2",
		wikiname, wikiname)
	resp, err := client.Get(urlstr)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error %d for %s", resp.StatusCode, urlstr)
	}

	u, err := url.Parse(urlstr)
	if err != nil {
		return "", err
	}
	filepath := path.Base(u.Path)

	var out io.WriteCloser
	out, err = os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return "", err
	}
	defer out.Close()

	logprint("downloading from %s to %s", urlstr, filepath)
	if logProgress && resp.ContentLength >= 0 {
		out = newPbWriter(out, resp.ContentLength)
	}
	_, err = io.Copy(out, resp.Body)
	logprint("download of %s done", urlstr)
	return filepath, nil
}

var partPattern = regexp.MustCompile(`pages-articles\d+\.xml.*\.bz2`)

// Download a dump in parts.
//
// wikiNameVersion should be, e.g., "enwiki/20150304".
//
// Use "latest" for version to get the latest dump.
func DownloadParts(wikiNameVersion string) (filenames []string, err error) {
	index := fmt.Sprintf("https://dumps.wikimedia.org/%s/", wikiNameVersion)
	doc, err := goquery.NewDocument(index)
	if err != nil {
		return
	}

	remotePaths, err := parseDumpIndex(doc)
	if err != nil {
		return
	}

	// We don't try to download in parallel so as not to make WikiMedia upset.
	for _, p := range remotePaths {
		u := (&url.URL{
			Scheme: "https",
			Host: "dumps.wikimedia.org",
			Path: p,
		}).String()

		localName := path.Base(p)

		resp, err := http.Get(u)
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("HTTP error %d for %s", resp.StatusCode, u)
			continue
		}

		var out io.WriteCloser
		out, err = os.OpenFile(localName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
		if err != nil {
			continue
		}
		defer out.Close()

		log.Printf("downloading %s", u)
		out = newPbWriter(out, resp.ContentLength)
		_, err = io.Copy(out, resp.Body)

		filenames = append(filenames, localName)
	}
	return
}

const partlink = `a[href *= "pages-articles"]`

// Find not yet recombined dump parts, so we can process these in parallel.
func parseDumpIndex(doc *goquery.Document) (remotePaths []string, err error) {
	doc.Find(partlink).Each(func(i int, s *goquery.Selection) {
		for _, node := range s.Nodes {
			for _, attr := range node.Attr {
				if partPattern.MatchString(attr.Val) {
					if attr.Val[0] != '/' {
						// Laziness on my part.
						err = errors.New("cannot handle relative URL")
						return
					}
					remotePaths = append(remotePaths, attr.Val)
				}
			}
		}
	})
	return
}
