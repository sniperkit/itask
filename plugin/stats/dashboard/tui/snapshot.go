package tui

import (
	"time"
)

type Snapshot struct {

	// time
	timestamp           time.Time
	timeSinceStart      time.Duration
	averageResponseTime time.Duration

	// counters
	numberOfWorkers              int
	totalNumberOfRequests        int
	numberOfSuccessfulRequests   int
	numberOfUnsuccessfulRequests int
	numberOfRequestsPerSecond    float64

	numberOfRequestsByStatusCode  map[int]int
	numberOfRequestsByContentType map[string]int

	// size
	totalSizeInBytes   int
	averageSizeInBytes int
}

func (s Snapshot) Timestamp() time.Time {
	return s.timestamp
}

func (s Snapshot) NumberOfWorkers() int {
	return s.numberOfWorkers
}

func (s Snapshot) NumberOfErrors() int {
	return s.numberOfUnsuccessfulRequests
}

func (s Snapshot) TotalNumberOfRequests() int {
	return s.totalNumberOfRequests
}

func (s Snapshot) TotalSizeInBytes() int {
	return s.totalSizeInBytes
}

func (s Snapshot) AverageSizeInBytes() int {
	return s.averageSizeInBytes
}

func (s Snapshot) AverageResponseTime() time.Duration {
	return s.averageResponseTime
}

func (s Snapshot) RequestsPerSecond() float64 {
	return s.numberOfRequestsPerSecond
}
