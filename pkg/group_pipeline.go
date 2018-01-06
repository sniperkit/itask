package xtask

import (
	"log"
	"reflect"
	"time"
)

// RunPipeline
func (tlist *TaskGroup) RunPipeline(concurrency int, queue int, interval time.Duration) {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()
	// defer tlist.Close()

	log.Println("initializing pipeline runner... concurrency=", concurrency, ", queue=", queue, "interval: ", interval)

	if tlist.state != nil {
		go tlist.AsyncMonitor()
	}

	tasks := make(chan *Task, queue)
	// elems := make(chan *list.Element, queue)
	// results := make(chan *Task)

	workers := make(chan int, concurrency)
	for workerID := 1; workerID <= concurrency; workerID++ {
		tlist.counters.Increment("workers", 1)
		workers <- workerID
	}

	// tlist.list = removeDuplicate(tlist.list)
	for e := tlist.list.Front(); e != nil; e = e.Next() {
		if task, ok := e.Value.(*Task); ok && !task.isCompleted {
			if task != nil {
				tlist.counters.Increment("enqueued", 1)
				log.Println("enqueuing task.name: ", task.name)
				tasks <- task
			} else {
				tlist.counters.Increment("skipped", 1)
				log.Println("somwthing is wrong with task.name: ", task.name)
			}
		}
	}

	stop := make(chan bool)
	log.Println("starting pipeline runner...")

	go func() {
		for {
			select {
			case <-stop:
				return
			// case tlist.pipeline.New <- true:
			case task := <-tasks:
				go func() {
					workerID := <-workers
					// time.Sleep(time.Duration(random(150, 250)) * time.Millisecond)
					log.Println("run task.name: ", task.name, "workerID: ", workerID)
					res := tlist.Process(task, workerID, cap(workers))
					if res.Result.Error != nil {
						log.Fatalln("res err task.name: ", res.Result.Error.Error())
					}
					log.Println("res task.name: ", res.Result.Result)
					log.Println("res task.name: ", res.Result.Result)
					// results <- tlist.Process(task, workerID, cap(workers))
					workers <- workerID
					log.Println("[FINISHED] task.name: ", task.name, "workerID: ", workerID, "tlist.Len()", tlist.Len(), "counters=", tlist.counters.Snapshot())
				}()
				// results <- executeWork(workerID, cap(workers), targetURL, urls)

			case <-time.After(time.Second * 10):
				if tlist.counters != nil {
					log.Println("counters=", tlist.counters.Snapshot())
				}

			case <-time.After(time.Second * 1):
				// log.Println("luc.............")
				log.Println("[CHECKUP] task.Len(): ", tlist.Len(), "task.completed: ", tlist.counters.Get("completed"))
				if (len(workers) == cap(workers)) && (tlist.counters.Get("completed") == tlist.Len()) {
					log.Println("all done")
					stop <- true
					return
				}
			}
		}
	}()

	log.Println("exiting pipeline runner...")
	<-stop
	tlist.rate.Close()

	tlist.list = nil
	tlist.rate = nil

	if workers != nil {
		close(workers)
	}
	if tasks != nil {
		close(tasks)
	}
	if stop != nil {
		close(stop)
	}

	/*
		// update the statistics with the results
		allStatisticsHaveBeenUpdated := make(chan bool)
		go func() {
			for {
				select {
				case <-tlist.pipeline.Done:
					allStatisticsHaveBeenUpdated <- true
					return

				case result := <-results:
					updateStatistics(result)
				}
			}
		}()

		<-allStatisticsHaveBeenUpdated
	*/

}

func (tlist *TaskGroup) Close() {
	/*
		if workers != nil {
			close(workers)
		}
		if tasks != nil {
			close(tasks)
		}
		if stop != nil {
			close(stop)
		}
	*/
}

// Process takes a task and does the work on it.
func (tlist *TaskGroup) Process(t *Task, workerID, numberOfWorkers int) *Task {

	t.once.Do(func() {
		t.wait.Add(1)

		if tlist.rate != nil {
			tlist.rate.Wait()
		}

		tlist.counters.Increment("started", 1)
		// Use context.Context to stop running goroutines
		// ctx, cancel := context.WithCancel(context.Background())
		// defer cancel()

		if t.name == "" {
			t.name = t.hash.String()
		}

		if t.delay.Nanoseconds() > 0 {
			tlist.counters.Increment("delayed", 1)
			log.Println("delay task.name: ", t.name, "delay=", t.delay.Seconds(), " seconds, workerID: ", workerID)
			time.Sleep(t.delay)
		}

		defer func() {
			t.isCompleted = true
			if t.continueWith != nil {
				result := t.Result
				for element := t.continueWith.Back(); element != nil; element = element.Prev() {
					if tt, ok := element.Value.(ContinueWithHandler); ok {
						tt(result)
					}
				}
				tlist.counters.Increment("continue", 1)
			}
			log.Println("done task.name: ", t.name, "workerID: ", workerID) //, "counters=", tlist.counters.Snapshot())
			t.wait.Done()
			tlist.counters.Increment("completed", 1)
		}()

		fn := reflect.ValueOf(t.fn)
		fnType := fn.Type()
		if fnType.Kind() != reflect.Func && fnType.NumIn() != len(t.args) {
			tlist.counters.Increment("not_expected", 1)
			log.Fatal("Expected a function")
		}

		var args []reflect.Value
		for _, arg := range t.args {
			args = append(args, reflect.ValueOf(arg))
		}

		res := fn.Call(args)
		for _, val := range res {
			log.Println("result for task.name: ", t.name, ", response:", val.Interface())
		}

		t.Result = TaskResult{
			Result: res,
		}

		if t.repeat {
			log.Println("repeat task.name: ", t.name, ", interval:", t.interval)
			tlist.EnqueueFuncEvery(t.name, t.interval, t.fn, t.args)
		}

	})
	log.Println("exit processing for task.name: ", t.name)
	return t
}
