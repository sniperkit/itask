package main

import (
	"net/http"
	"time"

	"github.com/jinzhu/configor"
	"github.com/sniperkit/xtask/plugin/aggregate/service"
)

var (
	config               Config
	defaultOffsetStarred = 1
	defaultOffsetSearch  = 1
	defaultBeatConfig    = BeatConfig{
		Period:     30 * time.Second,
		JobTimeout: 10 * time.Second,
	}
	writersList = []string{
		"search_trends",
		"search_codes",
		"stars",
		"latest_sha",
		"repos",
		"readmes",
		"topics",
		"langs",
		"files",
		"tasks",
		"users",
		"user_following",
		"user_followers",
		"user_nodes",
		"release_tags",
		"graph",
	}
)

type BeatConfig struct {
	Period      time.Duration `config:"period"`
	JobTimeout  time.Duration `config:"period"`
	Repos       []string      `config:"repos"`
	Orgs        []string      `config:"orgs"`
	AccessToken string        `config:"access_token"`
}

type Config struct {
	App struct {
		Name    string `default:"xtask-vcs" json:"name" yaml:"name" toml:"name"`
		Verbose bool   `default:"true" json:"verbose" yaml:"verbose" toml:"verbose"`
		Debug   bool   `default:"false" json:"debug" yaml:"debug" toml:"debug"`
	} `json:"app" yaml:"app" toml:"app"`

	Service struct {
		LibrariesIO struct {
			Owner   string           `default:"roscopecoltran" json:"owner" yaml:"owner" toml:"owner"`
			Tokens  []*service.Token `json:"tokens" yaml:"tokens" toml:"tokens"`
			Ignore  []string         `json:"ignore" yaml:"ignore" toml:"ignore"`
			PerPage int              `default:"20" json:"per_page" yaml:"per_page" toml:"per_page"`
			Offset  int              `default:"1" json:"offset" yaml:"offset" toml:"offset"`
			MaxPage int              `default:"-1" json:"max_page" yaml:"max_page" toml:"max_page"`
		} `json:"librariesio" yaml:"librariesio" toml:"librariesio"`

		Github struct {
			Owner    string           `default:"roscopecoltran" json:"owner" yaml:"owner" toml:"owner"`
			Runner   string           `default:"roscopecoltran" json:"runner" yaml:"runner" toml:"runner"`
			Accounts []string         `json:"accounts" yaml:"accounts" toml:"accounts"`
			Token    string           `json:"token" yaml:"token" toml:"token"`
			Tokens   []*service.Token `json:"tokens" yaml:"tokens" toml:"tokens"`
			Search   struct {
				Offset  int      `default:"1" json:"offset" yaml:"offset" toml:"offset"`
				MaxPage int      `default:"-1" json:"max_page" yaml:"max_page" toml:"max_page"`
				Repo    []string `json:"repo" yaml:"repo" toml:"repo"`
				Code    []string `json:"code" yaml:"code" toml:"code"`
				Issue   []string `json:"issue" yaml:"issue" toml:"issue"`
				Commit  []string `json:"commit" yaml:"commit" toml:"commit"`
			} `json:"search" yaml:"search" toml:"search"`
			ClientID     string `json:"client_id" yaml:"client_id" toml:"client_id"`
			ClientSecret string `json:"client_secret" yaml:"client_secret" toml:"client_secret"`
			PerPage      int    `default:"20" json:"per_page" yaml:"per_page" toml:"per_page"`
			Offset       int    `default:"1" json:"offset" yaml:"offset" toml:"offset"`
			MaxPage      int    `default:"-1" json:"max_page" yaml:"max_page" toml:"max_page"`
		} `json:"github" yaml:"github" toml:"github"`
	} `json:"service" yaml:"service" toml:"service"`

	Forward struct {
		Beat BeatConfig
	}

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

	Logger struct {
		Disabled      bool                   `default:"false" json:"disabled" toml:"disabled" yaml:"disabled"`
		Backend       string                 `default:"logrus" json:"backend" toml:"backend" yaml:"backend"`
		Level         string                 `default:"info" json:"level" toml:"level" yaml:"level"`
		Encoding      string                 `default:"json" json:"encoding" toml:"encoding" yaml:"encoding"`
		DisableCaller bool                   `default:"false" json:"disable_caller" toml:"disable_caller" yaml:"disable_caller"`
		OutputFile    string                 `json:"output_file" yaml:"output_file" toml:"output_file"`
		InitialFields map[string]interface{} `json:"fields" yaml:"fields" toml:"fields"`
	} `json:"logger" yaml:"logger" toml:"logger"`

	Stats struct {
		Engine struct {
			Disabled  bool   `default:"false" json:"disabled" yaml:"disabled" toml:"disabled"`            // disable stats forwarding to stats engine
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

func loadConfig() {
	configor.Load(&config, "shared/config/config.yaml")
}
