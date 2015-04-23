package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestCandidateJSON(t *testing.T) {
	in := candidate{"Wikipedia", 1, 0.0115}
	enc, _ := json.Marshal(in)

	var got candidate
	json.Unmarshal(enc, &got)

	if !reflect.DeepEqual(in, got) {
		t.Errorf("marshalled %v, got %v", in, got)
	}

	enc = []byte(`{"target":"Wikipedia","commonness":1,"senseprob":0.0115}`)
	err := json.Unmarshal(enc, &got)
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(got, in) {
		t.Errorf("could not unmarshal %q, got %v", enc, got)
	}
}
