package github

import (
	"time"

	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"

	"github.com/sniperkit/cuckoofilter"

	"github.com/sniperkit/xtask/plugin/aggregate/service"
	"github.com/sniperkit/xtask/plugin/counter"
	"github.com/sniperkit/xtask/plugin/rate"
)

// var log = logger.GetLogger("discovery")

var (
	Service             *Github
	rateLimiters        = map[string]*rate.RateLimiter{} // make(map[string]*rate.RateLimiter)
	isBackoff           bool
	defaultOpts         *Options
	defaultRetryDelay   time.Duration = 3 * time.Second
	defaultAbuseDelay   time.Duration = 3 * time.Second
	defaultRetryAttempt uint64        = 1
	defaultPrefixApi                  = "https://api.github.com/"
)

var (
	cfMax    *uint32
	cfDone   *cuckoofilter.Filter
	cf404    *cuckoofilter.Filter
	counters *counter.Oc
)

var (
	CacheEngine     = "badger"
	CachePrefixPath = "./shared/data/cache/http"
	xcache          httpcache.Cache
)

func New(tokens []*service.Token, opts *Options) *Github {
	g := &Github{
		ctoken:       tokens[0].Key,
		ctokens:      tokens,
		coptions:     opts,
		rateLimiters: make(map[string]*rate.RateLimiter, len(tokens)), // map[string]*rate.RateLimiter{}, //
		counters:     counter.NewOc(),
	}
	g.getClient(tokens[0].Key)
	return g

}

func Init() {
	defaultOpts = &Options{}
	defaultOpts.Page = 1
	defaultOpts.PerPage = 100
	Service = New(nil, defaultOpts)
}

func (Github) ProviderName() string {
	return serviceName
}

func (Github) PrefixApi() string {
	return defaultPrefixApi
}

type Context struct {
	Runner string
	Target *Target
}

type Profile struct {
	Owner    string
	Contribs bool
	Followed bool
	Starred  bool
}

type Target struct {
	Owner  string
	Name   string
	Branch string
	Ref    string
}

type Options struct {
	Runner               string
	Accounts             []string
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
