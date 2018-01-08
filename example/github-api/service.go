package main

import (
	"time"

	"github.com/sniperkit/xtask/plugin/aggregate/service"
	"github.com/sniperkit/xtask/plugin/aggregate/service/github"
)

// github client
var ghClient *github.Github

func githubClient() *github.Github {
	t := time.Now()
	client := github.New(config.Service.Github.Tokens, &github.Options{
		Page:    config.Service.Github.Offset,
		PerPage: config.Service.Github.PerPage,
		Runner:  config.Service.Github.Runner,
	})
	addMetrics(t, 1, false)
	return client
}

// service abstraction (github, gitlab, bitbucket, arxiv, ...)
var svc service.Service
