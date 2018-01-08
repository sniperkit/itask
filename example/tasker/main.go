package main

import (
	"log"

	"github.com/sniperkit/go-tasker"
)

func print_task(msg string) tasker.Task {
	return func() error {
		log.Println(msg)
		return nil
	}
}

func main() {
	tr, err := tasker.NewTasker(2)
	if err != nil {
		log.Fatal(err)
	}

	err = tr.Add("d", nil, print_task("d"))
	if err != nil {
		log.Fatal(err)
	}

	err = tr.Add("a", []string{"b", "c"}, print_task("a"))
	if err != nil {
		log.Fatal(err)
	}

	err = tr.Add("b", nil, print_task("b"))
	if err != nil {
		log.Fatal(err)
	}

	err = tr.Add("c", []string{"d"}, print_task("c"))
	if err != nil {
		log.Fatal(err)
	}

	if err = tr.Run(); err != nil {
		log.Fatal(err)
	}
}
