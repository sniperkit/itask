package main

import (
	"fmt"

	"github.com/sniperkit/xtask/pkg"
	"github.com/sniperkit/xtask/plugin/aggregate/service/github"
)

func main() {

	loadConfig()
	newGlobalTaskers()

	newFact()

	ghClient, errComponent = githubClient(&config)
	if errComponent != nil {
		log.Fatal(errComponent.Error())
	}

	_, resp, err := ghClient.GetFunc("getStars", &github.Options{
		Page:     1,
		PerPage:  config.Service.Github.PerPage,
		Runner:   config.Service.Github.Runner,
		Accounts: config.Service.Github.Accounts,
	})
	if err != nil {
		log.Fatalln("error: ", err)
	}

	if config.App.Verbose {
		log.Println("LastPage:", resp.LastPage)
	}

	if config.Service.Github.MaxPage < 0 {
		config.Service.Github.MaxPage = resp.LastPage
	}

	// move separately ?!
	// ghClient.LoadCache(config.Service.Github.MaxPage*config.Service.Github.PerPage*5, ghClient.PrefixApi(), ghClient.PrefixApi(), nil) //, []string{"/starred"})
	writer := "test"
	newWriter(writer)
	for i := config.Service.Github.Offset; i <= config.Service.Github.MaxPage; i++ {
		taskName := fmt.Sprintf("activity-starred-%d", i)
		starTasker.Add(taskName, nil, getStarsFunc(taskName, i)).ContinueWithHandler(exportInterface).ContinueWithHandler(func(result xtask.TaskInfo) {
			fmt.Println("ContinueWithHandler().taskName", taskName)
		})
		runGlobalTaskers()
		writers[writer].Flush()
		newGlobalTaskers()
	}
	runGlobalTaskers()

}
