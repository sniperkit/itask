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

type TaskRecoverMsg struct {
	FuncName  string
	StartTime int64 // in nano seconds
	Err       interface{}
}

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

	// preHandlers  []*Handler // sync execute
	// handlers     []*Handler // can be sync or parallel
	// postHandlers []*Handler // sync execute
	// onRecover    func(*TaskRecoverMsg)
}

// ContinueWith
func (task *Task) ContinueWith(handler ContinueWithHandler) *Task {
	// task.lock.Lock()
	// defer task.lock.Unlock()

	task.continueWith.PushFront(handler)
	return task
}

// ContinueWithFunc
func (task *Task) ContinueWithFunc(name string, fn interface{}, args ...interface{}) *Task {
	// task.lock.Lock()
	// defer task.lock.Unlock()

	handler := NewTask(name, fn, args...)
	task.continueWith.PushFront(handler)
	return task
}

// ContinueWithTask
func (task *Task) ContinueWithTask(handler ContinueWithHandler) *Task {
	// task.lock.Lock()
	// defer task.lock.Unlock()

	task.continueWith.PushFront(handler)
	return task
}

// Delay
func (task *Task) Delay(delay time.Duration) *Task {
	// task.lock.Lock()
	// defer task.lock.Unlock()

	task.delay = delay
	return task
}

// Wait
func (task *Task) Wait() {
	// task.lock.Lock()
	// defer task.lock.Unlock()

	task.wait.Wait()
}

func (task *Task) RunInGroup(tq *TaskQueue) *Task {
	// task.lock.Lock()
	// defer task.lock.Unlock()

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

			if task.continueWith.Len() > 0 {
				// if task.continueWith != nil {
				task.counters.Increment("chained", 1)

				result := task.Result
				for element := task.continueWith.Back(); element != nil; element = element.Prev() {
					if tt, ok := element.Value.(ContinueWithHandler); ok {
						tt(result)
					}

				}
			}
			// tq.worker.complete <- task
			// tq.LogTaskFinished(tq.worker, task)
			task.wait.Done()
			tq.counters.Increment("completed", 1)
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
			tq.EnqueueFuncEvery(task.name, task.interval, task.fn, task.args)
			// tq.EnqueueTaskEvery(task)
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

			//if task.continueWith != nil {
			if task.continueWith.Len() > 0 {
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
	// task.lock.Lock()
	// defer task.lock.Unlock()

	return task.hash.String()
}

func (task *Task) SetUUID(input string) *Task {
	// task.lock.Lock()
	// defer task.lock.Unlock()

	var err error
	task.hash, err = uuid.FromString(input) // "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
	if err != nil {
		log.Printf("error while trying to parse uuuid fron string input: %s", err)
	}
	return task
}

// RunAsync
func (task *Task) RunAsync() *Task {
	// task.lock.Lock()
	// defer task.lock.Unlock()

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
				// if task.continueWith != nil {
				if task.continueWith.Len() > 0 {
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

func NewTaskOld(name string, fn interface{}, args ...interface{}) *Task {
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

// NewTask
func NewHandler(handler TaskHanlder, params ...TaskParameter) *Task {
	handlerValue := reflect.ValueOf(handler)
	if handlerValue.Kind() == reflect.Func {
		task := Task{
			wait:         &sync.WaitGroup{},
			handler:      handlerValue,
			isCompleted:  false,
			continueWith: list.New(),
			delay:        0 * time.Second,
			params:       make([]reflect.Value, 0),
		}
		if paramNum := len(params); paramNum > 0 {
			task.params = make([]reflect.Value, paramNum)
			for index, v := range params {
				log.Println(index)
				task.params[index] = reflect.ValueOf(v)
			}
		}
		return &task
	}
	panic("handler not func")
}

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

func NewFunc(name string, fn interface{}, args ...interface{}) *Task {
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

// to cleanup!!
// NewTaskStart
func NewTaskStart(name string, fn interface{}, args ...interface{}) *Task {
	// task := NewTask(handler, params)
	task := NewFunc(name, fn, args...)
	task.RunAsync()
	return task
}

// NewFuncStart
func NewFuncStart(name string, fn interface{}, args ...interface{}) *Task {
	// task := NewTask(handler, params)
	task := NewFunc(name, fn, args...)
	task.RunAsync()
	return task
}

// NewHandlerStart
func NewHandlerStart(name string, handler TaskHanlder, params ...TaskParameter) *Task {
	task := NewHandler(name, handler, params)
	task.Run()
	return task
}
