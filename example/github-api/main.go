package main

import (
	"fmt"
	"time"

	"github.com/sniperkit/xtask/plugin/aggregate/service/github"
)

func main() {

	loadConfig()
	newGlobalTaskers()

	ghClient = githubClient()
	// mapFunc("Get", "Get") // Get or GetFunc

	t := time.Now()
	_, resp, err := ghClient.Get("getStars", &github.Options{
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

	// move separately ?!
	t = time.Now()
	ghClient.LoadCache(config.Service.Github.MaxPage*config.Service.Github.PerPage*5, ghClient.PrefixApi(), ghClient.PrefixApi(), nil) //, []string{"/starred"})
	addMetrics(t, 1, false)

	for i := config.Service.Github.Offset; i <= config.Service.Github.MaxPage; i++ {

		t = time.Now()
		taskName := fmt.Sprintf("activity-starred-%d", i)
		err := starTasker.Add(taskName, nil, getStarsFunc(i))
		if err != nil {
			log.Fatal(err.Error())
		}
		addMetrics(t, 1, false)

		t = time.Now()
		runGlobalTaskers()
		addMetrics(t, 1, false)

		t = time.Now()
		newGlobalTaskers()
		addMetrics(t, 1, false)

	}

	t = time.Now()
	runGlobalTaskers()
	addMetrics(t, 1, false)

}
