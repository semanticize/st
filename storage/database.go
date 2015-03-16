// Database stuff.
package storage

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
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

    create table linkstats (
        ngramhash integer not NULL,
        target    text    not NULL,
        count     float   not NULL
    );

    create index target on linkstats(target);
    create index hash_target on linkstats(ngramhash, target);
`)

func MakeDB(path string, overwrite bool) (db *sql.DB, err error) {
	if overwrite {
		os.Remove(path)
	}
	db, err = sql.Open("sqlite3", path)
	if err != nil {
		return
	}
	db.Ping()

	_, err = db.Exec(create)
	return
}

func Finalize(db *sql.DB) (err error) {
	_, err = db.Exec("drop index target;")
	_, err = db.Exec("drop index hash_target;")
	if err != nil {
		return
	}
	_, err = db.Exec("vacuum;")
	return
}

type linkCount struct {
	hash  int64
	count float64
}

func ProcessRedirects(db *sql.DB, redirs map[string]string) error {
	counts := make([]linkCount, 0)

	for from, to := range redirs {
		rows, err := db.Query(`select ngramhash, count from linkstats
							   where target = ?`, from)
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

		_, err = db.Exec(`delete from linkstats where target = ?`, from)
		if err != nil {
			return err
		}

		for _, c := range counts {
			_, err = db.Exec(`insert or ignore into linkstats values (?, ?, 0)`,
				c.hash, to)
			if err != nil {
				return err
			}

			_, err = db.Exec(`update linkstats set count = count + ?
							  where target = ? and ngramhash = ?`,
				c.count, to, c.hash)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
