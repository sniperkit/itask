package github

import (
	"github.com/google/go-github/github"
	"github.com/sniperkit/xtask/pkg/rate"
)

var (
	Service      *Github
	defaultOpts  *Options
	rateLimiters = map[string]*rate.RateLimiter{}
)

var (
	CacheEngine     = "badger"
	CachePrefixPath = "./shared/data/cache/http"
)

type Github struct {
	ctoken       string
	coptions     *Options
	client       *github.Client
	rateLimiters map[string]*rate.RateLimiter
}

func New(token *string, opts *Options) *Github {
	return &Github{
		ctoken:       *token,
		coptions:     opts,
		client:       getClient(*token),
		rateLimiters: make(map[string]*rate.RateLimiter),
	}
}

func (Github) ProviderName() string {
	return serviceName
}

type Context struct {
	Runner string
	Target *Target
}

type Target struct {
	Owner string
	Name  string
}

type Options struct {
	Runner               string
	Page                 int
	PerPage              int
	Target               *Target
	ActivityListStarred  *github.ActivityListStarredOptions
	RepositoryContentGet *github.RepositoryContentGetOptions
	Project              *github.ProjectOptions
	List                 *github.ListOptions
	Raw                  *github.RawOptions
	Search               *github.SearchOptions
}
