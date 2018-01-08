package xtask

import (
	"container/list"
	"log"
	"math/rand"
	"reflect"
	"sync"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/sniperkit/xtask/plugin/counter"
	"github.com/sniperkit/xtask/plugin/rate"
	"github.com/sniperkit/xtask/plugin/stats/tachymeter"
)

// TaskQueue is a threadsafe container for tasks to be processed
type TaskQueue struct {
	name string
	id   int
	hash uuid.UUID

	list *list.List
	lock *sync.RWMutex
	wg   *sync.WaitGroup

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
	results    *TaskQueueResult

	// running bool
	// signal chan bool
	// FinishedTasks:  make(chan *Task, 100),
	stop    chan bool
	tasks   chan *Task
	workers chan int
	runner  *Runner

	// w *worker

	// workers  chan *Worker
	// complete chan *Task

	// preHandlers  []*Handler // sync execute
	// handlers     []*Handler // can be sync or parallel
	// postHandlers []*Handler // sync execute
	// onRecover    func(*TaskRecoverMsg)
}

func NewTaskQueue() *TaskQueue {
	hash := uuid.NewV4()
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	tq := &TaskQueue{
		id:       random.Intn(10000),
		hash:     hash,
		list:     list.New(),
		counters: counter.NewOc(),
		limiter:  NewLimiter(hash.String()),
		lock:     &sync.RWMutex{},
		// runner:   NewRunner(0, 0), // default values concurency=5, queue=20
		// wg:       &sync.WaitGroup{},
		// stop:     make(chan bool),
		// tasks:    make(chan *Task),
		// workers:  make(chan int),
		// w:        NewWorker(),
		// signal:   make(chan bool, 1000),
		// newTasks: make(chan *Task, 1000),
		// FinishedTasks:  make(chan *Task, 100),
	}
	// tq.runner = NewRunner(0, 0, tq)
	return tq
}

type Runner struct {
	Queue *TaskQueue
	// running bool
	NewTask chan bool
	// New     chan *Task
	// FinishedTasks:  make(chan *Task, 100),
	Stop    chan bool
	Tasks   chan *Task
	Workers chan int
}

func NewRunner(concurrency int, pool int, queue *TaskQueue) *Runner {

	if concurrency <= 0 {
		concurrency = 5
	}

	if pool <= 0 {
		pool = 20
	}

	return &Runner{
		Queue:   queue,
		Stop:    make(chan bool),
		Tasks:   make(chan *Task, pool),
		Workers: make(chan int, concurrency),
		NewTask: make(chan bool, pool),
	}
}

func (r *Runner) Close() {
	close(r.Stop)
	close(r.Tasks)
	close(r.Workers)
	close(r.NewTask)
}

// Add
func (tq *TaskQueue) AddTask(task *Task) *TaskQueue {
	tq.lock.Lock()
	defer tq.lock.Unlock()

	tq.counters.Increment("add.task", 1)
	task.nextRun = time.Now()
	tq.list.PushFront(task)

	//if tq.runner != nil {
	//	// tq.runner.Tasks <- task
	//	tq.runner.NewTask <- true
	//}
	//go func() {
	// log.Println("[NEW] new task: ", task.name)
	// tq.signal <- true // tq.Pop()
	// }()
	return tq
}

// WaitAll
func (tq *TaskQueue) WaitAll() {
	tq.lock.RLock()
	defer tq.lock.RUnlock()

	for element := tq.list.Front(); element != nil; element = element.Next() {
		if task, ok := element.Value.(*Task); ok && !task.isCompleted {
			task.wait.Wait()
		}
	}
	if tq.runner != nil {
		tq.runner.Close()
	}
}

/*
func (tq *TaskQueue) PerformRequests() {
	rate := time.Second / 10
	burstLimit := 100
	tick := time.NewTicker(rate)
	defer tick.Stop()
	throttle := make(chan time.Time, burstLimit)
	go func() {
		for t := range tick.C {
			select {
			case throttle <- t:
			default:
			}
		}
	}()
	i := len(q.Requests)
	for key, req := range q.Requests {
		<-throttle
		go req.Action()
		tq.lock.Lock()
		defer tq.lock.Unlock()
		delete(tq.Requests, key)
		i = i - 1
	}
}
*/

func (tq *TaskQueue) Pool(concurrency int, workerInterval time.Duration) *TaskQueue {
	tq.lock.RLock()
	defer tq.lock.RUnlock()

	if tq.tachymeter != nil {
		tq.wallTime = time.Now() // Start wall time for all Goroutines to get accurate metrics with the tachymeter.
	}

	log.Println("initializing pipeline runner... concurrency=", concurrency, "workerInterval=", workerInterval)
	wp := NewWorkerPool(concurrency)
	for e := tq.list.Front(); e != nil; e = e.Next() {
		if task, ok := e.Value.(*Task); ok && !task.isCompleted {
			if task != nil {
				tq.counters.Increment("enqueued", 1)
				log.Println("enqueuing task.name: ", task.name)
				wp.Submit(tq.ProcessOnce(task))
			} else {
				tq.counters.Increment("skipped", 1)
				log.Println("somwthing is wrong with task.name: ", task.name)
			}
		}
	}

	wp.Stop()

	if tq.tachymeter != nil {
		tq.tachymeter.SetWallTime(time.Since(tq.wallTime))
		tq.results.Metrics = tq.tachymeter.Calc()
		log.Println(tq.tachymeter.Calc().String())
	}

	// tq.runner.Close()

	return tq
}

// RunPipeline
func (tq *TaskQueue) Pipeline(concurrency int, queue int, workerInterval time.Duration) *TaskQueue {
	tq.lock.RLock()
	defer tq.lock.RUnlock()

	if tq.tachymeter != nil {
		tq.wallTime = time.Now() // Start wall time for all Goroutines to get accurate metrics with the tachymeter.
	}

	log.Println("initializing pipeline runner... concurrency=", concurrency, ", queue=", queue, "workerInterval=", workerInterval)

	//if tq.state != nil {
	//	go tq.AsyncMonitor()
	//}

	runner := NewRunner(concurrency, queue, tq)
	// tq.runner = runner

	// stop = make(chan bool)
	// tasks = make(chan *Task, queue)
	// workers = make(chan int, concurrency)

	// tq.signal = make(chan bool, concurrency)
	// tq.wg = &sync.WaitGroup{}

	// results := make(chan *Task)

	for workerID := 1; workerID <= concurrency; workerID++ {
		tq.counters.Increment("workers", 1)
		runner.Workers <- workerID
	}

	for e := tq.list.Front(); e != nil; e = e.Next() {
		if task, ok := e.Value.(*Task); ok && !task.isCompleted {
			if task != nil {
				tq.counters.Increment("enqueued", 1)
				log.Println("enqueuing task.name: ", task.name)
				runner.Tasks <- task
			} else {
				tq.counters.Increment("skipped", 1)
				log.Println("somwthing is wrong with task.name: ", task.name)
			}
		}
	}

	log.Println("starting pipeline runner...")

	go func() {
		for {
			select {
			case <-runner.Stop:
				return

			case <-runner.NewTask:
				log.Println("received new task...")

			case task := <-runner.Tasks:
				go func() {
					workerID := <-runner.Workers

					if tq.rate != nil {
						tq.rate.Wait()
					}

					time.Sleep(workerInterval)
					// time.Sleep(time.Duration(random(150, 250)) * time.Millisecond)
					log.Println("run task.name: ", task.name, "workerID: ", workerID)

					res := runner.Process(task, workerID, cap(tq.workers))
					// res := tq.Process(task, workerID, cap(tq.workers))
					if res.Result.Error != nil {
						log.Fatalln("res err task.name: ", res.Result.Error.Error())
					}

					log.Println("res task.name: ", res.Result.Result)
					log.Println("[FINISHED] task.name: ", task.name, "workerID: ", workerID, "tq.Len()", tq.Len())

					// results <- tq.Process(task, workerID, cap(workers))
					runner.Workers <- workerID

				}()

			//case <-time.After(100 * time.Millisecond):
			//	w.Sleep()
			//case <-tq.signal:
			//	log.Println("[SIGNAL] ********** signal")
			//	tq.signal <- false

			case <-time.After(time.Millisecond * 500):
				// log.Println("luc.............")
				log.Println("[CHECKUP] task.Len(): ", tq.Len(), "task.completed: ", tq.counters.Get("completed"))
				// if (len(runner.Workers) == cap(runner.Workers)) && (tq.counters.Get("completed") == tq.Len()) {
				if (len(runner.Workers) == cap(runner.Workers)) && (tq.counters.Get("completed") == tq.Len()) {
					log.Println("all done")
					runner.Stop <- true
					return
				}

				//default:
				//	log.Println("[DEFAULT] pipeline runner")

			}
		}
	}()

	log.Println("exiting pipeline runner...")
	<-runner.Stop

	// tq.rate.Close()

	if tq.tachymeter != nil {
		tq.tachymeter.SetWallTime(time.Since(tq.wallTime))
		tq.results.Metrics = tq.tachymeter.Calc()
		log.Println(tq.tachymeter.Calc().String())
	}

	runner.Close()

	/*
		allStatsUpdated := make(chan bool) // update the stats with the task's results
		go func() {
			for {
				select {
				case <-tq.pipeline.Done:
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
	return tq
}

/*
func (r *Runner) Listen(workerInterval time.Duration, timeout time.Duration) {
	for {
		select {
		case <-tq.signal:
			if tq.list.Len() > 0 {
				workerID := <-tq.workers
				task := tq.Pop()
				if tq.rate != nil {
					tq.rate.Wait()
				}

				time.Sleep(workerInterval)
				// time.Sleep(time.Duration(random(150, 250)) * time.Millisecond)
				log.Println("run task.name: ", task.name, "workerID: ", workerID)

				res := tq.Process(task, workerID, cap(tq.workers))
				if res.Result.Error != nil {
					log.Fatalln("res err task.name: ", res.Result.Error.Error())
				}

				log.Println("res task.name: ", res.Result.Result)
				log.Println("[FINISHED] task.name: ", task.name, "workerID: ", workerID, "tq.Len()", tq.Len())

				// results <- tq.Process(task, workerID, cap(workers))
				tq.workers <- workerID
			}
			//case <-time.After(100 * time.Millisecond):
			//	tq.Sleep()
		}
	}
}
*/

func (tq *TaskQueue) ProcessOnce(t *Task) *Task {
	// tq.lock.Lock()
	// defer tq.lock.Unlock()

	t.once.Do(func() {
		t.wait.Add(1)

		if t.name == "" {
			t.name = t.hash.String()
		}

		if t.delay.Nanoseconds() > 0 {
			tq.counters.Increment("delayed", 1)
			log.Println("delay task.name: ", t.name, "delay=", t.delay.Seconds(), " seconds")
			time.Sleep(t.delay)
		}

		start := time.Now()
		tq.counters.Increment("started", 1)

		defer func() {
			t.isCompleted = true

			if t.continueWith.Len() > 0 {
				tq.counters.Increment("chained", 1)
				result := t.Result
				for element := t.continueWith.Back(); element != nil; element = element.Prev() {
					if tt, ok := element.Value.(ContinueWithHandler); ok {
						tt(result)
					}
				}
			}

			log.Println("done task.name (Once): ", t.name) //, "counters=", tq.counters.Snapshot())
			t.wait.Done()
			tq.tachymeter.AddTime(t.name, time.Since(start))
			tq.counters.Increment("completed", 1)
			// r.Tasks <- t

		}()

		fn := reflect.ValueOf(t.fn)
		fnType := fn.Type()
		if fnType.Kind() != reflect.Func && fnType.NumIn() != len(t.args) {
			tq.counters.Increment("unexpected", 1)
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

		// if tq.results != nil {
		//	 tq.results.Results = append(tq.results.Results, t.Result)
		// }

		if t.repeat {
			tq.counters.Increment("repeat.every", 1)
			log.Println("repeat task.name: ", t.name, ", interval:", t.interval)
			tq.EnqueueFuncEvery(t.name, t.interval, t.fn, t.args)
		}

	})
	log.Println("exit processing for task.name: ", t.name)
	return t
}

// Process takes a task and does the work on it.
func (r *Runner) Process(t *Task, workerID, numberOfWorkers int) *Task {
	// tq.lock.Lock()
	// defer tq.lock.Unlock()

	t.once.Do(func() {
		t.wait.Add(1)

		if t.name == "" {
			t.name = t.hash.String()
		}

		if t.delay.Nanoseconds() > 0 {
			r.Queue.counters.Increment("delayed", 1)
			log.Println("delay task.name: ", t.name, "delay=", t.delay.Seconds(), " seconds, workerID: ", workerID)
			time.Sleep(t.delay)
		}

		start := time.Now()
		r.Queue.counters.Increment("started", 1)

		defer func() {
			t.isCompleted = true

			if t.continueWith.Len() > 0 {
				r.Queue.counters.Increment("chained", 1)
				result := t.Result
				for element := t.continueWith.Back(); element != nil; element = element.Prev() {
					if tt, ok := element.Value.(ContinueWithHandler); ok {
						tt(result)
					}
				}
			}

			log.Println("done task.name: ", t.name, "workerID: ", workerID) //, "counters=", tq.counters.Snapshot())
			t.wait.Done()
			r.Queue.tachymeter.AddTime(t.name, time.Since(start))
			r.Queue.counters.Increment("completed", 1)
			r.Tasks <- t

		}()

		fn := reflect.ValueOf(t.fn)
		fnType := fn.Type()
		if fnType.Kind() != reflect.Func && fnType.NumIn() != len(t.args) {
			r.Queue.counters.Increment("unexpected", 1)
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

		// if tq.results != nil {
		//	 tq.results.Results = append(tq.results.Results, t.Result)
		// }

		if t.repeat {
			r.Queue.counters.Increment("repeat.every", 1)
			log.Println("repeat task.name: ", t.name, ", interval:", t.interval)
			r.Queue.EnqueueFuncEvery(t.name, t.interval, t.fn, t.args)
		}

	})
	log.Println("exit processing for task.name: ", t.name)
	return t
}

func (tq *TaskQueue) Close() {}

// Process takes a task and does the work on it.
func (tq *TaskQueue) Process(t *Task, workerID, numberOfWorkers int) *Task {
	// tq.lock.Lock()
	// defer tq.lock.Unlock()

	t.once.Do(func() {
		t.wait.Add(1)

		if t.name == "" {
			t.name = t.hash.String()
		}

		if t.delay.Nanoseconds() > 0 {
			tq.counters.Increment("delayed", 1)
			log.Println("delay task.name: ", t.name, "delay=", t.delay.Seconds(), " seconds, workerID: ", workerID)
			time.Sleep(t.delay)
		}

		start := time.Now()
		tq.counters.Increment("started", 1)

		defer func() {
			t.isCompleted = true

			if t.continueWith.Len() > 0 {
				tq.counters.Increment("chained", 1)
				result := t.Result
				for element := t.continueWith.Back(); element != nil; element = element.Prev() {
					if tt, ok := element.Value.(ContinueWithHandler); ok {
						tt(result)
					}
				}
			}

			log.Println("done task.name: ", t.name, "workerID: ", workerID) //, "counters=", tq.counters.Snapshot())
			t.wait.Done()
			tq.tachymeter.AddTime(t.name, time.Since(start))
			tq.counters.Increment("completed", 1)
			tq.runner.Tasks <- t

		}()

		fn := reflect.ValueOf(t.fn)
		fnType := fn.Type()
		if fnType.Kind() != reflect.Func && fnType.NumIn() != len(t.args) {
			tq.counters.Increment("unexpected", 1)
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

		// if tq.results != nil {
		//	 tq.results.Results = append(tq.results.Results, t.Result)
		// }

		if t.repeat {
			tq.counters.Increment("repeat.every", 1)
			log.Println("repeat task.name: ", t.name, ", interval:", t.interval)
			tq.EnqueueFuncEvery(t.name, t.interval, t.fn, t.args)
		}

	})
	log.Println("exit processing for task.name: ", t.name)
	return t
}

// Push adds a new task into the front of the TaskQueue
func (tq *TaskQueue) Len() int {
	tq.lock.RLock()
	defer tq.lock.RUnlock()

	return tq.list.Len()
}

func (tq *TaskQueue) Next() *Task {
	tq.lock.Lock()
	defer tq.lock.Unlock()

	for element := tq.list.Front(); element != nil; element = element.Next() {
		if task, ok := element.Value.(*Task); ok && !task.isCompleted {
			//if time.Since(task.nextRun) > 0 {
			return task
			//}
		}
	}
	return nil
}

// Get checks if a key exists in our dequeued task list
func (tq *TaskQueue) FindByUUID(hash uuid.UUID) bool {
	tq.lock.Lock()
	defer tq.lock.Unlock()

	if _, ok := tq.registry[hash]; ok {
		return true
	}
	return false
}

// Remove deletes the dequeued entry once we are done with it
func (tq *TaskQueue) RemoveByUUID(hash uuid.UUID) {
	tq.lock.Lock()
	defer tq.lock.Unlock()

	delete(tq.registry, hash)
}

// Push adds a new task into the front of the TaskQueue
func (tq *TaskQueue) Push(task *Task) {
	tq.lock.Lock()
	defer tq.lock.Unlock()
	tq.counters.Increment("push", 1)

	tq.list.PushFront(task)
	// tq.pipeline.New <- true
}

// Pop grabs the last task from the TaskQueue
func (tq *TaskQueue) Pop() *Task {
	tq.lock.Lock()
	defer tq.lock.Unlock()
	tq.counters.Increment("pop", 1)

	task := tq.list.Remove(tq.list.Back())
	return task.(*Task)
}

// AddFunc
func (tq *TaskQueue) AddFunc(name string, fn interface{}, args ...interface{}) *TaskQueue {
	tq.lock.Lock()
	defer tq.lock.Unlock()

	task := NewTask(name, fn, args...)
	task.nextRun = time.Now()
	tq.list.PushFront(task)

	tq.counters.Increment("add.func", 1)
	return tq
}

// AddRange
func (tq *TaskQueue) AddRange(tasks ...*Task) *TaskQueue {
	tq.lock.Lock()
	defer tq.lock.Unlock()
	tq.counters.Increment("add.range", 1)

	for _, task := range tasks {
		tq.list.PushFront(task)
	}
	return tq
}

// RunTaskAsync
func (tq *TaskQueue) RunTaskAsync() *TaskQueue {
	for element := tq.list.Front(); element != nil; element = element.Next() {
		if task, ok := element.Value.(*Task); ok && !task.isCompleted {
			task.RunAsync()
		}
	}
	return tq
}

// enqueue is an internal function used to asynchronously push a task onto the queue and log the state to the terminal.
func (tq *TaskQueue) enqueue(task *Task) *TaskQueue {
	tq.lock.Lock()
	defer tq.lock.Unlock()

	tq.Push(task)
	return tq
}

/*
func (tq *TaskQueue) PreProcess(name string, f interface{}, args ...interface{}) *Task {
	h := NewFunc(name, f, args...)
	tq.preHandlers = append(tq.preHandlers, h)
	return t
}

func (tq *TaskQueue) Process(name string, f interface{}, args ...interface{}) *Task {
	h := NewFunc(name, f, args...)
	tq.handlers = append(tq.handlers, h)
	return t
}

func (tq *TaskQueue) PostProcess(name string, f interface{}, args ...interface{}) *Task {
	h := NewFunc(name, f, args...)
	tq.postHandlers = append(tq.postHandlers, h)
	return t
}

func (tq *TaskQueue) SetRecover(f func(*RecoverMsg)) {
	tq.onRecover = f
}

func (tq *TaskQueue) deferFunc(wg *sync.WaitGroup, h *Handler, startTime int64) {
	if wg != nil {
		wg.Done()
	}
	if r := recover(); r != nil {
		if tq.onRecover != nil {
			tq.onRecover(&RecoverMsg{GetFuncName(h), startTime, r})
		} else {
			log.Println(GetFuncName(h), r)
		}
	}
}

// sync execution
func (tq *TaskQueue) RunChain() {
	var h Handler
	defer tq.deferFunc(nil, &h, time.Now().UnixNano())
	for i := range tq.preHandlers {
		h = *tq.preHandlers[i]
		h.Call()
	}
	for i := range tq.handlers {
		h = *tq.handlers[i]
		h.Call()
	}
	for i := range tq.postHandlers {
		h = *tq.postHandlers[i]
		h.Call()
	}
}

// parallel execute, waiting for all go routine finished
func (tq *TaskQueue) Parallel() {
	var h Handler
	defer tq.deferFunc(nil, &h, time.Now().UnixNano())
	for i := range tq.preHandlers {
		h = *tq.preHandlers[i]
		h.Call()
	}

	wg := &sync.WaitGroup{}
	for i := range t.handlers {
		wg.Add(1)
		go func(f *Handler) {
			defer tq.deferFunc(wg, f, time.Now().UnixNano())
			f.Call()
		}(t.handlers[i])
	}
	wg.Wait()

	for i := range t.postHandlers {
		h = *t.postHandlers[i]
		h.Call()
	}
}

// parallel execute with timeout
func (tq *TaskQueue) ParallelWithTimeout(timeout time.Duration) error {
	var h Handler
	defer tq.deferFunc(nil, &h, time.Now().UnixNano())
	for i := range tq.preHandlers {
		h = *tq.preHandlers[i]
		h.Call()
	}

	wg := &sync.WaitGroup{}
	for i := range tq.handlers {
		wg.Add(1)
		go func(f *Handler) {
			defer tq.deferFunc(wg, f, time.Now().UnixNano())
			f.Call()
		}(tq.handlers[i])
	}

	done := make(chan int)
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done: // all done
	case <-time.After(timeout): // timeout
		return errExecuteTimeout
	}

	for i := range tq.postHandlers {
		h = *tq.postHandlers[i]
		h.Call()
	}
	return nil
}

// submit async job
func (tq *TaskQueue) Async() {
	// TODO: write manager to schedule job
	go func() {
		var h Handler
		defer tq.deferFunc(nil, &h, time.Now().UnixNano())
		for i := range tq.preHandlers {
			h = *tq.preHandlers[i]
			h.Call()
		}

		for i := range tq.handlers {
			go func(f *Handler) {
				defer tq.deferFunc(nil, f, time.Now().UnixNano())
				f.Call()
			}(tq.handlers[i])
		}

		for i := range tq.postHandlers {
			h = *t.postHandlers[i]
			h.Call()
		}
	}()
}
*/
