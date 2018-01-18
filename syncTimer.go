package gosync

import (
	"time"
)

// 30分钟超时时间
func setTimer(fresher, ender, stop chan struct{}, i int) {
	n := 0
	tick := time.Tick(1 * time.Minute)
END:
	for {
		select {
		case <-fresher:
			n = 0
		case <-ender:
			close(fresher)
			close(ender)
			close(stop)
			break END
		case <-tick:
			n++
			if n == i {
				stop <- struct{}{}
				close(fresher)
				close(ender)
				close(stop)
				break END
			}
		}
	}
}
