package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/marcboeker/go-duckdb"
	// _ "github.com/marcboeker/go-duckdb"
)

const (
	DBFILE = "./anime.db"
	TABLE  = "animes"
)

func databaseExists() bool {
	_, err := os.Stat(DBFILE)
	return !errors.Is(err, os.ErrNotExist)
}

func createDatabase() {
	db, err := sql.Open("duckdb", DBFILE)
	if err != nil {
		panic(err)
	}

	_, err = db.Exec(`
        CREATE TABLE animes (
            id UBIGINT, 
            title VARCHAR,
            synopsis VARCHAR,
            start_date VARCHAR,
            end_date VARCHAR
        )
    `)
	if err != nil {
		panic(err)
	}
}

func getInserter(ch chan []AnimeRecord) {
	connector, err := duckdb.NewConnector(DBFILE, nil)
	if err != nil {
		panic(err)
	}
	defer connector.Close()

	conn, err := connector.Connect(context.Background())
	if err != nil {
		panic(err)
	}

	appender, err := duckdb.NewAppenderFromConn(conn, "", TABLE)
	if err != nil {
		panic(err)
	}
	defer appender.Close()

	for batch := range ch {
		for _, row := range batch {
			id, _ := strconv.ParseUint(row.Id, 10, 64)
			attrs := row.Attributes
			err = appender.AppendRow(
				id,
				attrs.CanonicalTitle,
				attrs.Synopsis,
				attrs.StartDate,
				attrs.EndDate,
			)
			if err != nil {
				println("ITS ME")
				panic(err)
			}
		}
		appender.Flush()
	}

	appender.Flush()
}

func fetchAnimes() []AnimeItem {
	animes := make([]AnimeItem, 0)

	db, err := sql.Open("duckdb", DBFILE)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	sql := `
    SELECT title, synopsis, start_date, end_date
    FROM %s
    ORDER BY title
    `

	rows, err := db.Query(fmt.Sprintf(sql, TABLE))
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() {
		anime := new(AnimeItem)
		if err := rows.Scan(
			&anime.Title,
			&anime.Synopsis,
			&anime.StartDate,
			&anime.EndDate,
		); err != nil {
			panic(err)
		}
		animes = append(animes, *anime)
	}

	return animes
}
