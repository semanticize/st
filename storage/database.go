// Database stuff.
package storage

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/semanticize/dumpparser/hash/countmin"
	"os"
)

const create = (`
	pragma foreign_keys = on;
	pragma journal_mode = off;
	pragma synchronous = off;

	drop table if exists linkstats;
	drop table if exists ngramfreq;

	create table parameters (
		key   text primary key not NULL,
		value text default NULL
	);

	create table ngramfreq (
		row   integer not NULL,
		col   integer not NULL,
		count integer not NULL
	);

	-- XXX I tried to put the link targets in a separate table with a foreign
	-- key in this one, but inserting into that table would sometimes fail.
	create table linkstats (
		ngramhash integer not NULL,
		target    string  not NULL,		-- actually UTF-8
		count     float   not NULL
	);

	create index target on linkstats(target);
	create unique index hash_target on linkstats(ngramhash, target);
`)

func MakeDB(path string, overwrite bool) (db *sql.DB, err error) {
	if overwrite {
		os.Remove(path)
	}
	db, err = sql.Open("sqlite3", path)
	defer func() {
		if err != nil && db != nil {
			db.Close()
			db = nil
		}
	}()

	if err == nil {
		err = db.Ping()
	}
	if err == nil {
		_, err = db.Exec(create)
	}
	return
}

func Finalize(db *sql.DB) (err error) {
	_, err = db.Exec("drop index target;")
	if err != nil {
		return
	}
	_, err = db.Exec("vacuum;")
	return
}

// Prepares statement; panics on error.
func MustPrepare(db *sql.DB, statement string) *sql.Stmt {
	stmt, err := db.Prepare(statement)
	if err != nil {
		panic(err)
	}
	return stmt
}

type linkCount struct {
	hash  int64
	count float64
}

func ProcessRedirects(db *sql.DB, redirs map[string]string) error {
	counts := make([]linkCount, 0)

	old := MustPrepare(db,
		`select ngramhash, count from linkstats where target = ?`)
	del := MustPrepare(db, `delete from linkstats where target = ?`)
	ins := MustPrepare(db, `insert or ignore into linkstats values (?, ?, 0)`)
	update := MustPrepare(db,
		`update linkstats set count = count + ? where target = ? and ngramhash = ?`)

	for from, to := range redirs {
		rows, err := old.Query(from)
		if err != nil {
			return err
		}

		// SQLite won't let us INSERT or UPDATE while doing a SELECT.
		for counts = counts[:0]; rows.Next(); {
			var count float64
			var hash int64
			rows.Scan(&hash, &count)
			counts = append(counts, linkCount{hash, count})
		}
		rows.Close()
		err = rows.Err()
		if err != nil {
			return err
		}

		_, err = del.Exec(from)
		if err != nil {
			return err
		}

		for _, c := range counts {
			_, err = ins.Exec(c.hash, to)
			if err != nil {
				return err
			}

			_, err = update.Exec(c.count, to, c.hash)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Store count-min sketch into table ngramfreq.
func StoreCM(db *sql.DB, sketch *countmin.Sketch) (err error) {
	insCM, err := db.Prepare(`insert into ngramfreq values (?, ?, ?)`)
	if err != nil {
		return
	}

	for i, row := range sketch.Counts() {
		for j, v := range row {
			_, err = insCM.Exec(i, j, int(v))
			if err != nil {
				return
			}
		}
	}
	return
}
