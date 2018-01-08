package github

import (
	"context"
	"log"

	"github.com/fatih/structs"
	"github.com/google/go-github/github"

	"github.com/sniperkit/flatten"
	"github.com/sniperkit/xtask/util/runtime"
)

func (g *Github) Get(entity string, opts *Options) (map[string]interface{}, *github.Response, error) {
	if g.client == nil {
		g.client = getClient(g.ctoken)
	}

	g.counters.Increment("github.Get()", 1)
	switch entity {
	case "getStars":
		return g.getStars(opts)
	case "getRepo":
		return g.getRepo(opts)
	case "getReadme":
		return g.getReadme(opts)
	case "getTree":
		return g.getTree(opts)
	case "getTopics":
		return g.getTopics(opts)
	case "getLatestSHA":
		return g.getLatestSHA(opts)
	}
	return nil, nil, nil
}

func (g *Github) getStars(opts *Options) (map[string]interface{}, *github.Response, error) {
	if g.client == nil {
		g.client = getClient(g.ctoken)
	}
	// g.mu.RLock()
	// defer g.mu.RUnlock()

	var stars []*github.StarredRepository
	var response *github.Response
	get := func() error {
		var err error
		stars, response, err = g.client.Activity.ListStarred(
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

		if status := g.limitHandler(response.StatusCode, response.Rate, response.Header, err); status != nil {
			log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
			return err
		}
		return nil
	}

	if err := g.retryRegistrationFunc(get); err != nil {
		log.Println("error: ", err, "debug=", runtime.WhereAmI())
		return nil, response, err
	}

	res := make(map[string]interface{})
	for _, star := range stars {
		key := star.Repository.GetFullName()
		res[key] = star.GetStarredAt()
	}

	return res, response, nil
}

func (g *Github) getRepo(opts *Options) (map[string]interface{}, *github.Response, error) {
	if g.client == nil {
		g.client = getClient(g.ctoken)
	}
	g.mu.RLock()
	defer g.mu.RUnlock()

	var repo *github.Repository
	var response *github.Response
	get := func() error {
		var err error
		repo, response, err = g.client.Repositories.Get(context.Background(), opts.Target.Owner, opts.Target.Name)
		if status := g.limitHandler(response.StatusCode, response.Rate, response.Header, err); status != nil {
			log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
			return err
		}
		return nil
	}

	if err := g.retryRegistrationFunc(get); err != nil {
		log.Println("error: ", err, "debug=", runtime.WhereAmI())
		return nil, response, err
	}

	// log.Println("repo=", repo)
	if repo == nil {
		return nil, response, errorMarshallingResponse
	}

	if !structs.IsStruct(repo) {
		return nil, response, errorMarshallingResponse
	}

	var fm map[string]interface{}
	var err error
	m := structs.Map(repo)
	fm, err = flatten.Flatten(m, "", flatten.DotStyle)
	if err != nil {
		log.Fatalln("error: ", err.Error(), "debug=", runtime.WhereAmI())
		return nil, response, err
	}
	return fm, response, nil
}

func (g *Github) getTopics(opts *Options) (map[string]interface{}, *github.Response, error) {
	if g.client == nil {
		g.client = getClient(g.ctoken)
	}
	g.mu.RLock()
	defer g.mu.RUnlock()

	var topics []string
	var response *github.Response
	get := func() error {
		var err error

		// g.rateLimiter().Wait()
		topics, response, err = g.client.Repositories.ListAllTopics(context.Background(), opts.Target.Owner, opts.Target.Name)
		if status := g.limitHandler(response.StatusCode, response.Rate, response.Header, err); status != nil {
			log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
			return err
		}
		return nil
	}

	if err := g.retryRegistrationFunc(get); err != nil {
		log.Println("error: ", err, "debug=", runtime.WhereAmI())
		return nil, response, err
	}

	if topics == nil {
		return nil, response, errorMarshallingResponse
	}

	fm := make(map[string]interface{}, 1)
	fm["topics"] = topics

	return fm, response, nil
}

func (g *Github) getLatestSHA(opts *Options) (map[string]interface{}, *github.Response, error) {
	if g.client == nil {
		g.client = getClient(g.ctoken)
	}
	g.mu.RLock()
	defer g.mu.RUnlock()

	var ref *github.Reference
	var response *github.Response
	get := func() error {
		var err error
		if opts.Target.Branch == "" {
			opts.Target.Branch = "master"
		}

		// g.rateLimiter().Wait()
		ref, response, err = g.client.Git.GetRef(context.Background(), opts.Target.Owner, opts.Target.Name, "refs/heads/"+opts.Target.Branch)
		if status := g.limitHandler(response.StatusCode, response.Rate, response.Header, err); status != nil {
			log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
			return err
		}
		return nil
	}

	if err := g.retryRegistrationFunc(get); err != nil {
		log.Println("error: ", err, "debug=", runtime.WhereAmI())
		return nil, response, err
	}

	if ref == nil {
		return nil, response, errorMarshallingResponse
	}

	fm := make(map[string]interface{}, 1)
	fm["sha"] = *ref.Object.SHA

	return fm, response, nil
}

func (g *Github) getTree(opts *Options) (map[string]interface{}, *github.Response, error) {
	if g.client == nil {
		g.client = getClient(g.ctoken)
	}
	g.mu.RLock()
	defer g.mu.RUnlock()

	var tree *github.Tree
	var response *github.Response
	get := func() error {
		var err error

		// g.rateLimiter().Wait()
		tree, response, err = g.client.Git.GetTree(context.Background(), opts.Target.Owner, opts.Target.Name, opts.Target.Ref, true)
		if status := g.limitHandler(response.StatusCode, response.Rate, response.Header, err); status != nil {
			return err
		}
		return nil
	}

	if err := g.retryRegistrationFunc(get); err != nil {
		log.Println("error: ", err, "debug=", runtime.WhereAmI())
		return nil, response, err
	}

	if tree == nil {
		return nil, response, errorMarshallingResponse
	}

	if !structs.IsStruct(tree) {
		return nil, response, errorMarshallingResponse
	}

	var fm map[string]interface{}
	var err error
	m := structs.Map(tree)
	fm, err = flatten.Flatten(m, "", flatten.DotStyle)
	if err != nil {
		log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
		return nil, response, err
	}

	return fm, response, nil
}

func (g *Github) getReadme(opts *Options) (map[string]interface{}, *github.Response, error) {
	if g.client == nil {
		g.client = getClient(g.ctoken)
	}
	g.mu.RLock()
	defer g.mu.RUnlock()

	var readme *github.RepositoryContent
	var response *github.Response
	get := func() error {
		var err error

		// g.rateLimiter().Wait()
		readme, response, err = g.client.Repositories.GetReadme(context.Background(), opts.Target.Owner, opts.Target.Name, nil)
		if status := g.limitHandler(response.StatusCode, response.Rate, response.Header, err); status != nil {
			return err
		}
		return nil
	}

	if err := g.retryRegistrationFunc(get); err != nil {
		log.Println("error: ", err, "debug=", runtime.WhereAmI())
		return nil, response, err
	}

	if readme == nil {
		return nil, response, errorMarshallingResponse
	}

	if !structs.IsStruct(readme) {
		return nil, response, errorMarshallingResponse
	}

	content, _ := readme.GetContent()
	readme.Content = &content

	var fm map[string]interface{}
	var err error
	m := structs.Map(readme)
	fm, err = flatten.Flatten(m, "", flatten.DotStyle)
	if err != nil {
		log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
		return nil, response, err
	}

	return fm, response, err
}
