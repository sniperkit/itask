package main

import (
	"time"

	"github.com/sniperkit/xstats/client/datadog"
	"github.com/sniperkit/xstats/client/influxdb"
	"github.com/sniperkit/xstats/pkg"
)

// stats/metrics engine
var (
	statsEngine *stats.Engine
	statsTags   []*stats.Tag
)

// stats storage client(s)
var (
	influxClient      *influxdb.Client
	influxClientConf  *influxdb.ClientConfig
	datadogClient     *datadog.Client
	datadogClientConf *datadog.ClientConfig
)

/*
	*** InfluxDB (API) ***
	- Install: `brew install influxdb`
	- Run:
		- To have launchd start influxdb now and restart at login: `brew services start influxdb`
		- Or, if you don't want/need a background service you can just run: `influxd -config /usr/local/etc/influxdb.conf`

	*** Chronograf (UI) ***
	- Install: `brew install chronograf`
	- Run:
		- To have launchd start chronograf now and restart at login: `brew services start chronograf`
		- Or, if you don't want/need a background service you can just run: `chronograf`
*/

func newStatsEngine() {
	switch config.Stats.Engine.Name {
	case "datadog":
		statsEngine = nil
	case "influxdb":
		fallthrough
	default:
		statsEngine = nil
	}
}

func statsWithTags() {}

type funcMetrics struct {
	calls struct {
		count  int           `metric:"count" type:"counter"`
		failed int           `metric:"failed" type:"counter"`
		time   time.Duration `metric:"time"  type:"histogram"`
	} `metric:"func.calls"`
}

func addMetrics(start time.Time, incr int, failed bool) {
	callTime := time.Now().Sub(start)
	m := &funcMetrics{}
	m.calls.count = incr
	m.calls.time = callTime
	if failed {
		m.calls.failed = incr
	}
	stats.Report(m)
}
