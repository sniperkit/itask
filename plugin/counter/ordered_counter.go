package counter

import (
	"container/list"
	"sync"
	"unicode/utf8"
)

type (
	Oc struct {
		set  map[string]*list.Element
		list *list.List
		lock *sync.RWMutex
		cur  *list.Element
	}

	element struct {
		key string
		ct  int
	}
	order int
)

const (
	ASC  order = 1
	DESC order = -1
)

func NewOc() *Oc {
	return &Oc{
		set:  make(map[string]*list.Element),
		list: list.New(),
		lock: &sync.RWMutex{},
	}
}

func (o *Oc) Increment(key string, val int) {
	o.lock.Lock()
	defer o.lock.Unlock()

	if el, exists := o.set[key]; exists {
		el.Value.(*element).ct += val
	} else {
		o.set[key] = o.list.PushBack(&element{key: key, ct: val})
	}
}

func (o *Oc) Decrement(key string, val int) {
	o.lock.Lock()
	defer o.lock.Unlock()

	o.Increment(key, -val)
}

func (o *Oc) Delete(key string) {
	o.lock.Lock()
	defer o.lock.Unlock()

	if el, exists := o.set[key]; exists {
		o.list.Remove(el)
		delete(o.set, key)
	}
}

func (o *Oc) Get(key string) int {
	o.lock.Lock()
	defer o.lock.Unlock()

	if el, exists := o.set[key]; exists {
		return el.Value.(*element).ct
	}

	return 0
}

func (o *Oc) Len() int {
	// o.lock.Lock()
	// defer o.lock.Unlock()

	length := len(o.set)
	return length
}

func (o *Oc) Next() bool {
	o.lock.Lock()
	defer o.lock.Unlock()

	// first time through
	if o.cur == nil {
		o.cur = o.list.Front()
		return true
	}

	o.cur = o.cur.Next()

	return o.cur != nil

}

func (o *Oc) Snapshot() map[string]int {
	stats := make(map[string]int, o.Len())
	o.SortByKey(ASC)
	for o.Next() {
		if o.cur != nil {
			key, value := o.KeyValue()
			stats[key] = value
		}
	}

	return stats
}

func keyValue(o *Oc) (string, int) {
	e := o.cur.Value.(*element)
	return e.key, e.ct
}

func (o *Oc) KeyValue() (string, int) {
	// o.lock.Lock()
	// defer o.lock.Unlock()

	e := o.cur.Value.(*element)

	return e.key, e.ct
}

func (o *Oc) SortByKey(dir order) {
	// o.lock.Lock()
	// defer o.lock.Unlock()

	cursor := o.list.Front()
	d := int(dir)

	for cursor != nil {

		// grab prev to process and next so we don't lose our place
		prev, next := cursor.Prev(), cursor.Next()

		// move backward until a cmp has been found
		for prev != nil && strcmp(prev.Value.(*element).key, cursor.Value.(*element).key)*d > 0 {
			prev = prev.Prev()
		}

		if prev == nil {
			o.list.MoveToFront(cursor)
		} else if prev != cursor.Prev() {
			o.list.MoveAfter(cursor, prev)
		}

		cursor = next

	}
}

func (o *Oc) SortByCt(dir order) {
	// o.lock.Lock()
	// defer o.lock.Unlock()

	cursor := o.list.Front()
	d := int(dir)

	for cursor != nil {

		// grab prev to process and next so we don't lose our place
		prev, next := cursor.Prev(), cursor.Next()

		// move backward until a cmp has been found
		for prev != nil && (prev.Value.(*element).ct-cursor.Value.(*element).ct)*d > 0 {
			prev = prev.Prev()
		}

		if prev == nil {
			o.list.MoveToFront(cursor)
		} else if prev != cursor.Prev() {
			o.list.MoveAfter(cursor, prev)
		}
		cursor = next
	}

}

func strcmp(a, b string) int {
	for len(a) > 0 && len(b) > 0 {
		ra, sizea := utf8.DecodeRuneInString(a)
		rb, sizeb := utf8.DecodeRuneInString(b)
		if ra != rb {
			return int(ra - rb)
		}
		a, b = a[sizea:], b[sizeb:]
	}

	// return the shorter
	return len(a) - len(b)

}
