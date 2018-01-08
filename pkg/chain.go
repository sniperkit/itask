package xtask

/*
type Chain struct {
	terminate bool
}

func NewChain() *Chain {
	res := new(Chain)
	return res
}

// Method for check condition.
// Input function must return bool value, return `false` will terminate the chain call.
// When MapIf func return false, the following calls will not be executed.
func (c *Chain) MapIf(name string, f interface{}, args ...interface{}) *Chain {
	if c.terminate {
		return c
	}
	h := NewFunc(name, f, args...)
	c.terminate = !h.BoolCall()
	return c
}

// Continue call iff the chain is not terminated
func (c *Chain) Call(name string, f interface{}, args ...interface{}) *Chain {
	if c.terminate {
		return c
	}
	h := NewFunc(name, f, args...)
	h.Call()
	return c
}

func (c *Chain) Result() bool {
	return !c.terminate
}
*/
