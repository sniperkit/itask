package xtask

import (
	"time"
)

// EnqueueFunc schedules a task to run a certain amount of time from the current time. This allows us to schedule tasks to run in intervals.
func (tq *TaskQueue) EnqueueFunc(name string, fn interface{}, args ...interface{}) *TaskQueue {
	// tq.lock.Lock()
	// defer tq.lock.Unlock()

	task := NewTask(name, fn, args...)
	task.nextRun = time.Now()
	go tq.enqueue(task)

	return tq
}

// EnqueueFuncEvery schedules a task to run and reschedule itself on a regular interval. It works like EnqueueFuncIn but repeats
func (tq *TaskQueue) EnqueueFuncEvery(name string, period time.Duration, fn interface{}, args ...interface{}) *TaskQueue {
	// tq.lock.Lock()
	// defer tq.lock.Unlock()

	task := NewTask(name, fn, args...)
	task.nextRun = time.Now().Add(period)
	task.interval = period
	task.repeat = true
	go tq.enqueue(task)

	return tq
}

// EnqueueFuncIn schedules a task to run a certain amount of time from the current time. This allows us to schedule tasks to run in intervals.
func (tq *TaskQueue) EnqueueFuncIn(name string, period time.Duration, task *Task) *TaskQueue {
	// tq.lock.Lock()
	// defer tq.lock.Unlock()

	task.nextRun = time.Now().Add(period)
	go tq.enqueue(task)

	return tq
}

// EnqueueAt schedules a task to run at a certain time in the future.
func (tq *TaskQueue) EnqueueFuncAt(name string, period time.Time, fn interface{}, args ...interface{}) *TaskQueue {
	// tq.lock.Lock()
	// defer tq.lock.Unlock()

	task := NewTask(name, fn, args...)
	task.nextRun = period
	go tq.enqueue(task)

	return tq
}
