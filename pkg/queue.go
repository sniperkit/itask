package xtask

import (
	"log"
	"math/rand"
	"reflect"
	"sync"
	"time"
	// "container/list"

	uuid "github.com/satori/go.uuid"
	"github.com/sniperkit/xtask/plugin/counter"
	"github.com/sniperkit/xtask/plugin/rate"
	"github.com/sniperkit/xtask/plugin/stats/tachymeter"
)

func NewTaskGroup() *TaskGroup {
	hash := uuid.NewV4()
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &TaskGroup{
		// list:     list.New(),
		list:     NewSearchableQueue(),
		id:       random.Intn(10000),
		hash:     hash,
		limiter:  NewLimiter(hash.String()),
		lock:     &sync.RWMutex{},
		counters: counter.NewOc(),
	}
}

// TaskGroupResult
type TaskGroupResult struct {
	Results []TaskResult
	Metrics interface{}
}

// TaskGroup is a threadsafe container for tasks to be processed
type TaskGroup struct {
	name string
	id   int
	hash uuid.UUID

	// list       *list.List
	list       *SearchableQueue
	lock       *sync.RWMutex
	state      *State
	pipeline   *Pipeline
	shared     interface{} // shared context between tasks
	registry   map[uuid.UUID]bool
	limiter    *Limiter
	counters   *counter.Oc
	completed  int
	rate       *rate.RateLimiter
	tachymeter *tachymeter.Tachymeter
	wallTime   time.Time
	startTime  time.Time //
	endTime    time.Time
	results    *TaskGroupResult
}

func (tlist *TaskGroup) Aggregate() *TaskGroup {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()

	if tlist.results == nil {
		tlist.results = &TaskGroupResult{}
	}
	return tlist
}

// RunPipeline
func (tlist *TaskGroup) RunPipeline(concurrency int, queue int, interval time.Duration) {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()
	// defer tlist.Close()

	if tlist.tachymeter != nil {
		tlist.wallTime = time.Now() // Start wall time for all Goroutines to get accurate metrics with the tachymeter.
	}

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

	if tlist.tachymeter != nil {
		tlist.tachymeter.SetWallTime(time.Since(tlist.wallTime))
		tlist.results.Metrics = tlist.tachymeter.Calc()
		log.Println(tlist.tachymeter.Calc().String())
	}

	tlist.rate.Close()
	/*
		// what if we want to add a scheduler and create a cron tasker
		tlist.rate = nil
		tlist.list = nil
	*/

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
		allStatsUpdated := make(chan bool) // update the stats with the task's results
		go func() {
			for {
				select {
				case <-tlist.pipeline.Done:
					allStatsUpdated <- true
					return
				case result := <-results:
					updateStats(result)
				}
			}
		}()
		<-allStatsUpdated

		if allStatsUpdated != nil {
			close(allStatsUpdated)
		}
	*/

}

func (tlist *TaskGroup) Close() {}

// Process takes a task and does the work on it.
func (tlist *TaskGroup) Process(t *Task, workerID, numberOfWorkers int) *Task {

	t.once.Do(func() {
		t.wait.Add(1)

		if tlist.rate != nil {
			tlist.rate.Wait()
		}

		start := time.Now()
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
			tlist.counters.Increment("completed", 1)
			if t.continueWith.Len() > 0 {
				tlist.counters.Increment("chained", 1)
				result := t.Result
				for element := t.continueWith.Back(); element != nil; element = element.Prev() {
					if tt, ok := element.Value.(ContinueWithHandler); ok {
						tt(result)
					}
				}
			}
			log.Println("done task.name: ", t.name, "workerID: ", workerID) //, "counters=", tlist.counters.Snapshot())

			tlist.tachymeter.AddTime(time.Since(start))
			t.wait.Done()
		}()

		fn := reflect.ValueOf(t.fn)
		fnType := fn.Type()
		if fnType.Kind() != reflect.Func && fnType.NumIn() != len(t.args) {
			tlist.counters.Increment("unexpected", 1)
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
			Error:  nil,
		}

		if tlist.results != nil {
			tlist.results.Results = append(tlist.results.Results, t.Result)
		}

		if t.repeat {
			tlist.counters.Increment("repeat.every", 1)
			log.Println("repeat task.name: ", t.name, ", interval:", t.interval)
			tlist.EnqueueFuncEvery(t.name, t.interval, t.fn, t.args)
		}

	})
	log.Println("exit processing for task.name: ", t.name)
	return t
}

func (tlist *TaskGroup) AddLimiter(limit int, interval time.Duration, key string) string {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()
	tlist.counters.Increment("add.limiter", 1)

	return tlist.limiter.Add(limit, interval, key)
}

// Push adds a new task into the front of the TaskGroup
func (tlist *TaskGroup) Len() int {
	tlist.lock.RLock()
	defer tlist.lock.RUnlock()

	return tlist.list.Len()
}

func (tlist *TaskGroup) Next() *Task {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()

	for element := tlist.list.Front(); element != nil; element = element.Next() {
		if task, ok := element.Value.(*Task); ok && !task.isCompleted {
			//if time.Since(task.nextRun) > 0 {
			return task
			//}
		}
	}
	return nil
}

// Get checks if a key exists in our dequeued task list
func (tlist *TaskGroup) FindByUUID(hash uuid.UUID) bool {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()

	if _, ok := tlist.registry[hash]; ok {
		return true
	}
	return false
}

// Remove deletes the dequeued entry once we are done with it
func (tlist *TaskGroup) RemoveByUUID(hash uuid.UUID) {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()

	delete(tlist.registry, hash)
}

// Push adds a new task into the front of the TaskGroup
func (tlist *TaskGroup) Push(task *Task) {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()
	tlist.counters.Increment("push", 1)

	tlist.list.PushFront(task)
	// tlist.pipeline.New <- true
}

// Pop grabs the last task from the TaskGroup
func (tlist *TaskGroup) Pop() *Task {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()
	tlist.counters.Increment("pop", 1)

	task := tlist.list.Remove(tlist.list.Back())
	return task.(*Task)
}

// Add
func (tlist *TaskGroup) AddTask(task *Task) *TaskGroup {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()

	tlist.counters.Increment("add.task", 1)
	task.nextRun = time.Now()
	tlist.list.PushFront(task)
	return tlist
}

// AddFunc
func (tlist *TaskGroup) AddFunc(name string, fn interface{}, args ...interface{}) *TaskGroup {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()

	task := NewTask(name, fn, args...)
	task.nextRun = time.Now()
	tlist.list.PushFront(task)

	tlist.counters.Increment("add.func", 1)
	return tlist
}

// AddRange
func (tlist *TaskGroup) AddRange(tasks ...*Task) *TaskGroup {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()
	tlist.counters.Increment("add.range", 1)

	for _, task := range tasks {
		tlist.list.PushFront(task)
	}
	return tlist
}

// RunTaskAsync
func (tlist *TaskGroup) RunTaskAsync() *TaskGroup {
	for element := tlist.list.Front(); element != nil; element = element.Next() {
		if task, ok := element.Value.(*Task); ok && !task.isCompleted {
			task.RunAsync()
		}
	}
	return tlist
}

// WaitAll
func (tlist *TaskGroup) WaitAll() {
	for element := tlist.list.Front(); element != nil; element = element.Next() {
		if task, ok := element.Value.(*Task); ok && !task.isCompleted {
			task.wait.Wait()
		}
	}
}

// enqueue is an internal function used to asynchronously push a task onto the queue and log the state to the terminal.
func (tlist *TaskGroup) enqueue(task *Task) *TaskGroup {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()

	tlist.Push(task)
	return tlist
}
