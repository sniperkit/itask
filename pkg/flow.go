package xtask

/*
import (
	"sync"
)
*/

/*
	Refs:
	- https://github.com/rafaeldias/async/blob/master/funcs.go
	- https://github.com/xh3b4sd/taskq
*/

/*
// RunTasks() registers tasks in the specified order.
func (q *Queue) RunTasks(tasks ...Task) error {
	if len(tasks) == 0 {
		return Mask(ErrNoTasks)
	}

	if err := inSeries(q.Ctx, tasks...); err != nil {
		return Mask(err)
	}

	return nil
}

// InSeries() executes the given tasks in a serial order, one after another.
func InSeries(tasks ...Task) Task {
	if len(tasks) == 0 {
		return func(ctx interface{}) error {
			return Mask(ErrNoTasks)
		}
	}

	return func(ctx interface{}) error {
		if err := inSeries(ctx, tasks...); err != nil {
			return Mask(err)
		}

		return nil
	}
}

// InParallel() executes the given tasks in a parallel order, all at the same time.
func InParallel(tasks ...Task) Task {
	if len(tasks) == 0 {
		return func(ctx interface{}) error {
			return Mask(ErrNoTasks)
		}
	}

	return func(ctx interface{}) error {
		// Create error for current tasks. If there occurs one error, all remaining
		// tasks will be canceled.
		var err error
		errCatched := false

		// Create waitgroup to keep track of parallel tasks by registering the count
		// of them.
		var wg sync.WaitGroup
		wg.Add(len(tasks))

		// Start a goroutine for each task to run them in parallel.
		for _, t := range tasks {
			go func(t Task, ctx interface{}) {
				if e := t(ctx); e != nil {
					// Just catch the first occuring error.
					if !errCatched {
						err = e
						errCatched = true
					}
				}

				wg.Done()
			}(t, ctx)
		}

		// Wait until the waitgroup count is 0.
		wg.Wait()

		if err != nil {
			return Mask(err)
		}

		return nil
	}
}

func inSeries(ctx interface{}, tasks ...Task) error {
	for _, t := range tasks {
		if err := t(ctx); err != nil {
			return Mask(err)
		}
	}

	return nil
}
*/
