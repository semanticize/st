package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"net"
	"net/http"
	"os"

	"github.com/semanticize/st/internal/storage"
	"github.com/semanticize/st/linking"
)

var infoTemplate = template.Must(template.New("info").Parse(`<html>
<head><title>Semanticizest</title></head>
  <body>
    <h1>Semanticizest</h1>
  	<p>
      Serving <code>{{.Dumpname}}</code>
      with maximum n-gram length {{.MaxNGram}}.
    </p>
    <p>Endpoints take data via POST requests and produce JSON:
      <ul>
        <li>
          <code>/all</code>
          gives all candidate entities occurring anywhere in a string
        </li>
        <li>
          <code>/bestpath</code> gives the entities according to a
          Viterbi algorithm</code>
        </li>
		<li>
          <code>/exactmatch</code>
          gives all candidate entities for a string (but not its substrings)
        </li>
      </ul>
    </p>
    <p>&copy; 2015 Netherlands eScience Center/University of Amsterdam.</p>
  </body>
</html>`))

func info(w http.ResponseWriter, settings *storage.Settings) {
	infoTemplate.Execute(w, settings)
}

type allHandler struct{ *linking.Semanticizer }

func (h allHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	serveEntities(w, req, h.All)
}

type stringHandler struct{ *linking.Semanticizer }

func (h stringHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	serveEntities(w, req, h.ExactMatch)
}

func serveEntities(w http.ResponseWriter, req *http.Request,
	method func(string) ([]linking.Entity, error)) {

	text, err := ioutil.ReadAll(req.Body)
	if len(text) == 0 {
		err = errors.New("received no data")
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cands, err := method(string(text))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if cands == nil {
		// Report "[]" to caller, not "null".
		cands = make([]linking.Entity, 0)
	}

	json.NewEncoder(w).Encode(cands)
}

// Determine actual port used by l and write it to path (followed by a newline).
//
// This is useful for random ports, as assigned when using port number 0.
func writePort(l net.Listener, path string) (err error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		return
	}
	fmt.Fprintln(f, port)
	return
}

func restServer(addr, portfile string, sem *linking.Semanticizer,
	s *storage.Settings) (err error) {

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		info(w, s)
	})
	http.Handle("/all", allHandler{sem})
	http.Handle("/exactmatch", stringHandler{sem})

	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer l.Close()

	if portfile != "" {
		if err = writePort(l, portfile); err != nil {
			return
		}
	}

	return http.Serve(l, nil)
}
