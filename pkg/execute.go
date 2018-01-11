package xtask

import (
	//"sync"

	"github.com/anacrolix/sync"
)

// RunTasks() registers tasks in the specified order.
func (tq *TaskQueue) Execute(tasks ...*Task) error {
	tq.lock.Lock()
	defer tq.lock.Unlock()

	if len(tasks) == 0 {
		log.Println("warning! errNoTasks")
	}

	if err := inSeries(tasks...); err != nil {
		return mask(err)
	}
	return nil
}

// InSeries() executes the given tasks in a serial order, one after another.
func (task *Task) InSeries(tasks ...*Task) *Task {
	task.lock.Lock()
	defer task.lock.Unlock()

	if len(tasks) == 0 {
		log.Println("warning! errNoTasks")
		return task
	}

	if err := inSeries(tasks...); err != nil {
		log.Println("warning! errNoTasks", mask(err))
		return task
	}
	return task
}

// InParallel() executes the given tasks in a parallel order, all at the same time.
func (task *Task) InParallel(tasks ...*Task) *Task {
	task.lock.Lock()
	defer task.lock.Unlock()

	if len(tasks) == 0 {
		log.Println("warning! errNoTasks")
		return task
	}

	// Create error for current tasks. If there occurs one error, all remaining tasks will be canceled.
	var err error
	errCatched := false

	// Create waitgroup to keep track of parallel tasks by registering the count of them.
	var wg sync.WaitGroup
	wg.Add(len(tasks))

	for _, t := range tasks { // Start a goroutine for each task to run them in parallel.
		go func(t Task) {
			if res := t.RunAsync(); res.Result.Error != nil {
				if !errCatched { // Just catch the first occuring error.
					err = res.Result.Error
					errCatched = true
				}
			}
			wg.Done()
		}(*t)
	}

	wg.Wait() // Wait until the waitgroup count is 0.
	if err != nil {
		log.Fatalln("Error occured while waiting until the waitgroup count to be reset to zero")
		// return nil
	}

	return task
}

func inSeries(tasks ...*Task) error {
	for _, t := range tasks {
		if res := t.RunAsync(); res.Result.Error != nil {
			return mask(res.Result.Error)
		}
	}
	return nil
}
