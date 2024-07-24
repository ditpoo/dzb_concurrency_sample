package tmdb

import (
	"time"
)

type TMDB struct {
}

func NewTMDB() *TMDB {
	return &TMDB{}
}



type MovieDetails struct {
	MovieResults []map[string]string `json:"movie_results"`
}

func (t *TMDB) GetContentDetails(imdbID string) MovieDetails {
	movieDetails := MovieDetails{
		[]map[string]string{
			{
				"description": "This is a test description from tmdb",
			},
		},
	}
	time.Sleep(time.Millisecond * 250)

	return movieDetails
}