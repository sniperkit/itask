package github

import (
	"errors"
	"net/http"
	"time"

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

var (
	defaultCLI *github.Client
)

type Github struct {
	ctoken  string
	ctokens []*service.Token
	// ref. https://github.com/pantheon-systems/baryon/blob/master/source/gh/gh.go
	exBackoff   *exBackoff.Backoff
	tokens      map[string]*service.TokenProfile
	coptions    *Options
	client      *github.Client
	tokenSource oauth2.TokenSource
	httpClient  *http.Client
	transport   *httpcache.Transport
	isPaused    bool

	rateLimiters map[string]*rate.RateLimiter
	mu           sync.Mutex
	xcache       httpcache.Cache
	manager      *ClientManager
	rateLimits   [categories]Rate
	timer        *time.Timer
	rateMu       sync.Mutex
	wg           sync.WaitGroup

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

func NewCacheWithTransport(xc httpcache.Cache) *httpcache.Transport {
	defer funcTrack(time.Now())

	var httpTransport = http.DefaultTransport
	httpTransport = httpstats.NewTransport(httpTransport)
	http.DefaultTransport = httpTransport

	cachingTransport := httpcache.NewTransportFrom(xc, httpTransport) // httpcache.NewMemoryCacheTransport()
	cachingTransport.MarkCachedResponses = true

	return cachingTransport
}

var clientManager *ClientManager //= NewManager(tokens)

// ClientManager used to manage the valid client.
type ClientManager struct {
	Dispatch chan *Github
	reclaim  chan *Github
	shutdown chan struct{}
}

// initTimer initialize client timer.
func (g *Github) startTimer(resetAt time.Time) {
	timer := time.NewTimer(resetAt.Sub(time.Now()) + time.Second*2)
	g.timer = timer
	return
}

func changeClient34(g *Github, resp *github.Response) *Github {
	// g.mu.Lock()
	// defer g.mu.Unlock()

	// g.wg.Add(1)
	// defer g.wg.Done()

	log.Info("changeClient")
	//g.wg.Add(1)
	//go func() {
	//	g.Reclaim((*resp).Reset.Time)
	//}()

	//g.wg.Done()
	return g.manager.Fetch()

	/*
		var wg sync.WaitGroup

		go func() {
			wg.Add(1)
			defer wg.Done()
			Reclaim(g, resp)
		}()

		// g = g.manager.Fetch()
		return g.manager.Fetch()
	*/
}

// NewManager create a new client manager based on tokens.
func NewManager(tokens []string, opts *Options, xc *httpcache.Cache) *ClientManager {
	defer funcTrack(time.Now())

	// log.Fatalln("len(tokens)=", len(tokens))

	var cm *ClientManager = &ClientManager{
		reclaim:  make(chan *Github),
		Dispatch: make(chan *Github, len(tokens)),
		shutdown: make(chan struct{}),
	}
	clients := newClients(tokens, opts, xc)
	go cm.start()

	go func() {
		for _, c := range clients {
			if !c.isLimited() {
				c.manager = cm
				cm.reclaim <- c
			}
		}
	}()

	return cm
}

//func init() {
//	xcache, xtransport = initCacheTransport()
//}

func InitCache(xc httpcache.Cache, xt *httpcache.Transport) {
	xcache = xc
	xtransport = xt
}

// newClients create a client list based on tokens.
func newClients(tokens []string, opts *Options, xc *httpcache.Cache) []*Github {
	var clients []*Github

	for _, t := range tokens {
		gClient, gTokenSource, gHttpClient := getClientSharedCache(t, xc)
		if gClient != nil {

			ghClient := &Github{
				ctoken:       t,
				coptions:     opts,
				rateLimiters: make(map[string]*rate.RateLimiter, 1),
				counters:     counter.NewOc(),
				client:       gClient,
				httpClient:   gHttpClient,
				tokenSource:  gTokenSource,
				xcache:       *xc,
			}

			if !ghClient.isValidToken(gHttpClient) {
				continue
			}

			if ghClient.isLimited() {
				continue
			}

			clients = append(clients, ghClient)
		}
	}

	return clients
}

func getClientSharedCache(token string, xc *httpcache.Cache) (*github.Client, oauth2.TokenSource, *http.Client) {
	defer funcTrack(time.Now())

	if token != "" {

		oauth2Source := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)

		transport := NewCacheWithTransport(*xc)

		httpClient := &http.Client{
			Transport: &oauth2.Transport{
				Base:   transport,
				Source: oauth2Source,
			},
		}

		g := github.NewClient(httpClient)

		return g, oauth2Source, httpClient

	}

	return nil, nil, nil
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
