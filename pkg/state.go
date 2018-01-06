package xtask

import (
	"log"
	"time"

	"github.com/fatih/color"
)

type State struct {
	Pipeline PipelineState
	Worker   WorkerState
	Task     TaskState
	Limiter  LimiterState
}

type PipelineState struct {
	// Pipeline Channels
	Close    chan bool
	Started  chan *Worker
	Sleeping chan *Worker
}

type WorkerState struct {
	// Worker Channels
	Started  chan *Worker
	Sleeping chan *Worker
}

type TaskState struct {
	// Task Channels
	Scheduled chan *Task
	Expired   chan *Task
	Aborted   chan *Task
	Dequeued  chan *Task
	Started   chan map[*Worker]*Task
	Finished  chan map[*Worker]*Task
}

type LimiterState struct {
	// limiter channels
	Started chan *Limiter
	Waiting chan *Limiter
}

func (tlist *TaskGroup) Monitor() *TaskGroup {
	tlist.state = &State{}

	// Worker Channels
	tlist.state.Worker.Started = make(chan *Worker)
	tlist.state.Worker.Sleeping = make(chan *Worker)

	// Task Channels
	tlist.state.Task.Scheduled = make(chan *Task)
	tlist.state.Task.Aborted = make(chan *Task)
	tlist.state.Task.Expired = make(chan *Task)
	tlist.state.Task.Started = make(chan map[*Worker]*Task)
	tlist.state.Task.Finished = make(chan map[*Worker]*Task)

	return tlist
}

// LogTaskAbortd sends a signal to the TaskAbortd channel triggering the output text.
func (tlist *TaskGroup) LogTaskScheduled(t *Task) {
	tlist.state.Task.Scheduled <- t
}

// LogTaskAbortd sends a signal to the TaskAbortd channel triggering the output text.
func (tlist *TaskGroup) LogTaskAbortd(t *Task) {
	tlist.state.Task.Aborted <- t
}

// LogTaskStarted sends a signal to the TaskStarted channel triggering the output text.
func (tlist *TaskGroup) LogTaskStarted(w *Worker, t *Task) {
	tlist.state.Task.Started <- map[*Worker]*Task{w: t}
}

// LogTaskFinished sends a signal to the TaskFinished channel triggering the output text.
func (tlist *TaskGroup) LogTaskFinished(w *Worker, t *Task) {
	tlist.state.Task.Finished <- map[*Worker]*Task{w: t}
}

// LogWorkerSleeping sends a signal to the WorkerSleeping channel triggering the
// output text.
func (tlist *TaskGroup) LogWorkerSleeping(w *Worker) {
	tlist.state.Worker.Sleeping <- w
}

// StateMonitor provides a sane way to listen for state changes in the application. New state is passed via channels outputting logs from anywhere in the application.
func (tlist *TaskGroup) AsyncMonitor() {

	var stateMonitor bool
	for {
		select {
		case status := <-tlist.state.Pipeline.Close:
			stateMonitor = status
		case worker := <-tlist.state.Worker.Started:
			color.Set(color.Bold, color.FgBlue)
			log.Println("[WorkerStarted] Started Worker", worker.id)
			color.Unset()
		case worker := <-tlist.state.Worker.Sleeping:
			color.Set(color.Faint)
			log.Println("[WorkerSleeping] Worker", worker.id, "sleeping for", tlist.pipeline.WorkInterval, "milliseconds")
			color.Unset()
		case task := <-tlist.state.Task.Scheduled:
			color.Set(color.Bold, color.FgYellow)
			log.Println("[TaskScheduled] Task.id", task.id, "Task.name ==", task.name, "== Task.hash", task.hash, "scheduled to run at", task.nextRun.Format(time.UnixDate))
			color.Unset()
		case data := <-tlist.state.Task.Started:
			color.Set(color.Bold)
			for worker, task := range data {
				log.Println("[TaskStarted] Worker", worker.id, "picked up Task.id", task.id, "Task.name ==", task.name, "== Task.hash", task.hash)
			}
			color.Unset()
		case data := <-tlist.state.Task.Finished:
			color.Set(color.Bold, color.FgGreen)
			for worker, task := range data {
				log.Println("[TaskFinished] Worker", worker.id, "finished: Task.id", task.id, "Task.name ==", task.name, "== Task.hash", task.hash)
			}
			color.Unset()
		}

		if stateMonitor {
			break
		}
	}
}
