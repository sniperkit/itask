package github

import (
	"net/http"
	"time"

	"github.com/gregjones/httpcache"
	"github.com/segmentio/stats/httpstats"
)

var (
	CacheEngine     = "badger"
	CachePrefixPath = "./shared/data/cache/http"
	xcache          httpcache.Cache
	xtransport      *httpcache.Transport
)

/*
keepAliveTimeout:= 600 * time.Second
timeout:= 2 * time.Second
defaultTransport := &http.Transport{
    Dial: (&net.Dialer{
                     KeepAlive: keepAliveTimeout,}
           ).Dial,
    MaxIdleConns: 100,
    MaxIdleConnsPerHost: 100,
}
client:= &http.Client{
           Transport: defaultTransport,
           Timeout:   timeout,
}
*/

func initCacheTransport() (httpcache.Cache, *httpcache.Transport) {
	defer funcTrack(time.Now())

	backendCache, err := newCacheBackend(CacheEngine, CachePrefixPath)
	if err != nil {
		log.Fatal("cache err", err.Error())
	}

	var httpTransport = http.DefaultTransport
	httpTransport = httpstats.NewTransport(httpTransport)
	http.DefaultTransport = httpTransport

	cachingTransport := httpcache.NewTransportFrom(backendCache, httpTransport) // httpcache.NewMemoryCacheTransport()
	cachingTransport.MarkCachedResponses = true

	return backendCache, cachingTransport
}

func setCacheExpire(key string, date time.Time) bool {
	defer funcTrack(time.Now())

	return true
}
