package xtask

/*
import (
	//"sync"

	"github.com/anacrolix/sync"
)

// Each return value of handler indicates whether to continue chain call
// return [true]: continue call
// return [false]: intercept call
type ChainBool struct {
	handlers []*Handler
	result   bool
}

func NewChainBool() *ChainBool {
	res := new(ChainBool)
	return res
}

func (c *ChainBool) Append(f interface{}, args ...interface{}) *ChainBool {
	h := NewHandler(f, args...)
	c.handlers = append(c.handlers, h)
	return c
}

func (c *ChainBool) Prepend(f interface{}, args ...interface{}) *ChainBool {
	h := NewHandler(f, args...)
	c.handlers = append([]*Handler{h}, c.handlers...)
	return c
}

func (c *ChainBool) Result() bool {
	return c.result
}

func (c *ChainBool) Call(f interface{}, args ...interface{}) *ChainBool {
	if !c.result {
		return c
	}
	h := NewHandler(f, args...)
	c.handlers = append(c.handlers, h)
	c.result = h.BoolCall()
	return c
}

// Check all the conditions in pipeline, funcs executed following the slice order.
// Once func returns false, the following func will not be executed.
func (c *ChainBool) Run() bool {
	for _, f := range c.handlers {
		if !f.BoolCall() {
			return false
		}
	}
	return true
}

// Check all the conditions in parallel, wait for result.
// All funcs will be executed, but return false if one func returns false.
func (c *ChainBool) Parallel() bool {
	// run parallel
	wg := &sync.WaitGroup{}
	for i := range c.handlers {
		wg.Add(1)
		go func(f *Handler) {
			f.BoolCall()
			wg.Done()
		}(c.handlers[i])
	}
	wg.Wait()

	// check results
	for _, f := range c.handlers {
		if len(f.ret) == 0 || !f.ret[0].Bool() {
			return false
		}
	}
	return true
}
*/
