package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"

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
            title VARCHAR,
            synopsis VARCHAR
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
			// fmt.Println("INSERTING", row.Attributes.CanonicalTitle)
			err = appender.AppendRow(row.Attributes.CanonicalTitle, row.Attributes.Synopsis)
			if err != nil {
				panic(err)
			}
		}
		appender.Flush()
	}

	appender.Flush()
	fmt.Println("FLUSHED")
}

func fetchAnimes() []AnimeItem {
	animes := make([]AnimeItem, 0)

	db, err := sql.Open("duckdb", DBFILE)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	rows, err := db.Query(fmt.Sprintf(`SELECT * FROM %s ORDER BY Title`, TABLE))
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() {
		anime := new(AnimeItem)
		if err := rows.Scan(&anime.Title, &anime.Synopsis); err != nil {
			panic(err)
		}
		animes = append(animes, *anime)
	}

	return animes
}
