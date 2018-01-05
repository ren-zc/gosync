package gosync

import (
	"math/rand"
	"time"
)

func RandId() int {
	now := time.Now()
	rand.Seed(now.UnixNano())
	n := float64(now.Hour()*10000+now.Minute()*100+now.Second()) + rand.Float64()
	return int(n * 1000000)
}
