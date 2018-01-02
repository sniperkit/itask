package xtask

import (
	"container/list"
	// "log"
	"sync"
	// "time"
	// "github.com/sniperkit/xtask/pkg/rate"
)

// TaskList is a threadsafe container for tasks to be processed
type TaskList struct {
	list       *list.List
	lock       *sync.Mutex
	numWorkers int
	numRunning int
	// workers    chan struct{}
	// slotNum int
	// s       *slots
	// taskHoler *taskHolder
	// locker    sync.RWMutex
}

func NewTaskList(workers int) *TaskList {
	if workers < 1 {
		workers = 1
	}
	return &TaskList{
		list:       list.New(),
		lock:       &sync.Mutex{},
		numWorkers: workers,
		numRunning: 0,
	}
}

// Add
func (tlist *TaskList) Add(task *Task) *TaskList {
	tlist.list.PushFront(task)
	return tlist
}

// AddRange
func (tlist *TaskList) AddRange(tasks ...*Task) *TaskList {
	for _, task := range tasks {
		tlist.list.PushFront(task)
	}
	return tlist
}

// Push adds a new task into the front of the TaskList
func (tlist *TaskList) Len() int {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()

	return tlist.list.Len()
}

// Push adds a new task into the front of the TaskList
func (tlist *TaskList) Push(t *Task) {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()

	tlist.list.PushFront(t)

	AppConfig.NewTasks <- true
}

// Pop grabs the last task from the TaskList
func (tlist *TaskList) Pop() *Task {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()

	task := tlist.list.Remove(tlist.list.Back())
	return task.(*Task)
}

// Run
func (tlist *TaskList) Run() *TaskList {
	for element := tlist.list.Front(); element != nil; element = element.Next() {
		if task, ok := element.Value.(*Task); ok && !task.isCompleted {
			task.Run()
		}
	}
	return tlist
}

// WaitAll
func (tlist *TaskList) WaitAll() {
	for element := tlist.list.Front(); element != nil; element = element.Next() {
		if task, ok := element.Value.(*Task); ok && !task.isCompleted {
			task.wait.Wait()
		}
	}
}
