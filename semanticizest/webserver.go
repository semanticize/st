package main

import (
	"encoding/json"
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
        <li><code>/all</code> gives all candidate entities
        </li>
      </ul>
    </p>
    <p>&copy; 2015 Netherlands eScience Center/University of Amsterdam.</p>
  </body>
</html>`))

func info(w http.ResponseWriter, settings *storage.Settings) {
	//io.WriteString(w, settings.Dumpname)
	infoTemplate.Execute(w, settings)
}

type restHandler struct {
	*semanticizer
}

func (h restHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	text, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	cands, err := h.semanticizer.allCandidates(string(text))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	json.NewEncoder(w).Encode(cands)
}

func restServer(addr string, sem *semanticizer, s *storage.Settings) error {
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		info(w, s)
	})
	http.Handle("/all", restHandler{sem})
	return http.ListenAndServe(addr, nil)
}
