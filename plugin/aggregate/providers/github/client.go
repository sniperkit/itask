package github

import (
	"context"
	"errors"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"

	"github.com/sniperkit/cuckoofilter"
	"github.com/sniperkit/xtask/pkg/util"
	"github.com/sniperkit/xtask/plugin/counter"
	"github.com/sniperkit/xtask/plugin/rate"
	"github.com/sniperkit/xtask/test/service"

	"github.com/segmentio/stats/httpstats"

	"github.com/gregjones/httpcache"
	"github.com/sniperkit/xcache/backend/default/badger"
	"github.com/sniperkit/xcache/backend/default/diskv"
	// "github.com/shurcooL/githubql"
	// "github.com/sniperkit/xapi/pkg"
	// "github.com/sniperkit/xapi/service/github"
	// "github.com/segmentio/stats"
	// "github.com/segmentio/stats/influxdb"
	// "github.com/sniperkit/xcache/pkg"
)

/*
	Refs:
	- https://github.com/smook1980/sourcegraph/blob/master/pkg/githubutil/client.go#L69
	- https://github.com/smook1980/sourcegraph/blob/master/pkg/githubutil/client.go#L176
	- https://github.com/sourcegraph/apiproxy
	-
*/

type Github struct {
	ctoken       string
	ctokens      []*service.Token
	tokens       map[string]*service.TokenProfile
	coptions     *Options
	client       *github.Client
	rateLimiters map[string]*rate.RateLimiter
	mu           sync.RWMutex
	mc           sync.RWMutex
	xcache       httpcache.Cache
	manager      *ClientManager
	rateLimits   [categories]Rate
	timer        *time.Timer
	rateMu       sync.Mutex
	cfMax        *uint32
	cfDone       *cuckoofilter.Filter
	cf404        *cuckoofilter.Filter
	counters     *counter.Oc
}

type GHClient struct {
	Client     *github.Client
	Manager    *ClientManager
	rateLimits [categories]Rate
	timer      *time.Timer
	rateMu     sync.Mutex
}

func (g *Github) getClient(token string) *github.Client {
	g.mc.Lock()
	defer g.mc.Unlock()

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

	// g.rateLimiter().Wait()
	log.Println("#1 / g.ctoken=", g.ctoken, "resetClient=", resetClient, "g.xcache=", g.xcache == nil)

	if g.xcache == nil {
		var err error

		util.EnsureDir(CachePrefixPath)
		CacheEngine = strings.ToLower(CacheEngine)

		switch CacheEngine {
		case "diskv":
			cacheStoragePrefixPath := filepath.Join(CachePrefixPath, "cacher.diskv")
			util.EnsureDir(cacheStoragePrefixPath)
			g.xcache = diskcache.New(cacheStoragePrefixPath)

		case "badger":
			cacheStoragePrefixPath := filepath.Join(CachePrefixPath, "cacher.badger")
			util.EnsureDir(cacheStoragePrefixPath)
			g.xcache, err = badgercache.New(
				&badgercache.Config{
					ValueDir:    "api.github.com.v3.snappy",
					StoragePath: cacheStoragePrefixPath,
					SyncWrites:  false,
					Debug:       false,
					Compress:    true,
				})

		case "memory":
			g.xcache = httpcache.NewMemoryCache()

		default:
			g.xcache = nil

		}

		if err != nil {
			log.Fatal("cache err", err.Error())
		}

	}

	log.Println("#2 / g.ctoken=", g.ctoken, "resetClient=", resetClient, "g.xcache=", g.xcache == nil)

	if g.client != nil && !resetClient {
		return g.client
	}

	var hc http.Client

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	t := httpcache.NewTransport(g.xcache)
	t.MarkCachedResponses = true

	hc.Transport = httpstats.NewTransport(t)

	ghClient := github.NewClient(&http.Client{
		Transport: &oauth2.Transport{
			Base:   hc.Transport,
			Source: ts,
		},
	})

	g.client = ghClient

	/*
		go func() {
			for {
				select {
				case <-stop:
					return
				case <-time.After(time.Second * 10):
					log.Println("counters=", g.counters.Snapshot())
				}
			}
		}()
	*/

	return g.client
}

/*
func Close() {
	<-stop
	close(stop)
}
*/

func newClientWithNoContext(token string) *http.Client {
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

func (g *Github) retryRegistrationFunc(f func() error) error {
	return backoff.Retry(f, backoff.WithMaxTries(backoff.WithContext(backoff.NewConstantBackOff(defaultRetryDelay), context.Background()), defaultRetryAttempt))
}

func (g *Github) retryNotifyRegistrationFunc(f func() error) error {
	return backoff.RetryNotify(f, backoff.WithMaxTries(backoff.WithContext(backoff.NewConstantBackOff(defaultRetryDelay), context.Background()), defaultRetryAttempt), g.notifyAttempts)
}

// isValidToken check if token is valid.
func (g *Github) isValidToken(httpClient *http.Client) bool {
	resp, err := g.makeRequest(httpClient)
	if err != nil {
		return false
	}

	err = github.CheckResponse(resp)
	if _, ok := err.(*github.TwoFactorAuthError); ok {
		return false
	}

	return true
}

// newClients create a client list based on tokens.
func newClients(tokens []string) []*Github {
	var clients []*Github

	for _, t := range tokens {
		client, err := newClient(t)
		if err != nil {
			continue
		}

		clients = append(clients, client)
	}

	return clients
}

// ClientManager used to manage the valid client.
type ClientManager struct {
	Dispatch chan *Github
	reclaim  chan *Github
	shutdown chan struct{}
}

// start start reclaim and dispatch the client.
func (cm *ClientManager) start() {
	for {
		select {
		case v := <-cm.reclaim:
			cm.Dispatch <- v
		case <-cm.shutdown:
			close(cm.Dispatch)
			close(cm.reclaim)
			return
		}
	}
}

// NewManager create a new client manager based on tokens.
func NewManager(tokens []string) *ClientManager {
	var cm *ClientManager = &ClientManager{
		reclaim:  make(chan *Github),
		Dispatch: make(chan *Github, len(tokens)),
		shutdown: make(chan struct{}),
	}

	clients := newClients(tokens)

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

// Fetch fetch a valid client.
func (cm *ClientManager) Fetch() *Github {
	return <-cm.Dispatch
}

// Reclaim reclaim client while the client is valid.
// resp: The response returned when calling the client.
func Reclaim(client *Github, resp *github.Response) {
	client.initTimer(resp)

	select {
	case <-client.timer.C:
		client.manager.reclaim <- client
	}
}

// Shutdown shutdown the client manager.
func (cm *ClientManager) Shutdown() {
	close(cm.shutdown)
}

func getClient(token string) *github.Client {
	if token != "" {
		var err error
		util.EnsureDir(CachePrefixPath)
		CacheEngine = strings.ToLower(CacheEngine)

		switch CacheEngine {
		case "diskv":
			cacheStoragePrefixPath := filepath.Join(CachePrefixPath, "cacher.diskv")
			util.EnsureDir(cacheStoragePrefixPath)
			xcache = diskcache.New(cacheStoragePrefixPath)

		case "badger":
			cacheStoragePrefixPath := filepath.Join(CachePrefixPath, "cacher.badger")
			util.EnsureDir(cacheStoragePrefixPath)
			xcache, err = badgercache.New(
				&badgercache.Config{
					ValueDir:    "api.github.com.v3.snappy",
					StoragePath: cacheStoragePrefixPath,
					SyncWrites:  false,
					Debug:       false,
					Compress:    true,
				})

		case "memory":
			xcache = httpcache.NewMemoryCache()

		default:
			xcache = nil

		}

		if err != nil {
			log.Fatal("cache err", err.Error())
		}

		var hc http.Client
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)

		t := httpcache.NewTransport(xcache)
		t.MarkCachedResponses = true

		hc.Transport = httpstats.NewTransport(t)
		ghClient := github.NewClient(&http.Client{
			Transport: &oauth2.Transport{
				Base:   hc.Transport,
				Source: ts,
			},
		})

		return ghClient
	}

	return github.NewClient(nil)
}
