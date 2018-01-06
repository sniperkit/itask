package main

import (
	"fmt"
	"time"

	"github.com/rafaeldias/async"
)

func main() {

	res, e := async.Parallel(async.MapTasks{
		"one": func() int {
			for i := 'a'; i < 'a'+26; i++ {
				fmt.Printf("%c \n", i)
			}

			return 1
		},
		"two": func() int {
			time.Sleep(2 * time.Microsecond)
			for i := 0; i < 27; i++ {
				fmt.Printf("%d \n", i)
			}

			return 2
		},
		"three": func() int {
			for i := 'z'; i >= 'a'; i-- {
				fmt.Printf("%c \n", i)
			}

			return 3
		},
	})

	if e != nil {
		fmt.Printf("Errors [%s]\n", e.Error())
	}

	fmt.Println("Results from task 'two': %v", res.Key("two"))
}
