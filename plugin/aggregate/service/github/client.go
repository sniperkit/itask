package github

import (
	"errors"
	"net/http"
	"time"
	// "sync"

	"github.com/anacrolix/sync"

	exBackoff "github.com/jpillora/backoff"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	// "golang.org/x/oauth2/github"

	cuckoo "github.com/seiflotfy/cuckoofilter"
	"github.com/sniperkit/cuckoofilter"

	"github.com/sniperkit/xtask/plugin/aggregate/service"
	"github.com/sniperkit/xtask/plugin/counter"
	"github.com/sniperkit/xtask/plugin/rate"

	"github.com/gregjones/httpcache"
	"github.com/segmentio/stats/httpstats"
)

type Github struct {
	ctoken  string
	ctokens []*service.Token
	// ref. https://github.com/pantheon-systems/baryon/blob/master/source/gh/gh.go
	exBackoff    *exBackoff.Backoff
	tokens       map[string]*service.TokenProfile
	coptions     *Options
	client       *github.Client
	rateLimiters map[string]*rate.RateLimiter
	mu           sync.Mutex
	xcache       httpcache.Cache
	manager      *ClientManager
	rateLimits   [categories]Rate
	timer        *time.Timer
	// rateMu       sync.Mutex
	cfMax     *uint32
	cfVisited *cuckoo.CuckooFilter
	cfDone    *cuckoofilter.Filter
	cf404     *cuckoofilter.Filter
	counters  *counter.Oc
}

func (g *Github) getClient(token string) *github.Client {
	defer funcTrack(time.Now())

	g.mu.Lock()
	defer g.mu.Unlock()

	if g.ctoken == "" {
		g.ctoken = token
	}
	resetClient := false
	if g.ctoken != token {
		g.ctoken = token
		resetClient = true
	}
	if g.client == nil {
		resetClient = true
	}
	if g.rateLimiters == nil {
		g.rateLimiters = make(map[string]*rate.RateLimiter, len(g.tokens))
	}
	log.Println("#1 / g.ctoken=", g.ctoken, "resetClient=", resetClient, "g.xcache=", g.xcache == nil)
	if g.xcache == nil {
		var err error
		g.xcache, err = newCacheBackend(CacheEngine, CachePrefixPath)
		if err != nil {
			log.Fatal("cache err", err.Error())
		}
	}
	log.Println("#2 / g.ctoken=", g.ctoken, "resetClient=", resetClient, "g.xcache=", g.xcache == nil)
	if g.client != nil && !resetClient {
		return g.client
	}

	/*
		// ref. https://github.com/golang/build/blob/master/maintner/github.go
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
		hc := oauth2.NewClient(ctx, ts)
		//if tr, ok := hc.Transport.(*http.Transport); ok {
		//	defer tr.CloseIdleConnections()
		//}
		directTransport := hc.Transport
		cachingTransport := &httpcache.Transport{
			Transport:           directTransport,
			Cache:               &githubCache{Cache: httpcache.NewMemoryCache()},
			MarkCachedResponses: true, // adds "X-From-Cache: 1" response header.
		}
	*/

	var httpTransport = http.DefaultTransport
	httpTransport = httpstats.NewTransport(httpTransport)
	http.DefaultTransport = httpTransport

	cachingTransport := httpcache.NewTransportFrom(g.xcache, httpTransport) // httpcache.NewMemoryCacheTransport()
	cachingTransport.MarkCachedResponses = true
	// reqModifyingTransport := newCacheRevalidationTransport(cachingTransport, revalidationDefaultMaxAge)

	oauth2Source := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	/*
		// Configure the default http transport which is used for forwarding the requests.
		http.DefaultTransport = httpstats.NewTransport(nil, &http.Transport{
			DialContext:            dialer(config.DialTimeout),
			IdleConnTimeout:        config.IdleTimeout,
			MaxIdleConns:           config.MaxIdleConns,
			MaxIdleConnsPerHost:    config.MaxIdleConnsPerHost,
			ResponseHeaderTimeout:  config.ReadTimeout,
			ExpectContinueTimeout:  config.ReadTimeout,
			MaxResponseHeaderBytes: int64(config.MaxHeaderBytes),
			DisableCompression:     !config.EnableCompression,
		})
	*/

	// http.DefaultTransport = httpcache.NewBlockingTransport(httpTransport)
	httpClient := &http.Client{
		Transport: &oauth2.Transport{
			Base: cachingTransport,
			// Base:   reqModifyingTransport,
			Source: oauth2Source,
		},
	}

	g.client = github.NewClient(httpClient)

	return g.client
}

func getClient(token string) *github.Client {
	defer funcTrack(time.Now())

	if token != "" {
		backendCache, err := newCacheBackend(CacheEngine, CachePrefixPath)
		if err != nil {
			log.Fatal("cache err", err.Error())
		}

		var httpTransport = http.DefaultTransport
		httpTransport = httpstats.NewTransport(httpTransport)
		http.DefaultTransport = httpTransport

		cachingTransport := httpcache.NewTransportFrom(backendCache, httpTransport) // httpcache.NewMemoryCacheTransport()
		cachingTransport.MarkCachedResponses = true
		// reqModifyingTransport := newCacheRevalidationTransport(cachingTransport, revalidationDefaultMaxAge)

		oauth2Source := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)

		httpClient := &http.Client{
			Transport: &oauth2.Transport{
				Base: cachingTransport,
				// Base:   reqModifyingTransport,
				Source: oauth2Source,
			},
		}

		return github.NewClient(httpClient)
	}
	return github.NewClient(nil)
}

func newClientWithNoContext(token string) *http.Client {
	defer funcTrack(time.Now())

	return oauth2.NewClient(
		oauth2.NoContext,
		oauth2.StaticTokenSource(
			&oauth2.Token{
				AccessToken: token,
			},
		),
	)
}

// newClient create client based on token.
func newClient(token string) (client *Github, err error) {
	defer funcTrack(time.Now())

	if token == "" {
		client = new(Github)
		tokenSource := new(oauth2.TokenSource)
		if !client.init(*tokenSource) {
			err = errors.New("failed to create client")
			return nil, err
		}

		return client, nil
	}

	client = new(Github)
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	if !client.init(tokenSource) {
		err = errors.New("failed to create client")
		return nil, err
	}

	return client, nil
}

// init initializes the client, returns true if available, or returns false.
func (g *Github) init(tokenSource oauth2.TokenSource) bool {
	defer funcTrack(time.Now())

	httpClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	ghClient := github.NewClient(httpClient)
	g.client = ghClient
	if !g.isValidToken(httpClient) {
		return false
	}
	if g.isLimited() {
		return false
	}
	return true
}

// makeRequest sends an HTTP GET request and returns an HTTP response, following
// policy (such as redirects, cookies, auth) as configured on the client.
func (g *Github) makeRequest(httpClient *http.Client) (*http.Response, error) {
	defer funcTrack(time.Now())

	req, err := g.client.NewRequest("GET", "", nil)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
