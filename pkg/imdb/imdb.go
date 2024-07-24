package imdb

import (
	"time"
)

type IMDB struct {
}

func NewIMDB() *IMDB {
	return &IMDB{}
}

func (i *IMDB) GetImdbId(title string) string {
	time.Sleep(time.Millisecond * 200)
	return "tt115447"
}
