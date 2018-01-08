package github

import (
	"log"
	"strings"

	cuckoo "github.com/seiflotfy/cuckoofilter"
	"github.com/sniperkit/cuckoofilter"
	"github.com/sniperkit/xtask/plugin/counter"
)

/*
	Refs:
	- https://github.com/queirozfcom/tracker-api/blob/master/src/github.com/queirozfcom/trackerapi/service.go
*/

/*
func (g *Github) Dump() *Github {}
*/

var (
	cfMax     *uint32
	cfVisited *cuckoo.CuckooFilter
	cfDone    *cuckoofilter.Filter
	cf404     *cuckoofilter.Filter
	counters  *counter.Oc
)

func (g *Github) CacheCount() int {
	if g.cfVisited == nil {
		return 0
	}
	return int(g.cfVisited.Count())
}

func (g *Github) LoadCache(max int, prefix string, remove string, stopPatterns []string) bool {
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
	return g.cfVisited.Lookup([]byte(slug))
}

// g.cfDone.Delete([]byte(slug))
// g.cfDone.Count()

func getCached(cnt *counter.Oc, maxKeys uint32, prefix *string, remove *string, stopPatterns *[]string, existingKeys *map[string]*interface{}) (*cuckoo.CuckooFilter, int) {
	registry := cuckoo.NewDefaultCuckooFilter()
	for key, _ := range *existingKeys {
		slug := key
		if prefix != nil {
			if !strings.HasPrefix(slug, *prefix) {
				if cnt != nil {
					cnt.Increment("github.cache.skipped", 1)
				}
				// log.Println("[missing prefix] skipping slug ", slug)
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
					// log.Println("[contains stop word] skipping slug ", slug)
					skip = true
					break
				}
			}
			if skip {
				continue
			}
		}

		if remove != nil {
			// func Replace(s, old, new string, n int) string
			slug = strings.Replace(slug, *remove, "", -1)
			if cnt != nil {
				cnt.Increment("github.cache.remove.substring", 1)
			}
		}

		slug = strings.Replace(slug, "/", ".", -1)
		// log.Println("adding cache.slug=", slug)

		registry.InsertUnique([]byte(slug))
		if cnt != nil {
			cnt.Increment("github.cache.entry", 1)
		}

	}

	return registry, int(registry.Count())
}

/*
func (g *Github) LoadCache2(max int, prefix string, remove string, stopPatterns []string) bool {
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
		g.cfDone, loaded = getCached(g.counters, maxKeys, &prefix, &remove, &stopPatterns, &existingKeys)
		g.cf404 = cuckoofilter.New(maxKeys)
		log.Println("[g.LoadCache] loaded=", loaded, " / total.cache=", len(existingKeys))
		g.counters.Increment("cache.keys.loaded", loaded)
		g.counters.Increment("cache.keys.existing", len(existingKeys))
		return true
	}
	return false
}

func (g *Github) CacheSlugExists2(slug string) bool {
	if g.cfDone != nil {
		exists := g.cfDone.Contains([]byte(slug))
		// log.Println("slug:", slug, ", exists=", exists)
		return exists
	} else {
		log.Fatal("g.cfDone is nil")
	}
	return false
}

func getCached2(cnt *counter.Oc, maxKeys uint32, prefix *string, remove *string, stopPatterns *[]string, existingKeys *map[string]*interface{}) (*cuckoofilter.Filter, int) {
	i := 0
	registry := cuckoofilter.New(maxKeys)
	for key, _ := range *existingKeys {
		slug := key
		if prefix != nil {
			if !strings.HasPrefix(slug, *prefix) {
				if cnt != nil {
					cnt.Increment("github.cache.skipped", 1)
				}
				// log.Println("[missing prefix] skipping slug ", slug)
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
					// log.Println("[contains stop word] skipping slug ", slug)
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

		// log.Println("adding cache.slug=", slug, "cache.key=", key)
		registry.Add([]byte(slug))
		if cnt != nil {
			cnt.Increment("github.cache.entry", 1)
		}
		i++

	}
	return registry, i
}

func CacheSlugExists(curl string) (url string, ok bool) {
	// log.Println("input:", curl, "exists=", exists)
	ok = cfDone.Contains([]byte(url))
	url = curl
	return
}

func LoadCache(max int, prefix string, remove string, stopPatterns []string, cnt *counter.Oc) bool {
	if xcache != nil {
		maxKeys := uint32(max)
		cfMax = &maxKeys
		existingKeys, _ := xcache.Action("getKeys", nil)
		loaded := 0
		cfDone, loaded = getCached(cnt, maxKeys, &prefix, &remove, &stopPatterns, &existingKeys)
		cf404 = cuckoofilter.New(maxKeys)
		log.Println("[LoadCache] loaded=", loaded, " / total.cache=", len(existingKeys))
		return true
	}
	return false
}
*/
