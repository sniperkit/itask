package xtask

import (
	"log"
	"math/rand"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/sniperkit/xtask/plugin/rate"
	"github.com/sniperkit/xtask/plugin/rate/cpu"
	// "github.com/sniperkit/xtask/plugin/rate/max"
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

func (tq *TaskQueue) Limiter(limit int, interval time.Duration) *TaskQueue {
	tq.lock.Lock()
	defer tq.lock.Unlock()

	tq.counters.Increment("set.limiter.default", 1)
	tq.rate = rate.New(limit, interval)
	return tq
}

func (tq *TaskQueue) LimiterWithKey(limit int, interval time.Duration, key string) string {
	tq.lock.Lock()
	defer tq.lock.Unlock()
	tq.counters.Increment("add.limiter", 1)

	return tq.limiter.Add(limit, interval, key)
}

func (tq *TaskQueue) CPU() *TaskQueue {
	tq.lock.Lock()
	defer tq.lock.Unlock()

	tq.counters.Increment("set.limiter.cpu", 1)
	lcpu, err := cpu.New(nil)
	if err != nil {
		log.Fatal("could not instanciate the cpu limiter")
		return tq
	}
	tq.limiter.cpu = lcpu
	return tq
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
