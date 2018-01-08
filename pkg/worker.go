package xtask

import (
	"log"
	"os"
	"reflect"
	"time"

	uuid "github.com/satori/go.uuid"
)

// Worker represents a background worker which picks up tasks and communicates its progress on its set channels
type Worker struct {
	id       int
	hash     uuid.UUID
	tasks    TaskQueue
	workers  chan *Worker
	complete chan *Task
}

// ProcessTask takes a task and does the work on it.
func (w *Worker) ProcessTask(t *Task, g *TaskQueue) {
	if g != nil {
		g.LogTaskStarted(w, t)
	}

	if t.name == "" {
		t.name = t.hash.String()
	}

	fn := reflect.ValueOf(t.fn)
	fnType := fn.Type()
	if fnType.Kind() != reflect.Func && fnType.NumIn() != len(t.args) {
		// log.Panic("Expected a function")
		log.Print("Expected a function")
		os.Exit(1)
	}

	var args []reflect.Value
	for _, arg := range t.args {
		args = append(args, reflect.ValueOf(arg))
	}

	res := fn.Call(args)
	for _, val := range res {
		log.Println("Response:", val.Interface())
	}

	if t.repeat {
		if g != nil {
			g.EnqueueFuncEvery(t.name, t.interval, t.fn, t.args)
		}
	}

	w.complete <- t
	if g != nil {
		g.LogTaskFinished(w, t)
	}
}

// Sleep pauses the worker before its next run
func (w *Worker) Sleep(g *TaskQueue) {
	if g != nil {
		g.LogWorkerSleeping(w)
	}
	time.Sleep(time.Duration(g.pipeline.WorkInterval) * time.Second)
	w.workers <- w
}
