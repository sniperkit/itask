package github

import (
	"strings"
	"time"

	cuckoo "github.com/seiflotfy/cuckoofilter"
	"github.com/sniperkit/cuckoofilter"
	"github.com/sniperkit/xtask/plugin/counter"
)

var (
	cfMax     *uint32
	cfVisited *cuckoo.CuckooFilter
	cfDone    *cuckoofilter.Filter
	cf404     *cuckoofilter.Filter
)

func (g *Github) CacheCount() int {
	defer funcTrack(time.Now())

	if g.cfVisited == nil {
		return 0
	}
	return int(g.cfVisited.Count())
}

func (g *Github) LoadCache(max int, prefix string, remove string, stopPatterns []string) bool {
	defer funcTrack(time.Now())

	if g.client == nil {
		g.client = getClient(g.ctoken)
	}
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.cfMax == nil && g.xcache != nil {
		g.counters.Increment("cache.load", 1)
		g.counters.Increment("cache.maxKeys", max)
		maxKeys := uint32(max)
		g.cfMax = &maxKeys
		existingKeys, _ := g.xcache.Action("getKeys", nil)
		loaded := 0
		g.cfVisited, loaded = getCached(g.counters, maxKeys, &prefix, &remove, &stopPatterns, &existingKeys)
		log.Println("[g.LoadCache] loaded=", loaded, " / total.cache=", len(existingKeys))
		g.counters.Increment("cache.keys.loaded", loaded)
		g.counters.Increment("cache.keys.existing", len(existingKeys))
		return true
	}
	return false
}

func (g *Github) CacheSlugExists(slug string) bool {
	defer funcTrack(time.Now())

	return g.cfVisited.Lookup([]byte(slug))
}

// g.cfDone.Delete([]byte(slug))
// g.cfDone.Count()

func getCached(cnt *counter.Oc, maxKeys uint32, prefix *string, remove *string, stopPatterns *[]string, existingKeys *map[string]*interface{}) (*cuckoo.CuckooFilter, int) {
	defer funcTrack(time.Now())

	registry := cuckoo.NewDefaultCuckooFilter()
	for key, _ := range *existingKeys {
		slug := key
		if prefix != nil {
			if !strings.HasPrefix(slug, *prefix) {
				if cnt != nil {
					cnt.Increment("github.cache.skipped", 1)
				}
				continue
			}
		}
		if stopPatterns != nil {
			var skip bool
			for _, pattern := range *stopPatterns {
				if strings.Contains(slug, pattern) {
					if cnt != nil {
						cnt.Increment("github.cache.skipped", 1)
					}
					skip = true
					break
				}
			}
			if skip {
				continue
			}
		}
		if remove != nil {
			slug = strings.Replace(slug, *remove, "", -1)
			if cnt != nil {
				cnt.Increment("github.cache.remove.substring", 1)
			}
		}
		registry.InsertUnique([]byte(slug))
		if cnt != nil {
			cnt.Increment("github.cache.entry", 1)
		}
	}
	return registry, int(registry.Count())
}

/*
func (g *Github) Dump() *Github {}
*/
