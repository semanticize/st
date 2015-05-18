package main

import (
	"encoding/json"
	"errors"
	"github.com/semanticize/st/linking"
	"github.com/semanticize/st/storage"
	"html/template"
	"io/ioutil"
	"net/http"
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
        <li><code>/all</code> gives all candidate entities</li>
        <li>
          <code>/bestpath</code> gives the entities according to a
          Viterbi algorithm</code>
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

type bestPathHandler struct{ *linking.Semanticizer }

func (h bestPathHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	serveEntities(w, req, h.BestPath)
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

func restServer(addr string, sem *linking.Semanticizer, s *storage.Settings) error {
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		info(w, s)
	})
	http.Handle("/all", allHandler{sem})
	http.Handle("/bestpath", bestPathHandler{sem})
	return http.ListenAndServe(addr, nil)
}
