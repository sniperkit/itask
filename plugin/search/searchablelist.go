/*
	Copyright 2017 Travis Clarke. All rights reserved.
	Use of this source code is governed by a Apache-2.0 license that can be found in the LICENSE file.
*/

package searchablelist

import (
	"container/list"
)

// SearchableList list.List
type SearchableList struct {
	*list.List
}

// New () *list.List
func New() *SearchableList {
	return &SearchableList{new(list.List).Init()}
}

// (l *SearchableList)

// ContainsElement (t *list.Element) bool
func (l *SearchableList) ContainsElement(t *list.Element) bool {
	if l.Len() > 0 {
		for e := l.Front(); e != nil; e = e.Next() {
			if e == t {
				return true
			}
		}
	}
	return false
}

// Contains (t *list.Element) bool
// alias -> ContainsElement
func (l *SearchableList) Contains(t *list.Element) bool {
	return l.ContainsElement(t)
}

// ContainsValue (v interface{}) bool
func (l *SearchableList) ContainsValue(v interface{}) bool {
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
func (l *SearchableList) FindFirst(v interface{}) *list.Element {
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
func (l *SearchableList) FindLast(v interface{}) *list.Element {
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
func (l *SearchableList) FindAll(v interface{}) []*list.Element {
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
