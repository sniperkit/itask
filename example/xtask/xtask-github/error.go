package main

import (
	"errors"
)

var (
	errComponent            error
	errGithubClientsManager       = errors.New("error occured while trying to create github API clients manager.")
	errGithubClient               = errors.New("error occured while trying to connect to github API client.")
	errNotFoundLatestSHA          = errors.New("error occured while reading repo latest SHA.")
	errRepoInfo                   = errors.New("error occured while reading repository info.")
	errUserInfo                   = errors.New("error occured while reading user info.")
	errNoRowsToExport             = errors.New("error occured while exporting results.")
	ErrParamsType           error = errors.New("Params type error")
	ErrParamsFormat         error = errors.New("Params format error")
	ErrTableNotFound        error = errors.New("Not found table")
	ErrUnSupportedType      error = errors.New("Unsupported type error")
	ErrNotExist             error = errors.New("Not exist error")
	ErrCacheFailed          error = errors.New("Cache failed")
	ErrNeedDeletedCond      error = errors.New("Delete need at least one condition")
	ErrNotImplemented       error = errors.New("Not implemented.")
)
