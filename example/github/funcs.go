package main

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/sniperkit/xtask/pkg"
	"github.com/sniperkit/xtask/plugin/aggregate/service/github"
	"github.com/sniperkit/xtask/util/runtime"

	"github.com/segmentio/stats"
)

func printTask(msg string) xtask.Tsk {
	return func() error {
		log.Println(msg)
		return nil
	}
}

func taskerTest() {
	tr, err := xtask.NewTasker(10)
	if err != nil {
		log.Fatal(err)
	}

	err = tr.Add("d", nil, printTask("d"))
	if err != nil {
		log.Fatal(err)
	}

	err = tr.Add("a", []string{"b", "c"}, printTask("a"))
	if err != nil {
		log.Fatal(err)
	}

	err = tr.Add("b", nil, printTask("b"))
	if err != nil {
		log.Fatal(err)
	}

	err = tr.Add("c", []string{"d"}, printTask("c"))
	if err != nil {
		log.Fatal(err)
	}

	if err = tr.Run(); err != nil {
		log.Fatal(err)
	}
}

func getStarsFunc(page int) xtask.Tsk {

	return func() error {

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

				r := strings.Split(k, "/")
				taskName := strings.Replace(fmt.Sprintf("repos.%s.topics", k), "/", ".", -1)
				if !visisted(taskName) {
					log.Println("[ADD] taskName ", taskName)
					err := preloadTasker.Add(taskName, nil, getTopicsFunc(r[0], r[1]))
					if err != nil {
						log.Println("preloadTasker.error(): ", err)
					}
				} else {
					// log.Println("[SKIP] taskName ", taskName, "visited", visisted(taskName))
					counters.Increment("getTopics.task.dequeued", 1)
				}

				taskName = strings.Replace(fmt.Sprintf("repos.%s.readme", k), "/", ".", -1)
				if !visisted(taskName) {
					log.Println("[ADD] taskName ", taskName)
					err := preloadTasker.Add(taskName, nil, getReadmeFunc(r[0], r[1]))
					if err != nil {
						log.Println("preloadTasker.error(): ", err)
					}
				} else {
					// log.Println("[SKIP] taskName ", taskName, "visited", visisted(taskName))
					counters.Increment("getReadme.task.dequeued", 1)
				}

				taskName = strings.Replace(fmt.Sprintf("repos.%s", k), "/", ".", -1)
				//if !visisted(taskName) {
				log.Println("[ADD] taskName ", taskName)
				err := preloadTasker.Add(taskName, nil, getRepoFunc(r[0], r[1]))
				if err != nil {
					log.Println("preloadTasker.error(): ", err)
				}
				//} else {
				//	log.Println("[SKIP] taskName ", taskName, "visited", visisted(taskName))
				//	counters.Increment("getRepo.task.dequeued", 1)
				//}
			}
		}

		callTime := time.Now().Sub(t)
		m := &funcMetrics{}
		m.calls.count = 1
		m.calls.time = callTime
		if err != nil {
			m.calls.failed = 1
		}
		stats.Report(m)

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

		callTime := time.Now().Sub(t)
		m := &funcMetrics{}
		m.calls.count = 1
		m.calls.time = callTime
		if err != nil {
			m.calls.failed = 1
		}
		stats.Report(m)

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

		callTime := time.Now().Sub(t)
		m := &funcMetrics{}
		m.calls.count = 1
		m.calls.time = callTime
		if err != nil {
			m.calls.failed = 1
		}
		stats.Report(m)

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

		/*
			if counters.Get("getTree.task.failure") > 5 {
				log.Fatalln("too much errors... debug=", runtime.WhereAmI())
			}
		*/

		callTime := time.Now().Sub(t)
		m := &funcMetrics{}
		m.calls.count = 1
		m.calls.time = callTime
		if err != nil {
			m.calls.failed = 1
		}
		stats.Report(m)

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
				return errors.New("error latest SHA")
			}

			counters.Increment("getLatestSHA.task.success", 1)
			// pp.Println(sha["sha"])
			shaStr := sha["sha"].(string)
			taskName := strings.Replace(fmt.Sprintf("%s.%s.%s.%s.tree", owner, name, branch, shaStr), "/", ".", -1)
			if !visisted(taskName) {
				log.Println("[ADD] taskName ", taskName)
				err := treeTasker.Add(taskName, nil, getTreeFunc(owner, name, shaStr))
				if err != nil {
					log.Println("extraTasker.error(): ", err)
				}
			} else {
				// log.Println("[SKIP] taskName ", taskName, "visited", visisted(taskName))
				counters.Increment("getLatestSHA.task.dequeued", 1)
			}
		}

		callTime := time.Now().Sub(t)
		m := &funcMetrics{}
		m.calls.count = 1
		m.calls.time = callTime
		if err != nil {
			m.calls.failed = 1
		}

		stats.Report(m)

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
			counters.Increment("getRepo.task.success", 1)

			branch := *repo["DefaultBranch"].(*string)
			taskName := strings.Replace(fmt.Sprintf("repos.%v.%v.git.refs.heads.%v", owner, name, branch), "/", ".", -1)
			if !visisted(taskName) {
				err := extraTasker.Add(taskName, nil, getLatestSHAFunc(owner, name, branch))
				if err != nil {
					log.Println("extraTasker.error(): ", err)
				}

			} else {
				// log.Println("[SKIP] taskName ", taskName, "visited", visisted(taskName))
				counters.Increment("getLatestSHA.task.dequeued", 1)
			}
		}

		callTime := time.Now().Sub(t)
		m := &funcMetrics{}
		m.calls.count = 1
		m.calls.time = callTime
		if err != nil {
			m.calls.failed = 1
		}
		stats.Report(m)

		return err
	}

}
