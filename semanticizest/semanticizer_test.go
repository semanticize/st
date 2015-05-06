package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

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
