package xtask

// Config contains the base configuration for the work queue.
type Pipeline struct {
	// NumWorkers specifies the maximum number of active workers to run at any given time.
	NumWorkers int
	// WorkInterval is the time it takes for a worker to sleep before it checks the task queue for more work to do.
	WorkInterval int
	// ScheduledTasks is the default queue used to decide what is available for the workers to consume.
	Scheduled TaskGroup
	// CancelledTasks is a queue which is checked before a task is executed to see if the task has been cancelled.
	Aborted TaskGroup
	// NewTasks is a signal channel to express that a new task has been pushed to the ScheduledTasks queue.
	NewTasks chan bool
	// WorkerPool in a channel to wait for a worker when a job comes in and we send workers back into it when they are done.
	WorkerPool chan *Worker
	// FinishedTasks is a channel which cleans up after a task has finished.
	FinishedTasks chan *Task
}
