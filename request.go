package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type AnimeRecord struct {
	Id         string
	Attributes struct {
		CanonicalTitle string
		Synopsis       string
		StartDate      string
		EndDate        string
		CoverImage     struct {
			Original string
		}
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
	ar := new(AnimeResponse)
	for {
		resp, err := http.Get(url)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		if err := json.NewDecoder(resp.Body).Decode(ar); err != nil {
			fmt.Println(url)
			time.Sleep(time.Second * 5)
			// panic(err)
		} else {
			break
		}
	}

	return ar
}
