package limiter

import (
	"sync"
	"time"

	"github.com/sniperkit/xtask/plugin/rate/cache"
)

type secondRate struct {
	sec      int64
	rate     int
	baserate int
}

type limiter struct {
	sync.RWMutex
	m map[string]secondRate
}

func (l *limiter) add(k string, rate int) {
	l.Lock()
	if _, ok := l.m[k]; !ok {
		sr := secondRate{
			sec:      time.Now().Unix(),
			rate:     rate,
			baserate: rate,
		}
		l.m[k] = sr
	}
	l.Unlock()
}

func (l *limiter) upd(k string, rate int) {
	l.Lock()
	if v, ok := l.m[k]; ok {
		v.baserate = rate
		l.m[k] = v
	}
	l.Unlock()
}

func (l *limiter) get(k string) (secondRate, bool) {
	l.RLock()
	defer l.RUnlock()

	v, ok := l.m[k]
	return v, ok
}

func (l *limiter) del(k string) {
	l.Lock()
	delete(l.m, k)
	l.Unlock()
}

func (l *limiter) exist(k string) bool {
	l.RLock()
	defer l.RUnlock()
	_, ok := l.m[k]
	return ok
}

type RateLimiter struct {
	rls   *limiter
	cache *cache.Cache
}

func newlimiter() *limiter {
	return &limiter{
		m: make(map[string]secondRate),
	}
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		rls:   newlimiter(),
		cache: cache.NewCache(defExpireSecond),
	}
}

//add k element just when the k is not existed
func (l *RateLimiter) AddElement(k string, rate int) {
	l.rls.add(k, rate)
}

//update rate
func (l *RateLimiter) UpdElement(k string, rate int) {
	l.rls.upd(k, rate)
}

//check k element exist
func (l *RateLimiter) ExistElement(k string) bool {
	return l.rls.exist(k)
}

//delete
func (l *RateLimiter) DelElemnt(k string) {
	l.rls.del(k)
	l.cache.Del(k)
}

func (l *RateLimiter) Limit(k string) bool {
	ok := l.cache.Exist(k)
	sr, e := l.rls.get(k)
	if !e {
		if ok {
			l.cache.Del(k)
		}
		return true
	}
	if !ok {
		sr.rate -= 1
		sr.sec = time.Now().Unix()
		l.cache.Set(k, sr)
		return true
	}

	cb := func(exist bool, newValue interface{}, oldValue interface{}) (interface{}, bool) {
		nowSec := time.Now().Unix()
		sr := oldValue.(secondRate)

		flag := true
		if nowSec == sr.sec {
			if sr.rate > 0 {
				sr.rate -= 1
			} else {
				sr.rate = 0
				flag = false
			}
		} else {
			sr.sec = nowSec
			if exist {
				raw := newValue.(secondRate)
				sr.rate = raw.baserate - 1
			} else {
				sr.rate = sr.baserate - 1
			}
		}

		return sr, flag
	}

	return l.cache.UpdateAtomic(k, e, sr, cb)
}

func (l *RateLimiter) Close() {
	l.cache.Close()
}
