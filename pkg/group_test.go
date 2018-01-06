package xtask

import (
	"fmt"
	"testing"
	"time"
)

func TestTaskGroup_Add(t *testing.T) {
	fmt.Println("==================================TestTaskGroup_Add=================================")
	handler1 := func() {
		fmt.Println(time.Now())
		fmt.Println("TestTaskGroup_Add: handler1")
	}
	param2 := "TestTaskGroup_Add:aaaaaaaaaaaaaaaaaaaaaa"
	handler2 := func(p string) {
		fmt.Println("TestTaskGroup_Add:handler2", time.Now())
		fmt.Println(p)
	}
	param3 := "TestTaskGroup_Add: bbbbbbbbbbbbbbbbbbbbbbbbbb"
	handler3 := func(p string) string {
		fmt.Println(p)
		return p + "111111111111111"
	}

	task1 := NewTask(handler1)
	task2 := NewTask(handler2, param2)
	task3 := NewTask(handler3, param3)

	taskList := NewTaskGroup()

	taskList.AddRange(task1, task2, task3).Run().WaitAll()
}
