package wikidump

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
)

type loggingWriter struct {
	w           io.WriteCloser
	done, total int64
	threshold   float32
}

func (w *loggingWriter) Write(p []byte) (n int, err error) {
	n, err = w.w.Write(p)
	w.done += int64(n)
	if float32(w.done) > w.threshold*float32(w.total) {
		log.Printf("%3d%% done (%d of %d)",
			int(100*w.threshold), w.done, w.total)
		w.threshold += .05
	}
	return
}

func (w loggingWriter) Close() error {
	return w.w.Close()
}

func nullLogger(string, ...interface{}) {
}

// Download database dump for wikiname (e.g., "en", "sco", "nds_nl") from
// WikiMedia.
//
// Returns the local file path of the dump, derived from the URL.
//
// Logs its progress on the standard log if logProgress is true.
func Download(wikiname string, logProgress bool) (filepath string, err error) {
	logprint := nullLogger
	if logProgress {
		logprint = log.Printf
	}

	urlstr := fmt.Sprintf(
		"https://dumps.wikimedia.org/%s/latest/%s-latest-pages-articles.xml.bz2",
		wikiname, wikiname)
	resp, err := http.Get(urlstr)
	defer resp.Body.Close()
	if err != nil {
		return
	} else if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("HTTP error %d for %s", resp.StatusCode, urlstr)
		return "", errors.New(msg)
	}

	u, err := url.Parse(urlstr)
	if err != nil {
		return
	}
	filepath = path.Base(u.Path)

	var out io.WriteCloser
	out, err = os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0666)
	defer out.Close()
	if err != nil {
		return
	}

	logprint("downloading from %s to %s", urlstr, filepath)
	if logProgress && resp.ContentLength >= 0 {
		out = &loggingWriter{
			w:     out,
			total: resp.ContentLength,
		}
	}
	_, err = io.Copy(out, resp.Body)
	logprint("download of %s done", urlstr)
	return
}
