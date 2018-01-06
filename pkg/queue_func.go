package xtask

import (
	"time"
)

// EnqueueFunc schedules a task to run a certain amount of time from the current time. This allows us to schedule tasks to run in intervals.
func (tlist *TaskGroup) EnqueueFunc(name string, fn interface{}, args ...interface{}) *TaskGroup {
	// tlist.lock.Lock()
	// defer tlist.lock.Unlock()

	task := NewTask(name, fn, args...)
	task.nextRun = time.Now()
	go tlist.enqueue(task)

	return tlist
}

// EnqueueFuncEvery schedules a task to run and reschedule itself on a regular interval. It works like EnqueueFuncIn but repeats
func (tlist *TaskGroup) EnqueueFuncEvery(name string, period time.Duration, fn interface{}, args ...interface{}) *TaskGroup {
	// tlist.lock.Lock()
	// defer tlist.lock.Unlock()

	task := NewTask(name, fn, args...)
	task.nextRun = time.Now().Add(period)
	task.interval = period
	task.repeat = true
	go tlist.enqueue(task)

	return tlist
}

// EnqueueFuncIn schedules a task to run a certain amount of time from the current time. This allows us to schedule tasks to run in intervals.
func (tlist *TaskGroup) EnqueueFuncIn(name string, period time.Duration, task *Task) *TaskGroup {
	// tlist.lock.Lock()
	// defer tlist.lock.Unlock()

	task.nextRun = time.Now().Add(period)
	go tlist.enqueue(task)

	return tlist
}

// EnqueueAt schedules a task to run at a certain time in the future.
func (tlist *TaskGroup) EnqueueFuncAt(name string, period time.Time, fn interface{}, args ...interface{}) *TaskGroup {
	// tlist.lock.Lock()
	// defer tlist.lock.Unlock()

	task := NewTask(name, fn, args...)
	task.nextRun = period
	go tlist.enqueue(task)

	return tlist
}
