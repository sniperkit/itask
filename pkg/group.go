package xtask

import (
	"container/list" // Package list implements a doubly linked list.
	"math/rand"
	"sync"
	"time"
	// "sync/atomic"

	uuid "github.com/satori/go.uuid"
	"github.com/sniperkit/xtask/plugin/counter"
	"github.com/sniperkit/xtask/plugin/rate"
	"github.com/sniperkit/xtask/plugin/tachymeter"
)

//

// TaskGroup is a threadsafe container for tasks to be processed
type TaskGroup struct {
	name       string
	id         int
	hash       uuid.UUID
	list       *list.List
	lock       *sync.RWMutex
	state      *State
	pipeline   *Pipeline
	registry   map[uuid.UUID]bool
	limiter    *Limiter
	counters   *counter.Oc
	completed  int
	rate       *rate.RateLimiter
	tachymeter *tachymeter.Tachymeter
}

func NewTaskGroup() *TaskGroup {
	hash := uuid.NewV4()
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &TaskGroup{
		id:       random.Intn(10000),
		hash:     hash,
		limiter:  NewLimiter(hash.String()),
		list:     list.New(),
		lock:     &sync.RWMutex{},
		counters: counter.NewOc(),
	}
}

func (tlist *TaskGroup) AddLimiter(limit int, interval time.Duration, key string) string {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()
	tlist.counters.Increment("add.limiter", 1)

	return tlist.limiter.Add(limit, interval, key)
}

// Push adds a new task into the front of the TaskGroup
func (tlist *TaskGroup) Len() int {
	tlist.lock.RLock()
	defer tlist.lock.RUnlock()

	return tlist.list.Len()
}

func (tlist *TaskGroup) Next() *Task {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()

	for element := tlist.list.Front(); element != nil; element = element.Next() {
		if task, ok := element.Value.(*Task); ok && !task.isCompleted {
			//if time.Since(task.nextRun) > 0 {
			return task
			//}
		}
	}
	return nil
}

// Get checks if a key exists in our dequeued task list
func (tlist *TaskGroup) FindByUUID(hash uuid.UUID) bool {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()

	if _, ok := tlist.registry[hash]; ok {
		return true
	}
	return false
}

// Remove deletes the dequeued entry once we are done with it
func (tlist *TaskGroup) RemoveByUUID(hash uuid.UUID) {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()

	delete(tlist.registry, hash)
}

/*
// Push adds a new task into the front of the TaskGroup
func (tlist *TaskGroup) Delete(task *Task) error {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()

	// err := tlist.list.Remove(task)
	// tlist.pipeline.Remove <- true
	// return err
	return tlist.list.Remove(task)
}
*/

// Push adds a new task into the front of the TaskGroup
func (tlist *TaskGroup) Push(task *Task) {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()
	tlist.counters.Increment("push", 1)

	tlist.list.PushFront(task)
	// tlist.pipeline.New <- true
}

// Pop grabs the last task from the TaskGroup
func (tlist *TaskGroup) Pop() *Task {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()
	tlist.counters.Increment("pop", 1)

	task := tlist.list.Remove(tlist.list.Back())
	return task.(*Task)
}

// Add
func (tlist *TaskGroup) AddTask(task *Task) *TaskGroup {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()

	tlist.counters.Increment("add.task", 1)
	task.nextRun = time.Now()
	tlist.list.PushFront(task)
	return tlist
}

// AddFunc
func (tlist *TaskGroup) AddFunc(name string, fn interface{}, args ...interface{}) *TaskGroup {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()

	task := NewTask(name, fn, args...)
	task.nextRun = time.Now()
	tlist.list.PushFront(task)

	tlist.counters.Increment("add.func", 1)
	return tlist
}

// AddRange
func (tlist *TaskGroup) AddRange(tasks ...*Task) *TaskGroup {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()
	tlist.counters.Increment("add.range", 1)

	for _, task := range tasks {
		tlist.list.PushFront(task)
	}
	return tlist
}

// RunTaskAsync
func (tlist *TaskGroup) RunTaskAsync() *TaskGroup {
	for element := tlist.list.Front(); element != nil; element = element.Next() {
		if task, ok := element.Value.(*Task); ok && !task.isCompleted {
			task.RunAsync()
		}
	}
	return tlist
}

// WaitAll
func (tlist *TaskGroup) WaitAll() {
	for element := tlist.list.Front(); element != nil; element = element.Next() {
		if task, ok := element.Value.(*Task); ok && !task.isCompleted {
			task.wait.Wait()
		}
	}
}
