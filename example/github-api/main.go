package main

import (
	"fmt"

	"github.com/sniperkit/xtask/plugin/aggregate/service/github"
)

func main() {

	writers["stars"] = newWriterJSON2CSV("stars")
	writers["latest_sha"] = newWriterJSON2CSV("latest_sha")
	writers["users"] = newWriterJSON2CSV("users")
	writers["repos"] = newWriterJSON2CSV("repos")
	writers["readmes"] = newWriterJSON2CSV("readmes")

	writers["topics"] = newWriterJSON2CSV("topics")

	// writers["topics_list"] = newWriterJSON2CSV("topics_list")
	// writers["topic_details"] = newWriterJSON2CSV("topic_details")
	// writers["topics_all"] = newWriterJSON2CSV("topics_all")

	// writers["topics_flatten"] = newWriterJSON2CSV("topics_flatten")
	// writers["topics_list_flatten"] = newWriterJSON2CSV("topics_list_flatten")
	// writers["topic_details_flatten"] = newWriterJSON2CSV("topic_details_flatten")

	writers["files"] = newWriterJSON2CSV("files")

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

	for i := config.Service.Github.Offset; i <= config.Service.Github.MaxPage; i++ {
		taskName := fmt.Sprintf("activity-starred-%d", i)
		starTasker.Add(taskName, "starred", nil, getStarsFunc(taskName, i)) // .ContinueWithHandler(exportInterface) // .ContinueWithHandler(func(result *xtask.TaskInfo) { fmt.Println("ContinueWithHandler().taskName", taskName) })
		runGlobalTaskers()
		flushAllWriters()
		newGlobalTaskers()
	}

	runGlobalTaskers()
	flushAllWriters()

}
