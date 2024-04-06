package main

import (
	"fmt"
	"math/rand/v2"
	"strings"

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
	w.Resize(fyne.NewSize(1400, 800))
	w.SetFixedSize(true)
	w.SetMaster()
	go runStartup()
	w.ShowAndRun()
}

type AnimeItem struct {
	Title     string
	Synopsis  string
	StartDate string
	EndDate   string
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

	detailContainer := container.NewStack()

	detailView := widget.NewLabel("empty")
	detailView.Wrapping = fyne.TextWrapWord

	animeList.OnSelected = func(id widget.ListItemID) {
		detailContainer.RemoveAll()
		anime := &animes[id]
		md := "# %s (%s)\n---\n %s"
		text := fmt.Sprintf(md, anime.Title, dateString(anime.StartDate, anime.EndDate), anime.Synopsis)
		textBox := widget.NewRichTextFromMarkdown(text)
		textBox.Wrapping = fyne.TextWrapWord
		detailContainer.Add(textBox)
	}

	searcher := widget.NewEntry()
	searcher.PlaceHolder = "search"
	searcher.OnChanged = func(s string) {
		for i, a := range animes {
			if strings.Contains(strings.ToLower(a.Title), strings.ToLower(s)) {
				animeList.ScrollTo(i)
				animeList.Select(i)
				return
			}
		}
		animeList.ScrollToTop()
	}

	listContainer := container.NewBorder(searcher, nil, nil, nil, animeList)

	content := container.NewHSplit(listContainer, detailContainer)
	content.SetOffset(0.3)

	animeList.Select(rand.IntN(len(animes)))
	animeList.ScrollToTop()

	w.SetContent(content)
	w.Resize(fyne.NewSize(1400, 800))
}

func dateString(start, end string) string {
	if end == "" || start == end {
		year := strings.Split(start, "-")[0]
		return year
	}
	startYear := strings.Split(start, "-")[0]
	endYear := strings.Split(end, "-")[0]
	if startYear == endYear {
		return startYear
	}
	return fmt.Sprintf("%s-%s", startYear, endYear)
}

func buildDatabase(w fyne.Window) {
	waitChan := make(chan int, 1)
	button := widget.NewButton("Get Dem Animes", func() {
		downloadPage(w)
		waitChan <- 1
	})
	w.SetContent(container.NewCenter(button))
	<-waitChan
}

func downloadPage(w fyne.Window) {
	text := widget.NewLabel("Gettin dem animes")
	pbar := widget.NewProgressBar()
	w.SetContent(container.NewCenter(container.NewVBox(text, pbar)))

	dl := NewDownloader(10)
	pbar.Max = float64(dl.TotalPages)
	go dl.Download()

	progress := 0
	for p := range dl.Progress {
		progress += p
		pbar.SetValue(float64(progress))
	}
}
