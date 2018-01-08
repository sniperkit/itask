package main

import (
	"fmt"
	"log"
	"time"

	"github.com/jinzhu/configor"
	"github.com/k0kubun/pp"
	"github.com/segmentio/stats"

	"github.com/sniperkit/xtask/pkg"
	"github.com/sniperkit/xtask/plugin/aggregate/service/github"
	"github.com/sniperkit/xtask/util/runtime"
)

func main() {

	starredTasker, _ = xtask.NewTasker(20)
	preloadTasker, _ = xtask.NewTasker(20)
	extraTasker, _ = xtask.NewTasker(20)
	treeTasker, _ = xtask.NewTasker(20)
	taskerTest()

	t := time.Now()
	configor.Load(&config, "config.yaml")
	if config.App.Debug {
		pp.Println("config: ", config)
	}
	callTime := time.Now().Sub(t)
	m := &funcMetrics{}
	m.calls.count = 1
	m.calls.time = callTime
	stats.Report(m)

	t = time.Now()
	gh = github.New(config.Service.Github.Tokens, &github.Options{
		Page:    1,
		PerPage: config.Service.Github.PerPage,
		Runner:  config.Service.Github.Runner,
	})
	callTime = time.Now().Sub(t)
	m = &funcMetrics{}
	m.calls.count = 1
	m.calls.time = callTime
	stats.Report(m)

	t = time.Now()
	_, resp, err := gh.Get("getStars", &github.Options{ // gh.GetFunc
		Page:     1,
		PerPage:  config.Service.Github.PerPage,
		Runner:   config.Service.Github.Runner,
		Accounts: config.Service.Github.Accounts,
	})
	callTime = time.Now().Sub(t)
	m = &funcMetrics{}
	m.calls.count = 1
	m.calls.time = callTime
	if err != nil {
		m.calls.failed = 1
		log.Fatalln("error: ", err.Error(), "debug=", runtime.WhereAmI())
	}
	stats.Report(m)

	if config.App.Verbose {
		log.Println("LastPage:", resp.LastPage)
	}

	if config.App.Debug {
		pp.Println("starList:", starredList)
		pp.Println("preloadList:", preloadList)
	}

	if config.Service.Github.MaxPage < 0 {
		config.Service.Github.MaxPage = resp.LastPage
	}

	gh.LoadCache(config.Service.Github.MaxPage*config.Service.Github.PerPage*5, gh.PrefixApi(), gh.PrefixApi(), nil) //, []string{"/starred"})

	for i := config.Service.Github.Offset; i <= config.Service.Github.MaxPage; i++ {
		taskName := fmt.Sprintf("activity-starred-%d", i)

		err := starredTasker.Add(taskName, nil, getStarsFunc(i))
		if err != nil {
			log.Fatal(err)
		}

		starredTasker.Limiter(75, time.Minute).Tachymeter().Run()
		preloadTasker.Limiter(17, time.Second).Tachymeter().Run()
		extraTasker.Limiter(17, time.Second).Tachymeter().Run()
		treeTasker.Limiter(17, time.Second).Tachymeter().Run()

		starredTasker, _ = xtask.NewTasker(20)
		preloadTasker, _ = xtask.NewTasker(20)
		extraTasker, _ = xtask.NewTasker(20)
		treeTasker, _ = xtask.NewTasker(20)
	}

	if config.App.Debug {
		pp.Println("starList:", starredList)
		pp.Println("preloadList:", preloadList)
	}

	starredTasker.Limiter(75, time.Minute).Tachymeter().Run()
	preloadTasker.Limiter(17, time.Second).Tachymeter().Run() // 15
	extraTasker.Limiter(17, time.Second).Tachymeter().Run()
	treeTasker.Limiter(17, time.Second).Tachymeter().Run()

}
