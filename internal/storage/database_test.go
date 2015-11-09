package storage

import (
	"database/sql"
	"reflect"
	"strconv"
	"testing"

	"github.com/semanticize/st/countmin"
	"github.com/semanticize/st/wikidump"
)

func TestMakeDB(t *testing.T) {
	db, err := MakeDB("/", true, &Settings{"blawiki-latest", 2})
	if db != nil {
		t.Error("got non-nil for invalid path name")
	}
	if err == nil {
		t.Error("got no error for invalid path name")
	}
}

func TestLoadModel(t *testing.T) {
	var err error
	var s *Settings
	check := func() {
		if err != nil {
			t.Fatal(err)
		}
	}

	db, err := MakeDB(":memory:", true, &Settings{"foowiki", 6})
	check()
	defer db.Close()

	s, err = loadModel(db)
	check()
	if s == nil {
		t.Fatal("got nil *Settings, but no error")
	}
	if s.MaxNGram != 6 {
		t.Errorf("expected 6, got %d for maxNGram", s.MaxNGram)
	}
}

func TestRedirects(t *testing.T) {
	var err error
	check := func() {
		if err != nil {
			panic(err)
		}
	}

	db, err := MakeDB(":memory:", true, &Settings{"somewiki", 5})
	check()

	_, err = db.Exec(`insert or ignore into titles values (NULL, "Architekt")`)
	check()
	_, err = db.Exec(`insert into linkstats values
		(42, (select id from titles where title = "Architekt"), 10)`)
	check()

	redirects := []wikidump.Redirect{
		{Title: "Architekt", Target: "Architect"},
		{Title: "Non existent", Target: "Non-existent"},
	}

	err = StoreRedirects(db, redirects, nil)
	check()
	err = Finalize(db)
	check()

	rows, err := db.Query(`select * from linkstats`)
	if err != nil {
		t.Fatal(err)
	}
	if !rows.Next() {
		t.Fatal("no rows in database")
	}

	var count float64
	var hash int64
	var toId int64
	var title string
	err = rows.Scan(&hash, &toId, &count)
	if hash != 42 {
		t.Fatalf("wrong hash: %d", hash)
	}
	if count != 10 {
		t.Fatalf("wrong count: %f", count)
	}
	if rows.Next() {
		t.Fatal("too many rows (original not deleted?)")
	}
	rows.Close()

	// Check that the redirect target was stored in the [titles] table,
	// and the original removed.
	err = db.QueryRow(`select title from titles where id = ?`,
		toId).Scan(&title)
	if err != nil {
		t.Fatal(err)
	}
	if title != "Architect" {
		t.Fatalf("wrong title: %q", title)
	}
	for _, title := range []string{
		"Architekt", "Non existent", "Non-existent",
	} {
		err = db.QueryRow(`select id from titles where title = ?`,
			title).Scan(&toId)
		if err != sql.ErrNoRows {
			t.Fatalf("expected ErrNoRows, got %q", err)
		}
	}
}

func TestCM(t *testing.T) {
	var err error
	check := func() {
		if err != nil {
			t.Fatal(err)
		}
	}

	cm, _ := countmin.New(5, 16)
	db, err := MakeDB(":memory:", true, &Settings{"foowiki.xml.bz2", 8})
	check()

	keys := []int{1, 6, 13, 7, 8, 20, 44}
	for _, k := range keys {
		cm.Add([]byte(strconv.Itoa(k)))
	}
	freq := make(map[int]uint32)
	for _, k := range keys {
		freq[k] = cm.Get([]byte(strconv.Itoa(k)))
	}

	err = StoreCM(db, cm)
	check()

	got, err := LoadCM(db)
	check()

	if !reflect.DeepEqual(cm.Counts(), got.Counts()) {
		t.Errorf("expected %v, got %v", cm.Counts(), got)
	}
	// Assert that the hash function still works after load-restore.
	for _, k := range keys {
		if f := got.Get([]byte(strconv.Itoa(k))); f != freq[k] {
			t.Errorf("expected count %d for %d, got %d", freq[k], k, f)
		}
	}
}
