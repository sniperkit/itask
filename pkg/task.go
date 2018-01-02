package xtask

import (
	"container/list"
	"log"
	"math/rand"
	"reflect"
	"sync"
	"time"

	uuid "github.com/satori/go.uuid"
)

// TaskParameter
type TaskParameter interface{}

// TaskHanlder
type TaskHanlder interface{}

// ContinueWithHandler
type ContinueWithHandler func(TaskResult)

// TaskResult
type TaskResult struct {
	Result interface{}
	Error  error
}

// Task represents a task to run. It can be scheduled to run later or right away.
type Task struct {
	id           int
	name         string
	fn           interface{}
	args         []interface{}
	hash         uuid.UUID
	nextRun      time.Time
	interval     time.Duration
	repeat       bool
	wait         *sync.WaitGroup
	handler      reflect.Value
	params       []reflect.Value
	once         sync.Once
	continueWith *list.List
	delay        time.Duration

	Result      TaskResult
	isCompleted bool
}

// Run
func (task *Task) Run() *Task {
	task.once.Do(func() {
		task.wait.Add(1)
		if task.delay.Nanoseconds() > 0 {
			time.Sleep(task.delay)
		}

		go func() {
			defer func() {
				task.isCompleted = true
				if task.continueWith != nil {
					result := task.Result
					for element := task.continueWith.Back(); element != nil; element = element.Prev() {
						if tt, ok := element.Value.(ContinueWithHandler); ok {
							tt(result)
						}

					}
				}
				task.wait.Done()
			}()
			values := task.handler.Call(task.params)
			task.Result = TaskResult{
				Result: values,
			}

		}()
	})
	return task
}

// Wait
func (task *Task) Wait() {
	task.wait.Wait()
}

// ContinueWith
func (task *Task) ContinueWith(handler ContinueWithHandler) *Task {
	task.continueWith.PushFront(handler)
	return task
}

// Delay
func (task *Task) Delay(delay time.Duration) *Task {
	task.delay = delay
	return task
}

// NewTask
func NewTask(handler TaskHanlder, params ...TaskParameter) *Task {
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	handlerValue := reflect.ValueOf(handler)
	if handlerValue.Kind() == reflect.Func {
		task := Task{
			id:           random.Intn(10000),
			hash:         uuid.NewV4(),
			wait:         &sync.WaitGroup{},
			handler:      handlerValue,
			repeat:       false,
			continueWith: list.New(),
			delay:        0 * time.Second,
			params:       make([]reflect.Value, 0),
			isCompleted:  false,
		}
		if paramNum := len(params); paramNum > 0 {
			task.params = make([]reflect.Value, paramNum)
			for index, v := range params {
				task.params[index] = reflect.ValueOf(v)
				log.Println("param: key=", index, ", value=", reflect.ValueOf(v))
			}
		}
		return &task
	}
	panic("handler not func")
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
func StartNew(handler TaskHanlder, params ...TaskParameter) *Task {
	task := NewTask(handler, params)
	task.Run()
	return task
}

// enqueue is an internal function used to asynchronously push a task onto the
// queue and log the state to the terminal.
func enqueue(task *Task) {
	AppConfig.ScheduledTasks.Push(task)
	LogTaskScheduled(task)
}

// Enqueue schedules a task to run as soon as the next worker is available.
func Enqueue(handler TaskHanlder, params ...TaskParameter) *Task {
	task := NewTask(handler, params)
	task.nextRun = time.Now()
	go enqueue(task)
	return task
}

// EnqueueIn schedules a task to run a certain amount of time from the current time. This allows us to schedule tasks to run in intervals.
func EnqueueIn(period time.Duration, handler TaskHanlder, params ...TaskParameter) *Task {
	task := NewTask(handler, params)
	task.nextRun = time.Now().Add(period)
	go enqueue(task)
	return task
}

// EnqueueAt schedules a task to run at a certain time in the future.
func EnqueueAt(period time.Time, handler TaskHanlder, params ...TaskParameter) *Task {
	task := NewTask(handler, params)
	task.nextRun = period
	go enqueue(task)
	return task
}

// EnqueueEvery schedules a task to run and reschedule itself on a regular interval. It works like EnqueueIn but repeats
func EnqueueEvery(period time.Duration, handler TaskHanlder, params ...TaskParameter) *Task {
	task := NewTask(handler, params)
	task.nextRun = time.Now().Add(period)
	task.interval = period
	task.repeat = true
	go enqueue(task)
	return task
}

/*
// Enqueue schedules a task to run as soon as the next worker is available.
// func Enqueue(fn interface{}, args ...interface{}) uuid.UUID {
func Enqueue(handler TaskHanlder, params ...TaskParameter) uuid.UUID {
	task := NewTask(handler, params)
	task.nextRun = time.Now()
	go enqueue(task)
	return task.hash
}

// EnqueueIn schedules a task to run a certain amount of time from the current time. This allows us to schedule tasks to run in intervals.
// func EnqueueIn(period time.Duration, fn interface{}, args ...interface{}) uuid.UUID {
func EnqueueIn(period time.Duration, handler TaskHanlder, params ...TaskParameter) uuid.UUID {
	task := NewTask(handler, params)
	task.nextRun = time.Now().Add(period)
	go enqueue(task)
	return task.hash
}

// EnqueueAt schedules a task to run at a certain time in the future.
// func EnqueueAt(period time.Time, fn interface{}, args ...interface{}) uuid.UUID {
func EnqueueAt(period time.Time, handler TaskHanlder, params ...TaskParameter) uuid.UUID {
	task := NewTask(handler, params)
	task.nextRun = period
	go enqueue(task)
	return task.hash
}

// EnqueueEvery schedules a task to run and reschedule itself on a regular interval. It works like EnqueueIn but repeats
// func EnqueueEvery(period time.Duration, fn interface{}, args ...interface{}) uuid.UUID {
func EnqueueEvery(period time.Duration, handler TaskHanlder, params ...TaskParameter) uuid.UUID {
	task := NewTask(handler, params)
	task.nextRun = time.Now().Add(period)
	task.interval = period
	task.repeat = true
	go enqueue(task)
	return task.hash
}
*/
