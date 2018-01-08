package rate

import (
	"container/list"
	"errors"
	// "log"
	"sync"
	"time"
)

//ErrClosed designates that a Limiter is already closed in calls to Push and Close.
var ErrClosed = errors.New("ratelimit: limiter already closed")

// A RateLimiter limits the rate at which an action can be performed.  It
// applies neither smoothing (like one could achieve in a token bucket system)
// nor does it offer any conception of warmup, wherein the rate of actions
// granted are steadily increased until a steady throughput equilibrium is
// reached.
type RateLimiter struct {
	limit    int
	interval time.Duration
	mtx      sync.Mutex
	times    list.List
	// values   chan interface{}
	// nextTime time.Time
}

// New creates a new rate limiter for the limit and interval.
func New(limit int, interval time.Duration) *RateLimiter {
	lim := &RateLimiter{
		limit:    limit,
		interval: interval,
	}
	lim.times.Init()
	return lim
}

// Wait blocks if the rate limit has been reached.  Wait offers no guarantees
// of fairness for multiple actors if the allowed rate has been temporarily
// exhausted.
func (r *RateLimiter) Wait() {
	for {
		ok, remaining := r.Try()
		if ok {
			// log.Println("RateLimiter().Wait() remaining: ", remaining.Nanoseconds())
			break
		}
		// log.Println("RateLimiter().Wait().Sleep().remaining=", remaining.Seconds())
		time.Sleep(remaining)
	}
}

//Close closes l and prevents any more values from being pushed.
//Note that values not yet popped are still available to receive.
//
//If l is already closed, then ErrClosed is returned, otherwise err is nil.
func (r *RateLimiter) Close() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = ErrClosed
		}
	}()
	//close(r.values)
	return
}

// Try returns true if under the rate limit, or false if over and the
// remaining time before the rate limit expires.
func (r *RateLimiter) Try() (ok bool, remaining time.Duration) {
	r.mtx.Lock()
	defer r.mtx.Unlock()
	now := time.Now()
	if l := r.times.Len(); l < r.limit {
		r.times.PushBack(now)
		return true, 0
	}
	frnt := r.times.Front()
	if diff := now.Sub(frnt.Value.(time.Time)); diff < r.interval {
		return false, r.interval - diff
	}
	frnt.Value = now
	r.times.MoveToBack(frnt)
	return true, 0
}
