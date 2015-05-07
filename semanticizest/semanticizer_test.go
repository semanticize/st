package main

import (
	"encoding/json"
	"github.com/semanticize/st/hash"
	"github.com/semanticize/st/hash/countmin"
	"github.com/semanticize/st/storage"
	"reflect"
	"testing"
)

func TestAllCandidates(t *testing.T) {
	cm, _ := countmin.New(10, 4)
	db, _ := storage.MakeDB(":memory:", true, &storage.Settings{MaxNGram: 2})
	sem := semanticizer{db, cm, 2}
	_ = sem

	for _, h := range hash.NGrams([]string{"Hello", "world"}, 2, 2) {
		_, err := db.Exec(`insert into linkstats values (?, 0, 1)`, h)
		if err == nil {
			_, err = db.Exec(`insert into titles values (0, "dmr")`)
		}
		if err != nil {
			panic(err)
		}
	}

	all, err := sem.allCandidates("Hello world")
	if err != nil {
		t.Error(err)
	}
	if len(all) != 1 {
		t.Errorf("expected one candidate, got %v", all)
	} else if tgt := all[0].Target; tgt != "dmr" {
		t.Errorf(`expected target "dmr", got %q`, tgt)
	}
}

func TestBestPath(t *testing.T) {
	cands := []candidate{
		{Target: "foo", Offset: 4, Length: 6, Senseprob: .8},
		{Target: "bar", Offset: 3, Length: 7, Senseprob: .9},
		{Target: "baz", Offset: 1, Length: 2, Senseprob: .1},
	}
	best := bestPath(cands)
	if len(best) != 2 {
		t.Errorf("too many entities in path: %d (wanted 2)", len(best))
	}
	for _, e := range best {
		if e.Target != "foo" && e.Target != "baz" {
			t.Errorf("unexpected entity %q in best path", e.Target)
		}
	}
}

func TestCandidateJSON(t *testing.T) {
	in := candidate{"Wikipedia", 4, 10, .9, 0.0115, 0, 9}
	enc, _ := json.Marshal(in)

	var got candidate
	json.Unmarshal(enc, &got)

	if !reflect.DeepEqual(in, got) {
		t.Errorf("marshalled %v, got %v", in, got)
	}

	enc = []byte(
		`{"offset": 0,"target":"Wikipedia", "commonness":0.9,"ngramcount": 4 ,
		  "linkcount": 10, "length": 9,"senseprob":0.0115}`)
	err := json.Unmarshal(enc, &got)
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(got, in) {
		t.Errorf("could not unmarshal %q, got %v", enc, got)
	}
}
