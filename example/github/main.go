package main

import (
	"fmt"
	"log"
	"time"

	"github.com/jinzhu/configor"
	"github.com/sniperkit/xtask/plugin/aggregate/service/github"
)

func main() {
	newGlobalTaskers()

	t := time.Now()
	configor.Load(&config, "config/config.yaml")
	addMetrics(t, 1, false)

	t = time.Now()
	gh = github.New(config.Service.Github.Tokens, &github.Options{
		Page:    1,
		PerPage: config.Service.Github.PerPage,
		Runner:  config.Service.Github.Runner,
	})
	addMetrics(t, 1, false)

	t = time.Now()
	_, resp, err := gh.Get("getStars", &github.Options{
		Page:     1,
		PerPage:  config.Service.Github.PerPage,
		Runner:   config.Service.Github.Runner,
		Accounts: config.Service.Github.Accounts,
	})
	addMetrics(t, 1, err != nil)

	if config.App.Verbose {
		log.Println("LastPage:", resp.LastPage)
	}

	if config.Service.Github.MaxPage < 0 {
		config.Service.Github.MaxPage = resp.LastPage
	}

	gh.LoadCache(config.Service.Github.MaxPage*config.Service.Github.PerPage*5, gh.PrefixApi(), gh.PrefixApi(), nil) //, []string{"/starred"})

	for i := config.Service.Github.Offset; i <= config.Service.Github.MaxPage; i++ {
		taskName := fmt.Sprintf("activity-starred-%d", i)
		err := starTasker.Add(taskName, nil, getStarsFunc(i))
		if err != nil {
			log.Fatal(err)
		}
		runGlobalTaskers()
		newGlobalTaskers()
	}
	runGlobalTaskers()

}
