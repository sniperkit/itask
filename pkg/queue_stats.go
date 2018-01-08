package xtask

import (
	//"log"

	// "github.com/rcrowley/go-metrics"
	"github.com/sniperkit/xtask/plugin/stats/tachymeter"
	// "github.com/sniperkit/xtask/plugin/stats/collection"
)

func (tq *TaskQueue) Tachymeter(size int, safe bool, hbuckets int) *TaskQueue {
	tq.lock.Lock()
	defer tq.lock.Unlock()

	if tq.results == nil {
		tq.results = &TaskQueueResult{}
	}

	tq.tachymeter = tachymeter.New(
		&tachymeter.Config{
			Size:     size,
			Safe:     safe,
			HBuckets: hbuckets,
		})

	return tq
}

func (tq *TaskQueue) StatsCollection(collectors []string) *TaskQueue {
	tq.lock.Lock()
	defer tq.lock.Unlock()

	return tq
}

//
func (tq *TaskQueue) iostats() *TaskQueue {
	tq.lock.Lock()
	defer tq.lock.Unlock()

	return tq
}

func (tq *TaskQueue) netstats() *TaskQueue {
	tq.lock.Lock()
	defer tq.lock.Unlock()

	return tq
}

func (tq *TaskQueue) procstats() *TaskQueue {
	tq.lock.Lock()
	defer tq.lock.Unlock()

	return tq
}

func (tq *TaskQueue) httpstats() *TaskQueue {
	tq.lock.Lock()
	defer tq.lock.Unlock()

	return tq
}

func (tq *TaskQueue) redisStats() *TaskQueue {
	tq.lock.Lock()
	defer tq.lock.Unlock()

	return tq
}
