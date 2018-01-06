// Package collider provides a lock-free circular data structure for arbitrary Go objects.
package xtask

import (
	"errors"
	"sync/atomic"
)

// Ring is a write-once read-many circular data
// structure with slots that hold arbitrary Go objects.
type Ring struct {
	p0    [64]byte
	pos   uint64
	p1    [64]byte
	mask  *uint64
	p2    [64]byte
	slots []interface{}
}

// New takes a size n and returns
// a ring buffer with n slots.
func Collider(s int) (*Ring, error) {
	if (s & (s - 1)) != 0 {
		return nil, errors.New("size must be a power of 2")
	}
	m := uint64(s - 1)
	ring := &Ring{
		slots: make([]interface{}, s),
		mask:  &m,
	}

	return ring, nil
}

// Get returns the object at the current
// slot and atomically increments the index.
func (r *Ring) Get() interface{} {
	return r.slots[(atomic.AddUint64(&r.pos, 1)-1)&(*r.mask)]
}

// Add adds an item at the current slot and
// atomically increments the index.
func (r *Ring) Add(u interface{}) {
	r.slots[(atomic.AddUint64(&r.pos, 1)-1)&(*r.mask)] = u
}
