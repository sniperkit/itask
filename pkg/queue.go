package xtask

import (
	"container/list"
	"sync"
	"time"
)

const defaultSlotNum = 512
const defaultDuration = time.Second

// TaskQueue is a threadsafe container for tasks to be processed
type TaskQueue struct {
	list       *list.List
	lock       *sync.Mutex
	numWorkers int
}

// NewQueue returns a new instance of a TaskQueue
func NewQueue(workers int) TaskQueue {
	return TaskQueue{
		list:       list.New(),
		lock:       &sync.Mutex{},
		numWorkers: workers,
	}
}

// Push adds a new task into the front of the TaskQueue
func (q *TaskQueue) Len() int {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.list.Len()
}

// Push adds a new task into the front of the TaskQueue
func (q *TaskQueue) Push(t *Task) {
	q.lock.Lock()
	defer q.lock.Unlock()

	q.list.PushFront(t)
	AppConfig.NewTasks <- true
}

// Pop grabs the last task from the TaskQueue
func (q *TaskQueue) Pop() *Task {
	q.lock.Lock()
	defer q.lock.Unlock()

	task := q.list.Remove(q.list.Back())
	return task.(*Task)
}
