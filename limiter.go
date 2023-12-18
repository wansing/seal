package seal

import (
	"sync"
	"time"
)

func Limit(interval time.Duration, size int, fn func()) func() bool {
	var tokens = size
	var backlog = false
	var lock sync.Mutex
	go func() {
		for ; true; <-time.Tick(interval) {
			lock.Lock()
			tokens = size // refill
			if backlog {
				backlog = false
				tokens--
				fn()
			}
			lock.Unlock()
		}
	}()
	return func() bool {
		lock.Lock()
		defer lock.Unlock()
		if tokens > 0 {
			tokens--
			fn()
			return true
		} else {
			backlog = true
			return false
		}
	}
}
