package yt_scraper

import (
	"log"
	"time"
)

type YtScrapper struct {
}

func NewYtScrapper() *YtScrapper {
	return &YtScrapper{}
}

type Video struct {
	VideoID string `json:"videoId"`
	Title string `json:"title"`
	Url string `json:"url"`
	YtDescription string `json:"ytDescription"`
	PublishedTimeText string `json:"publishedTimeText"`
	LenghtText string	`json:"lengthText"`
	ViewCountText string `json:"viewCountText"`
}

func (y *YtScrapper) GetVideosFromChannel(chanName string) []Video {
	videoItems := []Video{
		{
			VideoID: "10101",
			Title: "Youtube: Movie Trailer #1 (2024)",
			Url: "www.youtube.com",
			YtDescription: "This is a test description",
			PublishedTimeText: time.Now().String(),
			LenghtText: "2.40",
			ViewCountText: "10,0000",
		},
		{
			VideoID: "10101",
			Title: "Youtube: Movie Trailer #1 (2024)",
			Url: "www.youtube.com",
			YtDescription: "This is a test description",
			PublishedTimeText: time.Now().String(),
			LenghtText: "2.40",
			ViewCountText: "10,0000",
		},
		{
			VideoID: "10101",
			Title: "Youtube: Movie Trailer #1 (2024)",
			Url: "www.youtube.com",
			YtDescription: "This is a test description",
			PublishedTimeText: time.Now().String(),
			LenghtText: "2.40",
			ViewCountText: "10,0000",
		},
		{
			VideoID: "10101",
			Title: "Youtube: Movie Trailer #1 (2024)",
			Url: "www.youtube.com",
			YtDescription: "This is a test description",
			PublishedTimeText: time.Now().String(),
			LenghtText: "2.40",
			ViewCountText: "10,0000",
		},
		{
			VideoID: "10101",
			Title: "Youtube: Movie Trailer #1 (2024)",
			Url: "www.youtube.com",
			YtDescription: "This is a test description",
			PublishedTimeText: time.Now().String(),
			LenghtText: "2.40",
			ViewCountText: "10,0000",
		},
	}
	for i := 0; i < 2; i++ {
		time.Sleep(time.Second * 1)
		videoItems = append(videoItems, videoItems...)
		log.Printf("Feteched %d videos for %s channel", len(videoItems), chanName)
	}

	return videoItems
}