package xtask

import (
	"container/list"
	"sync"
)

// SearchableList list.List
type SearchableQueue struct {
	*list.List
	*sync.RWMutex
}

// New () *list.List
func NewSearchableQueue() *SearchableQueue {
	return &SearchableQueue{
		new(list.List).Init(),
		&sync.RWMutex{},
	}
}

// (l *NewSearchableQueue)

// ContainsElement (t *list.Element) bool
func (l *SearchableQueue) ContainsElement(t *list.Element) bool {
	// l.Lock()
	// defer l.Unlock()

	if l.Len() > 0 {
		for e := l.Front(); e != nil; e = e.Next() {
			if e == t {
				return true
			}
		}
	}
	return false
}

// Push adds a new task into the front of the TaskQueue
func (l *SearchableQueue) Len() int {
	// l.RLock()
	// defer l.RUnlock()

	return l.Len()
}

// Contains (t *list.Element) bool
// alias -> ContainsElement
func (l *SearchableQueue) Contains(t *list.Element) bool {
	// l.Lock()
	// defer l.Unlock()

	return l.ContainsElement(t)
}

// ContainsValue (v interface{}) bool
func (l *SearchableQueue) ContainsValue(v interface{}) bool {
	// l.Lock()
	// defer l.Unlock()

	if l.Len() > 0 {
		for e := l.Front(); e != nil; e = e.Next() {
			if e.Value == v {
				return true
			}
		}
	}
	return false
}

// FindFirst (v interface{}) *list.Element
func (l *SearchableQueue) FindFirst(v interface{}) *list.Element {
	// l.Lock()
	// defer l.Unlock()

	if l.Len() > 0 {
		for e := l.Front(); e != nil; e = e.Next() {
			if e.Value == v {
				return e
			}
		}
	}
	return nil
}

// FindLast (v interface{}) *list.Element
func (l *SearchableQueue) FindLast(v interface{}) *list.Element {
	// l.Lock()
	// defer l.Unlock()

	if l.Len() > 0 {
		for e := l.Back(); e != nil; e = e.Prev() {
			if e.Value == v {
				return e
			}
		}
	}
	return nil
}

// FindAll (v interface{}) []*list.Element
func (l *SearchableQueue) FindAll(v interface{}) []*list.Element {
	// l.Lock()
	// defer l.Unlock()

	if l.Len() > 0 {
		elList := []*list.Element{}
		for e := l.Front(); e != nil; e = e.Next() {
			if e.Value == v {
				elList = append(elList, e)
			}
		}
		return elList
	}
	return nil
}
