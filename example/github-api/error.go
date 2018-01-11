package main

import (
	"errors"
)

var (
	errComponent         error
	errGithubClient      = errors.New("error occured while trying to connect to github API clinet.")
	errNotFoundLatestSHA = errors.New("error occured while reading repo latest SHA.")
	errRepoInfo          = errors.New("error occured while reading repository info.")
	errUserInfo          = errors.New("error occured while reading user info.")
)
