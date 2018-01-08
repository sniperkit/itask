package xtask

import (
	"time"
)

// EnqueueTask schedules a task to run as soon as the next worker is available.
func (tq *TaskQueue) EnqueueTask(task *Task) *Task {
	task.nextRun = time.Now()
	go tq.enqueue(task)
	return task
}

// EnqueueTaskIn schedules a task to run a certain amount of time from the current time. This allows us to schedule tasks to run in intervals.
func (tq *TaskQueue) EnqueueTaskIn(name string, period time.Duration, task *Task) *TaskQueue {
	task.nextRun = time.Now().Add(period)
	go tq.enqueue(task)
	return tq
}

// EnqueueTaskAt schedules a task to run at a certain time in the future.
func (tq *TaskQueue) EnqueueTaskAt(name string, period time.Time, task *Task) *TaskQueue {
	task.nextRun = period
	go tq.enqueue(task)
	return tq
}

// EnqueueTaskEvery schedules a task to run and reschedule itself on a regular interval. It works like EnqueueIn but repeats
func (tq *TaskQueue) EnqueueTaskEvery(name string, period time.Duration, task *Task) *TaskQueue {
	task.nextRun = time.Now().Add(period)
	task.interval = period
	task.repeat = true
	go tq.enqueue(task)
	return tq
}
