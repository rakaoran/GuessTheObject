package game

import "time"

func (t *ticker) Create(duration time.Duration) <-chan time.Time {
	return time.NewTicker(duration).C
}

func NewTickerGen() ticker {
	return ticker{}
}
