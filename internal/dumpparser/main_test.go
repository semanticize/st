package dumpparser

import (
	"testing"

	"github.com/semanticize/st/internal/storage"
	"github.com/semanticize/st/wikidump"
)

func TestStoreLinks(t *testing.T) {
	db, _ := storage.MakeDB(":memory:", true,
		&storage.Settings{Dumpname: "bla", MaxNGram: 3})

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

	processed := make(chan *processedLink)
	go func() {
		for linkFreq := range links {
			for link, freq := range linkFreq {
				processed <- processLink(&link, freq, 3)
			}
		}
		close(processed)
	}()

	if err := storeLinks(db, processed); err != nil {
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
