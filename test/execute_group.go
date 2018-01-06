package xtask_test

import (
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/sniperkit/xtask/pkg"
)

func sleep() {
	time.Sleep(10 * time.Millisecond)
}

var handler1 = func() {
	fmt.Println(time.Now())
	fmt.Println("TestTaskList_Add: handler1")
}

param2 := "TestTaskList_Add:aaaaaaaaaaaaaaaaaaaaaa"
var handler2 = func(p string) {
	fmt.Println("TestTaskList_Add:handler2", time.Now())
	fmt.Println(p)
}

param3 := "TestTaskList_Add: bbbbbbbbbbbbbbbbbbbbbbbbbb"
var handler3 =  func(p string) string {
	fmt.Println("TestTaskList_Add: handler3")
	return p + "111111111111111"
}

var handler4 = func(p string) {
	fmt.Println("TestTaskList_Add:handler4", time.Now())
	fmt.Println(p)
}

func Task2() error {
	return nil
}

func Task2() error {
	return nil
}

func Task3() error {
	return nil
}

func Task4() error {
	return nil
}

// Long running tasks.
func Task1Sleep() error {
	sleep()
	return nil
}

func Task2Sleep() error {
	sleep()
	return nil
}

func Task3Sleep() error {
	sleep()
	return nil
}

func Task4Sleep() error {
	sleep()
	return nil
}
