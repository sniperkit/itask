package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/sniperkit/xtask/pkg"
	"github.com/sniperkit/xtask/plugin/aggregate/service/github"
	"github.com/sniperkit/xtask/util/runtime"
)

var (
	starTasker *xtask.Tasker
	metaTasker *xtask.Tasker
	refTasker  *xtask.Tasker
	treeTasker *xtask.Tasker
)

func newGlobalTaskers() {
	starTasker, _ = xtask.NewTasker(20)
	metaTasker, _ = xtask.NewTasker(20)
	refTasker, _ = xtask.NewTasker(20)
	treeTasker, _ = xtask.NewTasker(20)
}

func runGlobalTaskers() {
	starTasker.Limiter(75, time.Minute).Tachymeter().Run()
	metaTasker.Limiter(17, time.Second).Tachymeter().Run()
	refTasker.Limiter(17, time.Second).Tachymeter().Run()
	treeTasker.Limiter(17, time.Second).Tachymeter().Run()
}

func getStarsFunc(page int) xtask.Tsk {
	return func() error {
		t := time.Now()
		counters.Increment("getStars.task.queued", 1)
		stars, _, err := gh.Get("getStars", &github.Options{
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
					err := metaTasker.Add(taskName, nil, getTopicsFunc(r[0], r[1]))
					if err != nil {
						log.Println("preloadTasker.error(): ", err)
					}
				} else {
					counters.Increment("getTopics.task.dequeued", 1)
				}
				// readme
				taskName = fmt.Sprintf("repos/%s/readme", k)
				if !visisted(taskName) {
					log.Println("[ADD] taskName ", taskName)
					err := metaTasker.Add(taskName, nil, getReadmeFunc(r[0], r[1]))
					if err != nil {
						log.Println("preloadTasker.error(): ", err)
					}
				} else {
					counters.Increment("getReadme.task.dequeued", 1)
				}
				// repo refs
				taskName = fmt.Sprintf("repos/%s", k)
				log.Println("[ADD] taskName ", taskName)
				err := metaTasker.Add(taskName, nil, getRepoFunc(r[0], r[1]))
				if err != nil {
					log.Println("preloadTasker.error(): ", err)
				}
			}
		}
		addMetrics(t, 1, err != nil)
		return err
	}
}

func getLatestSHAFunc(owner, name, branch string) xtask.Tsk {
	return func() error {
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
			return err
		} else {
			if sha == nil {
				return errNotFoundLatestSHA
			}
			counters.Increment("getLatestSHA.task.success", 1)
			shaStr := sha["sha"].(string)
			taskName := strings.Replace(fmt.Sprintf("%s.%s.%s.%s.tree", owner, name, branch, shaStr), "/", ".", -1)
			if !visisted(taskName) {
				log.Println("[ADD] taskName ", taskName)
				err := treeTasker.Add(taskName, nil, getTreeFunc(owner, name, shaStr))
				if err != nil {
					log.Println("treeTasker.error(): ", err)
				}
			} else {
				counters.Increment("getLatestSHA.task.dequeued", 1)
			}
		}
		addMetrics(t, 1, err != nil)
		return err
	}
}

func getRepoFunc(owner, name string) xtask.Tsk {
	return func() error {
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
			if repo == nil {
				return errRepoInfo
			}
			counters.Increment("getRepo.task.success", 1)
			branch := *repo["DefaultBranch"].(*string)
			taskName := strings.Replace(fmt.Sprintf("repos.%v.%v.git.refs.heads.%v", owner, name, branch), "/", ".", -1)
			if !visisted(taskName) {
				err := refTasker.Add(taskName, nil, getLatestSHAFunc(owner, name, branch))
				if err != nil {
					log.Println("refTasker.error(): ", err)
				}
			} else {
				counters.Increment("getLatestSHA.task.dequeued", 1)
			}
		}
		addMetrics(t, 1, err != nil)
		return err
	}
}
func getReadmeFunc(owner, name string) xtask.Tsk {
	return func() error {
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
			return err
		} else {
			counters.Increment("getReadme.task.success", 1)
		}
		addMetrics(t, 1, err != nil)
		return err
	}
}

func getTopicsFunc(owner, name string) xtask.Tsk {
	return func() error {
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
		addMetrics(t, 1, err != nil)
		return err
	}
}

func getTreeFunc(owner, name, ref string) xtask.Tsk {
	return func() error {
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
			return err
		} else {
			counters.Increment("getTree.task.success", 1)
		}
		addMetrics(t, 1, err != nil)
		return err
	}
}
