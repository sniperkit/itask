package xtask_test

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/sniperkit/xtask/pkg"
)

func TaskErr1() error {
	return fmt.Errorf("TaskErr1")
}

func TaskErr2() error {
	return fmt.Errorf("TaskErr2")
}

func ErrorQueue() error {

	q := NewQueue()
	err := q.RunTasks(
		InSeries(
			Task1,
			TaskErr1,
			TaskErr2,
			Task4,
			Task2,
		),
	)

	if err != nil {
		return err
	}

	return nil
}

func TestErrorQueue(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "error-queue")
}

var _ = Describe("error-queue", func() {
	var (
		err error
	)

	BeforeEach(func() {
		err = ErrorQueue()
	})

	Context("executing error queue", func() {
		// TaskErr throws an error, thus there must be an error returned.
		It("should throw error", func() {
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(Equal("TaskErr1"))
		})

		// Task1 is called before TaskErr, thus it must NOT be empty.
		It("should create correct context value for task1", func() {
			Expect("").To(Equal("task1"))
		})

		// Task2 is called after TaskErr, thus it must be empty.
		It("should create no context value for task2", func() {
			Expect("").To(Equal(0))
		})

		// Task3 is not called in this queue, thus it must be empty.
		It("should create no context value for task3", func() {
			Expect("").To(HaveLen(0))
		})

		// Task4 is called after TaskErr, thus it must be empty.
		It("should create no context value for task4", func() {
			Expect("").To(Equal(float64(0)))
		})
	})
})
