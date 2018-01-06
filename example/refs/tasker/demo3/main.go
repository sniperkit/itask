package main

import (
	"log"
	"time"

	"github.com/sniperkit/xtask/pkg"
	"github.com/sniperkit/xtask/test/service/github"
)

var token string

func main() {

	token = "3a2cd56ac1985bb5da567a14723fb104dc7be97f"
	opts := &github.Options{
		Page:    1,
		PerPage: 100,
		Runner:  "roscopecoltran",
	}

	gh := github.New(&token, opts)
	gh.Get("getStars", opts)

	handler1 := func() {
		log.Println(time.Now())
		log.Println("TestTaskList_Add: handler1")
	}
	param2 := "TestTaskList_Add:aaaaaaaaaaaaaaaaaaaaaa"

	handler2 := func(p string) {
		log.Println("TestTaskList_Add:handler2", time.Now())
		log.Println(p)
	}
	param3 := "TestTaskList_Add: bbbbbbbbbbbbbbbbbbbbbbbbbb"
	handler3 := func(p string) string {
		log.Println(p)
		return p + "111111111111111"
	}

	task1 := xtask.NewTask(handler1)
	task2 := xtask.NewTask(handler2, param2)
	task3 := xtask.NewTask(handler3, param3)

	taskList := xtask.NewTaskList()

	taskList.AddRange(task1, task2, task3).Run().WaitAll()

}
