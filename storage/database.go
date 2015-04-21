// Database stuff.
package storage

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/semanticize/dumpparser/hash/countmin"
	"log"
	"os"
	"strconv"
)

const create = `
	--pragma foreign_keys = on;
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

	create table titles (
		id    integer primary key,
		title text    unique not NULL
	);

	create table linkstats (
		ngramhash integer not NULL,
		targetid  integer not NULL,
		count     float   not NULL
		-- Can't get the following to work.
		--foreign key(targetid) references titles(id)
	);

	create index target on linkstats(targetid);
	create unique index hash_target on linkstats(ngramhash, targetid);
`

type Settings struct {
	Dumpname string // Filename of dump
	MaxNGram uint   // Max. length of n-grams
}

func MakeDB(path string, overwrite bool, s *Settings) (db *sql.DB, err error) {
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
	if err == nil {
		_, err = db.Exec(`insert into parameters values ("dumpname", ?)`,
			s.Dumpname)
	}
	if err == nil {
		_, err = db.Exec(`insert into parameters values ("maxngram", ?)`,
			strconv.FormatUint(uint64(s.MaxNGram), 10))
	}
	return
}

// XXX move this elsewhere
const DefaultMaxNGram = 7

// XXX Load and return the n-gram count-min sketch as well?
func LoadModel(path string) (db *sql.DB, s *Settings, err error) {
	db, err = sql.Open("sqlite3", path)
	defer func() {
		if err != nil && db != nil {
			db.Close()
			db = nil
		}
	}()

	if err == nil {
		db.Ping()
	}
	if err == nil {
		s, err = loadModel(db)
	}
	return
}

func loadModel(db *sql.DB) (s *Settings, err error) {
	s = new(Settings)

	var maxNGramStr string
	rows := db.QueryRow(`select value from parameters where key = "maxngram"`)
	err = rows.Scan(&maxNGramStr)
	if err == sql.ErrNoRows {
		log.Printf("no maxngram setting in database, using default=%d",
			DefaultMaxNGram)
		s.MaxNGram = DefaultMaxNGram
	} else if maxNGramStr == "" {
		// go-sqlite3 seems to return this if the parameter is not set...
		s.MaxNGram = DefaultMaxNGram
	} else {
		var max64 int64
		max64, err = strconv.ParseInt(maxNGramStr, 10, 0)
		if max64 <= 0 {
			err = fmt.Errorf("invalid value maxngram=%d, must be >0")
		} else {
			s.MaxNGram = uint(max64)
		}
	}

	rows = db.QueryRow(`select value from parameters where key = "dumpname"`)
	if err = rows.Scan(&s.Dumpname); err != nil && err != sql.ErrNoRows {
		s = nil
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

	titleId := MustPrepare(db,
		`select id from titles where title = ?`)
	old := MustPrepare(db,
		`select ngramhash, count from linkstats where targetid = ?`)
	del := MustPrepare(db, `delete from linkstats where targetid = ?`)
	delTitle := MustPrepare(db, `delete from titles where id = ?`)
	insTitle := MustPrepare(db,
		`insert or ignore into titles values (NULL, ?)`)
	ins := MustPrepare(db,
		`insert or ignore into linkstats values
		 (?, (select id from titles where title = ?), 0)`)
	update := MustPrepare(db,
		`update linkstats set count = count + ?
		 where targetid = (select id from titles where title = ?)
		       and ngramhash = ?`)

	for from, to := range redirs {
		var fromId int
		err := titleId.QueryRow(from).Scan(&fromId)
		if err != nil {
			return err
		}

		rows, err := old.Query(fromId)
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

		if err == nil {
			_, err = del.Exec(fromId)
		}
		if err == nil {
			_, err = delTitle.Exec(fromId)
		}

		if err != nil {
			return err
		}

		for _, c := range counts {
			if err == nil {
				_, err = insTitle.Exec(to)
			}
			if err == nil {
				_, err = ins.Exec(c.hash, to)
			}
			if err == nil {
				_, err = update.Exec(c.count, to, c.hash)
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// Load count-min sketch from table ngramfreq.
func LoadCM(db *sql.DB) (sketch *countmin.Sketch, err error) {
	var nrows, ncols int
	shapequery := "select max(row) + 1, max(col) + 1 from ngramfreq"
	err = db.QueryRow(shapequery).Scan(&nrows, &ncols)
	if err != nil {
		return
	}

	cmrows := make([][]uint32, nrows)
	for i := 0; i < nrows; i++ {
		cmrows[i] = make([]uint32, ncols)
	}
	dbrows, err := db.Query("select row, col, count from ngramfreq")
	if err != nil {
		return
	}
	for dbrows.Next() {
		var i, j, count uint32
		if err = dbrows.Scan(&i, &j, &count); err != nil {
			return
		}
		cmrows[i][j] = count
	}
	sketch, err = countmin.NewFromCounts(cmrows)
	return
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
