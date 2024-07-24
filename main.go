package main

import (
	"context"
	"dzb/pkg/imdb"
	"dzb/pkg/tmdb"
	yt_scraper "dzb/pkg/yt_scrapper"
	"log"
	"strconv"
	"strings"
	"sync"
	rate "golang.org/x/time/rate"
)

type DeazyB struct {
	imdb       *imdb.IMDB
	tmdb       *tmdb.TMDB
	yt_scraper *yt_scraper.YtScrapper
}

func NewDeazyB() *DeazyB {
	return &DeazyB{
		imdb:       imdb.NewIMDB(),
		tmdb:       tmdb.NewTMDB(),
		yt_scraper: yt_scraper.NewYtScrapper(),
	}
}

type ChannelItem struct {
	Channel_Name    string
	Category_Name   string
	Label_name      string
	Title_structure string
}

type CategoryItem struct {
	Category_name string
	Channels      []ChannelItem
}

type Video struct {
	Video_Details    yt_scraper.Video
	Extracted_Title  string
	Release_Year     int
	IMDB_ID          string
	TMDB_Description tmdb.MovieDetails
	
}

type VideoItem struct {
	Channel_Name     string
	Category_Name    string
	Label_Name       string
}

type VideoData struct {
	Video Video
	VideoItem VideoItem
}

type ChannelData struct {
	Videos      []VideoData
	ChannelItem ChannelItem
}

type CategoryData struct {
	Videos       []VideoData
	CategoryItem CategoryItem
}

func (c *CategoryItem) PrepareYoutubeChannelNames(ctx context.Context) <-chan ChannelItem {
	ytChanItems := make(chan ChannelItem)
	go func() {
		defer close(ytChanItems)
		for _, item := range c.Channels {
			select {
				case <-ctx.Done():
					return
				case ytChanItems <- item:
			}
		}
	}()
	return ytChanItems
}

func (c *CategoryItem) VideoScrapperWorker(ctx context.Context, chanItem ChannelItem, deazyb *DeazyB) chan ChannelData {
	outputChan := make(chan ChannelData)
	go func() {
		videoItems := deazyb.yt_scraper.GetVideosFromChannel(chanItem.Channel_Name)
		var channelVideos []VideoData
		for _, videoItem := range videoItems {
			video := VideoData{
				Video: Video{
					Video_Details: videoItem,
				},
				VideoItem: VideoItem{
					Channel_Name:  chanItem.Channel_Name,
					Category_Name: chanItem.Category_Name,
					Label_Name:    chanItem.Label_name,
				},
			}
			video.ExtractTitle(ctx)
			channelVideos = append(channelVideos, video)
		}
		outputChan <- ChannelData{
			Videos: channelVideos,
			ChannelItem: chanItem,
		}
		defer close(outputChan)
	}()
	return outputChan
}


func (c *CategoryItem) VideoScrapper(ctx context.Context, chanItems <-chan ChannelItem, deazyb *DeazyB) []chan ChannelData {
	var vidSracpChan []chan ChannelData
	for chanItem := range chanItems {
		select {
		case <-ctx.Done():
			return vidSracpChan
		default:
			if len(chanItem.Channel_Name) > 0 {
				vidSracpChan = append(vidSracpChan, c.VideoScrapperWorker(ctx, chanItem, deazyb))
			}
		}
	}
	return vidSracpChan
}

func (c *CategoryItem) MergeVideoOuputChannels(ctx context.Context, outputchans []chan ChannelData) <- chan ChannelData {
	outputChan := make(chan ChannelData)
	var wg sync.WaitGroup
	for _, item := range outputchans {
		wg.Add(1)
		go func (item chan ChannelData) {
			defer wg.Done()
			for data := range item {
				select {
					case <- ctx.Done():
						return
					default:
						outputChan <- data				
				}
			}
		}(item)
	}
	go func() {
		wg.Wait()
		close(outputChan)
	}()
	return outputChan
}

func (v *VideoData) ExtractTitle(ctx context.Context) {
	movie_title := ""
	title := v.Video.Video_Details.Title
	if strings.Contains(title, "Trailer") || strings.Contains(title, "Teaser") {
		if strings.Contains(title, "Teaser") {
			movie_title = strings.Split(strings.Split(title, " Trailer ")[0], " Teaser")[0];
		} else {
			movie_title = strings.Split(title, " Trailer ")[0];
		}

		if strings.Contains(title, "(") && strings.Contains(title, ")") {
			release_year, err := strconv.Atoi(strings.Split(strings.Split(title, "(")[1], ")")[0])
			if err != nil {
				v.Video.Release_Year = 2024
			} else {
				v.Video.Release_Year = release_year
			}
		}
	}
	v.Video.Extracted_Title = movie_title
}

func (v *VideoData) IMDBWorker(imdb_limiter *rate.Limiter, deazyb *DeazyB) error {
	err := imdb_limiter.Wait(context.Background())
	if err != nil {
		log.Println(err)
		return err
	}
	imdb_id := deazyb.imdb.GetImdbId(v.Video.Extracted_Title)
	log.Printf("Found imdb id %s for title %s", imdb_id, v.Video.Extracted_Title)
	v.Video.IMDB_ID =  imdb_id
	return nil
}

func (v *VideoData) TMDBWorker(tmdb_limiter *rate.Limiter, deazyb *DeazyB) error {
	err := tmdb_limiter.Wait(context.Background())
	if err != nil {
		log.Println(err)
		return err
	}
	tmdb_details := deazyb.tmdb.GetContentDetails(v.Video.IMDB_ID)
	log.Printf("Found tmdb details %d for imdb_id %s", len(tmdb_details.MovieResults), v.Video.IMDB_ID)
	v.Video.TMDB_Description = tmdb_details
	return nil
}

func (v *VideoData) DescriptionWorker(
	ctx context.Context, imdb_limiter *rate.Limiter, tmdb_limiter *rate.Limiter, deazyb *DeazyB) error {
	// find imdb id for video
	err := v.IMDBWorker(imdb_limiter, deazyb)
	if err != nil {
		return err
	}
	// find description for video
	err = v.TMDBWorker(tmdb_limiter, deazyb)
	if err != nil {
		return err
	}
	return nil
}

func (c *ChannelData) FindDescription(
	ctx context.Context, channelData *ChannelData, imdbLimiter *rate.Limiter, tmdbLimiter *rate.Limiter, deazyb *DeazyB) chan VideoData {
	videoDatachan := make(chan VideoData)
	var wg sync.WaitGroup
	for id, videoItem := range channelData.Videos {
		wg.Add(1)
		go func () {
			err := channelData.Videos[id].DescriptionWorker(ctx, imdbLimiter, tmdbLimiter, deazyb)
			if err != nil {
				log.Println(err, videoItem.Video.Extracted_Title)
			}
			videoDatachan <- videoItem
			defer wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		close(videoDatachan)
	}()
	return videoDatachan
}

func (c *ChannelData) MetaDataWorker(
	ctx context.Context, imdb_limiter *rate.Limiter, tmdb_limiter *rate.Limiter, doneChan chan bool, deazyb *DeazyB) {
	for range c.FindDescription(ctx, c, imdb_limiter, tmdb_limiter, deazyb) {
		// do nothing
	}
	doneChan <- true
}

// Worker processes tasks from the task channel.
func (c *CategoryItem) AddMetaData(ctx context.Context, dataChan <- chan ChannelData, deazyb *DeazyB) chan ChannelData {
	outputchans := make(chan ChannelData)
	const (
		imdb_rateLimit     = 10 // tasks per second
		imdb_burstLimit    = 10
		tmdb_rateLimit     = 10 // tasks per second
		tmdb_burstLimit    = 10
	)
	imdb_limiter := rate.NewLimiter(rate.Limit(imdb_rateLimit), imdb_burstLimit)
	tmdb_limiter := rate.NewLimiter(rate.Limit(tmdb_rateLimit), tmdb_burstLimit)
	var wg sync.WaitGroup
	for channelData := range dataChan {
		metaDataExtracDoneChan := make(chan bool) 
		select {
		case <- ctx.Done():
		default:
			wg.Add(1)
			go func () {
				defer wg.Done()
				go channelData.MetaDataWorker(ctx, imdb_limiter, tmdb_limiter, metaDataExtracDoneChan, deazyb)
				<- metaDataExtracDoneChan
				outputchans <- channelData
			}()
		}
	}
	go func() {
		wg.Wait()
		close(outputchans)
	}()
	
	return outputchans
}

// this is an example to showcase various concurrency patters in a single file
func main() {
	log.Println("In main")
	deazyb := NewDeazyB()
	movie_trailer_channel := []ChannelItem{
		{
			Channel_Name:    "RottenAppleTRAILERS",
			Title_structure: "movie_trailers",
			Label_name:      "RottenAppleTRAILERS",
		},
		{
			Channel_Name:    "RottenOrangeTRAILERS",
			Title_structure: "tv_trailers",
			Label_name:      "RottenOrangeTRAILERS",
		},
		{
			Channel_Name:    "RottenMangoTRAILERS",
			Title_structure: "tv_trailers",
			Label_name:      "RottenMangoTRAILERS",
		},
		{
			Channel_Name:    "RottenStrawberryTRAILERS",
			Title_structure: "tv_trailers",
			Label_name:      "RottenStrawberryTRAILERS",
		},
		{
			Channel_Name:    "RottenBannanaTRAILERS",
			Title_structure: "tv_trailers",
			Label_name:      "RottenBannanaTRAILERS",
		},
	}
	trailerCategory := CategoryItem{
		Category_name: "movie_trailers",
		Channels:      movie_trailer_channel,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// get channel with job items
	ytChannelChan := trailerCategory.PrepareYoutubeChannelNames(ctx)
	// get go channel that will return an array of channel which individually retuns the scrapped video
	// this is fanning out scrapping of individual yt channels
	scrappedYtChannelChan := trailerCategory.VideoScrapper(ctx, ytChannelChan, deazyb)
	// merge the ouputs from scrapped yt channels to a singe go channel 
	mergedScrappedChannelOuputChan := trailerCategory.MergeVideoOuputChannels(ctx, scrappedYtChannelChan)
	// again fan out process to get imdb id and then tmdb description and return videos on complete of processing
	// this are rate limited workers workign concurrently where the tmdb worker depends on ouput of imdb worker
	processedYtChannelDataChan := trailerCategory.AddMetaData(ctx, mergedScrappedChannelOuputChan, deazyb)
	var videos []VideoData
	for items := range processedYtChannelDataChan {
		videos = append(videos, items.Videos...)
	}
	log.Printf("Uploading %d items for Category %s", len(videos), trailerCategory.Category_name)
	log.Printf("%+v", videos[0])
}
