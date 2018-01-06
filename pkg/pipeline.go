package xtask

// Config contains the base configuration for the work queue.
type Pipeline struct {
	// NumWorkers specifies the maximum number of active workers to run at any given time.
	NumWorkers int
	// WorkInterval is the time it takes for a worker to sleep before it checks the task queue for more work to do.
	WorkInterval int
	// ScheduledTasks is the default queue used to decide what is available for the workers to consume.
	scheduled TaskGroup
	// CancelledTasks is a queue which is checked before a task is executed to see if the task has been cancelled.
	aborted TaskGroup
	// NewTasks is a signal channel to express that a new task has been pushed to the ScheduledTasks queue.
	newTasks chan bool
	// WorkerPool in a channel to wait for a worker when a job comes in and we send workers back into it when they are done.
	workerPool chan *Worker
	// FinishedTasks is a channel which cleans up after a task has finished.
	finishedTasks chan *Task
	// Shared context
	sharedContext interface{}
}

// Create a new queue with the given context. That can be pointer to a struct
// type.
func NewPipeline(sharedContext *Pipeline) *Pipeline {
	return &Pipeline{
		sharedContext: sharedContext,
	}
}
