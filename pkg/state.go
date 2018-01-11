package xtask

import (
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

func (tq *TaskQueue) Monitor() *TaskQueue {
	tq.state = &State{}

	// Worker Channels
	tq.state.Worker.Started = make(chan *Worker)
	tq.state.Worker.Sleeping = make(chan *Worker)

	// Task Channels
	tq.state.Task.Scheduled = make(chan *Task)
	tq.state.Task.Aborted = make(chan *Task)
	tq.state.Task.Expired = make(chan *Task)
	tq.state.Task.Started = make(chan map[*Worker]*Task)
	tq.state.Task.Finished = make(chan map[*Worker]*Task)

	return tq
}

// LogTaskAbortd sends a signal to the TaskAbortd channel triggering the output text.
func (tq *TaskQueue) LogTaskScheduled(t *Task) {
	tq.state.Task.Scheduled <- t
}

// LogTaskAbortd sends a signal to the TaskAbortd channel triggering the output text.
func (tq *TaskQueue) LogTaskAbortd(t *Task) {
	tq.state.Task.Aborted <- t
}

// LogTaskStarted sends a signal to the TaskStarted channel triggering the output text.
func (tq *TaskQueue) LogTaskStarted(w *Worker, t *Task) {
	tq.state.Task.Started <- map[*Worker]*Task{w: t}
}

// LogTaskFinished sends a signal to the TaskFinished channel triggering the output text.
func (tq *TaskQueue) LogTaskFinished(w *Worker, t *Task) {
	tq.state.Task.Finished <- map[*Worker]*Task{w: t}
}

// LogWorkerSleeping sends a signal to the WorkerSleeping channel triggering the
// output text.
func (tq *TaskQueue) LogWorkerSleeping(w *Worker) {
	tq.state.Worker.Sleeping <- w
}

// StateMonitor provides a sane way to listen for state changes in the application. New state is passed via channels outputting logs from anywhere in the application.
func (tq *TaskQueue) AsyncMonitor() {

	var stateMonitor bool
	for {
		select {
		case status := <-tq.state.Pipeline.Close:
			stateMonitor = status
		case worker := <-tq.state.Worker.Started:
			color.Set(color.Bold, color.FgBlue)
			log.Println("[WorkerStarted] Started Worker", worker.id)
			color.Unset()
		case worker := <-tq.state.Worker.Sleeping:
			color.Set(color.Faint)
			log.Println("[WorkerSleeping] Worker", worker.id, "sleeping for", tq.pipeline.WorkInterval, "milliseconds")
			color.Unset()
		case task := <-tq.state.Task.Scheduled:
			color.Set(color.Bold, color.FgYellow)
			log.Println("[TaskScheduled] Task.id", task.id, "Task.name ==", task.name, "== Task.hash", task.hash, "scheduled to run at", task.nextRun.Format(time.UnixDate))
			color.Unset()
		case data := <-tq.state.Task.Started:
			color.Set(color.Bold)
			for worker, task := range data {
				log.Println("[TaskStarted] Worker", worker.id, "picked up Task.id", task.id, "Task.name ==", task.name, "== Task.hash", task.hash)
			}
			color.Unset()
		case data := <-tq.state.Task.Finished:
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
