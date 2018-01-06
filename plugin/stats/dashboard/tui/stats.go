package tui

import (
	"fmt"
	"sync"
	"time"
)

func New() *Statistics {
	stats = Statistics{
		lock: sync.RWMutex{},
		numberOfRequestsByStatusCode:  make(map[int]int),
		numberOfRequestsByContentType: make(map[string]int),
	}
	return *stats
}

type Statistics struct {
	lock sync.RWMutex

	// rawResults  []WorkResult
	snapShots   []Snapshot
	logMessages []string

	startTime time.Time
	endTime   time.Time

	totalResponseTime time.Duration

	numberOfWorkers               int
	numberOfRequests              int
	numberOfSuccessfulRequests    int
	numberOfUnsuccessfulRequests  int
	numberOfRequestsByStatusCode  map[int]int
	numberOfRequestsByContentType map[string]int

	totalSizeInBytes int
}

func (s *Statistics) Add(workResult WorkResult) Snapshot {

	// update the raw results
	s.lock.Lock()
	defer s.lock.Unlock()
	s.rawResults = append(s.rawResults, workResult)

	// initialize start and end time
	if s.numberOfRequests == 0 {
		s.startTime = workResult.StartTime()
		s.endTime = workResult.EndTime()
	}

	// start time
	if workResult.StartTime().Before(s.startTime) {
		s.startTime = workResult.StartTime()
	}

	// end time
	if workResult.EndTime().After(s.endTime) {
		s.endTime = workResult.EndTime()
	}

	// update the total number of requests
	s.numberOfRequests = len(s.rawResults)

	// is successful
	if workResult.StatusCode() > 199 && workResult.StatusCode() < 400 {
		s.numberOfSuccessfulRequests += 1
	} else {
		s.numberOfUnsuccessfulRequests += 1
	}

	// number of workers
	s.numberOfWorkers = workResult.NumberOfWorkers()

	// number of requests by status code
	s.numberOfRequestsByStatusCode[workResult.StatusCode()] += 1

	// number of requests by content type
	s.numberOfRequestsByContentType[workResult.ContentType()] += 1

	// update the total duration
	responseTime := workResult.EndTime().Sub(workResult.StartTime())
	s.totalResponseTime += responseTime

	// size
	s.totalSizeInBytes += workResult.Size()
	averageSizeInBytes := s.totalSizeInBytes / s.numberOfRequests

	// average response time
	averageResponseTime := time.Duration(s.totalResponseTime.Nanoseconds() / int64(s.numberOfRequests))

	// number of requests per second
	requestsPerSecond := float64(s.numberOfRequests) / s.endTime.Sub(s.startTime).Seconds()

	// log messages
	s.logMessages = append(s.logMessages, workResult.String())

	// create a snapshot
	snapShot := Snapshot{
		// times
		// timestamp:           workResult.EndTime(),
		averageResponseTime: averageResponseTime,

		// counters
		numberOfWorkers:               s.numberOfWorkers,
		totalNumberOfRequests:         s.numberOfRequests,
		numberOfSuccessfulRequests:    s.numberOfSuccessfulRequests,
		numberOfUnsuccessfulRequests:  s.numberOfUnsuccessfulRequests,
		numberOfRequestsPerSecond:     requestsPerSecond,
		numberOfRequestsByStatusCode:  s.numberOfRequestsByStatusCode,
		numberOfRequestsByContentType: s.numberOfRequestsByContentType,

		// size
		totalSizeInBytes:   s.totalSizeInBytes,
		averageSizeInBytes: averageSizeInBytes,
	}

	s.snapShots = append(s.snapShots, snapShot)

	return snapShot
}

func (s *Statistics) LastSnapshot() Snapshot {
	s.lock.RLock()
	defer s.lock.RUnlock()

	lastSnapshotIndex := len(s.snapShots) - 1
	if lastSnapshotIndex < 0 {
		return Snapshot{}
	}

	return s.snapShots[lastSnapshotIndex]
}

func (s *Statistics) LastLogMessages(count int) []string {
	s.lock.RLock()
	defer s.lock.RUnlock()

	messages, err := getLatestLogMessages(s.logMessages, count)
	if err != nil {
		panic(err)
	}

	return messages
}

func getLatestLogMessages(messages []string, count int) ([]string, error) {
	if count < 0 {
		return nil, fmt.Errorf("The count cannot be negative")
	}

	numberOfMessges := len(messages)

	if count == numberOfMessges {
		return messages, nil
	}

	if count < numberOfMessges {
		return messages[numberOfMessges-count:], nil
	}

	if count > numberOfMessges {
		fillLines := make([]string, count-numberOfMessges)
		return append(fillLines, messages...), nil
	}

	panic("Unreachable")
}
