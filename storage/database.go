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
        target    text not NULL,
        count     integer not NULL
    );

    create index link_target on linkstats(target);
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
