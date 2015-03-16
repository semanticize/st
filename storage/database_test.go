package storage

import "testing"

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
