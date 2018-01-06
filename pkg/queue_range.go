package xtask

import (
	"time"
)

func (tlist *TaskGroup) EnqueueRange(name string, tasks ...*Task) *TaskGroup {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()

	for _, task := range tasks {
		task.nextRun = time.Now()
		go tlist.enqueue(task)
	}

	return tlist
}

func (tlist *TaskGroup) EnqueueRangeEvery(name string, period time.Duration, tasks ...*Task) *TaskGroup {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()

	for _, task := range tasks {
		task.nextRun = time.Now().Add(period)
		task.interval = period
		task.repeat = true
		go tlist.enqueue(task)
	}

	return tlist
}

func (tlist *TaskGroup) EnqueueRangeIn(name string, period time.Duration, tasks ...*Task) *TaskGroup {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()

	for _, task := range tasks {
		task.nextRun = time.Now().Add(period)
		task.interval = period
		task.repeat = true
		go tlist.enqueue(task)
	}

	return tlist
}

func (tlist *TaskGroup) EnqueueRangeAt(name string, period time.Duration, tasks ...*Task) *TaskGroup {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()

	for _, task := range tasks {
		task.nextRun = time.Now().Add(period)
		task.interval = period
		task.repeat = true
		go tlist.enqueue(task)
	}

	return tlist
}
