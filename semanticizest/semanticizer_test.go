package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

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

func TestViterbi(t *testing.T) {
	m := newTable(3, 5)
	for _, x := range []struct {
		i, j int
		v    float64
	}{
		{0, 1, .1}, {0, 2, .02}, {0, 3, .01},
		{1, 1, .1}, {1, 2, .9}, {1, 3, 0},
		{2, 0, .01}, {2, 2, 0}, {2, 3, .7}, {2, 4, 1},
	} {
		m.at(x.i, x.j).v = x.v
	}
	for i := 0; i < 3; i++ {
		t.Logf("%v", m.row(i))
	}
	path := m.viterbi()
	if len(path) != m.nrows() {
		t.Errorf("expected path of length %d, found %d", m.nrows(), len(path))
	}
}
