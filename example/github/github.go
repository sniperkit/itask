package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/sniperkit/xtask/pkg"
	"github.com/sniperkit/xtask/plugin/aggregate/service/github"
	"github.com/sniperkit/xtask/util/runtime"

	"github.com/segmentio/stats"
)

var gh *github.Github

var (
	starredFields []string = []string{"name", "owner.login"}
	readmeFields  []string = []string{"content"}
	repoFields    []string = []string{"name", "owner.login", "DefaultBranch"}
	treeFields    []string = []string{"name", "owner.login", "DefaultBranch"}
)

func visisted(taskName string) bool {
	return gh.CacheSlugExists(taskName)
}

var getStars = func(page int) {
	t := time.Now()
	counters.Increment("getStars.task.queued", 1)

	stars, _, err := gh.Get("getStars", &github.Options{ // gh.Get
		Page:    page,
		PerPage: config.Service.Github.PerPage,
		Runner:  config.Service.Github.Runner,
	})

	if err != nil {
		counters.Increment("getStars.task.failure", 1)
		log.Fatalln("error: ", err.Error(), "debug=", runtime.WhereAmI())
	} else {
		counters.Increment("getStars.task.success", 1)
		for k, _ := range stars {

			//

			r := strings.Split(k, "/")
			taskName := fmt.Sprintf("repos/%s/topics", k)
			if !visisted(taskName) {
				log.Println("[ADD] taskName ", taskName)
				preloadList.AddTask(xtask.NewTask(taskName, getTopics, r[0], r[1]))
			} else {
				log.Println("[SKIP] taskName ", taskName, "visited", visisted(taskName))
				counters.Increment("getTopics.task.dequeued", 1)
			}

			taskName = fmt.Sprintf("repos/%s/readme", k)
			if !visisted(taskName) {
				log.Println("[ADD] taskName ", taskName)
				preloadList.AddTask(xtask.NewTask(taskName, getReadme, r[0], r[1]))
			} else {
				log.Println("[SKIP] taskName ", taskName, "visited", visisted(taskName))
				counters.Increment("getReadme.task.dequeued", 1)
			}

			taskName = fmt.Sprintf("repos/%s", k)
			//if !visisted(taskName) {
			log.Println("[ADD] taskName ", taskName)
			preloadList.AddTask(xtask.NewTask(taskName, getRepo, r[0], r[1]))
			//} else {
			//	log.Println("[SKIP] taskName ", taskName, "visited", visisted(taskName))
			//	counters.Increment("getRepo.task.dequeued", 1)
			//}
		}
		// log.Println("cache.count()", gh.CacheCount(), "starredList.Len():", starredList.Len(), "preloadList.Len():", preloadList.Len())
	}

	// log.Println("counters.Snapshot()=", counters.Snapshot())

	callTime := time.Now().Sub(t)
	m := &funcMetrics{}
	m.calls.count = 1
	m.calls.time = callTime
	if err != nil {
		m.calls.failed = 1
	}
	stats.Report(m)
}

var getReadme = func(owner, name string) {
	t := time.Now()
	counters.Increment("getReadme.task.queued", 1)

	_, _, err := gh.Get("getReadme", &github.Options{
		Runner: config.Service.Github.Runner,
		Target: &github.Target{
			Owner: owner,
			Name:  name,
		},
	})
	if err != nil {
		counters.Increment("getReadme.task.failure", 1)
		log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
	} else {
		counters.Increment("getReadme.task.success", 1)
	}

	callTime := time.Now().Sub(t)
	m := &funcMetrics{}
	m.calls.count = 1
	m.calls.time = callTime
	if err != nil {
		m.calls.failed = 1
	}
	stats.Report(m)
}

var getTopics = func(owner, name string) {
	t := time.Now()
	counters.Increment("getTopics.task.queued", 1)

	_, _, err := gh.Get("getTopics", &github.Options{
		Runner: config.Service.Github.Runner,
		Target: &github.Target{
			Owner: owner,
			Name:  name,
		},
	})
	if err != nil {
		counters.Increment("getTopics.task.failure", 1)
		log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
	} else {
		counters.Increment("getTopics.task.success", 1)
	}

	callTime := time.Now().Sub(t)
	m := &funcMetrics{}
	m.calls.count = 1
	m.calls.time = callTime
	if err != nil {
		m.calls.failed = 1
	}
	stats.Report(m)
}

var getTree = func(owner, name, ref string) {
	t := time.Now()
	counters.Increment("getTree.task.queued", 1)

	_, _, err := gh.Get("getTree", &github.Options{
		Runner: config.Service.Github.Runner,
		Target: &github.Target{
			Owner: owner,
			Name:  name,
			Ref:   ref,
		},
	})
	if err != nil {
		counters.Increment("getTree.task.failure", 1)
		log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
	} else {
		counters.Increment("getTree.task.success", 1)
	}

	if counters.Get("getTree.task.failure") > 5 {
		log.Fatalln("too much errors... debug=", runtime.WhereAmI())
	}

	callTime := time.Now().Sub(t)
	m := &funcMetrics{}
	m.calls.count = 1
	m.calls.time = callTime
	if err != nil {
		m.calls.failed = 1
	}
	stats.Report(m)
}

var getLatestSHA = func(owner, name, branch string) {
	t := time.Now()
	counters.Increment("getLatestSHA.task.queued", 1)
	sha, _, err := gh.Get("getLatestSHA", &github.Options{
		Runner: config.Service.Github.Runner,
		Target: &github.Target{
			Owner:  owner,
			Name:   name,
			Branch: branch,
		},
	})

	if err != nil {
		counters.Increment("getLatestSHA.task.failure", 1)
		log.Fatalln("error: ", err.Error(), "debug=", runtime.WhereAmI())
	} else {
		counters.Increment("getLatestSHA.task.success", 1)
		shaStr := *sha["sha"].(*string)
		taskName := fmt.Sprintf("%s/%s/%s/%s/tree", owner, name, branch, shaStr)
		//if !visisted(taskName) {
		log.Println("[ADD] taskName ", taskName)
		// treeList.AddTask(xtask.NewTask(taskName, getTree, owner, name, sha))    //.Delay(requestDelay))    // .ContinueWith(outputResult))
		preloadList.AddTask(xtask.NewTask(taskName, getTree, owner, name, sha)) //.Delay(requestDelay)) // .ContinueWith(outputResult))
		//} else {
		//	log.Println("[SKIP] taskName ", taskName, "visited", visisted(taskName))
		//	counters.Increment("getTree.task.dequeued", 1)
		//}
	}

	callTime := time.Now().Sub(t)
	m := &funcMetrics{}
	m.calls.count = 1
	m.calls.time = callTime
	if err != nil {
		m.calls.failed = 1
	}

	stats.Report(m)
}

var getRepo = func(owner, name string) {
	t := time.Now()
	counters.Increment("getRepo.task.queued", 1)
	repo, _, err := gh.Get("getRepo", &github.Options{
		Runner: config.Service.Github.Runner,
		Target: &github.Target{
			Owner: owner,
			Name:  name,
		},
	})

	if err != nil {
		counters.Increment("getRepo.task.failure", 1)
		log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
	} else {
		counters.Increment("getRepo.task.success", 1)
		branch := *repo["DefaultBranch"].(*string)
		//if branch == "" {
		//	branch = "master"
		//}

		taskName := fmt.Sprintf("repos/%v/%v/git/refs/heads/%v", owner, name, branch)
		log.Println("[getLatestSHA]=", taskName)
		//if !visisted(taskName) {
		// shaList.AddTask(xtask.NewTask(taskName, getLatestSHA, owner, name, branch))     //.Delay(requestDelay))     // .ContinueWith(outputResult))
		preloadList.AddTask(xtask.NewTask(taskName, getLatestSHA, owner, name, branch)) //.Delay(requestDelay)) // .ContinueWith(outputResult))
		//} else {
		log.Println("[SKIP] taskName ", taskName, "visited", visisted(taskName))
		//	counters.Increment("getLatestSHA.task.dequeued", 1)
		//}
	}

	callTime := time.Now().Sub(t)
	m := &funcMetrics{}
	m.calls.count = 1
	m.calls.time = callTime
	if err != nil {
		m.calls.failed = 1
	}
	stats.Report(m)

}