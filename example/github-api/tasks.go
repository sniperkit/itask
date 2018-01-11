package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/sniperkit/xtask/pkg"
	"github.com/sniperkit/xtask/plugin/aggregate/service/github"
	"github.com/sniperkit/xtask/plugin/counter"
	"github.com/sniperkit/xtask/util/runtime"
)

var (
	starTasker     *xtask.Tasker
	metaTasker     *xtask.Tasker
	refTasker      *xtask.Tasker
	treeTasker     *xtask.Tasker
	requestDelay   time.Duration = 350 * time.Millisecond
	workerInterval time.Duration = time.Duration(random(150, 250)) * time.Millisecond
)

func printTasksInfo() {
	log.Println("tasks stats: ", getTasksInfo())
}

func mapFunc(funcName string, aliasName string) { // f func() error
	//	return backoff.Retry(f, backoff.WithMaxTries(backoff.WithContext(backoff.NewConstantBackOff(defaultRetryDelay), context.Background()), defaultRetryAttempt))
}

func getTasksInfo() map[string]int {
	info := make(map[string]int)
	counters.SortByKey(counter.ASC)
	for counters.Next() {
		if counters != nil {
			key, value := counters.KeyValue()
			info[key] = value
		}
	}
	return info
}

func postProcessor(input xtask.Tsk) xtask.Tsk {
	return input
}

func updateRequestDelay(beat int, unit string) (delay time.Duration) {
	input := time.Duration(beat)
	switch strings.ToLower(unit) {
	case "microsecond":
		delay = input * time.Microsecond
	case "millisecond":
		delay = input * time.Millisecond
	case "minute":
		delay = input * time.Minute
	case "hour":
		delay = input * time.Hour
	case "second":
		fallthrough
	default:
		delay = input * time.Second
	}
	return
}

func random(min, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}

func newGlobalTaskers() {
	starTasker, _ = xtask.NewTasker(20)
	metaTasker, _ = xtask.NewTasker(20)
	refTasker, _ = xtask.NewTasker(20)
	treeTasker, _ = xtask.NewTasker(20)
}

func runGlobalTaskers() {
	starTasker.Limiter(75, time.Minute).Tachymeter().Run()
	metaTasker.Limiter(28, time.Second).Tachymeter().Run()
	refTasker.Limiter(28, time.Second).Tachymeter().Run()
	treeTasker.Limiter(28, time.Second).Tachymeter().Run()
}

/*
	factdb.Let("dog").Has("name", "hou")
	factdb.Let("cat").Has("white", "black")
	store.AddQuad(quad.Make(siteURL, "type", "site", nil))
	store.AddQuad(quad.Make(siteURL, "name", "Red Hat Developers", nil))
	store.AddQuad(quad.Make(siteURL, "allows protocol", "http", nil))
	store.AddQuad(quad.Make(siteURL, "allows protocol", "https", nil))
	store.AddQuad(quad.Make(siteURL, "scores", 72, nil))
*/

func getStarsFunc(taskName string, page int) xtask.Tsk {
	return func() *xtask.TaskInfo {
		defer timeTrack(time.Now(), taskName)
		defer counterTrack("getStars.task.queued", 1)
		stars, _, err := ghClient.GetFunc("getStars", &github.Options{
			Page:    page,
			PerPage: config.Service.Github.PerPage,
			Runner:  config.Service.Github.Runner,
		})
		if err != nil {
			defer counterTrack("getStars.task.failure", 1)

			log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
		} else {
			defer counterTrack("getStars.task.success", 1)
			for k, _ := range stars {
				r := strings.Split(k, "/")
				// user info
				taskName := fmt.Sprintf("users/%s", r[0])
				log.Println("[ADD] taskName ", taskName)
				// NewTask(exporter, paramTest)
				metaTasker.Add(taskName, nil, getUserFunc(taskName, r[0])).ContinueWithHandler(exportInterface).ContinueWithHandler(func(result xtask.TaskInfo) {
					log.Println("ContinueWithHandler().taskName", taskName)
				})
				factdb.Let(r[0]).Has("repo", r[1])
				// topics
				taskName = fmt.Sprintf("repos/%s/topics", k)
				log.Println("[ADD] taskName ", taskName)
				factdb.Let("dog").Has("name", "hou")
				metaTasker.Add(taskName, nil, getTopicsFunc(taskName, r[0], r[1])).ContinueWithHandler(exportInterface).ContinueWithHandler(func(result xtask.TaskInfo) {
					log.Println("ContinueWithHandler().taskName", taskName)
				})
				// readme
				taskName = fmt.Sprintf("repos/%s/readme", k)
				log.Println("[ADD] taskName ", taskName)
				metaTasker.Add(taskName, nil, getReadmeFunc(taskName, r[0], r[1])).ContinueWithHandler(exportInterface).ContinueWithHandler(func(result xtask.TaskInfo) {
					log.Println("ContinueWithHandler().taskName", taskName)
				})
				// repo info
				taskName = fmt.Sprintf("repos/%s", k)
				log.Println("[ADD] taskName ", taskName)
				metaTasker.Add(taskName, nil, getRepoFunc(taskName, r[0], r[1])).ContinueWithHandler(exportInterface).ContinueWithHandler(func(result xtask.TaskInfo) {
					log.Println("ContinueWithHandler().taskName", taskName)
				})
			}
		}
		return &xtask.TaskInfo{
			Result: &xtask.TaskResult{
				Result: stars,
				Error:  err,
			},
		}
	}
}

func getLatestSHAFunc(taskName, owner, name, branch string) xtask.Tsk {
	return func() *xtask.TaskInfo {
		defer timeTrack(time.Now(), taskName)
		defer counterTrack("getLatestSHA.task.queued", 1)
		sha, _, err := ghClient.GetFunc("getLatestSHA", &github.Options{
			Runner: config.Service.Github.Runner,
			Target: &github.Target{
				Owner:  owner,
				Name:   name,
				Branch: branch,
			},
		})
		if err != nil {
			defer counterTrack("getLatestSHA.task.failure", 1)
			log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
		} else {
			if sha == nil {
				return &xtask.TaskInfo{
					Result: &xtask.TaskResult{
						Result: nil,
						Error:  errNotFoundLatestSHA},
				}
			}
			defer counterTrack("getLatestSHA.task.success", 1)
			shaStr := sha["sha"].(string)
			taskName := strings.Replace(fmt.Sprintf("%s.%s.%s.%s.tree", owner, name, branch, shaStr), "/", ".", -1)
			log.Println("[ADD] taskName ", taskName)
			treeTasker.Add(taskName, nil, getTreeFunc(taskName, owner, name, shaStr)).ContinueWithHandler(exportInterface).ContinueWithHandler(func(result xtask.TaskInfo) {
				log.Println("ContinueWithHandler().taskName", taskName)
			})
		}
		return &xtask.TaskInfo{
			Result: &xtask.TaskResult{
				Result: sha,
				Error:  err,
			},
		}
	}
}

func getUserFunc(taskName, owner string) xtask.Tsk {
	return func() *xtask.TaskInfo {
		defer timeTrack(time.Now(), taskName)
		defer counterTrack("getUser.task.queued", 1)
		user, _, err := ghClient.GetFunc("getUser", &github.Options{
			Runner: config.Service.Github.Runner,
			Target: &github.Target{
				Owner: owner,
			},
		})
		if err != nil {
			defer counterTrack("getUser.task.failure", 1)
			log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
		} else {
			if user == nil {
				return &xtask.TaskInfo{
					Result: &xtask.TaskResult{
						Result: nil,
						Error:  errUserInfo},
				}
			}
			defer counterTrack("getUser.task.success", 1)
		}
		return &xtask.TaskInfo{
			Result: &xtask.TaskResult{
				Result: user,
				Error:  err,
			},
		}
	}
}

func getRepoFunc(taskName, owner, name string) xtask.Tsk {
	return func() *xtask.TaskInfo {
		defer timeTrack(time.Now(), taskName)
		defer counterTrack("getRepo.task.queued", 1)
		repo, _, err := ghClient.GetFunc("getRepo", &github.Options{
			Runner: config.Service.Github.Runner,
			Target: &github.Target{
				Owner: owner,
				Name:  name,
			},
		})
		if err != nil {
			defer counterTrack("getRepo.task.failure", 1)
			log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
		} else {
			if repo == nil {
				return &xtask.TaskInfo{
					Result: &xtask.TaskResult{
						Result: nil,
						Error:  errRepoInfo},
				}
			}
			defer counterTrack("getRepo.task.success", 1)
			branch := *repo["DefaultBranch"].(*string)
			taskName := strings.Replace(fmt.Sprintf("repos.%v.%v.git.refs.heads.%v", owner, name, branch), "/", ".", -1)
			refTasker.Add(taskName, nil, getLatestSHAFunc(taskName, owner, name, branch)).ContinueWithHandler(exportInterface).ContinueWithHandler(func(result xtask.TaskInfo) {
				log.Println("ContinueWithHandler().taskName", taskName)
			})
		}
		return &xtask.TaskInfo{
			Result: &xtask.TaskResult{
				Result: repo,
				Error:  err,
			},
		}
	}
}

func getReadmeFunc(taskName, owner, name string) xtask.Tsk {
	return func() *xtask.TaskInfo {
		defer timeTrack(time.Now(), taskName)
		defer counterTrack("getReadme.task.queued", 1)
		readme, _, err := ghClient.GetFunc("getReadme", &github.Options{
			Runner: config.Service.Github.Runner,
			Target: &github.Target{
				Owner: owner,
				Name:  name,
			},
		})
		if err != nil {
			defer counterTrack("getReadme.task.failure", 1)
			log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
		} else {
			defer counterTrack("getReadme.task.success", 1)
		}
		return &xtask.TaskInfo{
			Result: &xtask.TaskResult{
				Result: readme,
				Error:  err,
			},
		}
	}
}

func getTopicsFunc(taskName, owner, name string) xtask.Tsk {
	return func() *xtask.TaskInfo {
		defer timeTrack(time.Now(), taskName)
		defer counterTrack("getTopics.task.queued", 1)
		topics, _, err := ghClient.GetFunc("getTopics", &github.Options{
			Runner: config.Service.Github.Runner,
			Target: &github.Target{
				Owner: owner,
				Name:  name,
			},
		})
		if err != nil {
			defer counterTrack("getTopics.task.failure", 1)
			log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
		} else {
			defer counterTrack("getTopics.task.success", 1)
		}
		return &xtask.TaskInfo{
			Result: &xtask.TaskResult{
				Result: topics,
				Error:  err,
			},
		}
	}
}

func getTreeFunc(taskName, owner, name, ref string) xtask.Tsk {
	return func() *xtask.TaskInfo {
		defer timeTrack(time.Now(), taskName)
		defer counterTrack("getTree.task.queued", 1)

		tree, _, err := ghClient.GetFunc("getTree", &github.Options{
			Runner: config.Service.Github.Runner,
			Target: &github.Target{
				Owner: owner,
				Name:  name,
				Ref:   ref,
			},
		})
		if err != nil {
			defer counterTrack("getTree.task.failure", 1)
			log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
		} else {
			defer counterTrack("getTree.task.success", 1)
		}
		return &xtask.TaskInfo{
			Result: &xtask.TaskResult{
				Result: tree,
				Error:  err,
			},
		}
	}
}
