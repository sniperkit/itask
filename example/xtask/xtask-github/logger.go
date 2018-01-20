package main

import (
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/sniperkit/xstats/pkg"
	"github.com/sniperkit/xtask/plugin/counter"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

var (
	log                         = logrus.New()
	logTasks     bool           = true
	counters     *counter.Oc    = counter.NewOc()
	counterAsync map[string]int = make(map[string]int)
)

// type logFields logrus.Fields
type Fields logrus.Fields

// WithFields is an alias for logrus.WithFields.
func LogWithFields(f Fields) *logrus.Entry {
	return logrus.WithFields(logrus.Fields(f))
}

type funcMetrics struct {
	calls struct {
		count  int           `metric:"count" type:"counter"`
		failed int           `metric:"failed" type:"counter"`
		time   time.Duration `metric:"time"  type:"histogram"`
	} `metric:"func.calls"`
}

func GetCaller() string {
	_, file, line, _ := runtime.Caller(3)
	return fmt.Sprintf("%s:%d", trimPath(file), line)
}

func timeTrack(startedAt time.Time, topic string) { //, reqInfo map[string]interface{}
	go func() {
		/*
			if req_url, ok := obj["request_url"].(string); ok {
				req_url = strings.Replace(req_url, defaultSvcDomain, "", -1)
				key = fmt.Sprintf("%s/%s", req_url, key)
				log.Debugln("saving task info with key=", key)
			}
		*/

		elapsed := time.Since(startedAt)

		if logTasks {

			completedAt := startedAt.Add(elapsed)
			expiredAt := completedAt.Add(taskTTL)

			task := make(map[string]interface{}, 10)
			task["topic"] = topic
			task["tags"] = []string{}
			task["service"] = "github"
			task["task_duration"] = elapsed.Seconds()
			task["task_creation_datetime"] = startedAt.String()
			task["task_creation_timestamp"] = strconv.FormatInt(startedAt.UTC().Unix(), 10)
			task["task_completed_datetime"] = completedAt.String()
			task["task_completed_timestamp"] = strconv.FormatInt(completedAt.UTC().Unix(), 10)
			task["task_expired_datetime"] = expiredAt
			task["task_expired_timestamp"] = strconv.FormatInt(expiredAt.UTC().Unix(), 10)
			cds.Append("tasks", task)
		}

		log.Printf("main().timeTrack() %s took %s", topic, elapsed)
		// addMetrics(start, 1, err != nil)
	}()
}

// See http://stackoverflow.com/a/7053871/199475
func Function(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func trimPath(path string) string {
	// For details, see https://github.com/uber-go/zap/blob/e15639dab1b6ca5a651fe7ebfd8d682683b7d6a8/zapcore/entry.go#L101
	if idx := strings.LastIndexByte(path, '/'); idx >= 0 {
		if idx := strings.LastIndexByte(path[:idx], '/'); idx >= 0 {
			// Keep everything after the penultimate separator.
			return path[idx+1:]
		}
	}
	return path
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

// return the source filename after the last slash
func chopPath(original string) string {
	i := strings.LastIndex(original, "/")
	if i == -1 {
		return original
	} else {
		return original[i+1:]
	}
}

func funcTrack(start time.Time) {
	return
	function, file, line, _ := runtime.Caller(1)
	go func() {
		elapsed := time.Since(start)
		log.Printf("main().funcTrack() %s took %s", fmt.Sprintf("%s:%s:%d", runtime.FuncForPC(function).Name(), chopPath(file), line), elapsed)
		// addMetrics(start, 1, err != nil)
	}()
}

func counterTrack(name string, incr int) {
	go func() {
		counters.Increment(name, incr)
	}()
}

var FullyQualifiedPath = false

// Err consumes an error, a string, or nil, and produces an error message prefixed with the name of the function that called it (or nil).
func Err(err interface{}) error {
	switch o := err.(type) {
	case string:
		return e(fmt.Errorf("%s", o))
	case error:
		return e(o)
	default:
		return nil
	}
}

// e returns an error, prefixed with the name of the function that triggered it. Originally by StackOverflow user svenwltr:
// http://stackoverflow.com/a/38551362/199475
func e(err error) error {
	pc, _, _, _ := runtime.Caller(2)

	fr := runtime.CallersFrames([]uintptr{pc})
	namer, _ := fr.Next()
	name := namer.Function

	if !FullyQualifiedPath {
		fn := strings.Split(name, "/")
		if len(fn) > 0 {
			return fmt.Errorf("%s: %s", fn[len(fn)-1], err.Error())
		}
	}

	return fmt.Errorf("%s: %s", name, err.Error())
}

func init() {
	log.Formatter = new(prefixed.TextFormatter)
	log.Level = logrus.DebugLevel
}
