package main

import (
	"fmt"
	"time"

	"github.com/shomali11/util/xconcurrency"
)

func main() {
	func1 := func() {
		for char := 'a'; char < 'a'+3; char++ {
			fmt.Printf("%c \n", char)
		}
	}

	func2 := func() {
		for number := 1; number < 4; number++ {
			fmt.Printf("%d \n", number)
		}
	}

	xconcurrency.Parallelize(func1, func2)                     // a 1 b 2 c 3
	xconcurrency.ParallelizeTimeout(time.Minute, func1, func2) // a 1 b 2 c 3

}
