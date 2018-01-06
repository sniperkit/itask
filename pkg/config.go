package xtask

import (
	"reflect"
)

const (
	DEFAULT_WORKER_COUNT         = 15
	DEFAULT_WORKER_TASK_INTERVAL = 1 // in seconds
	DEFAULT_QUEUE_SIZE           = 100000
	DEFAULT_TASK_TIMEOUT         = 120
)

var (
	emptyStr    string
	emptyError  error
	emptyResult []interface{}
	emptyArgs   []reflect.Value
)
