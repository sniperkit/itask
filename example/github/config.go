package main

import (
	"github.com/sniperkit/xtask/plugin/aggregate/service"
)

var config Config

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
}
