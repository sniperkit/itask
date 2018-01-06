package xtask

import (
	//"log"

	"github.com/sniperkit/xtask/plugin/stats/tachymeter"
	// "github.com/sniperkit/xtask/plugin/stats/collection"
)

func (tlist *TaskGroup) Tachymeter(size int, safe bool, hbuckets int) *TaskGroup {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()

	if tlist.results == nil {
		tlist.results = &TaskGroupResult{}
	}

	tlist.tachymeter = tachymeter.New(
		&tachymeter.Config{
			Size:     size,
			Safe:     safe,
			HBuckets: hbuckets,
		})

	return tlist
}

func (tlist *TaskGroup) StatsCollection(collectors []string) *TaskGroup {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()

	return tlist
}

//
func (tlist *TaskGroup) iostats() *TaskGroup {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()

	return tlist
}

func (tlist *TaskGroup) netstats() *TaskGroup {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()

	return tlist
}

func (tlist *TaskGroup) procstats() *TaskGroup {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()

	return tlist
}

func (tlist *TaskGroup) httpstats() *TaskGroup {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()

	return tlist
}

func (tlist *TaskGroup) redisStats() *TaskGroup {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()

	return tlist
}
