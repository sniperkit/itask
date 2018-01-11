package main

import (
	"time"

	"github.com/sniperkit/xtask/plugin/aggregate/service"
	"github.com/sniperkit/xtask/plugin/aggregate/service/github"
)

// github client
var ghClient *github.Github

func githubClient(config *Config) (client *github.Github, err error) {
	t := time.Now()
	ghCfg := config.Service.Github
	client = github.New(ghCfg.Tokens, &github.Options{
		Page:    ghCfg.Offset,
		PerPage: ghCfg.PerPage,
		Runner:  ghCfg.Runner,
	})
	addMetrics(t, 1, false)
	if client == nil {
		err = errGithubClient
	}
	return client, err
}

func visisted(taskName string) bool {
	return false
	// return ghClient.CacheSlugExists(taskName)
}

// service abstraction (github, gitlab, bitbucket, arxiv, ...)
var svc service.Service
