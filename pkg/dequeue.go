package xtask

import (
	"errors"
	"sync"

	uuid "github.com/satori/go.uuid"
)

// TaskDequeue is a threadsafe container for tasks to be dequeued
type TaskDequeue struct {
	list map[uuid.UUID]bool
	lock *sync.Mutex
}

// NewDequeue returns a new instance of a TaskQueue
func NewDequeue() TaskDequeue {
	return TaskDequeue{
		list: make(map[uuid.UUID]bool),
		lock: &sync.Mutex{},
	}
}

// Enqueue schedules a task to run as soon as the next worker is available.
func Dequeue(hash uuid.UUID) {
	go func() {
		if _, err := AppConfig.CancelledTasks.Push(hash); err != nil {
			// @TODO handle this properly
			panic(err)
		}
	}()
}

// Push adds a new task into the front of the TaskQueue
func (q *TaskDequeue) Push(hash uuid.UUID) (uuid.UUID, error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	if _, ok := q.list[hash]; !ok {
		q.list[hash] = true
		return hash, nil
	}

	// @TODO use proper error
	return hash, errors.New("Task is already scheduled to be dequeued")
}

// Get checks if a key exists in our dequeued task list
func (q *TaskDequeue) Get(hash uuid.UUID) bool {
	q.lock.Lock()
	defer q.lock.Unlock()

	if _, ok := q.list[hash]; ok {
		return true
	}
	return false
}

// Remove deletes the dequeued entry once we are done with it
func (q *TaskDequeue) Remove(hash uuid.UUID) {
	q.lock.Lock()
	defer q.lock.Unlock()

	delete(q.list, hash)
}
