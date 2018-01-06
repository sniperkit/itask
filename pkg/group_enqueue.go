package xtask

// enqueue is an internal function used to asynchronously push a task onto the queue and log the state to the terminal.
func (tlist *TaskGroup) enqueue(task *Task) *TaskGroup {
	tlist.lock.Lock()
	defer tlist.lock.Unlock()

	tlist.Push(task)
	return tlist
}
