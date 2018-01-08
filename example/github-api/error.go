package main

import (
	"errors"
)

var (
	errNotFoundLatestSHA = errors.New("error occured while reading latest SHA.")
	errRepoInfo          = errors.New("error occured while reading repository information.")
)
