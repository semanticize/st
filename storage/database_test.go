package storage

import (
	"github.com/semanticize/dumpparser/hash/countmin"
	"reflect"
	"testing"
)

func TestMakeDB(t *testing.T) {
	db, err := MakeDB("/", true)
	if db != nil {
		t.Error("got non-nil for invalid path name")
	}
	if err == nil {
		t.Error("got no error for invalid path name")
	}
}

func TestRedirects(t *testing.T) {
	var err error
	check := func() {
		if err != nil {
			panic(err)
		}
	}

	db, err := MakeDB(":memory:", true)
	check()

	_, err = db.Exec(`insert into linkstats values (42, "Architekt", 10)`)

	redirects := make(map[string]string)
	redirects["Architekt"] = "Architect"

	err = ProcessRedirects(db, redirects)
	check()
	err = Finalize(db)
	check()

	rows, err := db.Query(`select * from linkstats`)
	if !rows.Next() {
		t.Fatal("no rows in database")
	}

	var count float64
	var hash int64
	var title string
	err = rows.Scan(&hash, &title, &count)
	if hash != 42 {
		t.Fatalf("wrong hash: %d", hash)
	}
	if title != "Architect" {
		t.Fatalf("wrong title: %q", title)
	}
	if count != 10 {
		t.Fatalf("wrong count: %d", count)
	}

	if rows.Next() {
		t.Fatal("too many rows (original not deleted?)")
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
	db, err := MakeDB(":memory:", true)
	check()

	for _, i := range []uint32{1, 6, 13, 7, 8, 20, 44} {
		cm.Add(i, i + 5)
	}

	err = StoreCM(db, cm)
	check()

	got, err := LoadCM(db)
	check()

	if !reflect.DeepEqual(cm.Counts(), got.Counts()) {
		t.Errorf("expected %v, got %v", cm.Counts(), got)
	}
}
