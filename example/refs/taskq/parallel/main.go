package main

import (
	"log"
	"time"

	taskqPkg "github.com/xh3b4sd/taskq"
)

func main() {
	ctx := &Ctx{}
	q := taskqPkg.NewQueue(ctx)

	err := q.RunTasks(
		taskqPkg.InParallel(
			Task1,
			Task3,
			Task4,
			Task2,
		),
	)

	if err != nil {
		panic(err)
	}

}

type Ctx struct {
	Task1 string
	Task2 int
	Task3 []string
	Task4 float64
}

func sleep() {
	time.Sleep(10 * time.Millisecond)
}

// Simple tasks.
func Task1(ctx interface{}) error {
	ctx.(*Ctx).Task1 = "task1"
	log.Println("executing task1")
	return nil
}

func Task2(ctx interface{}) error {
	ctx.(*Ctx).Task2 = 2
	log.Println("executing task2")
	return nil
}

func Task3(ctx interface{}) error {
	ctx.(*Ctx).Task3 = []string{"task3"}
	log.Println("executing task3")
	return nil
}

func Task4(ctx interface{}) error {
	ctx.(*Ctx).Task4 = 4.4
	log.Println("executing task4")
	return nil
}

// Long running tasks.
func Task1Sleep(ctx interface{}) error {
	sleep()
	ctx.(*Ctx).Task1 = "task1"
	log.Println("sleeping task1")
	return nil
}

func Task2Sleep(ctx interface{}) error {
	sleep()
	ctx.(*Ctx).Task2 = 2
	log.Println("sleeping task2")
	return nil

}

func Task3Sleep(ctx interface{}) error {
	sleep()
	ctx.(*Ctx).Task3 = []string{"task3"}
	log.Println("sleeping task3")
	return nil
}

func Task4Sleep(ctx interface{}) error {
	sleep()
	ctx.(*Ctx).Task4 = 4.4
	log.Println("sleeping task4")
	return nil
}
