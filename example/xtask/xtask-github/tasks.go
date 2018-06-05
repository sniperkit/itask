package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	// "github.com/GianlucaGuarini/go-observable"
	"github.com/cnf/structhash"
	"github.com/sadlil/go-trigger"

	sync "github.com/sniperkit/xutil/plugin/concurrency/sync/debug"
	// llerrgroup "github.com/sniperkit/xutil/plugin/concurrency/llerrgroup"

	"github.com/sniperkit/xtask/pkg"
	"github.com/sniperkit/xtask/plugin/aggregate/service/github"
	"github.com/sniperkit/xtask/plugin/counter"
	"github.com/sniperkit/xtask/util/runtime"
	// "github.com/sniperkit/xutil/plugin/debug/pp"
)

var (
	wg             sync.WaitGroup
	searchTasker   *xtask.Tasker
	listTasker     *xtask.Tasker
	starTasker     *xtask.Tasker
	metaTasker     *xtask.Tasker
	refTasker      *xtask.Tasker
	treeTasker     *xtask.Tasker
	requestDelay   time.Duration = 350 * time.Millisecond
	workerInterval time.Duration = time.Duration(random(150, 250)) * time.Millisecond
	// obs                          = observable.New()
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

func process() {
	runTasks()
	flushWriters()
	newGlobalTaskers()
}

func newGlobalTaskers() {
	searchTasker, _ = xtask.NewTasker(2)
	starTasker, _ = xtask.NewTasker(20)
	listTasker, _ = xtask.NewTasker(20)
	metaTasker, _ = xtask.NewTasker(20)
	refTasker, _ = xtask.NewTasker(20)
	treeTasker, _ = xtask.NewTasker(20)
}

func runTasks() {
	if searchTasker.Count() > 0 {
		searchTasker.Limiter(29, 70*time.Second).Tachymeter().Run()
	}
	if starTasker.Count() > 0 {
		starTasker.Limiter(75, time.Minute).Tachymeter().Run()
	}
	if listTasker.Count() > 0 {
		listTasker.Limiter(20, time.Minute).Tachymeter().Run()
	}
	if metaTasker.Count() > 0 {
		metaTasker.Limiter(20, time.Second).Tachymeter().Run()
	}
	if refTasker.Count() > 0 {
		refTasker.Limiter(20, time.Second).Tachymeter().Run()
	}
	if treeTasker.Count() > 0 {
		treeTasker.Limiter(20, time.Second).Tachymeter().Run()
	}
}

/*
var onExceededRateLimit = func() {
	go func() {
		wg.Add(1)
		defer wg.Done()

		github.Reclaim(svc, resetAt)
	}()
	// ghClient = clientManager.Fetch()
}
*/

//func reclaimClient(resetAt time.Time) *github.Github {
//	github.Reclaim(svc, resetAt)
//}

func changeClient(resetAt time.Time) *github.Github {
	go func() {
		wg.Add(1)
		defer wg.Done()

		github.Reclaim(ghClient, resetAt)
	}()
	ghClient = clientManager.Fetch()
	wg.Wait()
	return ghClient
}

func inspectClient() {
	return
	if ok, resetAt := github.ExceededRateLimit(ghClient.Client); ok {
		_, err := trigger.Fire("onExceededRateLimit", resetAt)
		if err != nil {
			log.Errorln("onExceededRateLimit=", resetAt)
		}
		return // clientManager.Fetch()
	}
	return
}

func lastPage(count int) int {
	return (count / config.Service.Github.PerPage) + 1
}

func getStarsFunc(taskName string, page int) xtask.Tsk {
	return func() *xtask.TaskResult {
		defer timeTrack(time.Now(), taskName)
		defer counterTrack("getStars.task.queued", 1)

		f := github.NewFilter()
		f.Leafs.Paths = true
		f.Maps = map[string]string{
			"repo_remote_id":        "id",
			"repo_name":             "name",
			"repo_created_at":       "created_at.time",
			"repo_updated_at":       "updated_at.time",
			"repo_full_name":        "full_name",
			"repo_default_branch":   "default_branch",
			"repo_lang":             "language",
			"repo_forked":           "fork",
			"repo_forks":            "forks_count",
			"repo_has_pages":        "has_pages",
			"repo_has_wiki":         "has_wiki",
			"repo_has_projects":     "has_projects",
			"repo_stargazers_count": "stargazers_count",
			"repo_watchers_count":   "watchers_count",
			"repo_size":             "size",
			"owner_remote_id":       "owner.id",
			"owner_login":           "owner.login",
		}

		currentPage := 1
		lastPage := 1
		var list []map[string]interface{}
		var merr error
		for currentPage <= lastPage {

			stars, response, err := github.Do(ghClient, "getStars", &github.Options{
				Page:    currentPage,
				PerPage: config.Service.Github.PerPage,
				Runner:  config.Service.Github.Runner,
				Filter:  f,
			})

			if err != nil {
				defer counterTrack("getStars.task.failure", 1)
				log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
				inspectClient()

			} else {
				defer counterTrack("getStars.task.success", 1)
				defer cacheTaskResult(time.Now(), defaultSvc, taskName, stars)

				lastPage = response.LastPage // Set last page only if we didn't get an error
				log.Debugln("currentPage=", currentPage, "lastPage=", lastPage, "Request=", response.Request.URL)
				// addQuad(config.Service.Github.Runner, "starred_last_page", lastPage)

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

					addQuad(config.Service.Github.Runner, "starred", fmt.Sprintf("%s/%s", r[0], r[1]))

					// user info
					taskName := strings.Replace(fmt.Sprintf("%s.users.%s", r[0], defaultSvc), "/", ".", -1)
					taskHash := fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
					if !cuckflt.Lookup([]byte(taskName)) {
						cuckflt.InsertUnique([]byte(taskName))
						metaTasker.Add(taskName, "users", nil, getUserFunc(taskName, r[0]), priority) // .ContinueWithHandler(exportCSV)
						log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					} else {
						log.Debugln("[SKIP] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					}

					// topics
					taskName = strings.Replace(fmt.Sprintf("topics.%s.repos.%s.%s", r[1], r[0], defaultSvc), "/", ".", -1)
					taskHash = fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
					if !cuckflt.Lookup([]byte(taskName)) {
						cuckflt.InsertUnique([]byte(taskName))
						metaTasker.Add(taskName, "topics", nil, getTopicsFunc(taskName, r[0], r[1], rid), priority) //.ContinueWithHandler(exportCSV)
						log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					} else {
						log.Debugln("[SKIP] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					}

					// langs
					taskName = strings.Replace(fmt.Sprintf("langs.%s.repos.%s.%s", r[1], r[0], defaultSvc), "/", ".", -1)
					taskHash = fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
					if !cuckflt.Lookup([]byte(taskName)) {
						cuckflt.InsertUnique([]byte(taskName))
						metaTasker.Add(taskName, "langs", nil, getLanguagesFunc(taskName, r[0], r[1], rid), priority) //.ContinueWithHandler(exportCSV)
						log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					} else {
						log.Debugln("[SKIP] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					}

					// release tags
					taskName = strings.Replace(fmt.Sprintf("release.tags.%s.repos.%s.%s", r[1], r[0], defaultSvc), "/", ".", -1)
					taskHash = fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
					if !cuckflt.Lookup([]byte(taskName)) {
						cuckflt.InsertUnique([]byte(taskName))
						metaTasker.Add(taskName, "release_tags", nil, getReleaseTagsFunc(taskName, r[0], r[1], rid), priority) //.ContinueWithHandler(exportCSV)
						log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					} else {
						log.Debugln("[SKIP] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					}

					// readme
					taskName = strings.Replace(fmt.Sprintf("readme.%s.repos.%s.%s", r[1], r[0], defaultSvc), "/", ".", -1)
					taskHash = fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
					if !cuckflt.Lookup([]byte(taskName)) {
						cuckflt.InsertUnique([]byte(taskName))
						metaTasker.Add(taskName, "readmes", nil, getReadmeFunc(taskName, r[0], r[1], rid), priority) // .ContinueWithHandler(exportCSV)
						log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					} else {
						log.Debugln("[SKIP] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					}

					// repo info
					taskName = strings.Replace(fmt.Sprintf("info.%s.repos.%s.%s", r[1], r[0], defaultSvc), "/", ".", -1)
					taskHash = fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
					if !cuckflt.Lookup([]byte(taskName)) {
						cuckflt.InsertUnique([]byte(taskName))
						metaTasker.Add(taskName, "repos", nil, getRepoFunc(taskName, r[0], r[1], rid), priority) // .ContinueWithHandler(exportCSV)
						log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					} else {
						log.Debugln("[SKIP] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					}
				}
				process()
			}
			currentPage++ // Set the next page
		}

		return &xtask.TaskResult{
			Result: list,
			Error:  merr,
		}
	}
}

func getRepoListFunc(taskName, owner string) xtask.Tsk {
	return func() *xtask.TaskResult {
		f := github.NewFilter()
		f.Leafs.Paths = true
		f.Maps = map[string]string{
			"repo_remote_id":        "id",
			"repo_name":             "name",
			"repo_default_branch":   "default_branch",
			"repo_created_at":       "created_at.time",
			"repo_updated_at":       "updated_at.time",
			"repo_full_name":        "full_name",
			"repo_lang":             "language",
			"repo_forked":           "fork",
			"repo_forks":            "forks_count",
			"repo_has_pages":        "has_pages",
			"repo_has_wiki":         "has_wiki",
			"repo_has_projects":     "has_projects",
			"repo_stargazers_count": "stargazers_count",
			"repo_watchers_count":   "watchers_count",
			"repo_size":             "size",
			"owner_remote_id":       "owner.id",
			"owner_login":           "owner.login",
		}

		currentPage := 1
		lastPage := 1
		var list []map[string]interface{}
		var merr error
		for currentPage <= lastPage {

			repos, response, err := github.Do(ghClient, "getRepoList", &github.Options{
				Page:    currentPage,
				PerPage: config.Service.Github.PerPage,
				Runner:  config.Service.Github.Runner,
				Filter:  f,
				Target: &github.Target{
					Owner: owner,
				},
			})

			if err != nil {
				defer counterTrack("getRepoList.task.failure", 1)
				log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
				merr = err
				inspectClient()

			} else {
				defer counterTrack("getRepoList.task.success", 1)
				defer cacheTaskResult(time.Now(), defaultSvc, taskName, repos)

				lastPage = response.LastPage // Set last page only if we didn't get an error
				log.Debugln("currentPage=", currentPage, "lastPage=", lastPage, "Request=", response.Request.URL)

				for k, repo := range repos {
					r := strings.Split(k, "/")
					rid, err := strconv.Atoi(r[2])
					if err != nil {
						log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
					}
					cds.Append("repos", repo)

					priority, err := strconv.Atoi(r[3])
					if err != nil {
						priority = 0
					}

					addQuad(owner, "repos", fmt.Sprintf("%s/%s", r[0], r[1]))

					// topics
					taskName = strings.Replace(fmt.Sprintf("topics.%s.repos.%s.%s", r[1], r[0], defaultSvc), "/", ".", -1)
					taskHash := fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
					if !cuckflt.Lookup([]byte(taskName)) {
						cuckflt.InsertUnique([]byte(taskName))
						metaTasker.Add(taskName, "topics", nil, getTopicsFunc(taskName, r[0], r[1], rid), priority) //.ContinueWithHandler(exportCSV)
						log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					} else {
						log.Debugln("[SKIP] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					}

					// langs
					taskName = strings.Replace(fmt.Sprintf("langs.%s.repos.%s.%s", r[1], r[0], defaultSvc), "/", ".", -1)
					taskHash = fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
					if !cuckflt.Lookup([]byte(taskName)) {
						cuckflt.InsertUnique([]byte(taskName))
						metaTasker.Add(taskName, "langs", nil, getLanguagesFunc(taskName, r[0], r[1], rid), priority) //.ContinueWithHandler(exportCSV)
						log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					} else {
						log.Debugln("[SKIP] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					}

					// release tags
					taskName = strings.Replace(fmt.Sprintf("release.tags.%s.repos.%s.%s", r[1], r[0], defaultSvc), "/", ".", -1)
					taskHash = fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
					if !cuckflt.Lookup([]byte(taskName)) {
						cuckflt.InsertUnique([]byte(taskName))
						metaTasker.Add(taskName, "release_tags", nil, getReleaseTagsFunc(taskName, r[0], r[1], rid), priority) //.ContinueWithHandler(exportCSV)
						log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					} else {
						log.Debugln("[SKIP] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					}

					// readme
					taskName = strings.Replace(fmt.Sprintf("readme.%s.repos.%s.%s", r[1], r[0], defaultSvc), "/", ".", -1)
					taskHash = fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
					if !cuckflt.Lookup([]byte(taskName)) {
						cuckflt.InsertUnique([]byte(taskName))
						metaTasker.Add(taskName, "readmes", nil, getReadmeFunc(taskName, r[0], r[1], rid), priority) // .ContinueWithHandler(exportCSV)
						log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					} else {
						log.Debugln("[SKIP] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					}

					// repo info
					taskName = strings.Replace(fmt.Sprintf("info.%s.repos.%s.%s", r[1], r[0], defaultSvc), "/", ".", -1)
					taskHash = fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
					if !cuckflt.Lookup([]byte(taskName)) {
						cuckflt.InsertUnique([]byte(taskName))
						metaTasker.Add(taskName, "repos", nil, getRepoFunc(taskName, r[0], r[1], rid), priority) // .ContinueWithHandler(exportCSV)
						log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					} else {
						log.Debugln("[SKIP] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					}
				}
				process()
			}
			currentPage++ // Set the next page
		}

		return &xtask.TaskResult{
			Result: list,
			Error:  merr,
		}
	}
}

func getFollowers(taskName, owner string) xtask.Tsk {
	return func() *xtask.TaskResult {
		f := github.NewFilter()
		f.Leafs.Paths = true
		f.Maps = map[string]string{
			"remote_id":  "id",
			"login":      "login",
			"name":       "name",
			"created_at": "created_at.time",
		}

		currentPage := 1
		lastPage := 1
		var list []map[string]interface{}
		for currentPage <= lastPage {

			users, response, err := github.Do(ghClient, "getFollowers", &github.Options{
				Page:    currentPage,
				PerPage: config.Service.Github.PerPage,
				Runner:  config.Service.Github.Runner,
				Target: &github.Target{
					Owner: owner,
				},
				Filter: f,
			})

			if err != nil {
				defer counterTrack("getFollowersPage.task.failure", 1)
				inspectClient()
				log.Warnln("error: ", err.Error(), "debug=", runtime.WhereAmI())

			} else {
				defer counterTrack("getFollowersPage.task.success", 1)
				defer cacheTaskResult(time.Now(), defaultSvc, taskName, users)

				lastPage = response.LastPage // Set last page only if we didn't get an error
				log.Debugln("currentPage=", currentPage, "lastPage=", lastPage, "Request=", response.Request.URL)

				// defer cacheTaskResult(time.Now(), defaultSvc, taskName, users)
				cds.Append("user_followers", users)

				for k, _ := range users {
					r := strings.Split(k, "/")
					taskName := strings.Replace(fmt.Sprintf("%s.users.%s", r[0], defaultSvc), "/", ".", -1)
					taskHash := fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
					if !cuckflt.Lookup([]byte(taskName)) {
						cuckflt.InsertUnique([]byte(taskName))
						log.Warnln("[ADD] getFollowers().task=", taskName, "taskHash=", taskHash)
						metaTasker.Add(taskName, "users", nil, getUserFunc(taskName, r[0]), 0)
						// addQuad(r[0], "follows", owner)

					} else {
						log.Debugln("[SKIP] source=", config.Service.Github.Runner, "getFollowers().task=", taskName, "taskHash=", taskHash)
					}

					//userGraph := true
					//if userGraph {
					taskName = fmt.Sprintf("%s.user.%s.repos", defaultSvc, r[0])
					taskHash = fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
					if !cuckflt.Lookup([]byte(taskName)) {
						cuckflt.InsertUnique([]byte(taskName))
						listTasker.Add(taskName, "user_repos", nil, getRepoListFunc(taskName, r[0]), 0)
						log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
						//}
					} else {
						log.Debugln("[SKIP] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					}

				}
				process()
			}
			currentPage++ // Set the next page
		}

		return &xtask.TaskResult{
			Result: list,
			Error:  nil,
		}
	}
}

func getFollowing(taskName, owner string) xtask.Tsk {
	return func() *xtask.TaskResult {
		f := github.NewFilter()
		f.Leafs.Paths = true
		f.Maps = map[string]string{
			"remote_id":  "id",
			"login":      "login",
			"name":       "name",
			"created_at": "created_at.time",
		}

		currentPage := 1
		lastPage := 1
		var list []map[string]interface{}
		for currentPage <= lastPage {

			users, response, err := github.Do(ghClient, "getFollowing", &github.Options{
				Page:    currentPage,
				PerPage: config.Service.Github.PerPage,
				Runner:  config.Service.Github.Runner,
				Filter:  f,
				Target: &github.Target{
					Owner: owner,
				},
			})

			if err != nil {
				defer counterTrack("getFollowingPage.task.failure", 1)
				inspectClient()
				log.Warnln("error: ", err.Error(), "debug=", runtime.WhereAmI())
				continue

			} else {
				defer counterTrack("getFollowingPage.task.success", 1)
				defer cacheTaskResult(time.Now(), defaultSvc, taskName, users)
				cds.Append("user_following", users)

				lastPage = response.LastPage // Set last page only if we didn't get an error
				log.Println("currentPage=", currentPage, "lastPage=", lastPage)

				for k, _ := range users {
					r := strings.Split(k, "/")
					taskName := strings.Replace(fmt.Sprintf("%s.users.%s", r[0], defaultSvc), "/", ".", -1)
					taskHash := fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
					if !cuckflt.Lookup([]byte(taskName)) {
						cuckflt.InsertUnique([]byte(taskName))
						log.Warnln("[ADD] getFollowers().task=", taskName, "taskHash=", taskHash)
						metaTasker.Add(taskName, "users", nil, getUserFunc(taskName, r[0]), 0)
						// https://github.com/oren/pokemon#example-for-quads
						// addQuad(owner, "follows", r[0])
					} else {
						log.Debugln("[SKIP] source=", config.Service.Github.Runner, "getFollowers().task=", taskName, "taskHash=", taskHash)
					}

					//userGraph := true
					//if userGraph {
					taskName = fmt.Sprintf("%s.user.%s.repos", defaultSvc, r[0])
					taskHash = fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
					if !cuckflt.Lookup([]byte(taskName)) {
						cuckflt.InsertUnique([]byte(taskName))
						listTasker.Add(taskName, "user_repos", nil, getRepoListFunc(taskName, r[0]), 0)
						log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)

					} else {
						log.Debugln("[SKIP] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					}

				}
				process()
			}
			currentPage++ // Set the next page
		}

		return &xtask.TaskResult{
			Result: list,
			Error:  nil,
		}
	}
}

func getUserFunc(taskName, owner string) xtask.Tsk {
	return func() *xtask.TaskResult {
		defer timeTrack(time.Now(), taskName)
		defer counterTrack("getUser.task.queued", 1)

		f := github.NewFilter()
		f.Leafs.Paths = true
		f.Maps = map[string]string{
			"remote_id":          "id",
			"login":              "login",
			"name":               "name",
			"created_at":         "created_at.time",
			"blog":               "blog",
			"company":            "company",
			"email":              "email",
			"location":           "location",
			"hireable":           "hireable",
			"bio":                "bio",
			"repos_public_count": "public_repos",
			"gist_public_count":  "public_gists",
			"following_count":    "following",
			"followers_count":    "followers",
		}

		user, _, err := github.Do(ghClient, "getUser", &github.Options{
			Runner: config.Service.Github.Runner,
			Target: &github.Target{
				Owner: owner,
			},
			Filter: f,
		})

		if err != nil {
			defer counterTrack("getUser.task.failure", 1)
			inspectClient()
			log.Warnln("error: ", err.Error(), "debug=", runtime.WhereAmI())

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

			/*
				if is_hireable, ok := user["hireable"].(*bool); ok {
					if is_hireable != nil {
						addQuad(owner, "hireable", fmt.Sprintf("%t", *is_hireable))
					}
				}
			*/

			// user nodes
			userNodes := false
			if userNodes {
				taskName = fmt.Sprintf("node.%s.user.%s", owner, defaultSvc)
				taskHash := fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
				if !cuckflt.Lookup([]byte(taskName)) {
					cuckflt.InsertUnique([]byte(taskName))
					metaTasker.Add(taskName, "user_nodes", nil, getUserNode(taskName, owner), 0)
					log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
				} else {
					log.Debugln("[SKIP] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
				}
			}

			// user repositories
			userRepos := true
			if userRepos {
				// user repos list
				taskName = fmt.Sprintf("list.%s.repos.%s", owner, defaultSvc)
				taskHash := fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
				if !cuckflt.Lookup([]byte(taskName)) {
					cuckflt.InsertUnique([]byte(taskName))
					metaTasker.Add(taskName, "repos", nil, getRepoListFunc(taskName, owner), 0)
					log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
				} else {
					log.Debugln("[SKIP] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
				}
			}

		}
		return &xtask.TaskResult{
			Result: user,
			Error:  err,
		}
	}
}

func getUserNode(taskName string, owner string) xtask.Tsk {
	return func() *xtask.TaskResult {
		defer timeTrack(time.Now(), taskName)
		defer counterTrack("getUserNode.task.queued", 1)

		user_nodes, _, err := github.Do(ghClient, "getUserNode", &github.Options{
			Runner: config.Service.Github.Runner,
			Target: &github.Target{
				Owner:   owner,
				Start:   "",
				End:     "",
				Workers: 1,
			},
		})

		if err != nil {
			defer counterTrack("getUserNode.task.failure", 1)
			log.Warnln("error: ", err.Error(), "debug=", runtime.WhereAmI())
			inspectClient()

		} else {
			if user_nodes == nil {
				return &xtask.TaskResult{
					Result: nil,
					Error:  errRepoInfo,
				}
			}

			defer counterTrack("getUserNode.task.success", 1)
			defer cacheTaskResult(time.Now(), defaultSvc, taskName, user_nodes)
			cds.Append("user_nodes", user_nodes)

		}

		return &xtask.TaskResult{
			Result: user_nodes,
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
			"repo_remote_id":        "id",
			"repo_name":             "name",
			"repo_created_at":       "created_at.time",
			"repo_updated_at":       "updated_at.time",
			"repo_full_name":        "full_name",
			"repo_lang":             "language",
			"repo_forked":           "fork",
			"repo_forks":            "forks_count",
			"repo_has_pages":        "has_pages",
			"repo_has_wiki":         "has_wiki",
			"repo_has_projects":     "has_projects",
			"repo_stargazers_count": "stargazers_count",
			"repo_watchers_count":   "watchers_count",
			"repo_default_branch":   "default_branch",
			"repo_size":             "size",
			"owner_remote_id":       "owner.id",
			"owner_login":           "owner.login",
		}

		repo, _, err := github.Do(ghClient, "getRepo", &github.Options{
			Runner: config.Service.Github.Runner,
			Target: &github.Target{
				Owner: owner,
				Name:  name,
			},
			Filter: f,
		})

		if err != nil {
			defer counterTrack("getRepo.task.failure", 1)
			log.Warnln("error: ", err.Error(), "debug=", runtime.WhereAmI())
			inspectClient()

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

			branch := *repo["repo_default_branch"].(*string)
			// addQuad(owner, "repos", branch)
			// addQuad(name, "branch", branch)

			taskName := strings.Replace(fmt.Sprintf("refs.heads.%s.%v.repos.%v.%v", branch, name, owner, defaultSvc), "/", ".", -1)
			taskHash := fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
			if !cuckflt.Lookup([]byte(taskName)) {
				cuckflt.InsertUnique([]byte(taskName))
				refTasker.Add(taskName, "repos_latest_sha", nil, getLatestSHAFunc(taskName, owner, name, branch, rid), 1) // .ContinueWithHandler(exportInterface)
				log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
			} else {
				log.Debugln("[SKIP] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
			}

		}
		return &xtask.TaskResult{
			Result: repo,
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

		sha, _, err := github.Do(ghClient, "getLatestSHA", &github.Options{
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
			log.Warnln("error: ", err.Error(), "debug=", runtime.WhereAmI())
			inspectClient()

		} else {
			if sha != nil {

				if shaStr, ok := sha["sha"].(string); ok {

					defer counterTrack("getLatestSHA.task.success", 1)
					defer cacheTaskResult(time.Now(), defaultSvc, taskName, sha)
					cds.Append("latest_sha", sha)

					// shaStr := sha["sha"].(string)
					taskName := strings.Replace(fmt.Sprintf("%s.%s.%s.%s.%s.tree", branch, name, owner, shaStr, defaultSvc), "/", ".", -1)
					taskHash := fmt.Sprintf("%x", structhash.Sha1(taskName, 1))

					if !cuckflt.Lookup([]byte(taskName)) {
						cuckflt.InsertUnique([]byte(taskName))
						treeTasker.Add(taskName, "trees", nil, getTreeFunc(taskName, owner, name, shaStr, rid), 1) // .ContinueWithHandler(exportCSV)
						log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					} else {
						log.Debugln("[SKIP] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					}

				}

			} else {
				return &xtask.TaskResult{
					Result: nil,
					Error:  errNotFoundLatestSHA,
				}
			}
		}

		return &xtask.TaskResult{
			Result: sha,
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

		readme, _, err := github.Do(ghClient, "getReadme", &github.Options{
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
			log.Warnln("error: ", err.Error(), "debug=", runtime.WhereAmI())
			inspectClient()

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

func getLanguagesFunc(taskName, owner, name string, rid int) xtask.Tsk {
	return func() *xtask.TaskResult {
		defer timeTrack(time.Now(), taskName)
		defer counterTrack("getLanguagesFunc.task.queued", 1)

		langs, _, err := github.Do(ghClient, "getLanguages", &github.Options{
			Runner: config.Service.Github.Runner,
			Target: &github.Target{
				Owner:  owner,
				Name:   name,
				RepoId: rid,
			},
		})

		if err != nil {
			defer counterTrack("getLanguagesFunc.task.failure", 1)
			log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
			inspectClient()

		} else {
			defer counterTrack("getLanguagesFunc.task.success", 1)
			defer cacheTaskResult(time.Now(), defaultSvc, taskName, langs)

			for _, lang := range langs {
				// for label, lang := range langs {
				cds.Append("langs", lang)
				// addQuad(owner, "languages", label)
				// addQuad(name, "languages", label)
			}

		}
		return &xtask.TaskResult{
			Result: langs,
			Error:  err,
		}
	}
}

func getTopicsFunc(taskName, owner, name string, rid int) xtask.Tsk {
	return func() *xtask.TaskResult {
		defer timeTrack(time.Now(), taskName)
		defer counterTrack("getTopics.task.queued", 1)

		topics, _, err := github.Do(ghClient, "getTopics", &github.Options{
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
			inspectClient()

		} else {
			defer counterTrack("getTopics.task.success", 1)
			defer cacheTaskResult(time.Now(), defaultSvc, taskName, topics)

			for _, topic := range topics {
				//for label, topic := range topics {
				cds.Append("topics", topic)
				// addQuad(owner, "topics", label)
				// addQuad(name, "topics", label)
			}

		}
		return &xtask.TaskResult{
			Result: topics,
			Error:  err,
		}
	}
}

func getTrendsFunc(taskName, query string, page int) xtask.Tsk {
	return func() *xtask.TaskResult {
		f := github.NewFilter()
		f.Leafs.Paths = true
		f.Maps = map[string]string{
			"repo_remote_id":        "id",
			"repo_name":             "name",
			"repo_created_at":       "created_at.time",
			"repo_updated_at":       "updated_at.time",
			"repo_full_name":        "full_name",
			"repo_default_branch":   "default_branch",
			"repo_lang":             "language",
			"repo_forked":           "fork",
			"repo_forks":            "forks_count",
			"repo_has_pages":        "has_pages",
			"repo_has_wiki":         "has_wiki",
			"repo_has_projects":     "has_projects",
			"repo_stargazers_count": "stargazers_count",
			"repo_watchers_count":   "watchers_count",
			"repo_size":             "size",
			"owner_remote_id":       "owner.id",
			"owner_login":           "owner.login",
		}

		//currentPage := 1
		//lastPage := 10
		// if config.Service.Github.Search.MaxPage != -1 {
		// 	 lastPage = config.Service.Github.Search.MaxPage
		// }

		var list []map[string]interface{}
		var merr error
		//for currentPage <= lastPage {

		repos, response, err := github.Do(ghClient, "getTrends", &github.Options{
			Page:    page,
			PerPage: config.Service.Github.PerPage,
			Runner:  config.Service.Github.Runner,
			Filter:  f,
			Target: &github.Target{
				Query: query,
			},
		})

		// pp.Println("repos=", repos)

		if err != nil {
			defer counterTrack("getTrends.task.failure", 1)
			log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
			merr = err
			inspectClient()

		} else {
			defer counterTrack("getTrends.task.success", 1)
			defer cacheTaskResult(time.Now(), defaultSvc, taskName, repos)

			// if config.Service.Github.Search.MaxPage == -1 {
			//	 lastPage = response.LastPage // Set last page only if we didn't get an error
			// }

			// pp.Println("response.LastPage=", response.LastPage)
			// pp.Println("repos=", repos)
			// log.Debugln("currentPage=", currentPage, "lastPage=", lastPage, "Request=", response.Request.URL)
			log.Debugln("Request=", response.Request.URL)

			for k, repo := range repos {
				r := strings.Split(k, "/")
				rid, err := strconv.Atoi(r[2])
				if err != nil {
					log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
				}
				cds.Append("repos", repo)

				priority, err := strconv.Atoi(r[3])
				if err != nil {
					priority = 0
				}

				// topics
				taskName = strings.Replace(fmt.Sprintf("topics.%s.repos.%s.%s", r[1], r[0], defaultSvc), "/", ".", -1)
				taskHash := fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
				if !cuckflt.Lookup([]byte(taskName)) {
					cuckflt.InsertUnique([]byte(taskName))
					metaTasker.Add(taskName, "topics", nil, getTopicsFunc(taskName, r[0], r[1], rid), priority) //.ContinueWithHandler(exportCSV)
					log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
				} else {
					log.Debugln("[SKIP] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
				}

				// langs
				taskName = strings.Replace(fmt.Sprintf("langs.%s.repos.%s.%s", r[1], r[0], defaultSvc), "/", ".", -1)
				taskHash = fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
				if !cuckflt.Lookup([]byte(taskName)) {
					cuckflt.InsertUnique([]byte(taskName))
					metaTasker.Add(taskName, "langs", nil, getLanguagesFunc(taskName, r[0], r[1], rid), priority) //.ContinueWithHandler(exportCSV)
					log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
				} else {
					log.Debugln("[SKIP] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
				}

				// release tags
				taskName = strings.Replace(fmt.Sprintf("release.tags.%s.repos.%s.%s", r[1], r[0], defaultSvc), "/", ".", -1)
				taskHash = fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
				if !cuckflt.Lookup([]byte(taskName)) {
					cuckflt.InsertUnique([]byte(taskName))
					metaTasker.Add(taskName, "release_tags", nil, getReleaseTagsFunc(taskName, r[0], r[1], rid), priority) //.ContinueWithHandler(exportCSV)
					log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
				} else {
					log.Debugln("[SKIP] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
				}

				// readme
				taskName = strings.Replace(fmt.Sprintf("readme.%s.repos.%s.%s", r[1], r[0], defaultSvc), "/", ".", -1)
				taskHash = fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
				if !cuckflt.Lookup([]byte(taskName)) {
					cuckflt.InsertUnique([]byte(taskName))
					metaTasker.Add(taskName, "readmes", nil, getReadmeFunc(taskName, r[0], r[1], rid), priority) // .ContinueWithHandler(exportCSV)
					log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
				} else {
					log.Debugln("[SKIP] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
				}

				// repo info
				taskName = strings.Replace(fmt.Sprintf("info.%s.repos.%s.%s", r[1], r[0], defaultSvc), "/", ".", -1)
				taskHash = fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
				if !cuckflt.Lookup([]byte(taskName)) {
					cuckflt.InsertUnique([]byte(taskName))
					metaTasker.Add(taskName, "repos", nil, getRepoFunc(taskName, r[0], r[1], rid), priority) // .ContinueWithHandler(exportCSV)
					log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
				} else {
					log.Debugln("[SKIP] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
				}
				process()
			}

		}
		//	currentPage++ // Set the next page
		//}

		return &xtask.TaskResult{
			Result: list,
			Error:  merr,
		}
	}
}

func getSearchFunc(taskName, query string, page int) xtask.Tsk {
	return func() *xtask.TaskResult {
		f := github.NewFilter()
		f.Leafs.Paths = true
		f.Maps = map[string]string{
			"repo_remote_id":        "id",
			"repo_name":             "name",
			"repo_created_at":       "created_at.time",
			"repo_updated_at":       "updated_at.time",
			"repo_full_name":        "full_name",
			"repo_default_branch":   "default_branch",
			"repo_lang":             "language",
			"repo_forked":           "fork",
			"repo_forks":            "forks_count",
			"repo_has_pages":        "has_pages",
			"repo_has_wiki":         "has_wiki",
			"repo_has_projects":     "has_projects",
			"repo_stargazers_count": "stargazers_count",
			"repo_watchers_count":   "watchers_count",
			"repo_size":             "size",
			"owner_remote_id":       "owner.id",
			"owner_login":           "owner.login",
		}

		currentPage := 1
		lastPage := 10
		// if config.Service.Github.Search.MaxPage != -1 {
		// 	 lastPage = config.Service.Github.Search.MaxPage
		// }

		var list []map[string]interface{}
		var merr error
		for currentPage <= lastPage {

			repos, response, err := github.Do(ghClient, "getTrends", &github.Options{
				Page:    currentPage,
				PerPage: config.Service.Github.PerPage,
				Runner:  config.Service.Github.Runner,
				Filter:  f,
				Target: &github.Target{
					Query: query,
				},
			})

			// pp.Println("repos=", repos)

			if err != nil {
				defer counterTrack("getTrends.task.failure", 1)
				log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
				merr = err
				inspectClient()

			} else {
				defer counterTrack("getTrends.task.success", 1)
				defer cacheTaskResult(time.Now(), defaultSvc, taskName, repos)

				// if config.Service.Github.Search.MaxPage == -1 {
				//	 lastPage = response.LastPage // Set last page only if we didn't get an error
				// }

				// pp.Println("response.LastPage=", response.LastPage)
				// pp.Println("repos=", repos)
				log.Debugln("currentPage=", currentPage, "lastPage=", lastPage, "Request=", response.Request.URL)

				for k, repo := range repos {
					r := strings.Split(k, "/")
					rid, err := strconv.Atoi(r[2])
					if err != nil {
						log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
					}
					cds.Append("repos", repo)

					priority, err := strconv.Atoi(r[3])
					if err != nil {
						priority = 0
					}

					// topics
					taskName = strings.Replace(fmt.Sprintf("topics.%s.repos.%s.%s", r[1], r[0], defaultSvc), "/", ".", -1)
					taskHash := fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
					if !cuckflt.Lookup([]byte(taskName)) {
						cuckflt.InsertUnique([]byte(taskName))
						metaTasker.Add(taskName, "topics", nil, getTopicsFunc(taskName, r[0], r[1], rid), priority) //.ContinueWithHandler(exportCSV)
						log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					} else {
						log.Debugln("[SKIP] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					}

					// langs
					taskName = strings.Replace(fmt.Sprintf("langs.%s.repos.%s.%s", r[1], r[0], defaultSvc), "/", ".", -1)
					taskHash = fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
					if !cuckflt.Lookup([]byte(taskName)) {
						cuckflt.InsertUnique([]byte(taskName))
						metaTasker.Add(taskName, "langs", nil, getLanguagesFunc(taskName, r[0], r[1], rid), priority) //.ContinueWithHandler(exportCSV)
						log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					} else {
						log.Debugln("[SKIP] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					}

					// release tags
					taskName = strings.Replace(fmt.Sprintf("release.tags.%s.repos.%s.%s", r[1], r[0], defaultSvc), "/", ".", -1)
					taskHash = fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
					if !cuckflt.Lookup([]byte(taskName)) {
						cuckflt.InsertUnique([]byte(taskName))
						metaTasker.Add(taskName, "release_tags", nil, getReleaseTagsFunc(taskName, r[0], r[1], rid), priority) //.ContinueWithHandler(exportCSV)
						log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					} else {
						log.Debugln("[SKIP] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					}

					// readme
					taskName = strings.Replace(fmt.Sprintf("readme.%s.repos.%s.%s", r[1], r[0], defaultSvc), "/", ".", -1)
					taskHash = fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
					if !cuckflt.Lookup([]byte(taskName)) {
						cuckflt.InsertUnique([]byte(taskName))
						metaTasker.Add(taskName, "readmes", nil, getReadmeFunc(taskName, r[0], r[1], rid), priority) // .ContinueWithHandler(exportCSV)
						log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					} else {
						log.Debugln("[SKIP] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					}

					// repo info
					taskName = strings.Replace(fmt.Sprintf("info.%s.repos.%s.%s", r[1], r[0], defaultSvc), "/", ".", -1)
					taskHash = fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
					if !cuckflt.Lookup([]byte(taskName)) {
						cuckflt.InsertUnique([]byte(taskName))
						metaTasker.Add(taskName, "repos", nil, getRepoFunc(taskName, r[0], r[1], rid), priority) // .ContinueWithHandler(exportCSV)
						log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					} else {
						log.Debugln("[SKIP] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
					}
				}
				process()
			}
			currentPage++ // Set the next page
		}

		return &xtask.TaskResult{
			Result: list,
			Error:  merr,
		}
	}
}

func getCodeFunc(taskName, query string, page int) xtask.Tsk {
	return func() *xtask.TaskResult {
		f := github.NewFilter()
		f.Leafs.Paths = true

		f.Maps = map[string]string{
			"match_file_name":       "name",
			"match_file_path":       "path",
			"match_file_sha":        "sha",
			"repo_remote_id":        "repository.id",
			"repo_name":             "repository.name",
			"repo_created_at":       "repository.created_at.time",
			"repo_updated_at":       "repository.updated_at.time",
			"repo_full_name":        "repository.full_name",
			"repo_default_branch":   "repository.default_branch",
			"repo_lang":             "repository.language",
			"repo_forked":           "repository.fork",
			"repo_forks":            "repository.forks_count",
			"repo_has_pages":        "repository.has_pages",
			"repo_has_wiki":         "repository.has_wiki",
			"repo_has_projects":     "repository.has_projects",
			"repo_stargazers_count": "repository.stargazers_count",
			"repo_watchers_count":   "repository.watchers_count",
			"repo_size":             "repository.size",
			"owner_remote_id":       "repository.owner.id",
			"owner_login":           "repository.owner.login",
		}

		// currentPage := 1
		// lastPage := 10
		// if config.Service.Github.Search.MaxPage != -1 {
		//	 lastPage = config.Service.Github.Search.MaxPage
		// }

		var list []map[string]interface{}
		var merr error
		// for currentPage <= lastPage {

		repos, response, err := github.Do(ghClient, "getCode", &github.Options{
			Page:    page,
			PerPage: config.Service.Github.PerPage,
			Runner:  config.Service.Github.Runner,
			Filter:  f,
			Target: &github.Target{
				Query: query,
			},
		})

		// pp.Println("query=", query)
		// pp.Println("repos=", repos)

		if err != nil {
			log.Fatalln("err=", err)

			defer counterTrack("getCode.task.failure", 1)
			log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
			merr = err
			inspectClient()

		} else {
			defer counterTrack("getCode.task.success", 1)
			defer cacheTaskResult(time.Now(), defaultSvc, taskName, repos)

			// if config.Service.Github.Search.MaxPage == -1 {
			// 	 lastPage = response.LastPage // Set last page only if we didn't get an error
			// }

			log.Debugln("Request=", response.Request.URL)

			for k, repo := range repos {
				r := strings.Split(k, "/")
				rid, err := strconv.Atoi(r[2])
				if err != nil {
					log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
				}
				cds.Append("repos", repo)

				priority, err := strconv.Atoi(r[3])
				if err != nil {
					priority = 0
				}

				// topics
				taskName = strings.Replace(fmt.Sprintf("topics.%s.repos.%s.%s", r[1], r[0], defaultSvc), "/", ".", -1)
				taskHash := fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
				if !cuckflt.Lookup([]byte(taskName)) {
					cuckflt.InsertUnique([]byte(taskName))
					metaTasker.Add(taskName, "topics", nil, getTopicsFunc(taskName, r[0], r[1], rid), priority) //.ContinueWithHandler(exportCSV)
					log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
				} else {
					log.Debugln("[SKIP] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
				}

				// langs
				taskName = strings.Replace(fmt.Sprintf("langs.%s.repos.%s.%s", r[1], r[0], defaultSvc), "/", ".", -1)
				taskHash = fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
				if !cuckflt.Lookup([]byte(taskName)) {
					cuckflt.InsertUnique([]byte(taskName))
					metaTasker.Add(taskName, "langs", nil, getLanguagesFunc(taskName, r[0], r[1], rid), priority) //.ContinueWithHandler(exportCSV)
					log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
				} else {
					log.Debugln("[SKIP] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
				}

				// release tags
				taskName = strings.Replace(fmt.Sprintf("release.tags.%s.repos.%s.%s", r[1], r[0], defaultSvc), "/", ".", -1)
				taskHash = fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
				if !cuckflt.Lookup([]byte(taskName)) {
					cuckflt.InsertUnique([]byte(taskName))
					metaTasker.Add(taskName, "release_tags", nil, getReleaseTagsFunc(taskName, r[0], r[1], rid), priority) //.ContinueWithHandler(exportCSV)
					log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
				} else {
					log.Debugln("[SKIP] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
				}

				// readme
				taskName = strings.Replace(fmt.Sprintf("readme.%s.repos.%s.%s", r[1], r[0], defaultSvc), "/", ".", -1)
				taskHash = fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
				if !cuckflt.Lookup([]byte(taskName)) {
					cuckflt.InsertUnique([]byte(taskName))
					metaTasker.Add(taskName, "readmes", nil, getReadmeFunc(taskName, r[0], r[1], rid), priority) // .ContinueWithHandler(exportCSV)
					log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
				} else {
					log.Debugln("[SKIP] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
				}

				// repo info
				taskName = strings.Replace(fmt.Sprintf("info.%s.repos.%s.%s", r[1], r[0], defaultSvc), "/", ".", -1)
				taskHash = fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
				if !cuckflt.Lookup([]byte(taskName)) {
					cuckflt.InsertUnique([]byte(taskName))
					metaTasker.Add(taskName, "repos", nil, getRepoFunc(taskName, r[0], r[1], rid), priority) // .ContinueWithHandler(exportCSV)
					log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
				} else {
					log.Debugln("[SKIP] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
				}
			}
			process()
		}

		//currentPage++ // Set the next page
		// }

		return &xtask.TaskResult{
			Result: list,
			Error:  merr,
		}
	}
}

func getReleaseTagsFunc(taskName, owner, name string, rid int) xtask.Tsk {
	return func() *xtask.TaskResult {
		defer timeTrack(time.Now(), taskName)
		defer counterTrack("getReleaseTags.task.queued", 1)

		rts, _, err := github.Do(ghClient, "getReleaseTags", &github.Options{
			Runner: config.Service.Github.Runner,
			Target: &github.Target{
				Owner:  owner,
				Name:   name,
				RepoId: rid,
			},
		})

		if err != nil {
			defer counterTrack("getReleaseTags.task.failure", 1)
			log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
			inspectClient()

		} else {
			defer counterTrack("getReleaseTags.task.success", 1)
			defer cacheTaskResult(time.Now(), defaultSvc, taskName, rts)

			for _, rt := range rts {
				cds.Append("release_tags", rt)
			}

		}
		return &xtask.TaskResult{
			Result: rts,
			Error:  err,
		}
	}
}

func getTreeFunc(taskName, owner, name, ref string, rid int) xtask.Tsk {
	return func() *xtask.TaskResult {
		defer timeTrack(time.Now(), taskName)
		defer counterTrack("getTree.task.queued", 1)

		entries, _, err := github.Do(ghClient, "getTree", &github.Options{
			Runner: config.Service.Github.Runner,
			Target: &github.Target{
				Owner:  owner,
				Name:   name,
				Ref:    ref,
				RepoId: rid,
				// Filters: []string{`ignore`, `docs`, `manifest`},
			},
		})

		if err != nil {
			defer counterTrack("getTree.task.failure", 1)
			log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
			inspectClient()

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
