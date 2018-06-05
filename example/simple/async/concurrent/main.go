package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/rafaeldias/async"
)

func main() {

	res, e := async.Concurrent(async.Tasks{
		func() int {
			for i := 'a'; i < 'a'+26; i++ {
				fmt.Printf("%c \n", i)
			}
			return 0
		},
		func() error {
			time.Sleep(3 * time.Microsecond)
			for i := 0; i < 27; i++ {
				fmt.Printf("%d \n", i)
			}
			return errors.New("Error executing concurently")
		},
	})

	if e != nil {
		fmt.Printf("Errors [%s]\n", e.Error()) // output errors separated by space
	}

	fmt.Println("Result from function 0: %v", res.Index(0))
}
