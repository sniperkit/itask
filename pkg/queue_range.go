package xtask

import (
	"time"
)

func (tq *TaskQueue) EnqueueRange(name string, tasks ...*Task) *TaskQueue {
	tq.lock.Lock()
	defer tq.lock.Unlock()

	for _, task := range tasks {
		task.nextRun = time.Now()
		go tq.enqueue(task)
	}

	return tq
}

func (tq *TaskQueue) EnqueueRangeEvery(name string, period time.Duration, tasks ...*Task) *TaskQueue {
	tq.lock.Lock()
	defer tq.lock.Unlock()

	for _, task := range tasks {
		task.nextRun = time.Now().Add(period)
		task.interval = period
		task.repeat = true
		go tq.enqueue(task)
	}

	return tq
}

func (tq *TaskQueue) EnqueueRangeIn(name string, period time.Duration, tasks ...*Task) *TaskQueue {
	tq.lock.Lock()
	defer tq.lock.Unlock()

	for _, task := range tasks {
		task.nextRun = time.Now().Add(period)
		task.interval = period
		task.repeat = true
		go tq.enqueue(task)
	}

	return tq
}

func (tq *TaskQueue) EnqueueRangeAt(name string, period time.Duration, tasks ...*Task) *TaskQueue {
	tq.lock.Lock()
	defer tq.lock.Unlock()

	for _, task := range tasks {
		task.nextRun = time.Now().Add(period)
		task.interval = period
		task.repeat = true
		go tq.enqueue(task)
	}

	return tq
}
