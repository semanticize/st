package linking

import (
	"encoding/json"
	"github.com/semanticize/st/hash"
	"github.com/semanticize/st/hash/countmin"
	"github.com/semanticize/st/storage"
	"reflect"
	"testing"
)

var sem = makeSemanticizer()

func TestBestPath(t *testing.T) {
	sem.BestPath("   ") // should not crash
}

func makeSemanticizer() Semanticizer {
	cm, _ := countmin.New(10, 4)
	db, _ := storage.MakeDB(":memory:", true, &storage.Settings{MaxNGram: 2})
	sem := Semanticizer{db, cm, 2}

	for _, h := range hash.NGrams([]string{"Hello", "world"}, 2, 2) {
		_, err := db.Exec(`insert into linkstats values (?, 0, 1)`, h)
		if err == nil {
			_, err = db.Exec(`insert into titles values (0, "dmr")`)
		}
		if err != nil {
			panic(err)
		}
	}
	return sem
}

func TestCandidates(t *testing.T) {
	all, err := sem.All("Hello world")
	if err != nil {
		t.Error(err)
	}
	if len(all) != 1 {
		t.Errorf("expected one entity mention, got %v", all)
	} else if tgt := all[0].Target; tgt != "dmr" {
		t.Errorf(`expected target "dmr", got %q`, tgt)
	}
}

func TestExactMatch(t *testing.T) {
	all, err := sem.ExactMatch("Hello world")
	if err != nil {
		t.Error(err)
	}
	if len(all) != 1 {
		t.Errorf("expected one entity mention, got %v", all)
	}

	all, err = sem.ExactMatch("Hello world program")
	if err != nil {
		t.Error(err)
	}
	if len(all) != 0 {
		t.Errorf("expected no entity mentions, got %v", all)
	}
}

func TestJSON(t *testing.T) {
	in := Entity{"Wikipedia", 4, 10, .9, 0.0115, 0, 9}
	enc, _ := json.Marshal(in)

	var got Entity
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

func BenchmarkJSON(b *testing.B) {
	// XXX We need more examples.
	entities := []*Entity{
		{Target: "Wikipedia", NGramCount: 4, LinkCount: 10, Commonness: .9,
			Senseprob: 0.0115, Offset: 0, Length: 9},
		{Target: "Supercalifragilisticexpialidocious",
			Offset: 4, Length: 6, Senseprob: .8},
		{Target: "Barack_Obama", Offset: 3, Length: 7, Senseprob: .9},
		{Target: "List_of_stupid_examples_for_unit_tests",
			Offset: 1, Length: 2, Senseprob: .1, NGramCount: 100},
	}
	for i := 0; i < b.N; i++ {
		for _, e := range entities {
			json.Marshal(e)
		}
	}
}

func TestViterbi(t *testing.T) {
	cands := []Entity{
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
