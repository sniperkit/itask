package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	// "github.com/cnf/structhash"
	"github.com/gregjones/httpcache"
	cuckoo "github.com/seiflotfy/cuckoofilter"
	"github.com/willf/bloom"

	// "github.com/mandolyte/csv-utils"
	// "github.com/sniperkit/xtask/plugin/counter"

	"github.com/sniperkit/xcache/backend/default/badger"
	"github.com/sniperkit/xcache/backend/default/diskv"
	"github.com/sniperkit/xtask/util/fs" // move into a separate repo/package
)

var (
	CacheEngine     = "badger"
	CacheDrive      = "/Volumes/HardDrive/" // ""
	CachePrefixPath = CacheDrive + "./shared/data/cache/http"
	xcache          httpcache.Cache
	taskTTL         time.Duration = time.Duration(24 * 120 * time.Hour)
	blmflt                        = bloom.New(500000, 5)
	cuckflt                       = cuckoo.NewDefaultCuckooFilter()
)

func parseTimeStamp(utime string) (*time.Time, error) {
	i, err := strconv.ParseInt(utime, 10, 64)
	if err != nil {
		return nil, err
	}
	t := time.Unix(i, 0)
	return &t, nil
}

func getTasksRank() {}

func skipTasksWithTTL(filepath string) {
	fp, err := os.Open(filepath)
	if err != nil {
		log.Fatal(err)
	}
	defer fp.Close()

	// xz := xzReader(fp)
	csv := csv.NewReader(fp)
	lines := streamCsv(csv, Buffer)

	for line := range lines {

		expiresAt, err := parseTimeStamp(line.Get("task_expired_timestamp"))
		if err != nil {
			log.Errorln("[SKIP-ERROR] taskInfo, service=", line.Get("service"), "topic=", line.Get("topic"), "expiresTimestamp", line.Get("task_expired_timestamp"))
			continue
		}

		now := time.Now()
		if now.After(expiresAt.Add(taskTTL)) {
			log.Infoln("[TSK-ALLOW] task info, service=", line.Get("service"), "topic=", line.Get("topic"), "expiresAt=", expiresAt)
			continue
		}
		//taskHash := fmt.Sprintf("%x", structhash.Sha1(line.Get("topic"), 1))
		//cuckflt.InsertUnique([]byte(taskHash))
		cuckflt.InsertUnique([]byte(line.Get("topic")))

	}

	log.Warnln("[TSK-EXCLUDED] taskInfo, count=", cuckflt.Count())

	// log.Fatal("test...\n")

	// convertedLines := convertLine(lines)
	// completed := printStream(convertedLines)

	// halt until I am told we are done
	// x := <-completed
	// fmt.Printf("Done %d lines\n", x)

	// for k, line := range convertedLines {
	//	log.Println("line=", line)
	// }

	// cuckflt.

}

func getCache() httpcache.Cache {
	defer funcTrack(time.Now())

	backendCache, err := newCacheBackend(CacheEngine, CachePrefixPath)
	if err != nil {
		log.Fatal("cache err", err.Error())
	}

	return backendCache
}

/*

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

func listCache(cnt *counter.Oc, maxKeys uint32, prefix *string, remove *string, stopPatterns *[]string, existingKeys *map[string]*interface{}) (*cuckoo.CuckooFilter, int) {
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
*/

func newCacheBackend(engine string, prefixPath string) (backend httpcache.Cache, err error) {
	defer funcTrack(time.Now())

	fsutil.EnsureDir(prefixPath)
	engine = strings.ToLower(engine)

	switch CacheEngine {
	case "diskv":
		cacheStoragePrefixPath := filepath.Join(prefixPath, "cacher.diskv")
		fsutil.EnsureDir(cacheStoragePrefixPath)
		backend = diskcache.New(cacheStoragePrefixPath)

	case "badger":
		cacheStoragePrefixPath := filepath.Join(prefixPath, "cacher.badger")
		fsutil.EnsureDir(cacheStoragePrefixPath)
		backend, err = badgercache.New(
			&badgercache.Config{
				ValueDir:    "api.github.com.v3.snappy",
				StoragePath: cacheStoragePrefixPath,
				SyncWrites:  false,
				Debug:       false,
				Compress:    true,
				TTL:         time.Duration(120 * 24 * time.Hour),
			})

	case "memory":
		backend = httpcache.NewMemoryCache()

	default:
		backend = nil
	}

	return
}

const (
	defaultSvc       = "gh"
	defaultSvcDomain = "https://api.github.com/"
)

func cacheTaskResult(createdAt time.Time, service string, key string, obj map[string]interface{}) {
	//go func() {
	xcache.Set(key, toBytes(mapToString(obj)))
	//}()
}

func cacheFilter(filepath string) {}

func cacheSet(key string, obj map[string]interface{}) {
	xcache.Set(key, toBytes(mapToString(obj)))
}

func toBytes(input string) []byte {
	return []byte(input)
}

func mapToString(input map[string]interface{}) string {
	return toString(input)
}

func toString(obj interface{}) string {
	return fmt.Sprintf("%v", obj)
}

/*
func CacheItems(group string, items []interface{}) {
	// We create an encoder.
	// enc := ffjson.NewEncoder(out)

	for i, item := range items {
		// Encode into the buffer

		buf, err := ffjson.Marshal(&item)
		if err != nil {
			log.Fatalln("Encode error:", err)
		}

		key := fmt.Sprintf("%s/%s", group, item["hash"].(string))
		xcache.Set(key, buf)
	}
}
*/
