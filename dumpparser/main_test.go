package main

import (
	"github.com/semanticize/dumpparser/dumpparser/wikidump"
	"github.com/semanticize/dumpparser/storage"
	"testing"
)

func TestStoreLinks(t *testing.T) {
	db, _ := storage.MakeDB(":memory:", true, &storage.Settings{"bla", 3})

	links := make(chan map[wikidump.Link]int)
	go func() {
		links <- map[wikidump.Link]int{
			wikidump.Link{Anchor: "semanticizest", Target: "Entity_linking"}: 2,
			wikidump.Link{Anchor: "NER", Target: "Named_entity_recognition"}: 3,
		}
		links <- map[wikidump.Link]int{
			wikidump.Link{Anchor: "semanticizest", Target: "Entity_linking"}: 1,
		}
		close(links)
	}()

	if err := storeLinks(db, links, 3); err != nil {
		t.Error(err)
	}

	var count float64
	q := `select count from linkstats
	      where targetid = (select id from titles where title="Entity_linking")`
	err := db.QueryRow(q).Scan(&count)
	if err != nil {
		t.Fatal(err)
	} else if count != 3 {
		t.Errorf("expected count=3.0, got %f\n", count)
	}
}
