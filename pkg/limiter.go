package xtask

import (
	"log"
	"math/rand"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/sniperkit/xtask/plugin/rate"
	"github.com/sniperkit/xtask/plugin/rate/cpu"
	// "github.com/sniperkit/xtask/plugin/rate/limiter"
)

type Limiter struct {
	disabled   bool
	id         int
	name       string
	hash       uuid.UUID
	rates      map[string]*rate.RateLimiter
	throughput map[string]int
	cpu        *cpu.Limiter
}

func NewLimiter(name string) *Limiter {
	if name == "" {
		name = uuid.NewV4().String()
	}
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &Limiter{
		id:    random.Intn(10000),
		name:  name,
		hash:  uuid.NewV4(),
		rates: make(map[string]*rate.RateLimiter),
	}
}

func (tlist *TaskGroup) Limiter(limit int, interval time.Duration) *TaskGroup {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()

	tlist.counters.Increment("set.limiter.default", 1)
	tlist.rate = rate.New(limit, interval)
	return tlist
}

func (tlist *TaskGroup) CPU() *TaskGroup {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()

	tlist.counters.Increment("set.limiter.cpu", 1)
	lcpu, err := cpu.New(nil)
	if err != nil {
		log.Fatal("could not instanciate the cpu limiter")
		return tlist
	}
	tlist.limiter.cpu = lcpu
	return tlist
}

func (rl *Limiter) Add(limit int, interval time.Duration, key string) string {
	if key == "" {
		key = uuid.NewV4().String()
	}
	if len(rl.rates) <= 30 {
		rl.rates[key] = rate.New(limit, interval)
	}
	return key
}

func random(min, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}
