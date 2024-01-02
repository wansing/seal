package seal

import (
	"sync"
	"time"
)

// Limit returns a function which calls fn up to n times per interval.
// Calls to fn are synchronized.
// If the limit is exceeded, a call is scheduled for the beginning of the next interval.
// The returned function returns true if the call has been executed, false if it has been scheduled.
func Limit(interval time.Duration, n int, fn func()) func() bool {
	var tokens = n
	var backlog = false
	var lock sync.Mutex
	go func() {
		for ; true; <-time.Tick(interval) {
			lock.Lock()
			tokens = n // refill
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
