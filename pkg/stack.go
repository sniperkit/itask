package xtask

import (
	//"sync"

	"github.com/anacrolix/sync"
)

// string_stack is a generic slice with stack operations.
type stringStack struct {
	stack []string
	count int
	lock  *sync.RWMutex
}

// push adds an element onto the top of the stack.
func (s *stringStack) push(e string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.stack = append(s.stack, e)
	s.count++
}

// pop removes and returns the element on top of the stack. It returns an error is no element can be removed.
func (s *stringStack) pop() (string, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.count == 0 {
		return "", errStackIsEmpty // errors.New("Stack is empty")
	}
	s.count--
	e := s.stack[s.count]
	s.stack = s.stack[:s.count]
	return e, nil
}

// newStringStack returns a new empty stack.
func newStringStack() *stringStack {
	return &stringStack{make([]string, 0), 0, &sync.RWMutex{}}
}
