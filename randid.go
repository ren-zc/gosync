package gosync

import (
	"math"
	"math/rand"
	"time"
)

func RandId() int {
	now := time.Now()
	rand.Seed(now.UnixNano())
	// n := float64(now.Hour()*10000+now.Minute()*100+now.Second()) + rand.Float64()
	// return int(n * 1000000)
	randFloat := rand.Float64()
	randid := uint(now.Hour())<<48 | uint(now.Minute())<<40 | uint(now.Second())<<32 | uint(math.Float64bits(randFloat))>>32
	return int(randid)
}
