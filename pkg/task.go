package xtask

import (
	"container/list"
	"fmt"
	"log"
	"math/rand"
	"os"
	"reflect"
	"sync"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/sniperkit/xtask/plugin/counter"
	"github.com/sniperkit/xtask/plugin/rate"
)

// TaskParameter
type TaskParameter interface{}

// TaskHanlder
type TaskHanlder interface{}

// A Task is a function called in specified order by RunTasks(). It receives the queues configured context object to operate on.
type TaskQ func(ctx interface{}) error

// A Task is a function called in specified order by RunTasks(). It receives the queues configured context object to operate on.
// type Task func(ctx interface{}) error

// Task represents a task to run. It can be scheduled to run later or right away.
type Task struct {
	id           int
	name         string
	fn           interface{}
	args         []interface{}
	handler      reflect.Value
	params       []reflect.Value
	hash         uuid.UUID
	repeat       bool
	lock         *sync.RWMutex // Controls access to this task.
	wait         *sync.WaitGroup
	once         sync.Once
	continueWith *list.List
	delay        time.Duration
	nextRun      time.Time
	interval     time.Duration
	isCompleted  bool
	counters     *counter.Oc
	rate         *rate.RateLimiter
	Result       TaskResult
}

// ContinueWith
func (task *Task) ContinueWith(handler ContinueWithHandler) *Task {
	task.continueWith.PushFront(handler)
	return task
}

// ContinueWithFunc
func (task *Task) ContinueWithFunc(name string, fn interface{}, args ...interface{}) *Task {
	handler := NewTask(name, fn, args...)
	task.continueWith.PushFront(handler)
	return task
}

// ContinueWithTask
func (task *Task) ContinueWithTask(handler ContinueWithHandler) *Task {
	task.continueWith.PushFront(handler)
	return task
}

// Delay
func (task *Task) Delay(delay time.Duration) *Task {
	task.delay = delay
	return task
}

// Wait
func (task *Task) Wait() {
	task.wait.Wait()
}

func (task *Task) RunInGroup(tlist *TaskGroup) *Task {
	task.once.Do(func() {
		// Use context.Context to stop running goroutines
		// ctx, cancel := context.WithCancel(context.Background())
		// defer cancel()

		task.wait.Add(1)
		task.counters.Increment("started", 1)

		if task.delay.Nanoseconds() > 0 {
			task.counters.Increment("delayed", 1)
			time.Sleep(task.delay)
		}

		defer func() {
			task.isCompleted = true
			task.counters.Increment("completed", 1)

			if task.continueWith != nil {
				task.counters.Increment("chained", 1)

				result := task.Result
				for element := task.continueWith.Back(); element != nil; element = element.Prev() {
					if tt, ok := element.Value.(ContinueWithHandler); ok {
						tt(result)
					}

				}
			}
			// tlist.worker.complete <- task
			// tlist.LogTaskFinished(tlist.worker, task)
			task.wait.Done()
			tlist.counters.Increment("completed", 1)
		}()

		fn := reflect.ValueOf(task.fn)
		fnType := fn.Type()
		if fnType.Kind() != reflect.Func && fnType.NumIn() != len(task.args) {
			task.counters.Increment("unexpected", 1)
			// log.Panic("Expected a function")
			log.Print("Expected a function")
			os.Exit(1)
		}

		var args []reflect.Value
		for _, arg := range task.args {
			args = append(args, reflect.ValueOf(arg))
		}

		res := fn.Call(args)
		for _, val := range res {
			log.Println("Response:", val.Interface())
		}
		task.Result = TaskResult{
			Result: res,
		}

		if task.repeat {
			tlist.EnqueueFuncEvery(task.name, task.interval, task.fn, task.args)
			// tlist.EnqueueTaskEvery(task)
		}

	})
	return task
}

func (task *Task) Run() *Task {
	task.once.Do(func() {
		task.wait.Add(1)
		task.counters.Increment("started", 1)

		if task.delay.Nanoseconds() > 0 {
			task.counters.Increment("delayed", 1)
			time.Sleep(task.delay)
		}

		defer func() {
			task.isCompleted = true
			task.counters.Increment("completed", 1)
			if task.continueWith != nil {
				task.counters.Increment("chained", 1)
				result := task.Result
				for element := task.continueWith.Back(); element != nil; element = element.Prev() {
					if tt, ok := element.Value.(ContinueWithHandler); ok {
						tt(result)
					}

				}
			}
			task.wait.Done()
		}()

		fn := reflect.ValueOf(task.fn)
		fnType := fn.Type()
		if fnType.Kind() != reflect.Func && fnType.NumIn() != len(task.args) {
			task.counters.Increment("unexpected", 1)
			// log.Panic("Expected a function")
			log.Print("Expected a function")
			os.Exit(1)
		}

		var args []reflect.Value
		for _, arg := range task.args {
			args = append(args, reflect.ValueOf(arg))
		}

		res := fn.Call(args)
		for _, val := range res {
			log.Println("Response:", val.Interface())
		}
		task.Result = TaskResult{
			Result: res,
		}

	})
	return task
}

func (task *Task) GetUUID() string {
	return task.hash.String()
}

func (task *Task) SetUUID(input string) *Task {
	var err error
	task.hash, err = uuid.FromString(input) // "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
	if err != nil {
		log.Printf("error while trying to parse uuuid fron string input: %s", err)
	}
	return task
}

// RunAsync
func (task *Task) RunAsync() *Task {
	task.once.Do(func() {
		task.wait.Add(1)
		task.counters.Increment("started", 1)

		if task.delay.Nanoseconds() > 0 {
			task.counters.Increment("delayed", 1)
			time.Sleep(task.delay)
		}

		go func() {
			defer func() {
				task.isCompleted = true
				task.counters.Increment("completed", 1)
				if task.continueWith != nil {
					task.counters.Increment("chained", 1)
					result := task.Result
					for element := task.continueWith.Back(); element != nil; element = element.Prev() {
						if tt, ok := element.Value.(ContinueWithHandler); ok {
							tt(result)
						}

					}
				}
				task.wait.Done()
			}()

			fn := reflect.ValueOf(task.fn)
			fnType := fn.Type()
			if fnType.Kind() != reflect.Func && fnType.NumIn() != len(task.args) {
				// log.Panic("Expected a function")
				log.Print("Expected a function")
				os.Exit(1)
			}

			var args []reflect.Value
			for _, arg := range task.args {
				args = append(args, reflect.ValueOf(arg))
			}

			res := fn.Call(args)
			for _, val := range res {
				fmt.Println("Response:", val.Interface())
			}
			task.Result = TaskResult{
				Result: res,
			}

		}()
	})
	return task
}

// NewTask
func NewTask(name string, fn interface{}, args ...interface{}) *Task {
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	task := Task{
		id:           random.Intn(10000),
		hash:         uuid.NewV4(),
		wait:         &sync.WaitGroup{},
		lock:         &sync.RWMutex{},
		fn:           fn,
		args:         args,
		repeat:       false,
		continueWith: list.New(),
		delay:        0 * time.Second,
		isCompleted:  false,
		name:         name,
		nextRun:      time.Now(),
		counters:     counter.NewOc(),
		rate:         &rate.RateLimiter{},
	}

	return &task
}

// WaitAll
func WaitAll(tasks ...*Task) {
	wait := &sync.WaitGroup{}
	for _, task := range tasks {
		wait.Add(1)
		go func() {
			defer wait.Done()
			task.wait.Wait()
		}()
	}
	wait.Wait()
}

// StartNew
func StartNew(name string, fn interface{}, args ...interface{}) *Task {
	// task := NewTask(handler, params)
	task := NewTask(name, fn, args...)
	task.RunAsync()
	return task
}
