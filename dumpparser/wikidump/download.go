package wikidump

import (
	"fmt"
	"github.com/cheggaaa/pb"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
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
// Returns the local file path of the dump, derived from the URL.
//
// Logs its progress on the standard log if logProgress is true.
func Download(wikiname string, logProgress bool) (filepath string, err error) {
	return download(wikiname, ".", logProgress, http.DefaultClient)
}

func download(wikiname, directory string, logProgress bool,
	client *http.Client) (filepath string, err error) {

	logprint := nullLogger
	if logProgress {
		logprint = log.Printf
	}

	urlstr := fmt.Sprintf(
		"https://dumps.wikimedia.org/%s/latest/%s-latest-pages-articles.xml.bz2",
		wikiname, wikiname)
	resp, err := client.Get(urlstr)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("HTTP error %d for %s", resp.StatusCode, urlstr)
		return
	}

	u, err := url.Parse(urlstr)
	if err != nil {
		return
	}
	filepath = path.Base(u.Path)

	var out io.WriteCloser
	out, err = os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return
	}
	defer out.Close()

	logprint("downloading from %s to %s", urlstr, filepath)
	if logProgress && resp.ContentLength >= 0 {
		out = newPbWriter(out, resp.ContentLength)
	}
	_, err = io.Copy(out, resp.Body)
	logprint("download of %s done", urlstr)
	return
}
