package main

import (
	"context"
	"time"

	// "github.com/sniperkit/xutil/plugin/debug/pp"

	"github.com/sniperkit/xtask/plugin/aggregate/service"
	"github.com/sniperkit/xtask/plugin/aggregate/service/github"
)

var (
	svc           service.Service
	ghClient      *github.Github
	ghClients     []*github.Github
	clientManager *github.ClientManager
	ctx           context.Context
)

var (
	origin   string
	target   string
	seen     = make(map[string]bool, 0)
	done     = make(chan *github.UserNode)
	jobQueue = make(chan jobRequest, 1000)
)

func githubClients(config *Config) (cm *github.ClientManager) {
	defer timeTrack(time.Now(), "github.client.manager")
	defer counterTrack("github.client.manager", 1)

	var tokens []string
	ghCfg := config.Service.Github
	for _, token := range ghCfg.Tokens {
		if token.Key != "" {
			tokens = append(tokens, token.Key)
		}
	}
	xcache = getCache()
	cm = github.NewManager(tokens, &github.Options{
		Page:    ghCfg.Offset,
		PerPage: ghCfg.PerPage,
		Runner:  ghCfg.Runner,
	}, &xcache)

	return
}

func githubClient(config *Config) (client *github.Github, err error) {
	defer timeTrack(time.Now(), "githubClient")
	defer counterTrack("githubClient", 1)

	ghCfg := config.Service.Github
	client = github.New(ghCfg.Tokens, &github.Options{
		Page:    ghCfg.Offset,
		PerPage: ghCfg.PerPage,
		Runner:  ghCfg.Runner,
	})
	if client == nil {
		err = errGithubClient
	}
	return client, err
}

func visisted(taskName string) bool {
	return false
}
