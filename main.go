package main

import (
	"fmt"
	"net/url"
	"strconv"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const BASEURL = "https://kitsu.io/api/edge/anime"

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

	// left pane
	animeList := widget.NewList(
		func() int { return len(animes) },
		func() fyne.CanvasObject { return widget.NewLabel("empty") },
		func(id widget.ListItemID, ob fyne.CanvasObject) {
			ob.(*widget.Label).SetText(animes[id].Title)
		},
	)

	// right pane
	// detailTitle := widget.NewLabelWithStyle("empty", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	detailView := widget.NewLabel("empty")
	detailView.Wrapping = fyne.TextWrapWord

	detailContainer := container.NewStack()

	animeList.OnSelected = func(id widget.ListItemID) {
		detailContainer.RemoveAll()
		anime := &animes[id]
		// detailTitle := widget.NewRichTextWithText(anime.Title)
		//       detailTitle.TextStyle = fyne.TextStyle{Bold: true}
		//       detailTitle.Text
		// detailView.SetText(fmt.Sprintf(animes[id].Synopsis))
		// rightPanel := container.NewVBox(detailTitle, detailView)
		textBox := widget.NewRichTextFromMarkdown(fmt.Sprintf("# %s \n---\n %s", anime.Title, anime.Synopsis))
		textBox.Wrapping = fyne.TextWrapWord
		detailContainer.Add(textBox)
	}

	content := container.NewHSplit(animeList, detailContainer)
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
		next := fmt.Sprintf("%s?page[limit]=%d&page[offset]=%d", BASEURL, perPage, page*perPage)
		fmt.Println(next)
		urlChan <- next
		page += 1
	}

	wg.Wait()
}
