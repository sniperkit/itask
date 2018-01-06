package stack

// string_stack is a generic slice with stack operations.
type stringStack struct {
	stack []string
	count int
}

// push adds an element onto the top of the stack.
func (s *stringStack) Push(e string) {
	s.stack = append(s.stack, e)
	s.count++
}

// pop removes and returns the element on top of the stack. It returns an error is no element can be removed.
func (s *stringStack) Pop() (string, error) {
	if s.count == 0 {
		return "", errStackIsEmpty // errors.New("Stack is empty")
	}
	s.count--
	e := s.stack[s.count]
	s.stack = s.stack[:s.count]
	return e, nil
}

// newStringStack returns a new empty stack.
func NewStringStack() *stringStack {
	return &stringStack{make([]string, 0), 0}
}
