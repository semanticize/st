package main

import (
	"github.com/semanticize/dumpparser/dumpparser/wikidump"
	"github.com/semanticize/dumpparser/storage"
	"testing"
)

func TestStoreLinks(t *testing.T) {
	db, _ := storage.MakeDB(":memory:", true, 3)

	links := make(chan map[wikidump.Link]int, 1)
	links <- map[wikidump.Link]int{
		wikidump.Link{Anchor: "semanticizest", Target: "Entity_linking"}: 2,
		wikidump.Link{Anchor: "NER", Target: "Named_entity_recognition"}: 3,
	}
	close(links)

	if err := storeLinks(db, links, 3); err != nil {
		t.Error(err)
	}
}
