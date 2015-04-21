package main

import (
	"encoding/json"
	"github.com/semanticize/dumpparser/storage"
	"io/ioutil"
	"net/http"
)

type restHandler semanticizer

func (h *restHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	text, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	cands, err := semanticizer(*h).allCandidates(string(text))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	json.NewEncoder(w).Encode(cands)
}

func restServer(addr string, s *storage.Settings) error {
	return http.ListenAndServe(addr, nil)
}
