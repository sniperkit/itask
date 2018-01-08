package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/sniperkit/xtask/pkg"
	"github.com/sniperkit/xtask/plugin/aggregate/service/github"
	"github.com/sniperkit/xtask/util/runtime"
)

var (
	starredList *xtask.TaskQueue = xtask.NewTaskQueue()
	metaList    *xtask.TaskQueue = xtask.NewTaskQueue()
	refList     *xtask.TaskQueue = xtask.NewTaskQueue()
	treeList    *xtask.TaskQueue = xtask.NewTaskQueue()
)

var (
	starredFields []string = []string{"name", "owner.login"}
	readmeFields  []string = []string{"content"}
	repoFields    []string = []string{"name", "owner.login", "DefaultBranch"}
	treeFields    []string = []string{"name", "owner.login", "DefaultBranch"}
)

func visisted(taskName string) bool {
	return ghClient.CacheSlugExists(taskName)
}

var getStars = func(page int) {
	t := time.Now()
	counters.Increment("getStars.task.queued", 1)
	stars, _, err := ghClient.Get("getStars", &github.Options{
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
			r := strings.Split(k, "/")
			// topics
			taskName := fmt.Sprintf("repos/%s/topics", k)
			if !visisted(taskName) {
				log.Println("[ADD] taskName ", taskName)
				metaList.AddTask(xtask.NewTask(taskName, getTopics, r[0], r[1]))
			} else {
				counters.Increment("getTopics.task.dequeued", 1)
			}
			// readme
			taskName = fmt.Sprintf("repos/%s/readme", k)
			if !visisted(taskName) {
				log.Println("[ADD] taskName ", taskName)
				metaList.AddTask(xtask.NewTask(taskName, getReadme, r[0], r[1]))
			} else {
				counters.Increment("getReadme.task.dequeued", 1)
			}
			// repo info
			taskName = fmt.Sprintf("repos/%s", k)
			log.Println("[ADD] taskName ", taskName)
			metaList.AddTask(xtask.NewTask(taskName, getRepo, r[0], r[1]))
		}
	}
	addMetrics(t, 1, err != nil)
}

var getRepo = func(owner, name string) {
	t := time.Now()
	counters.Increment("getRepo.task.queued", 1)
	repo, _, err := ghClient.Get("getRepo", &github.Options{
		Runner: config.Service.Github.Runner,
		Target: &github.Target{
			Owner: owner,
			Name:  name,
		},
	})
	if err != nil {
		counters.Increment("getRepo.task.failure", 1)
		log.Fatalln("error: ", err.Error(), "debug=", runtime.WhereAmI())
	} else {
		if repo == nil {
			log.Fatalln("error: ", errRepoInfo, "debug=", runtime.WhereAmI())
		}
		counters.Increment("getRepo.task.success", 1)
		branch := *repo["DefaultBranch"].(*string)

		taskName := fmt.Sprintf("repos/%v/%v/git/refs/heads/%v", owner, name, branch)
		log.Println("[getLatestSHA]=", taskName)
		if !visisted(taskName) {
			refList.AddTask(xtask.NewTask(taskName, getLatestSHA, owner, name, branch)) //.Delay(requestDelay)).ContinueWith(outputResult))
		} else {
			counters.Increment("getLatestSHA.task.dequeued", 1)
		}
	}
	addMetrics(t, 1, err != nil)
}

var getLatestSHA = func(owner, name, branch string) {
	t := time.Now()
	counters.Increment("getLatestSHA.task.queued", 1)
	sha, _, err := ghClient.Get("getLatestSHA", &github.Options{
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
		if sha == nil {
			log.Fatalln("error: ", errNotFoundLatestSHA, "debug=", runtime.WhereAmI())
		}
		counters.Increment("getLatestSHA.task.success", 1)
		shaStr := sha["sha"].(string)
		taskName := fmt.Sprintf("%s/%s/%s/%s/tree", owner, name, branch, shaStr)
		if !visisted(taskName) {
			log.Println("[ADD] taskName ", taskName)
			treeList.AddTask(xtask.NewTask(taskName, getTree, owner, name, sha)) //.Delay(requestDelay)).ContinueWith(outputResult))
		} else {
			counters.Increment("getTree.task.dequeued", 1)
		}
	}
	addMetrics(t, 1, err != nil)
}

var getReadme = func(owner, name string) {
	t := time.Now()
	counters.Increment("getReadme.task.queued", 1)
	_, _, err := ghClient.Get("getReadme", &github.Options{
		Runner: config.Service.Github.Runner,
		Target: &github.Target{
			Owner: owner,
			Name:  name,
		},
	})
	if err != nil {
		counters.Increment("getReadme.task.failure", 1)
		log.Fatalln("error: ", err.Error(), "debug=", runtime.WhereAmI())
	} else {
		counters.Increment("getReadme.task.success", 1)
	}
	addMetrics(t, 1, err != nil)
}

var getTopics = func(owner, name string) {
	t := time.Now()
	counters.Increment("getTopics.task.queued", 1)
	_, _, err := ghClient.Get("getTopics", &github.Options{
		Runner: config.Service.Github.Runner,
		Target: &github.Target{
			Owner: owner,
			Name:  name,
		},
	})
	if err != nil {
		counters.Increment("getTopics.task.failure", 1)
		log.Fatalln("error: ", err.Error(), "debug=", runtime.WhereAmI())
	} else {
		counters.Increment("getTopics.task.success", 1)
	}
	addMetrics(t, 1, err != nil)
}

var getTree = func(owner, name, ref string) {
	t := time.Now()
	counters.Increment("getTree.task.queued", 1)
	_, _, err := ghClient.Get("getTree", &github.Options{
		Runner: config.Service.Github.Runner,
		Target: &github.Target{
			Owner: owner,
			Name:  name,
			Ref:   ref,
		},
	})
	if err != nil {
		counters.Increment("getTree.task.failure", 1)
		log.Fatalln("error: ", err.Error(), "debug=", runtime.WhereAmI())
	} else {
		counters.Increment("getTree.task.success", 1)
	}
	addMetrics(t, 1, err != nil)
}
