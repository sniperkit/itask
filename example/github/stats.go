package main

import (
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/perriv/go-tasker"
	"github.com/segmentio/stats"

	"github.com/sniperkit/xtask/plugin/counter"
)

var (
	requestDelay   time.Duration  = 350 * time.Millisecond
	workerInterval time.Duration  = time.Duration(random(150, 250)) * time.Millisecond
	counterAsync   map[string]int = make(map[string]int)
	counters       *counter.Oc    = counter.NewOc()
	tr             *tasker.Tasker
)

func showStats() {
	log.Println("tasks stats: ", getStats())
}

func getStats() map[string]int {
	stats := make(map[string]int)
	counters.SortByKey(counter.ASC)
	for counters.Next() {
		if counters != nil {
			key, value := counters.KeyValue()
			stats[key] = value
		}
	}
	return stats
}

func updateRequestDelay(beat int, unit string) (delay time.Duration) {

	input := time.Duration(beat)

	switch strings.ToLower(unit) {
	case "microsecond":
		delay = input * time.Microsecond
	case "millisecond":
		delay = input * time.Millisecond
	case "minute":
		delay = input * time.Minute
	case "hour":
		delay = input * time.Hour
	case "second":
		fallthrough
	default:
		delay = input * time.Second
	}
	return
}

func random(min, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}

type taskMetrics struct {
	calls struct {
		count int           `metric:"count" type:"counter"`
		time  time.Duration `metric:"time"  type:"histogram"`
	} `metric:"func.calls"`
}

type funcMetrics struct {
	calls struct {
		count  int           `metric:"count" type:"counter"`
		failed int           `metric:"failed" type:"counter"`
		time   time.Duration `metric:"time"  type:"histogram"`
	} `metric:"func.calls"`
}

func addMetrics(start time.Time, incr int, failed bool) {
	callTime := time.Now().Sub(start)
	m := &funcMetrics{}
	m.calls.count = incr
	m.calls.time = callTime
	if failed {
		m.calls.failed = incr
	}
	stats.Report(m)
}
