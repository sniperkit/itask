package xtask

const (
	// This value is the size of the queue that workers register their
	// availability to the dispatcher.  There may be hundreds of workers, but
	// only a small channel is needed to register some of the workers.
	readyQueueSize = 16

	// If worker pool receives no new work for this period of time, then stop
	// a worker goroutine.
	idleTimeoutSec = 5
)
