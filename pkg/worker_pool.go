package xtask

import (
	"time"
)

// New creates and starts a pool of worker goroutines.
//
// The maxWorkers parameter specifies the maximum number of workers that will
// execute tasks concurrently.  After each timeout period, a worker goroutine
// is stopped until there are no remaining workers.
func NewWorkerPool(maxWorkers int) *WorkerPool {
	// There must be at least one worker.
	if maxWorkers < 1 {
		maxWorkers = 1
	}

	// taskQueue is unbuffered since items are always removed immediately.
	pool := &WorkerPool{ // chan *Task
		maxWorkers:   maxWorkers,
		taskQueue:    make(chan *Task),
		readyWorkers: make(chan chan *Task, readyQueueSize),
		// taskQueue:    make(chan func()),
		// readyWorkers: make(chan chan func(), readyQueueSize),
		timeout:     time.Second * idleTimeoutSec,
		stoppedChan: make(chan struct{}),
	}

	// Start the task dispatcher.
	go pool.dispatch()

	return pool
}

// WorkerPool is a collection of goroutines, where the number of concurrent
// goroutines processing requests does not exceed the specified maximum.
type WorkerPool struct {
	maxWorkers   int
	timeout      time.Duration
	taskQueue    chan *Task
	readyWorkers chan chan *Task
	// taskQueue    chan func()
	// readyWorkers chan chan func()
	stoppedChan chan struct{}
}

// Stop stops the worker pool and waits for workers to complete.
//
// Since creating the worker pool starts at least one goroutine, for the
// dispatcher, this function should be called when the worker pool is no longer
// needed.
func (p *WorkerPool) Stop() {
	if p.Stopped() {
		return
	}
	close(p.taskQueue)
	<-p.stoppedChan
}

// Stopped returns true if this worker pool has been stopped.
func (p *WorkerPool) Stopped() bool {
	select {
	case <-p.stoppedChan:
		return true
	default:
	}
	return false
}

// Submit enqueues a function for a worker to execute.
//
// Any external values needed by the task function must be captured in a
// closure.  Any return values should be returned over a channel that is
// captured in the task function closure.
//
// Submit will not block regardless of the number of tasks submitted.  Each
// task is immediately given to an available worker or passed to a goroutine to
// be given to the next available worker.  If there are no available workers,
// the dispatcher adds a worker, until the maximum number of workers is
// running.
func (p *WorkerPool) Submit(task *Task) {
	if task != nil {
		p.taskQueue <- task
	}
}

/*
// SubmitWait enqueues the given function and waits for it to be executed.
func (p *WorkerPool) SubmitWait(task *Task) {
	if task == nil {
		return
	}
	doneChan := make(chan struct{})
	p.taskQueue <- f() {
		// task()
		task.Run()
		close(doneChan)
	}
	<-doneChan
}
*/

// dispatch sends the next queued task to an available worker.
func (p *WorkerPool) dispatch() {
	defer close(p.stoppedChan)
	timeout := time.NewTimer(p.timeout)
	var workerCount int
	var task *Task
	var ok bool
	var workerTaskChan chan *Task
	startReady := make(chan chan *Task)
Loop:
	for {
		timeout.Reset(p.timeout)
		select {
		case task, ok = <-p.taskQueue:
			if !ok {
				break Loop
			}
			// Got a task to do.
			select {
			case workerTaskChan = <-p.readyWorkers:
				// A worker is ready, so give task to worker.
				workerTaskChan <- task
			default:
				// No workers ready.
				// Create a new worker, if not at max.
				if workerCount < p.maxWorkers {
					workerCount++
					go func(t *Task) {
						startWorker(startReady, p.readyWorkers)
						// Submit the task when the new worker.
						taskChan := <-startReady
						taskChan <- t
					}(task)
				} else {
					// Start a goroutine to submit the task when an existing
					// worker is ready.
					go func(t *Task) {
						taskChan := <-p.readyWorkers
						taskChan <- t
					}(task)
				}
			}
		case <-timeout.C:
			// Timed out waiting for work to arrive.  Kill a ready worker.
			if workerCount > 0 {
				select {
				case workerTaskChan = <-p.readyWorkers:
					// A worker is ready, so kill.
					close(workerTaskChan)
					workerCount--
				default:
					// No work, but no ready workers.  All workers are busy.
				}
			}
		}
	}

	// Stop all remaining workers as they become ready.
	for workerCount > 0 {
		workerTaskChan = <-p.readyWorkers
		close(workerTaskChan)
		workerCount--
	}
}

// startWorker starts a goroutine that executes tasks given by the dispatcher.
//
// When a new worker starts, it registers its availability on the startReady
// channel.  This ensures that the goroutine associated with starting the
// worker gets to use the worker to execute its task.  Otherwise, the main
// dispatcher loop could steal the new worker and not know to start up another
// worker for the waiting goroutine.  The task would then have to wait for
// another existing worker to become available, even though capacity is
// available to start additional workers.
//
// A worker registers that is it available to do work by putting its task
// channel on the readyWorkers channel.  The dispatcher reads a worker's task
// channel from the readyWorkers channel, and writes a task to the worker over
// the worker's task channel.  To stop a worker, the dispatcher closes a
// worker's task channel, instead of writing a task to it.
func startWorker(startReady, readyWorkers chan chan *Task) {
	go func() {
		taskChan := make(chan *Task)
		var task *Task
		var ok bool
		// Register availability on starReady channel.
		startReady <- taskChan
		for {
			// Read task from dispatcher.
			task, ok = <-taskChan
			if !ok {
				// Dispatcher has told worker to stop.
				break
			}

			// Execute the task.
			task.Run()

			// Register availability on readyWorkers channel.
			readyWorkers <- taskChan
		}
	}()
}
