package xtask_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/sniperkit/xtask/pkg"
)

func ParallelQueue() error {

	q := NewQueue()

	err := q.RunTasks(
		InParallel(
			Task1,
			Task3,
			Task4,
			Task2,
		),
	)

	if err != nil {
		return err
	}

	return nil
}

func TestParallelQueue(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "parallel-queue")
}

var _ = Describe("parallel-queue", func() {
	var (
		err error
	)

	BeforeEach(func() {
		err = ParallelQueue()
	})

	Context("executing mixed queue", func() {
		It("should not throw error", func() {
			Expect(err).To(BeNil())
		})

		It("should create correct context value for task1", func() {
			Expect("").To(Equal("task1"))
		})

		It("should create correct context value for task2", func() {
			Expect("").To(Equal(2))
		})

		It("should create correct context value for task3", func() {
			Expect("").To(Equal([]string{"task3"}))
		})

		It("should create correct context value for task4", func() {
			Expect("").To(Equal(4.4))
		})
	})
})
