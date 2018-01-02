package github

import (
	"context"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"

	// "github.com/sniperkit/xapi/pkg"
	// "github.com/sniperkit/xapi/service/github"

	"github.com/sniperkit/xtask/pkg/rate"
	"github.com/sniperkit/xtask/pkg/util"

	"github.com/segmentio/stats/httpstats"
	// "github.com/segmentio/stats"
	// "github.com/segmentio/stats/influxdb"

	// "github.com/sniperkit/xcache/pkg"
	"github.com/gregjones/httpcache"
	"github.com/sniperkit/xcache/backend/default/badger"
	"github.com/sniperkit/xcache/backend/default/diskv"
)

func getClient(token string) *github.Client {
	if token != "" {

		var err error
		var hcache httpcache.Cache

		utils.EnsureDir(CachePrefixPath)
		CacheEngine = strings.ToLower(CacheEngine)

		switch CacheEngine {
		case "diskv":
			cacheStoragePrefixPath := filepath.Join(CachePrefixPath, "cacher.diskv")
			utils.EnsureDir(cacheStoragePrefixPath)
			hcache = diskcache.New(cacheStoragePrefixPath)

		case "badger":
			cacheStoragePrefixPath := filepath.Join(CachePrefixPath, "cacher.badger")
			utils.EnsureDir(cacheStoragePrefixPath)
			hcache, err = badgercache.New(
				&badgercache.Config{
					ValueDir:    "api.github.com.v3.snappy",
					StoragePath: cacheStoragePrefixPath,
					SyncWrites:  true,
					Debug:       false,
					Compress:    true,
				})

		case "memory":
			hcache = httpcache.NewMemoryCache()

		default:
			hcache = nil

		}

		if err != nil {
			log.Fatal("cache err", err.Error())
		}

		var hc http.Client
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)

		// t := httpcache.NewBlockingTransport(nil)
		// t := httpcache.NewTransport(transport)
		t := httpcache.NewTransport(hcache)
		t.MarkCachedResponses = false
		t.Transport = httpcache.NewBlockingTransport(nil)

		/*
			transport := &apiproxy.RevalidationTransport{
				Transport: t,
				Check: (&githubproxy.MaxAge{
					User:         time.Hour * 24,
					Repository:   time.Hour * 24,
					Readme:       time.Hour * 24,
					Languages:    time.Hour * 24,
					Topics:       time.Hour * 24,
					Repositories: time.Hour * 24,
					Activity:     time.Hour * 12,
				}).Validator(),
			}
		*/
		// hc.Transport = transport
		// hc.Transport = t

		/*
			influxConfig = influxdb.ClientConfig{
				Database:   "limo-httpstats",
				Address:    "127.0.0.1:8086",
				BufferSize: 2 * 1024 * 1024,
				Timeout:    5 * time.Second,
			}
			influxClient = influxdb.NewClientWith(influxConfig)
			influxClient.CreateDB("limo-httpstats")

			// stats.Register(influxClient)
			// defer stats.Flush()
			statsEngine = stats.NewEngine("limo", influxClient, statsTags...)
			// register engine
			// statsEngine.Register(influxClient)
			// defer statsEngine.Flush()

			hc.Transport = httpstats.NewTransportWith(statsEngine, t)
		*/

		hc.Transport = httpstats.NewTransport(t)
		// timeout := time.Duration(10 * time.Second)

		ghClient := github.NewClient(&http.Client{
			Transport: &oauth2.Transport{
				Base:   hc.Transport,
				Source: ts,
			},
			// Timeout: timeout,
		})

		// tc := oauth2.NewClient(context.Background(), ts)
		return ghClient
	}

	return github.NewClient(nil)
}

// GetRateLimit helps keep track of the API rate limit.
func (g *Github) getRateLimit() (int, error) {
	if g.client == nil {
		g.client = getClient(g.ctoken)
	}

	limits, _, err := g.client.RateLimits(context.Background())
	if err != nil {
		return 0, err
	}
	return limits.Core.Limit, nil
}

func rateLimiter(name string) *rate.RateLimiter {
	rl, ok := rateLimiters[name]
	if !ok {
		limit := 20
		if name == "" {
			limit = 5
		}
		rl = rate.New(limit, time.Minute)
		rateLimiters[name] = rl
	}
	return rl
}

func (g *Github) rateLimiter(name string) *rate.RateLimiter {
	rl, ok := rateLimiters[name]
	if !ok {
		limit := 20
		if name == "" {
			limit = 5
		}
		rl = rate.New(limit, time.Minute)
		rateLimiters[name] = rl
	}
	return rl
}
