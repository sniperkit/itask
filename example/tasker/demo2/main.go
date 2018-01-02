package main

import (
	"log"
	"strings"
	"time"

	// "github.com/k0kubun/pp"
	"github.com/sniperkit/xtask/pkg"
	"github.com/sniperkit/xtask/test/service/github"
)

var (
	token      string
	starsList  *xtask.TaskList = xtask.NewTaskList(15)
	readmeList *xtask.TaskList = xtask.NewTaskList(15)
	repoList   *xtask.TaskList = xtask.NewTaskList(15)
	maxPage    int             = 10
	counter    map[string]int  = make(map[string]int)
)

func main() {

	token = "3a2cd56ac1985bb5da567a14723fb104dc7be97f"
	opts := &github.Options{
		Page:    1,
		PerPage: 20,
		Runner:  "roscopecoltran",
	}

	xtask.Configure(15, 0)

	gh := github.New(&token, opts)

	_, resp, err := gh.Get("getStars", opts)
	if err != nil {
		log.Fatalln("error: ", err.Error())
	}
	log.Println("LastPage:", resp.LastPage)

	getReadme := func(owner, name string) {
		opts := &github.Options{Runner: "roscopecoltran"}
		opts.Target = &github.Target{Owner: owner, Name: name}
		_, _, err := gh.Get("getReadme", opts)
		if err != nil {
			log.Println("error: ", err.Error())
		}
		counter["readme"]++
		log.Println("readme count: ", counter["readme"])
	}

	_ = func(owner, name string) {
		opts := &github.Options{Runner: "roscopecoltran"}
		opts.Target = &github.Target{Owner: owner, Name: name}
		_, _, err := gh.Get("getRepo", opts)
		if err != nil {
			log.Println("error: ", err.Error())
		}
		counter["repo"]++
		log.Println("repo count: ", counter["repo"])
	}

	getStars := func(page int) {
		opts := &github.Options{
			Page:    page,
			PerPage: 20,
			Runner:  "roscopecoltran",
		}
		stars, _, err := gh.Get("getStars", opts)
		if err != nil {
			panic(err)
		}
		for k, _ := range stars {
			r := strings.Split(k, "/")
			counter["starred"]++
			log.Println("starred count: ", counter["starred"])

			// repoList.Add(xtask.Enqueue(getRepo, r[0], r[1]))
			starsList.Add(xtask.Enqueue(getReadme, r[0], r[1]))
			// repoList.Add(xtask.NewTask(getRepo, r[0], r[1]))
			// readmeList.Add(xtask.NewTask(getReadme, r[0], r[1]))
		}
		log.Println("starred count: ", counter["starred"])
	}

	for i := 1; i <= maxPage; i++ {
		//starsList.Add(xtask.Enqueue(getStars, i)) // .Delay(1 * time.Second))
		starsList.Add(xtask.NewTask(getStars, i).Delay(1 * time.Second))
	}

	// starsList.Run().WaitAll()
	// repoList.Run().WaitAll()
	// readmeList.Run().WaitAll()

	xtask.RunService()

}
