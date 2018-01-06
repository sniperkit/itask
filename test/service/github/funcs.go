package github

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/structs"
	"github.com/pkg/errors"

	"github.com/google/go-github/github"

	"github.com/sniperkit/flatten"
	"github.com/sniperkit/xtask/util/runtime"
)

func (g *Github) GetFunc(entity string, opts *Options) (map[string]interface{}, *github.Response, error) {
	if g.client == nil {
		g.client = getClient(g.ctoken)
	}

	g.counters.Increment("github.GetFunc()", 1)

	switch entity {
	case "getStars":
		return getStars(g, opts)
	case "getRepo":
		return getRepo(g, opts)
	case "getReadme":
		return getReadme(g, opts)
	case "getTree":
		return getTree(g, opts)
	case "getTopics":
		return getTopics(g, opts)
	case "getLatestSHA":
		return getLatestSHA(g, opts)
	}
	return nil, nil, nil
}

func getStars(g *Github, opts *Options) (map[string]interface{}, *github.Response, error) {
	// g.mu.RLock()
	// defer g.mu.RUnlock()

	var stars []*github.StarredRepository
	var response *github.Response
	get := func() error {
		var err error

		// g.rateLimiter().Wait()
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

		if status, nc := limitHandler(response.StatusCode, response.Rate, response.Header, err); status != nil {
			if nc {
				g.getClient(g.getNextToken(g.ctoken))
			}
			return err
		}

		return nil
	}

	if err := g.retryRegistrationFunc(get); err != nil {
		g.counters.Increment("request.retry", 1)
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

func getRepo(g *Github, opts *Options) (map[string]interface{}, *github.Response, error) {
	// g.mu.RLock()
	// defer g.mu.RUnlock()

	var repo *github.Repository
	var response *github.Response
	get := func() error {
		var err error

		// g.rateLimiter().Wait()
		repo, response, err = g.client.Repositories.Get(context.Background(), opts.Target.Owner, opts.Target.Name)

		if status, nc := limitHandler(response.StatusCode, response.Rate, response.Header, err); status != nil {
			if nc {
				g.getClient(g.getNextToken(g.ctoken))
			}
			return err
		}

		return nil
	}

	if err := g.retryRegistrationFunc(get); err != nil {
		g.counters.Increment("request.retry", 1)
		log.Println("error: ", err, "debug=", runtime.WhereAmI())
		return nil, response, err
	}

	var fm map[string]interface{}
	var err error
	m := structs.Map(repo)
	fm, err = flatten.Flatten(m, "", flatten.DotStyle)
	if err != nil {
		log.Println("error: ", err.Error(), "debug=", runtime.WhereAmI())
		return nil, response, err
	}
	return fm, response, nil
}

func getTopics(g *Github, opts *Options) (map[string]interface{}, *github.Response, error) {
	// g.mu.RLock()
	// defer g.mu.RUnlock()

	var topics []string
	var response *github.Response
	get := func() error {
		var err error

		// g.rateLimiter().Wait()
		topics, response, err = g.client.Repositories.ListAllTopics(context.Background(), opts.Target.Owner, opts.Target.Name)

		if status, nc := limitHandler(response.StatusCode, response.Rate, response.Header, err); status != nil {
			if nc {
				g.getClient(g.getNextToken(g.ctoken))
			}
			return err
		}

		return nil
	}

	if err := g.retryRegistrationFunc(get); err != nil {
		g.counters.Increment("request.retry", 1)
		log.Println("error: ", err, "debug=", runtime.WhereAmI())
		return nil, response, err
	}

	fm := make(map[string]interface{}, 1)
	fm["topics"] = topics

	return fm, response, nil
}

func getLatestSHA(g *Github, opts *Options) (map[string]interface{}, *github.Response, error) {
	// g.mu.RLock()
	// defer g.mu.RUnlock()

	var ref *github.Reference
	var response *github.Response
	get := func() error {
		var err error
		if opts.Target.Branch == "" {
			opts.Target.Branch = "master"
		}

		// g.rateLimiter().Wait()
		ref, response, err = g.client.Git.GetRef(context.Background(), opts.Target.Owner, opts.Target.Name, "refs/heads/"+opts.Target.Branch)

		if status, nc := limitHandler(response.StatusCode, response.Rate, response.Header, err); status != nil {
			if nc {
				g.getClient(g.getNextToken(g.ctoken))
			}
			return err
		}

		return nil
	}

	if err := g.retryRegistrationFunc(get); err != nil {
		g.counters.Increment("request.retry", 1)
		log.Println("error: ", err, "debug=", runtime.WhereAmI())
		return nil, response, err
	}

	fm := make(map[string]interface{}, 1)
	fm["sha"] = *ref.Object.SHA

	return fm, response, nil
}

func getTree(g *Github, opts *Options) (map[string]interface{}, *github.Response, error) {
	// g.mu.RLock()
	// defer g.mu.RUnlock()

	var tree *github.Tree
	var response *github.Response
	get := func() error {
		var err error

		// g.rateLimiter().Wait()
		tree, response, err = g.client.Git.GetTree(context.Background(), opts.Target.Owner, opts.Target.Name, opts.Target.Ref, true)

		if status, nc := limitHandler(response.StatusCode, response.Rate, response.Header, err); status != nil {
			if nc {
				g.getClient(g.getNextToken(g.ctoken))
			}
			return err
		}

		return nil
	}

	if err := g.retryRegistrationFunc(get); err != nil {
		g.counters.Increment("request.retry", 1)
		log.Println("error: ", err, "debug=", runtime.WhereAmI())
		return nil, response, err
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

func getReadme(g *Github, opts *Options) (map[string]interface{}, *github.Response, error) {
	// g.mu.RLock()
	// defer g.mu.RUnlock()

	var readme *github.RepositoryContent
	var response *github.Response
	get := func() error {
		var err error

		// g.rateLimiter().Wait()
		readme, response, err = g.client.Repositories.GetReadme(context.Background(), opts.Target.Owner, opts.Target.Name, nil)

		if status, nc := limitHandler(response.StatusCode, response.Rate, response.Header, err); status != nil {
			if nc {
				g.getClient(g.getNextToken(g.ctoken))
			}
			return err
		}

		return nil
	}

	if err := g.retryRegistrationFunc(get); err != nil {
		g.counters.Increment("request.retry", 1)
		log.Println("error: ", err, "debug=", runtime.WhereAmI())
		return nil, response, err
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

func limitHandler(statusCode int, rate github.Rate, hdrs http.Header, err error) (error, bool) {
	if err != nil {
		if v := hdrs["Retry-After"]; len(v) > 0 {
			// g.counters.Increment("request.retry.after", 1)
			retryAfterSeconds, _ := strconv.ParseInt(v[0], 10, 64) // Error handling is noop.
			retryAfter := time.Duration(retryAfterSeconds) * time.Second
			log.Println("error:", err.Error(), "Retry-After=", v, "retryAfterSeconds: ", retryAfterSeconds, "retryAfter=", retryAfter.Seconds(), "debug", runtime.WhereAmI())
			// time.Sleep(retryAfter)
			time.Sleep(2 * time.Second)
			return nil, false
		} else {
			log.Println("Retry-After not found")
		}

		// Get the underlying error, if this is a Wrapped error by the github.com/pkg/errors package.
		// If not, this will just return the error itself.
		underlyingErr := errors.Cause(err)

		switch underlyingErr.(type) {

		case *github.RateLimitError:
			// g.counters.Increment("request.err.rate.limit.consumed", 1)
			return nil, true

		default:
			if strings.Contains(err.Error(), "timeout") ||
				strings.Contains(err.Error(), "abuse detection") ||
				strings.Contains(err.Error(), "try again") {
				// g.counters.Increment("request.err.rate.limit.fallback", 1)

				time.Sleep(2 * time.Second)
				log.Println("error:", err.Error(), "underlyingErr.(type).default", underlyingErr, "debug", runtime.WhereAmI())
				return nil, false
			}

			return err, false
		}

	} else {
		// g.counters.Increment("request.rate.limit.check", 1)
		log.Println("statusCode:", statusCode, "rate.remaining=", rate.Remaining)
	}

	return nil, false
}
