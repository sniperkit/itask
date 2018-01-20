package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/anacrolix/sync"

	"github.com/sniperkit/xtask/pkg"
	"github.com/sniperkit/xtask/plugin/aggregate/service/github"
	"github.com/sniperkit/xtask/plugin/counter"
	"github.com/sniperkit/xtask/util/runtime"
)

var (
	wg             sync.WaitGroup
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

func runTasks() {
	starTasker.Limiter(75, time.Minute).Tachymeter().Run()
	metaTasker.Limiter(28, time.Second).Tachymeter().Run()
	refTasker.Limiter(28, time.Second).Tachymeter().Run()
	treeTasker.Limiter(28, time.Second).Tachymeter().Run()
}

func convertInterface2(input []map[string]interface{}) []interface{} {
	results := make([]interface{}, len(input))
	for _, result := range input {
		// resultSlice := result.(map[string]interface{})
		// pp.Println("resultSlice=", resultSlice)
		results = append(results, result)
	}
	return results
}

func convertInterface(input map[string]interface{}) []interface{} {
	results := make([]interface{}, len(input))
	for _, result := range input {
		resultSlice := result.(interface{})
		// pp.Println("resultSlice=", resultSlice)
		results = append(results, resultSlice)
	}
	return results
}

func getHeaders(filterMap map[string]string) []string {
	var hdrs []string
	for k, _ := range filterMap {
		hdrs = append(hdrs, k)
	}
	return hdrs
}

func getStarsFunc(taskName string, page int) xtask.Tsk {
	return func() *xtask.TaskResult {
		defer timeTrack(time.Now(), taskName)
		defer counterTrack("getStars.task.queued", 1)

		f := github.NewFilter()
		f.Leafs.Paths = true
		f.Maps = map[string]string{
			"repo_remote_id":  "id",
			"repo_name":       "name",
			"repo_full_name":  "full_name",
			"user_remote_id":  "owner.id",
			"user_name":       "owner.login",
			"user_avatar_url": "owner.avatar_url",
		}

		stars, _, err := ghClient.GetFunc("getStars", &github.Options{
			Page:    page,
			PerPage: config.Service.Github.PerPage,
			Runner:  config.Service.Github.Runner,
			Filter:  f,
		})

		if err != nil {
			defer counterTrack("getStars.task.failure", 1)
			log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())

		} else {
			defer counterTrack("getStars.task.success", 1)
			defer cacheTaskResult(time.Now(), defaultSvc, taskName, stars)

			for k, star := range stars {
				r := strings.Split(k, "/")
				rid, err := strconv.Atoi(r[2])
				if err != nil {
					log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
				}
				cds.Append("stars", star)

				priority, err := strconv.Atoi(r[3])
				if err != nil {
					priority = 0
				}

				// user info
				taskName := strings.Replace(fmt.Sprintf("%s.users.%s", defaultSvc, r[0]), "/", ".", -1)
				if !cuckflt.Lookup([]byte(taskName)) {
					metaTasker.Add(taskName, "users", nil, getUserFunc(taskName, r[0], rid), priority) // .ContinueWithHandler(exportCSV)
					factdb.Let(r[0]).Has("repo", r[1])
					log.Debugln("[ADD] task=", taskName)
				} else {
					log.Debugln("[SKIP] task=", taskName)
				}

				// topics
				taskName = strings.Replace(fmt.Sprintf("%s.repos.%s.topics", defaultSvc, k), "/", ".", -1)
				if !cuckflt.Lookup([]byte(taskName)) {
					factdb.Let("dog").Has("name", "hou")
					metaTasker.Add(taskName, "topics", nil, getTopicsFunc(taskName, r[0], r[1], rid), priority) //.ContinueWithHandler(exportCSV)
					log.Debugln("[ADD] task=", taskName)
				} else {
					log.Debugln("[SKIP] task=", taskName)
				}

				// readme
				taskName = strings.Replace(fmt.Sprintf("%s.repos.%s.readme", defaultSvc, k), "/", ".", -1)
				if !cuckflt.Lookup([]byte(taskName)) {
					metaTasker.Add(taskName, "readmes", nil, getReadmeFunc(taskName, r[0], r[1], rid), priority) // .ContinueWithHandler(exportCSV)
					log.Debugln("[ADD] task=", taskName)
				} else {
					log.Debugln("[SKIP] task=", taskName)
				}

				// repo info
				taskName = strings.Replace(fmt.Sprintf("%s.repos.%s", defaultSvc, k), "/", ".", -1)
				if !cuckflt.Lookup([]byte(taskName)) {
					metaTasker.Add(taskName, "repos", nil, getRepoFunc(taskName, r[0], r[1], rid), priority) // .ContinueWithHandler(exportCSV)
					log.Debugln("[ADD] task=", taskName)
				} else {
					log.Debugln("[SKIP] task=", taskName)
				}

			}

		}

		return &xtask.TaskResult{
			Result: stars,
			Error:  err,
		}
	}
}

func getLatestSHAFunc(taskName, owner, name, branch string, rid int) xtask.Tsk {
	return func() *xtask.TaskResult {
		defer timeTrack(time.Now(), taskName)
		defer counterTrack("getLatestSHA.task.queued", 1)

		f := github.NewFilter()
		f.Leafs.Paths = true
		f.Maps = map[string]string{
			"repo_sha": "sha",
		}

		sha, _, err := ghClient.GetFunc("getLatestSHA", &github.Options{
			Runner: config.Service.Github.Runner,
			Target: &github.Target{
				Owner:  owner,
				Name:   name,
				Branch: branch,
				RepoId: rid,
			},
			Filter: f,
		})

		if err != nil {
			defer counterTrack("getLatestSHA.task.failure", 1)
			log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())

		} else {
			if sha == nil {
				return &xtask.TaskResult{
					Result: nil,
					Error:  errNotFoundLatestSHA,
				}
			}
			defer counterTrack("getLatestSHA.task.success", 1)
			defer cacheTaskResult(time.Now(), defaultSvc, taskName, sha)

			cds.Append("latest_sha", sha)

			shaStr := sha["sha"].(string)
			taskName := strings.Replace(fmt.Sprintf("%s.%s.%s.%s.%s.tree", defaultSvc, owner, name, branch, shaStr), "/", ".", -1)
			// taskName := fmt.Sprintf("%s/%s/%s/%s/%s/tree", defaultSvc, owner, name, branch, shaStr)
			// log.Println("[ADD] taskName ", taskName)
			if !cuckflt.Lookup([]byte(taskName)) {
				treeTasker.Add(taskName, "trees", nil, getTreeFunc(taskName, owner, name, shaStr, rid), 1) // .ContinueWithHandler(exportCSV)
				log.Debugln("[ADD] task=", taskName)
			} else {
				log.Debugln("[SKIP] task=", taskName)
			}

		}

		return &xtask.TaskResult{
			Result: sha,
			Error:  err,
		}
	}
}

func getUserFunc(taskName, owner string, rid int) xtask.Tsk {
	return func() *xtask.TaskResult {
		defer timeTrack(time.Now(), taskName)
		defer counterTrack("getUser.task.queued", 1)

		f := github.NewFilter()
		f.Leafs.Paths = true

		f.Maps = map[string]string{
			"remote_id":  "id",
			"login":      "login",
			"name":       "name",
			"created_at": "created_at.time",
			"blog":       "blog",
			"company":    "company",
			"email":      "email",
			"location":   "location",
			"bio":        "bio",
			"following":  "following",
			"followers":  "followers",
		}

		user, _, err := ghClient.GetFunc("getUser", &github.Options{
			Runner: config.Service.Github.Runner,
			Target: &github.Target{
				Owner: owner,
			},
			Filter: f,
		})

		if err != nil {
			defer counterTrack("getUser.task.failure", 1)
			log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
		} else {
			if user == nil {
				return &xtask.TaskResult{
					Result: nil,
					Error:  errUserInfo,
				}
			}
			defer counterTrack("getUser.task.success", 1)
			defer cacheTaskResult(time.Now(), defaultSvc, taskName, user)

			cds.Append("users", user)
		}
		return &xtask.TaskResult{
			Result: user,
			Error:  err,
		}
	}
}

func getRepoFunc(taskName, owner, name string, rid int) xtask.Tsk {
	return func() *xtask.TaskResult {
		defer timeTrack(time.Now(), taskName)
		defer counterTrack("getRepo.task.queued", 1)

		f := github.NewFilter()
		f.Leafs.Paths = true
		f.Maps = map[string]string{
			"remote_rid":     "id",
			"full_name":      "full_name",
			"name":           "name",
			"user":           "owner.login",
			"remote_uid":     "owner.id",
			"created_at":     "created_at",
			"default_branch": "default_branch",
		}

		repo, _, err := ghClient.GetFunc("getRepo", &github.Options{
			Runner: config.Service.Github.Runner,
			Target: &github.Target{
				Owner: owner,
				Name:  name,
			},
			Filter: f,
		})

		if err != nil {
			defer counterTrack("getRepo.task.failure", 1)
			log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())

		} else {
			if repo == nil {
				return &xtask.TaskResult{
					Result: nil,
					Error:  errRepoInfo,
				}
			}
			defer counterTrack("getRepo.task.success", 1)
			defer cacheTaskResult(time.Now(), defaultSvc, taskName, repo)
			cds.Append("repos", repo)

			branch := *repo["default_branch"].(*string)
			taskName := strings.Replace(fmt.Sprintf("%s.repos.%v.%v.git.refs.heads.%v", defaultSvc, owner, name, branch), "/", ".", -1)
			// taskName := fmt.Sprintf("%s/repos/%v/%v/git/refs/heads/%v", defaultSvc, owner, name, branch)
			if !cuckflt.Lookup([]byte(taskName)) {
				refTasker.Add(taskName, "repos_latest_sha", nil, getLatestSHAFunc(taskName, owner, name, branch, rid), 1) // .ContinueWithHandler(exportInterface)
				log.Debugln("[ADD] task=", taskName)
			} else {
				log.Debugln("[SKIP] task=", taskName)
			}

		}
		return &xtask.TaskResult{
			Result: repo,
			Error:  err,
		}
	}
}

func getReadmeFunc(taskName, owner, name string, rid int) xtask.Tsk {
	return func() *xtask.TaskResult {
		defer timeTrack(time.Now(), taskName)
		defer counterTrack("getReadme.task.queued", 1)

		f := github.NewFilter()
		f.Leafs.Paths = true
		f.Maps = map[string]string{
			"readme": "content",
			"size":   "size",
			"sha":    "sha",
			"path":   "path",
			"name":   "name",
		}

		readme, _, err := ghClient.GetFunc("getReadme", &github.Options{
			Runner: config.Service.Github.Runner,
			Target: &github.Target{
				Owner:  owner,
				Name:   name,
				RepoId: rid,
			},
			Filter: f,
		})

		if err != nil {
			defer counterTrack("getReadme.task.failure", 1)
			log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
		} else {
			defer counterTrack("getReadme.task.success", 1)
			defer cacheTaskResult(time.Now(), defaultSvc, taskName, readme)

			cds.Append("readmes", readme)
		}
		return &xtask.TaskResult{
			Result: &readme,
			Error:  err,
		}
	}
}

func getTopicsFunc(taskName, owner, name string, rid int) xtask.Tsk {
	return func() *xtask.TaskResult {
		defer timeTrack(time.Now(), taskName)
		defer counterTrack("getTopics.task.queued", 1)

		topics, _, err := ghClient.GetFunc("getTopics", &github.Options{
			Runner: config.Service.Github.Runner,
			Target: &github.Target{
				Owner:  owner,
				Name:   name,
				RepoId: rid,
			},
		})

		if err != nil {
			defer counterTrack("getTopics.task.failure", 1)
			log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
		} else {
			defer counterTrack("getTopics.task.success", 1)
			defer cacheTaskResult(time.Now(), defaultSvc, taskName, topics)

			for _, topic := range topics {
				cds.Append("topics", topic)
			}

		}
		return &xtask.TaskResult{
			Result: topics,
			Error:  err,
		}
	}
}

func getTreeFunc(taskName, owner, name, ref string, rid int) xtask.Tsk {
	return func() *xtask.TaskResult {
		defer timeTrack(time.Now(), taskName)
		defer counterTrack("getTree.task.queued", 1)

		entries, _, err := ghClient.GetFunc("getTree", &github.Options{
			Runner: config.Service.Github.Runner,
			Target: &github.Target{
				Owner:  owner,
				Name:   name,
				Ref:    ref,
				RepoId: rid,
			},
		})

		if err != nil {
			defer counterTrack("getTree.task.failure", 1)
			log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
		} else {
			defer counterTrack("getTree.task.success", 1)
			defer cacheTaskResult(time.Now(), defaultSvc, taskName, entries)

			for _, entry := range entries {
				cds.Append("files", entry)
			}

		}
		return &xtask.TaskResult{
			Result: entries,
			Error:  err,
		}
	}
}
