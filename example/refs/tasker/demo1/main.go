package main

import (
	"log"

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
	// gh.Get("getStars", opts)
	/*
		handler := func() string {
			log.Println("aaaaaaa")
			return "finished"
		}
	*/

	task := xtask.NewTask(gh.Get("getStars", opts))
	task.Run().Wait()

	log.Println(task.Result)

}
