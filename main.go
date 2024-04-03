package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/marcboeker/go-duckdb"
	_ "github.com/marcboeker/go-duckdb"
)

var w fyne.Window

func main() {
	a := app.New()
	w = a.NewWindow("Yo Bitch")
	w.Resize(fyne.NewSize(800, 600))
	w.SetMaster()

	go runStartup()

	w.ShowAndRun()
}

type AnimeItem struct {
	Title    string
	Synopsis string
}

func runStartup() {
	if !databaseExists() {
		buildDatabase(w)
	}

	animes := fetchAnimes()
	detailView := widget.NewLabel("empty")
	detailView.Wrapping = fyne.TextWrapWord
	animeList := widget.NewList(
		func() int { return len(animes) },
		func() fyne.CanvasObject { return widget.NewLabel("empty") },
		func(id widget.ListItemID, ob fyne.CanvasObject) {
			ob.(*widget.Label).SetText(animes[id].Title)
		},
	)
	animeList.OnSelected = func(id widget.ListItemID) {
		detailView.SetText(fmt.Sprintf("%s - %s", animes[id].Title, animes[id].Synopsis))
	}
	content := container.NewHSplit(animeList, detailView)
	content.SetOffset(0.4)

	w.SetContent(content)
}

func buildDatabase(w fyne.Window) {
	waitChan := make(chan int, 1)
	button := widget.NewButton("Get Dem Animes", func() {
		downloadAnimes(w)
		waitChan <- 1
	})
	w.SetContent(container.NewCenter(button))
	<-waitChan
}

const BASEURL = "https://kitsu.io/api/edge/anime"

func downloadAnimes(w fyne.Window) {
	perPage := 10
	info := doRequest(fmt.Sprintf("%s?page[limit]=%d", BASEURL, perPage))
	lastURL, _ := url.Parse(info.Links.Last)
	params, _ := url.ParseQuery(lastURL.RawQuery)
	total, _ := strconv.Atoi(params["page[offset]"][0])

	total = 1000 // for testing

	pbar := widget.NewProgressBar()
	pbar.Max = float64(total)
	text := widget.NewLabel("Downloading dem animes")
	w.SetContent(container.NewCenter(container.NewVBox(text, pbar)))

	animeChan := make(chan AnimeResponse)
	urlChan := make(chan string)
	semChan := make(chan int, 64)
	createDatabase()

	insertChan := make(chan []AnimeRecord)
	go getInserter(insertChan)

	defer close(animeChan)

	var wg sync.WaitGroup

	go func() {
		for next := range urlChan {
			wg.Add(1)
			semChan <- 1
			go func(u string) {
				animeChan <- *doRequest(u)
				<-semChan
			}(next)
		}
	}()

	go func() {
		defer close(insertChan)
		current := 0.0
		for batch := range animeChan {
			insertChan <- batch.Data
			current += 1.0
			pbar.SetValue(current)
			wg.Done()
		}
	}()

	defer close(urlChan)
	page := 0
	for page <= total {
		page += 1
		next := fmt.Sprintf("%s?page[limit]=%d&page[offset]=%d", BASEURL, perPage, page)
		urlChan <- next
	}

	wg.Wait()
}

type AnimeRecord struct {
	Id         string
	Attributes struct {
		Slug           string
		CanonicalTitle string
		Synopsis       string
	}
}

type AnimeResponse struct {
	Links struct {
		First string
		Next  string
		Last  string
	}
	Data []AnimeRecord
}

func doRequest(url string) *AnimeResponse {
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	ar := new(AnimeResponse)
	if err := json.NewDecoder(resp.Body).Decode(ar); err != nil {
		panic(err)
	}

	return ar
}

func databaseExists() bool {
	_, err := os.Stat("./test.db")
	return !errors.Is(err, os.ErrNotExist)
}

func createDatabase() {
	db, err := sql.Open("duckdb", "./test.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

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
	connector, err := duckdb.NewConnector("./test.db", nil)
	if err != nil {
		panic(err)
	}
	defer connector.Close()

	conn, err := connector.Connect(context.Background())
	if err != nil {
		panic(err)
	}

	appender, err := duckdb.NewAppenderFromConn(conn, "", "animes")
	if err != nil {
		panic(err)
	}
	defer appender.Close()

	for batch := range ch {
		for _, row := range batch {
			fmt.Println("INSERTING", row.Attributes.CanonicalTitle)
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

	db, err := sql.Open("duckdb", "test.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	rows, err := db.Query(`SELECT * FROM animes`)
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
