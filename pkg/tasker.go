package xtask

import (
	"bytes"
	"container/list"
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"runtime"
	"time"
	// "sync"
	// "context"
	// "sync/atomic"

	"github.com/anacrolix/sync"
	"github.com/boz/go-throttle"
	"go.uber.org/ratelimit"
	// "github.com/k0kubun/pp"
	// "github.com/VividCortex/robustly"
	// "github.com/eapache/go-resiliency/semaphore"
	// "github.com/eapache/channels"

	uuid "github.com/satori/go.uuid"
	"github.com/sniperkit/xtask/plugin/counter"
	"github.com/sniperkit/xtask/plugin/rate"
	"github.com/sniperkit/xtask/plugin/stats/tachymeter"
)

func init() {
	sync.Enable()
}

type CycleError [][]string

func (ce CycleError) Error() string {
	defer funcTrack(time.Now())

	msg := "tasker: "
	if len(ce) > 1 {
		msg += "cycles"
	} else {
		msg += "cycle"
	}
	msg += " detected: "
	for i, c := range ce {
		for j, e := range c {
			msg += e
			if j < len(c)-1 {
				msg += " -> "
			}
		}
		if i < len(ce)-1 {
			msg += ", "
		}
	}
	return msg
}

type DepNotFoundError struct {
	v string
	w string
}

func NewDepNotFoundError(v, w string) *DepNotFoundError {
	return &DepNotFoundError{v, w}
}

func (dnfe *DepNotFoundError) Error() string {
	return fmt.Sprintf("tasker: %s not found, required by %s", dnfe.w, dnfe.v)
}

// A Task is a function called with no arguments that returns an error. If
// variable information is required, consider providing a closure.
// type Tsk func() *TaskInfo // , time.Duration
type Tsk func() *TaskResult // , time.Duration

type TaskGroup struct {
	Id       int
	Name     string
	Disabled bool
	Paused   bool
}

// task info holds run-time information related to a task identified by taskInfo.name.
type TaskInfo struct {
	Id           int
	Name         string
	Group        *string
	Hash         uuid.UUID
	Result       *TaskResult
	task         Tsk // The task itself.
	fn           interface{}
	args         []interface{}
	handler      reflect.Value
	params       []reflect.Value
	err          error       // Stores error on failure.
	done         bool        // Prevents running a task more than once.
	mux          *sync.Mutex // Controls access to this task.
	wait         *sync.WaitGroup
	continueWith *list.List
	delay        time.Duration
	nextRun      time.Time
	interval     time.Duration
	repeat       bool
	counters     *counter.Oc
	rate         *rate.RateLimiter

	// Elements used in cycle detection.
	index    int
	lowlink  int
	on_stack bool
}

func (ti *TaskInfo) lock() {
	ti.mux.Lock()
}

func (ti *TaskInfo) unlock() {
	ti.mux.Unlock()
}

func GetTaskFuncName(t *Task) string {
	defer funcTrack(time.Now())
	if t == nil || t.fn == nil {
		return ""
	}
	return runtime.FuncForPC(reflect.ValueOf(t.fn).Pointer()).Name()
}

func (tr *Tasker) ContinueWith2(name string, deps []string, task Tsk) *TaskInfo {
	defer funcTrack(time.Now())

	tr.Add(name, "jj", deps, task)
	err_ch := make(chan error)
	tr.runTask(name, err_ch)
	ti := tr.tis[name]
	ti.err = <-err_ch
	// Do not run this task if one of its dependencies fail.
	if ti.err != nil {
		err_ch <- ti.err
		return nil
	}
	return ti
}

func newTaskFunc(name string, fn interface{}, args ...interface{}) *TaskInfo {
	defer funcTrack(time.Now())
	random := rand.New(rand.NewSource(time.Now().UnixNano()))

	return &TaskInfo{
		Id:           random.Intn(10000),
		Hash:         uuid.NewV4(),
		Name:         name,
		wait:         &sync.WaitGroup{},
		mux:          &sync.Mutex{},
		fn:           fn,
		args:         args,
		repeat:       false,
		continueWith: list.New(),
		delay:        0 * time.Second,
		nextRun:      time.Now(),
		counters:     counter.NewOc(),
		rate:         &rate.RateLimiter{},
		index:        -1,
		lowlink:      -1,
		done:         false,
		err:          nil,
		on_stack:     false,
		Result:       &TaskResult{Error: nil, Result: nil},
	}
}

func newTaskInfo(task Tsk) *TaskInfo {
	defer funcTrack(time.Now())
	random := rand.New(rand.NewSource(time.Now().UnixNano()))

	return &TaskInfo{
		task:         task,
		Id:           random.Intn(10000),
		Hash:         uuid.NewV4(),
		done:         false,
		err:          nil,
		continueWith: list.New(),
		wait:         &sync.WaitGroup{},
		mux:          &sync.Mutex{},
		repeat:       false,
		delay:        0 * time.Second,
		nextRun:      time.Now(),
		index:        -1,
		lowlink:      -1,
		on_stack:     false,
		Result:       &TaskResult{Error: nil, Result: nil},
	}
}

type Tasker struct {
	Id      int
	Name    string
	Cluster string

	// Map of taskInfo's indexed by task name.
	tis  map[string]*TaskInfo
	list *list.List

	// Map of tasks names their dependencies. Its keys are identical to tis'.
	dep_graph map[string][]string

	// Semaphore implemented as a buffered boolean channel. May be nil.
	// See wait and signal.
	semaphore chan bool

	// Elements used in cycle detection.
	index  int
	stack  *stringStack
	cycles [][]string

	// Indicates whether Run has been called.
	was_run bool

	mux *sync.Mutex // Controls access to this task.
	// lock       *sync.RWMutex
	hash     uuid.UUID
	limiter  *Limiter
	state    *State
	counters *counter.Oc

	uberRate ratelimit.Limiter
	rate     *rate.RateLimiter
	throttle throttle.Throttle

	tachymeter *tachymeter.Tachymeter
	results    *TaskQueueResult

	wallTime  time.Time
	startTime time.Time
	endTime   time.Time
	prev      time.Time
	last      time.Time
}

// wait signals that a task is running and blocks until it may be run.
func (tr *Tasker) wait() {
	// defer funcTrack(time.Now())
	if tr.semaphore != nil {
		tr.semaphore <- true
	}
}

// signal signals that a task is done.
func (tr *Tasker) signal() {
	// defer funcTrack(time.Now())
	if tr.semaphore != nil {
		<-tr.semaphore
	}
}

// NewTasker returns a new Tasker that will run up to n number of tasks
// simultaneously. If n is -1, there is no such restriction.
//
// Returns an error if n is invalid.
func NewTasker(n int) (*Tasker, error) {
	defer funcTrack(time.Now())

	if n < -1 || n == 0 {
		return nil, fmt.Errorf("n must be positive or -1: %d", n)
	}

	var semaphore chan bool
	if n > 0 {
		semaphore = make(chan bool, n)
	} // else semaphore is nil

	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	hash := uuid.NewV4()

	tr := &Tasker{
		Id:         random.Intn(10000),
		tis:        make(map[string]*TaskInfo),
		dep_graph:  make(map[string][]string),
		semaphore:  semaphore,
		index:      -1,
		stack:      newStringStack(),
		cycles:     make([][]string, 0),
		was_run:    false,
		hash:       hash,
		counters:   counter.NewOc(),
		tachymeter: tachymeter.New(nil),
		results:    &TaskQueueResult{},
		mux:        &sync.Mutex{},
		limiter:    NewLimiter(hash.String()),
		uberRate:   ratelimit.New(n), // per second
	}

	return tr, nil
}

// Add adds a task to call a function. name is the unique name for the task.
// deps is a list of the names of tasks to run before this the one being added.
//
// name may not be the empty string.
//
// Any tasks specified in deps must be added before the Tasker can be run. deps
// may be nil, but task may not.
//
// An error is returned if name is not unique.
func (tr *Tasker) Add(name string, group string, deps []string, task Tsk) *TaskInfo {
	defer funcTrack(time.Now())

	tr.mux.Lock()
	defer tr.mux.Unlock()

	if name == "" {
		return nil // &Tsk{err: errors.New("name is empty")}
	}
	if _, ok := tr.tis[name]; ok {
		return tr.tis[name] // &Tsk{err: fmt.Errorf("task already added: %s", name)}
	}

	// Prevent the basic cyclic dependency of one from occuring.
	for _, dep := range deps {
		if name == dep {
			return tr.tis[name] // &Tsk{err: errors.New("task must not add itself as a dependency")}
		}
	}

	tr.tis[name] = newTaskInfo(task)
	tr.dep_graph[name] = deps

	tr.tis[name].Result = &TaskResult{Name: name}
	tr.tis[name].Group = &group

	return tr.tis[name]
}

type ContinueTaskWithHandler func(*TaskInfo)
type ContinueTaskWithFunc func(*TaskInfo)
type ContinueTaskWithTask func(*TaskInfo)

// ContinueWithTask
func (ti *TaskInfo) ContinueWithHandler(handler ContinueTaskWithTask) *TaskInfo {
	defer funcTrack(time.Now())
	ti.lock()
	defer ti.unlock()

	ti.continueWith.PushFront(handler)
	// pp.Println("TaskInfo:\n", ti)
	return ti
}

// ContinueWithFunc
func (ti *TaskInfo) ContinueWithFunc(name string, fn interface{}, args ...interface{}) *TaskInfo {
	defer funcTrack(time.Now())
	ti.lock()
	defer ti.unlock()

	handler := newTaskFunc(name, fn, args...)
	ti.continueWith.PushFront(handler)
	return ti
}

// Delay
func (ti *TaskInfo) Delay(delay time.Duration) *TaskInfo {
	defer funcTrack(time.Now())

	ti.lock()
	defer ti.unlock()

	ti.delay = delay
	return ti
}

func (tr *Tasker) ContinueWith(name string, deps []string, task Tsk) *Tasker {
	defer funcTrack(time.Now())

	/*
		if _, ok := tr.tis[name]; !ok {
			return tr
		}
	*/

	tr.tis[name].continueWith.PushFront(task)
	return tr
}

// xtask.Tsk

func (t Tsk) PostProcess() Tsk {
	defer funcTrack(time.Now())

	f := reflect.Indirect(reflect.ValueOf(t))
	if f.Kind() != reflect.Func {
		return nil
	}

	// pp.Println(f)
	// pp.Println(t)
	// var res *TaskInfo
	// res = t
	// LogWithFields(Fields{"Tsk": t}).Println("*Tsk.PostProcess()...")
	return t
}

func (t *TaskInfo) PostProcess() *TaskInfo {
	defer funcTrack(time.Now())

	t.mux.Lock()
	defer t.mux.Unlock()

	var msg string
	if t.Result.Error != nil {
		msg = t.Result.Error.Error()
	}
	LogWithFields(Fields{"error": msg, "result": t.Result.Result}).Println("PostProcess()...")
	return t
}

func (tr *Tasker) Tachymeter() *Tasker {
	defer funcTrack(time.Now())
	tr.mux.Lock()
	defer tr.mux.Unlock()

	size := tr.Count()

	tr.tachymeter = tachymeter.New(
		&tachymeter.Config{
			Size:     size,
			Safe:     true,
			HBuckets: 50,
		})

	return tr
}

func (tr *Tasker) Count() int {
	defer funcTrack(time.Now())
	// tr.mux.Lock()
	// defer tr.mux.Unlock()

	count := len(tr.tis)
	return count
}

// find_cycles implements Tarjan's Algorithm to construct a list of strongly
// connected components, or cycles, in the directed graph of tasks and their
// dependencies. It sets tr.cycles to a list of lists of task names. Each task
// name list denotes a strongly connected component of more than one vertex.
//
// It is called with the empty string, but called recursively with a task name.
func (tr *Tasker) findCycles(v string) {
	defer funcTrack(time.Now())

	if v == "" {
		// Initialize algorithm's state.
		tr.index = 0
		tr.stack = newStringStack()
		tr.cycles = make([][]string, 0)
		for v := range tr.dep_graph {
			ti := tr.tis[v]
			ti.index = -1
			ti.lowlink = -1
			ti.on_stack = false
		}
		// Find all cycles.
		for v := range tr.dep_graph {
			if tr.tis[v].index == -1 {
				// v has not yet been visited.
				tr.findCycles(v)
			}
		}
	} else {
		// Visit v: Set its index and lowlink and push it onto the stack.
		v_ti := tr.tis[v]
		v_ti.index = tr.index
		v_ti.lowlink = tr.index
		tr.index++
		tr.stack.push(v)
		v_ti.on_stack = true

		// Recursively consider dependencies of v.
		for _, w := range tr.dep_graph[v] {
			w_ti := tr.tis[w]
			if w_ti.index == -1 {
				// w has not yet been visited.
				tr.findCycles(w)
				// v's lowlink is the smallest index of any
				// recursive dependency of v. If w's lowlink is
				// smaller than v's, it follows that v's
				// lowlink must be set to w's, since w is a
				// dependency of v.
				if w_ti.lowlink < v_ti.lowlink {
					v_ti.lowlink = w_ti.lowlink
				}
			} else if w_ti.on_stack {
				if w_ti.index != w_ti.lowlink {
					panic("w's index and lowlink differ, how!?")
				}
				// w's presence on the stack means that it is
				// in the current scc. It's index is equal to
				// its lowlink because we are in one of its
				// recursive calls.
				if w_ti.index < v_ti.lowlink {
					v_ti.lowlink = w_ti.index
				}
			}
		}

		if v_ti.lowlink == v_ti.index {
			scc := make([]string, 0)
			for {
				w, err := tr.stack.pop()
				if err != nil {
					panic(err)
				}
				tr.tis[w].on_stack = false
				scc = append(scc, w)
				if w == v {
					break
				}
			}

			// Ignore sccs that only include itself, since
			// technically a root node with no dependencies is an
			// scc, and in the Add function we make sure that a
			// task never depends on itself.
			if len(scc) > 1 {
				tr.cycles = append(tr.cycles, scc)
			}
		}
	}
}

// verify returns an error if any task dependencies haven't been added or any
// cycles exist among the tasks.
func (tr *Tasker) verify() error {
	defer funcTrack(time.Now())

	for name, deps := range tr.dep_graph {
		for _, dep := range deps {
			if _, ok := tr.tis[dep]; !ok {
				return NewDepNotFoundError(name, dep)
			}
		}
	}
	tr.findCycles("")
	if len(tr.cycles) > 0 {
		return CycleError(tr.cycles)
	}
	return nil
}

/*
func (tr *Tasker) runTaskOnce(name string, err_ch chan error) {
	ti := tr.tis[name]

	ti.lock()
	defer ti.unlock()

	ti.once.Do(func() {
		ti.wait.Add(1)

		// ti.counters.Increment("started", 1)

		if ti.delay.Nanoseconds() > 0 {
			// ti.counters.Increment("delayed", 1)
			time.Sleep(ti.delay)
		}

		defer func() {
			ti.done = true
			// ti.counters.Increment("completed", 1)

			if ti.continueWith.Len() > 0 {
				// if task.continueWith != nil {
				ti.counters.Increment("chained", 1)

				if ti.Result.Error == nil {
					result := *ti.Result
					for element := ti.continueWith.Back(); element != nil; element = element.Prev() {
						if tt, ok := element.Value.(ContinueWithHandler); ok {
							tt(result)
						}
					}
				}
			}

			// tq.worker.complete <- task
			// tq.LogTaskFinished(tq.worker, task)
			ti.wait.Done()
			// tr.counters.Increment("completed", 1)
		}()

		fn := reflect.ValueOf(ti.fn)
		fnType := fn.Type()
		if fnType.Kind() != reflect.Func && fnType.NumIn() != len(ti.args) {
			// ti.counters.Increment("unexpected", 1)
			// log.Panic("Expected a function")
			log.Print("Expected a function")
			os.Exit(1)
		}

		var args []reflect.Value
		for _, arg := range ti.args {
			args = append(args, reflect.ValueOf(arg))
		}

		res := fn.Call(args)
		for _, val := range res {
			log.Println("Response:", val.Interface())
		}

		// Limit the number of consecutive tasks.
		tr.wait()
		defer tr.signal()
		defer tr.timeTrack(time.Now(), name)

		r := ti.task()

		ti.Result.Name = r.Result.Name
		ti.Result.Result = r.Result.Result
		ti.Result.Error = r.Result.Error
		ti.err = ti.Result.Error
		err_ch <- ti.err


	})
	// return ti
}
*/

// runTask is called recursivley as a goroutine to run tasks in parallel. It
// runs all dependencies before running the provided task. The first error it
// encounters will be send through err_ch, be it from a dependency or the task
// itself. It will not run the provided task if any dependency fails.
//
// It initially takes the task's lock and sets a flag so that a task is not run
// in any other goroutine. Other goroutines will wait for the lock, then see
// that the task has already been executed, and return whatever error it had
// produced.
//
// It further limits the number of consecutive tasks as defined by the size of
// the Tasker's semaphore.
// , tm *tachymeter.Tachymeter
func (tr *Tasker) runTask(name string, err_ch chan error) {
	defer funcTrack(time.Now())

	ti := tr.tis[name]

	ti.lock()
	defer ti.unlock()

	// Don't run this task if it has been handled by another goroutine and send
	// its error, which may be an error from running the task itself or from
	// running one of its dependencies.
	if ti.done {
		err_ch <- ti.err
		return
	}

	// Set this task to done.
	ti.done = true

	// Run all dependencies first. Do not continue with the current task if one
	// fails. If that happens, this task will inherit its error from the first
	// one that failed.
	deps := tr.dep_graph[name]

	dep_err_ch := make(chan error)
	for _, dep := range deps {
		go tr.runTask(dep, dep_err_ch)
	}
	for _ = range deps {
		ti.err = <-dep_err_ch
		// Do not run this task if one of its dependencies fail.
		if ti.err != nil {
			err_ch <- ti.err
			return
		}
	}

	if tr.rate != nil {
		tr.rate.Wait()
	}

	// Limit the number of consecutive tasks.
	tr.wait()
	ti.wait.Add(1)
	defer tr.signal()
	defer tr.timeTrack(time.Now(), name)

	output := ti.task()

	// ti.Result.Name = output.Result.Name
	ti.Result.Result = output.Result
	// ti.Result.Output = output.Output
	ti.Result.Error = output.Error
	ti.err = ti.Result.Error

	go func() {
		defer func() {
			ti.done = true
			if ti.continueWith != nil {
				for element := ti.continueWith.Back(); element != nil; element = element.Prev() {
					if tt, ok := element.Value.(ContinueTaskWithTask); ok {
						tt(ti)
					}

				}
			}
			ti.wait.Done()
		}()
	}()

	// robustly.Run(func() { ti.task() }, nil)
	err_ch <- ti.err
}

func (tr *Tasker) timeTrack(start time.Time, name string) {
	tr.mux.Lock()
	defer tr.mux.Unlock()

	elapsed := time.Since(start)
	log.Debugf("timeTrack() %s took %s", name, elapsed)
	tr.tachymeter.AddTime(name, elapsed)
}

func (tr *Tasker) Limiter(limit int, interval time.Duration) *Tasker {
	defer funcTrack(time.Now())

	tr.mux.Lock()
	defer tr.mux.Unlock()

	tr.counters.Increment("set.limiter.default", 1)
	tr.rate = rate.New(limit, interval)
	return tr
}

// runTasks runs a list of tasks using runTask and waits for them to finish.
func (tr *Tasker) runTasks(names ...string) error {
	defer funcTrack(time.Now())

	var buf bytes.Buffer

	if tr.tachymeter != nil {
		log.Println("has tachymeter")
		tr.wallTime = time.Now() // Start wall time for all Goroutines to get accurate metrics with the tachymeter.
	}

	err_ch := make(chan error)

	for _, name := range names {
		sync.PrintLockTimes(&buf)
		LogWithFields(Fields{"lock_times": buf.String()})
		go tr.runTask(name, err_ch)
	}

	// Wait for all tasks to finish. Return the first error encountered.
	var err error
	for _ = range names {

		e := <-err_ch
		if err == nil {
			err = e
		}
	}

	if tr.tachymeter != nil {
		tr.mux.Lock()
		defer tr.mux.Unlock()
		tr.tachymeter.SetWallTime(time.Since(tr.wallTime))
		tr.results.Metrics = tr.tachymeter.Calc()
		log.Println(tr.tachymeter.Calc().String())
	}

	return err
}

/*
	sem.Acquire(ctx, n)     // acquire n with context
	sem.TryAcquire(n)       // try acquire n without blocking
	...
	ctx := context.WithTimeout(context.Background(), time.Second)
	sem.Acquire(ctx, n)     // acquire n with timeout

	sem.Release(n)          // release n

	sem.SetLimit(new_limit) // set new semaphore limit

*/

// Run runs a list of tasks registered through Add in parallel. If not tasks
// are provided, then all tasks are run.
//
// All tasks are only run once, even if two or more other tasks depend on it.
// A task will not run if any dependency fails.
//
// The last error from a task is returned. Otherwise, Run returns
// nil.
func (tr *Tasker) Run(names ...string) error {

	if tr.was_run {
		return errors.New("tasker: already run")
	}

	if err := tr.verify(); err != nil {
		return err
	}

	if len(names) == 0 {
		names = make([]string, 0)
		for name, _ := range tr.tis {
			names = append(names, name)
		}
	} else {
		// Validate the provided tasks.
		for _, name := range names {
			if _, ok := tr.tis[name]; !ok {
				return fmt.Errorf("tasker: task not found: %s", name)
			}
		}
	}

	// This function must not be called again at this point.
	tr.was_run = true

	return tr.runTasks(names...)
}
