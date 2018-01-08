package xtask

import (
	"container/list"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
	// "context"
	// "sync/atomic"

	"github.com/boz/go-throttle"
	"go.uber.org/ratelimit"

	uuid "github.com/satori/go.uuid"
	"github.com/sniperkit/xtask/plugin/counter"
	"github.com/sniperkit/xtask/plugin/rate"
	"github.com/sniperkit/xtask/plugin/stats/tachymeter"
)

/*
	Refs:
	- https://github.com/beefsack/go-rate
	- https://github.com/alexurquhart/rlimit/blob/master/rlimit.go
*/

type CycleError [][]string

func (ce CycleError) Error() string {
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
type Tsk func() error

// task info holds run-time information related to a task identified by taskInfo.name.
type taskInfo struct {
	task Tsk         // The task itself.
	done bool        // Prevents running a task more than once.
	err  error       // Stores error on failure.
	mux  *sync.Mutex // Controls access to this task.

	// Elements used in cycle detection.
	index    int
	lowlink  int
	on_stack bool
}

func (ti *taskInfo) lock() {
	ti.mux.Lock()
}

func (ti *taskInfo) unlock() {
	ti.mux.Unlock()
}

func newTaskInfo(task Tsk) *taskInfo {
	return &taskInfo{task, false, nil, &sync.Mutex{}, -1, -1, false}
}

type Tasker struct {
	id   int
	name string

	// Map of taskInfo's indexed by task name.
	tis  map[string]*taskInfo
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
	if tr.semaphore != nil {
		tr.semaphore <- true
	}
}

// signal signals that a task is done.
func (tr *Tasker) signal() {
	if tr.semaphore != nil {
		<-tr.semaphore
	}
}

// NewTasker returns a new Tasker that will run up to n number of tasks
// simultaneously. If n is -1, there is no such restriction.
//
// Returns an error if n is invalid.
func NewTasker(n int) (*Tasker, error) {
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
		id:         random.Intn(10000),
		tis:        make(map[string]*taskInfo),
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
		// prev:       time.Time,
		// list:       list.New(),
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
func (tr *Tasker) Add(name string, deps []string, task Tsk) error {
	tr.mux.Lock()
	defer tr.mux.Unlock()

	if name == "" {
		return errors.New("name is empty")
	}
	if _, ok := tr.tis[name]; ok {
		return fmt.Errorf("task already added: %s", name)
	}

	// Prevent the basic cyclic dependency of one from occuring.
	for _, dep := range deps {
		if name == dep {
			return errors.New("task must not add itself as a dependency")
		}
	}

	tr.tis[name] = newTaskInfo(task)
	tr.dep_graph[name] = deps

	return nil
}

func (tr *Tasker) Tachymeter() *Tasker {
	size := tr.Count()

	tr.mux.Lock()
	defer tr.mux.Unlock()

	tr.tachymeter = tachymeter.New(
		&tachymeter.Config{
			Size:     size,
			Safe:     true,
			HBuckets: 50,
		})

	return tr
}

func (tr *Tasker) Count() int {
	tr.mux.Lock()
	defer tr.mux.Unlock()

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
	tr.mux.Lock()
	ti := tr.tis[name]
	tr.mux.Unlock()

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
	tr.mux.Lock()
	deps := tr.dep_graph[name]
	tr.mux.Unlock()

	dep_err_ch := make(chan error)
	for _, dep := range deps {
		// tr.mux.Lock()
		//if tr.rate != nil {
		//	tr.rate.Wait()
		//}
		// tr.mux.Unlock()
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
	defer tr.signal()

	start := time.Now()
	ti.err = ti.task()
	log.Println("runTask().tachymeter for ", name, "end", time.Since(start))

	// tr.mux.Lock()
	tr.tachymeter.AddTime(name, time.Since(start))
	// tr.mux.Unlock()

	err_ch <- ti.err
}

func (tr *Tasker) Limiter(limit int, interval time.Duration) *Tasker {
	tr.mux.Lock()
	defer tr.mux.Unlock()

	tr.counters.Increment("set.limiter.default", 1)
	tr.rate = rate.New(limit, interval)
	return tr
}

// runTasks runs a list of tasks using runTask and waits for them to finish.
func (tr *Tasker) runTasks(names ...string) error {

	if tr.tachymeter != nil {
		log.Println("has tachymeter")
		tr.wallTime = time.Now() // Start wall time for all Goroutines to get accurate metrics with the tachymeter.
	}

	err_ch := make(chan error)

	for _, name := range names {
		// tr.mux.Lock()
		// tr.mux.Unlock()
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
