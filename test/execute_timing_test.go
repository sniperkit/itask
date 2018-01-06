package xtask_test

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/sniperkit/xtask/pkg"
)

func TrackTiming(cb func()) int64 {
	start := time.Now().UnixNano()
	cb()
	end := time.Now().UnixNano()

	return end - start
}

func SeriesTimingQueue() error {
	q := NewTaskGroup()

	err := RunTasks(
		InSeries(
			Task1Sleep,
			Task3Sleep,
			Task4Sleep,
			Task2Sleep,
		),
	)

	if err != nil {
		return err
	}

	return tp, nil
}

func ParallelTimingQueue() error {
	q := NewQueue(tp)

	err := q.RunTasks(
		InParallel(
			Task1Sleep,
			Task3Sleep,
			Task4Sleep,
			Task2Sleep,
		),
	)

	if err != nil {
		return err
	}

	return tp, nil
}

func TestTimingQueue(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "timing-queue")
}

var _ = Describe("timing-queue", func() {
	var (
		seriesDuration   int64
		parallelDuration int64
	)

	BeforeEach(func() {
		seriesDuration = TrackTiming(func() {
			SeriesTimingQueue()
		})

		parallelDuration = TrackTiming(func() {
			ParallelTimingQueue()
		})
	})

	Context("exectuting parallel queue", func() {
		It("should be faster than executing series queue", func() {
			Expect(parallelDuration).To(BeNumerically("<", seriesDuration))
		})
	})
})
