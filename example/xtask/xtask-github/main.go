package main

import (
	"fmt"

	"github.com/sniperkit/xtask/plugin/aggregate/service/github"
)

func main() {

	loadConfig()
	newGlobalTaskers()
	newFact()
	initWriters(false, writersList...)
	skipTasksWithTTL("./shared/data/export/csv/tasks.csv")

	clientManager = githubClients(&config)
	defer clientManager.Shutdown()

	ghClient = clientManager.Fetch()

	_, resp, err := ghClient.GetFunc("getStars", &github.Options{
		Page:     1,
		PerPage:  config.Service.Github.PerPage,
		Runner:   config.Service.Github.Runner,
		Accounts: config.Service.Github.Accounts,
	})
	if err != nil {
		log.Errorln("error: ", err)
	}

	if config.App.Verbose {
		log.Println("LastPage:", resp.LastPage)
	}

	if config.Service.Github.MaxPage < 0 {
		config.Service.Github.MaxPage = resp.LastPage
	}

	for i := config.Service.Github.Offset; i <= config.Service.Github.MaxPage; i++ {
		taskName := fmt.Sprintf("%s.activity-starred-%d", defaultSvc, i)
		starTasker.Add(taskName, "starred", nil, getStarsFunc(taskName, i), 0) // .ContinueWithHandler(exportInterface) // .ContinueWithHandler(func(result *xtask.TaskInfo) { fmt.Println("ContinueWithHandler().taskName", taskName) })
		runTasks()
		flushWriters()
		newGlobalTaskers()
	}

	runTasks()
	flushWriters()

}
