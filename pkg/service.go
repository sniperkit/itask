package xtask

const (
	DEFAULT_WORKER_COUNT  = 5
	DEFAULT_WORK_INTERVAL = 5 // in seconds
)

// appConfig is the configuration object to use within the actual module.
var AppConfig *Config

// Config contains the base configuration for the work queue.
type Config struct {
	// NumWorkers specifies the maximum number of active workers to run at any
	// given time.
	NumWorkers int
	// WorkInterval is the time it takes for a worker to sleep before it
	// checks the task queue for more work to do.
	WorkInterval int
	// ScheduledTasks is the default queue used to decide what is available
	// for the workers to consume.
	ScheduledTasks TaskList
	// CancelledTasks is a queue which is checked before a task is executed to
	// see if the task has been cancelled.
	CancelledTasks TaskDequeue
	// NewTasks is a signal channel to express that a new task has been pushed
	// to the ScheduledTasks queue.
	NewTasks chan bool
	// WorkerPool in a channel to wait for a worker when a job comes in and
	// we send workers back into it when they are done.
	WorkerPool chan *Worker
	// FinishedTasks is a channel which cleans up after a task has finished.
	FinishedTasks chan *Task
}

// Configure sets up the base application confiuration options.
func Configure(numWorkers, workInterval int) *Config {
	scheduledTasks := NewTaskList(numWorkers)
	config := &Config{
		NumWorkers:     numWorkers,
		WorkInterval:   workInterval,
		ScheduledTasks: *scheduledTasks,
		CancelledTasks: NewDequeue(),
		NewTasks:       make(chan bool, 10000),
		FinishedTasks:  make(chan *Task, 10000),
		WorkerPool:     make(chan *Worker, numWorkers),
	}
	AppConfig = config
	return config
}

// DefaultConfig uses the defaults to configure the application.
func DefaultConfig() *Config {
	return Configure(DEFAULT_WORKER_COUNT, DEFAULT_WORK_INTERVAL)
}

// RunService starts a blocking loop allowing the goroutines to communicate
// without the program closing. We spawn the workers here and also fire off
// the StateMonitor to listen for state changes while processing.
func RunService() {
	if AppConfig == nil {
		AppConfig = DefaultConfig()
	}

	go StateMonitor()
	go SpawnWorkers()

	for {
		select {
		case <-AppConfig.FinishedTasks:
		case <-AppConfig.WorkerPool:
		}
	}
}
