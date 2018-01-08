package main

import (
	"math/rand"
	"strings"
	"time"

	"github.com/sniperkit/xtask/plugin/counter"
)

// delay requests
var (
	requestDelay   time.Duration = 350 * time.Millisecond
	workerInterval time.Duration = time.Duration(random(150, 250)) * time.Millisecond
)

func mapFunc(funcName string, aliasName string) {}

// task counters
var (
	counterAsync map[string]int = make(map[string]int)
	counters     *counter.Oc    = counter.NewOc()
)

func printTasksInfo() {
	log.Println("tasks stats: ", getTasksInfo())
}

func getTasksInfo() map[string]int {
	info := make(map[string]int)
	counters.SortByKey(counter.ASC)
	for counters.Next() {
		if counters != nil {
			key, value := counters.KeyValue()
			info[key] = value
		}
	}
	return info
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
