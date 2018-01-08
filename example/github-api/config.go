package main

import (
	"net/http"
	"time"

	"github.com/jinzhu/configor"
	"github.com/sniperkit/xtask/plugin/aggregate/service"
)

var config Config

func loadConfig() {
	t := time.Now()
	configor.Load(&config, "shared/config/config.yaml")
	addMetrics(t, 1, false)
}

type Config struct {
	App struct {
		Name    string `default:"xtask-vcs" json:"name" yaml:"name" toml:"name"`
		Verbose bool   `default:"true" json:"verbose" yaml:"verbose" toml:"verbose"`
		Debug   bool   `default:"false" json:"debug" yaml:"debug" toml:"debug"`
	} `json:"app" yaml:"app" toml:"app"`

	Service struct {
		Github struct {
			Runner       string           `default:"roscopecoltran" json:"runner" yaml:"runner" toml:"runner"`
			Accounts     []string         `json:"accounts" yaml:"accounts" toml:"accounts"`
			Token        string           `json:"token" yaml:"token" toml:"token"`
			Tokens       []*service.Token `json:"tokens" yaml:"tokens" toml:"tokens"`
			ClientID     string           `json:"client_id" yaml:"client_id" toml:"client_id"`
			ClientSecret string           `json:"client_secret" yaml:"client_secret" toml:"client_secret"`
			PerPage      int              `default:"20" json:"per_page" yaml:"per_page" toml:"per_page"`
			Offset       int              `default:"1" json:"offset" yaml:"offset" toml:"offset"`
			MaxPage      int              `default:"-1" json:"max_page" yaml:"max_page" toml:"max_page"`
		} `json:"github" yaml:"github" toml:"github"`
	} `json:"service" yaml:"service" toml:"service"`

	Flow struct {
		Concurrency int `default:"5" json:"concurrency" yaml:"concurrency" toml:"concurrency"`
		Interval    int `default:"50" json:"interval" yaml:"interval" toml:"interval"`
	} `json:"flow" yaml:"flow" toml:"flow"`

	Pipeline struct {
		Length   int `default:"10000" json:"length" yaml:"length" toml:"length"`
		Interval int `default:"5" json:"interval" yaml:"interval" toml:"interval"`
		Workers  struct {
			Count    int `default:"15" json:"count" yaml:"count" toml:"count"`
			Interval int `default:"0" json:"interval" yaml:"interval" toml:"interval"`
		} `json:"workers" yaml:"workers" toml:"workers"`
	} `json:"pipeline" yaml:"pipeline" toml:"pipeline"`

	Stats struct {
		Engine struct {
			Disabled  bool   `json:"disabled" yaml:"disabled" toml:"disabled"`                            // disable stats forwarding to stats engine
			Name      string `default:"influxdb" json:"engine" yaml:"engine" toml:"engine"`               // engine name to use
			RetryConn int    `default:"3" json:"retry_connect" yaml:"retry_connect" toml:"retry_connect"` // Maximun attempt to try to connect to the stats engine

			Clients struct {
				InfluxDB struct {
					Address    string            `default:"localhost:8086" json:"address" yaml:"address" toml:"address"`      // Address of the InfluxDB database to send metrics to.
					Database   string            `default:"stats" json:"database" yaml:"database" toml:"database"`            // Name of the InfluxDB database to send metrics to.
					Timeout    time.Duration     `default:"3s" json:"timeout" yaml:"timeout" toml:"timeout"`                  // Maximum amount of time that requests to InfluxDB may take.
					BufferSize int               `default:"2097152" json:"buffer_size" yaml:"buffer_size" toml:"buffer_size"` // Maximum size of batch of events sent to InfluxDB.
					Transport  http.RoundTripper `json:"-" yaml:"-" toml:"-"`                                                 // Transport configures the HTTP transport used by the client to send requests to InfluxDB. By default http.DefaultTransport is used.
				} `json:"influxdb" yaml:"influxdb" toml:"influxdb"` // The ClientConfig type is used to configure InfluxDB clients.

				DataDog struct {
					Address    string   `default:"localhost:8125" json:"address" yaml:"address" toml:"address"`   // Address of the datadog database to send metrics to.
					BufferSize int      `default:"1024" json:"buffer_size" yaml:"buffer_size" toml:"buffer_size"` // Maximum size of batch of events sent to datadog.
					Filters    []string `json:"filters" yaml:"filters" toml:"filters"`                            // List of tags to filter. If left nil is set to DefaultFilters.
				} `json:"datadog" yaml:"datadog" toml:"datadog"` // The ClientConfig type is used to configure datadog clients.
			} `json:"clients" yaml:"clients" toml:"clients"`
		} `json:"engine" yaml:"engine" toml:"engine"`
	} `json:"stats" yaml:"stats" toml:"stats"`
}
