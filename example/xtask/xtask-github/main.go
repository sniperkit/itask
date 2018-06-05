package main

import (
	"fmt"

	"github.com/cnf/structhash"

	"github.com/sniperkit/xtask/plugin/aggregate/service/github"
	"github.com/sniperkit/xutil/plugin/debug/pp"
)

func main() {
	loadConfig()
	// initFact()
	initWriters(false, writersList...)
	skipTasksWithTTL("./shared/data/export/csv/tasks.csv")
	newGlobalTaskers()

	clientManager = githubClients(&config)

	defer clientManager.Shutdown()
	github.GlobalFilters(wordFiltersMap)

	ghClient = clientManager.Fetch()

	// userData(config.Service.Github.Runner)

	accounts := shuffleStrings(config.Service.Github.Accounts)
	pp.Println("accounts=", accounts)

	for _, account := range accounts {
		config.Service.Github.Runner = account
		extractData()
	}

}

func extractData() {

	var taskName, taskHash string

	taskName = fmt.Sprintf("%s.activity.starred", defaultSvc)
	taskHash = fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
	starTasker.Add(taskName, "starred", nil, getStarsFunc(taskName, defaultOffsetStarred), 0) // .ContinueWithHandler(exportInterface) // .ContinueWithHandler(func(result *xtask.TaskInfo) { fmt.Println("ContinueWithHandler().taskName", taskName) })
	process()

	taskName = fmt.Sprintf("%s.user.%s.repos", defaultSvc, config.Service.Github.Runner)
	taskHash = fmt.Sprintf("%x", structhash.Sha1(taskName, 1))

	cuckflt.InsertUnique([]byte(taskName))
	listTasker.Add(taskName, "user_repos", nil, getRepoListFunc(taskName, config.Service.Github.Runner), 0)
	log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
	process()

	taskName = fmt.Sprintf("%s.user.%s.following", defaultSvc, config.Service.Github.Runner)
	taskHash = fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
	cuckflt.InsertUnique([]byte(taskName))
	listTasker.Add(taskName, "user_following", nil, getFollowing(taskName, config.Service.Github.Runner), 0)
	log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
	// process()

	taskName = fmt.Sprintf("%s.user.%s.followers", defaultSvc, config.Service.Github.Runner)
	taskHash = fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
	cuckflt.InsertUnique([]byte(taskName))
	listTasker.Add(taskName, "user_followers", nil, getFollowers(taskName, config.Service.Github.Runner), 0)
	log.Debugln("[ADD] source=", config.Service.Github.Runner, "task=", taskName, "taskHash=", taskHash)
	process()

}

func searchProcess() {

	var taskName, taskHash string
	searchGraph := false

	if searchGraph {
		for _, q := range config.Service.Github.Search.Code {
			log.Infoln("query=", q)
			for i := 1; i <= 10; i++ {
				taskName = fmt.Sprintf("%s.search.codes.%s.%d", defaultSvc, q, i)
				taskHash = fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
				if !cuckflt.Lookup([]byte(taskName)) {
					cuckflt.InsertUnique([]byte(taskName))
					searchTasker.Add(taskHash, "search_codes", nil, getCodeFunc(taskName, q, i), 0)
				} else {
					log.Debugln("[SKIP] task=", taskName)
				}
			}
		}
		// process()

		for _, q := range config.Service.Github.Search.Repo {
			log.Infoln("query=", q)
			for i := 1; i <= 10; i++ {
				taskName = fmt.Sprintf("%s.search.trends.%s.%d", defaultSvc, q, i)
				taskHash = fmt.Sprintf("%x", structhash.Sha1(taskName, 1))
				if !cuckflt.Lookup([]byte(taskName)) {
					cuckflt.InsertUnique([]byte(taskName))
					searchTasker.Add(taskHash, "search_trends", nil, getTrendsFunc(taskName, q, i), 0)
				} else {
					log.Debugln("[SKIP] task=", taskName)
				}
			}
		}
		// searchTasker.Limiter(30, 70*time.Second).Tachymeter().Run()
		process()

	}

}
