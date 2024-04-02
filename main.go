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

func main() {
	a := app.New()
	w := a.NewWindow("Yo Bitch")
	w.Resize(fyne.NewSize(800, 600))
	w.SetMaster()

	// var animes []Anime
	// if databaseExists() {
	// 	animes = fetchAnimes()
	// } else {
	// 	fmt.Println("BUILDING THE DATABASE")
	// }
	buildDatabase(w)

	// detailView := widget.NewLabel("empty")
	// animeList := widget.NewList(
	// 	func() int { return len(animes) },
	// 	func() fyne.CanvasObject { return widget.NewLabel("empty") },
	// 	func(id widget.ListItemID, ob fyne.CanvasObject) {
	// 		ob.(*widget.Label).SetText(animes[id].title)
	// 	},
	// )
	// animeList.OnSelected = func(id widget.ListItemID) {
	// 	detailView.SetText(fmt.Sprintf("%s - %s", animes[id].title, animes[id].synopsis))
	// }
	// content := container.NewHSplit(animeList, detailView)
	// content.SetOffset(0.4)
	//
	// w.SetContent(content)
	//
	// pb := widget.NewProgressBarInfinite()
	// pblab := widget.NewLabel("Gettin them animes")
	// w2 := a.NewWindow("Downloading")
	// w2.SetContent(container.NewStack(pb, pblab))
	// w2.Show()

	w.ShowAndRun()
}

func buildDatabase(w fyne.Window) {
	button := widget.NewButton("Get Dem Animes", func() { downloadAnimes(w) })
	w.SetContent(container.NewCenter(button))
}

const BASEURL = "https://kitsu.io/api/edge/anime"

func downloadAnimes(w fyne.Window) {
	perPage := 10
	info := doRequest(fmt.Sprintf("%s?page[limit]=%d", BASEURL, perPage))
	lastURL, _ := url.Parse(info.Links.Last)
	params, _ := url.ParseQuery(lastURL.RawQuery)
	total, _ := strconv.Atoi(params["page[offset]"][0])

	total = 100 // for testing

	pbar := widget.NewProgressBar()
	pbar.Max = float64(total)
	text := widget.NewLabel("Downloading dem animes")
	w.SetContent(container.NewCenter(container.NewVBox(text, pbar)))

	animeChan := make(chan AnimeResponse)
	urlChan := make(chan string)
	semChan := make(chan int, 16)
	createDatabase()

	insertChan := make(chan []Anime)
	defer close(insertChan)
	go getInserter(insertChan)

	defer close(animeChan)

	var wg sync.WaitGroup

	go func() {
		for next := range urlChan {
			semChan <- 1
			go func(u string) {
				animeChan <- *doRequest(u)
				<-semChan
			}(next)
		}
	}()

	go func() {
		current := 0.0
		for batch := range animeChan {
			insertChan <- batch.Data
			current += 1.0
			pbar.SetValue(current)
			wg.Done()
		}
	}()

	go func() {
		defer close(urlChan)
		page := 0
		for page <= total {
			page += 1
			wg.Add(1)
			next := fmt.Sprintf("%s?page[limit]=%d&page[offset]=%d", BASEURL, perPage, page)
			urlChan <- next
		}
	}()

	wg.Wait()
}

type Anime struct {
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
	Data []Anime
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

func getInserter(ch chan []Anime) {
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
			err = appender.AppendRow(row.Attributes.CanonicalTitle, row.Attributes.Synopsis)
			if err != nil {
				panic(err)
			}
		}
	}

	appender.Flush()
	fmt.Println("FLUSHED")
}

func fetchAnimes() []Anime {
	animes := make([]Anime, 0)

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
		anime := new(Anime)
		if err := rows.Scan(&anime.Attributes.CanonicalTitle, &anime.Attributes.Synopsis); err != nil {
			panic(err)
		}
		animes = append(animes, *anime)
	}

	return animes
}
