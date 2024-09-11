package pkg

import (
	"math/rand"
	"time"
)

func GenGoodRandom(min, max int, bad map[int]bool) int {
	n := rand.Intn(max-min) + min // generate a random number between min and max
	if bad[n] {
		return GenGoodRandom(min, max, bad)
	}
	return n
}

func GetFirstTimeOfCurrentWeek() time.Time {
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	return today.Truncate(time.Hour * 168)
}
