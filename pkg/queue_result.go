package xtask

// TaskQueueResult
type TaskQueueResult struct {
	Results []TaskResult
	Metrics interface{}
}

func (tq *TaskQueue) Aggregate() *TaskQueue {
	tq.lock.Lock()
	defer tq.lock.Unlock()
	if tq.results == nil {
		tq.results = &TaskQueueResult{}
	}
	return tq
}
