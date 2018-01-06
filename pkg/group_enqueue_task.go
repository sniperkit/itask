package xtask

import (
	"time"
)

// EnqueueTask schedules a task to run as soon as the next worker is available.
func (tlist *TaskGroup) EnqueueTask(task *Task) *Task {
	task.nextRun = time.Now()
	go tlist.enqueue(task)
	return task
}

// EnqueueTaskIn schedules a task to run a certain amount of time from the current time. This allows us to schedule tasks to run in intervals.
func (tlist *TaskGroup) EnqueueTaskIn(name string, period time.Duration, task *Task) *TaskGroup {
	task.nextRun = time.Now().Add(period)
	go tlist.enqueue(task)
	return tlist
}

// EnqueueTaskAt schedules a task to run at a certain time in the future.
func (tlist *TaskGroup) EnqueueTaskAt(name string, period time.Time, task *Task) *TaskGroup {
	task.nextRun = period
	go tlist.enqueue(task)
	return tlist
}

// EnqueueTaskEvery schedules a task to run and reschedule itself on a regular interval. It works like EnqueueIn but repeats
func (tlist *TaskGroup) EnqueueTaskEvery(name string, period time.Duration, task *Task) *TaskGroup {
	task.nextRun = time.Now().Add(period)
	task.interval = period
	task.repeat = true
	go tlist.enqueue(task)
	return tlist
}
