package seal

import (
	"sync"
	"time"
)

// Limit returns a function which calls fn up to n times per interval.
// Calls to fn are synchronized.
// If the limit is exceeded, a call is scheduled for the beginning of the next interval.
// The returned function returns true if the call has been executed, false if it has been scheduled.
// If a call is scheduled, the error value of the last execution is returned.
func Limit(interval time.Duration, n int, fn func() error) func() (bool, error) {
	var tokens = n
	var backlog = false
	var err error
	var lock sync.Mutex
	go func() {
		for ; true; <-time.Tick(interval) {
			lock.Lock()
			tokens = n // refill
			if backlog {
				backlog = false
				tokens--
				err = fn()
			}
			lock.Unlock()
		}
	}()
	return func() (bool, error) {
		lock.Lock()
		defer lock.Unlock()
		if tokens > 0 {
			tokens--
			err = fn()
			return true, err
		} else {
			backlog = true
			return false, err
		}
	}
}
