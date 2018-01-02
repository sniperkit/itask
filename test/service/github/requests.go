package github

import (
	"context"
	"log"

	"github.com/fatih/structs"
	"github.com/google/go-github/github"
	"github.com/sniperkit/flatten"
	// "github.com/sniperkit/async/utils"
)

func (g *Github) Get(entity string, opts *Options) (map[string]interface{}, *github.Response, error) {
	switch entity {
	case "getStars":
		return g.getStars(opts)
	case "getRepo":
		return g.getRepo(opts)
	case "getReadme":
		return g.getReadme(opts)
	}
	return nil, nil, nil
}

func (g *Github) getStars(opts *Options) (map[string]interface{}, *github.Response, error) {
	if g.client == nil {
		g.client = getClient(g.ctoken)
	}

	g.rateLimiter(opts.Runner).Wait()
	stars, response, err := g.client.Activity.ListStarred(
		context.Background(),
		opts.Runner,
		&github.ActivityListStarredOptions{
			Sort:      "updated",
			Direction: "desc", // desc
			ListOptions: github.ListOptions{
				Page:    opts.Page,
				PerPage: opts.PerPage,
			},
		},
	)

	res := make(map[string]interface{})
	for _, star := range stars {
		key := star.Repository.GetFullName()
		res[key] = star.GetStarredAt()
	}

	return res, response, err
}

func (g *Github) getRepo(opts *Options) (map[string]interface{}, *github.Response, error) {
	if g.client == nil {
		g.client = getClient(g.ctoken)
	}

	g.rateLimiter(opts.Runner).Wait()
	repo, response, err := g.client.Repositories.Get(context.Background(), opts.Target.Owner, opts.Target.Name)
	if err != nil {
		return nil, nil, err
	}

	var fm map[string]interface{}
	m := structs.Map(repo)
	// pp.Println("m: ", m)
	fm, err = flatten.Flatten(m, "", flatten.DotStyle)
	// fm, err = flatmap.Flatten(structs.Map(repo))
	if err != nil {
		log.Println("getRepo().error:", err.Error())
		return nil, response, err
	}
	// pp.Println("output.flatten: ", fm)
	return fm, response, err
}

func (g *Github) getReadme(opts *Options) (map[string]interface{}, *github.Response, error) {
	if g.client == nil {
		g.client = getClient(g.ctoken)
	}

	g.rateLimiter(opts.Runner).Wait()
	readme, response, err := g.client.Repositories.GetReadme(context.Background(), opts.Target.Owner, opts.Target.Name, nil)
	if err != nil {
		return nil, nil, err
	}
	content, _ := readme.GetContent()
	readme.Content = &content

	var fm map[string]interface{}
	m := structs.Map(readme)
	// pp.Println("m: ", m)
	fm, err = flatten.Flatten(m, "", flatten.DotStyle)
	if err != nil {
		log.Println("getReadme().error:", err.Error())
		return nil, response, err
	}
	// pp.Println("output.flatten: ", fm)
	return fm, response, err
}
