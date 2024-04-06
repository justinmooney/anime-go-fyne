package main

import (
	"fmt"
	"sync"
)

type Downloader struct {
	Progress   chan int
	baseURL    string
	pageSize   int
	TotalPages int
}

func NewDownloader(pageSize int) *Downloader {
	dl := new(Downloader)
	dl.pageSize = pageSize
	dl.baseURL = dl.getBaseURL()
	dl.TotalPages = dl.initialize()
	dl.Progress = make(chan int)
	return dl
}

func (dl *Downloader) initialize() int {
	// info := doRequest(dl.baseURL)
	// lastURL, _ := url.Parse(info.Links.Last)
	// params, _ := url.ParseQuery(lastURL.RawQuery)
	// total, _ := strconv.Atoi(params["page[offset]"][0])

	total := 100 // for testing
	return total
}

func (dl *Downloader) getBaseURL() string {
	return fmt.Sprintf("%s?page[limit]=%d", BASEURL, dl.pageSize)
}

func (dl *Downloader) getPageURL(offset int) string {
	return fmt.Sprintf("%s&page[offset]=%d", dl.baseURL, offset)
}

func (dl *Downloader) Download() {
	defer close(dl.Progress)

	semChan := make(chan int, 64)
	defer close(semChan)

	urlChan := make(chan string)
	defer close(urlChan)

	insertChan := NewInserter()
	defer close(insertChan)

	var wg sync.WaitGroup

	go func() {
		for next := range urlChan {
			wg.Add(1)
			semChan <- 1
			go func(u string) {
				batch := doRequest(u)
				<-semChan
				insertChan <- batch.Data
				dl.Progress <- 1
				wg.Done()
			}(next)
		}
	}()

	page := 0
	for page <= dl.TotalPages {
		urlChan <- dl.getPageURL(page * dl.pageSize)
		page += 1
	}

	wg.Wait()
}
