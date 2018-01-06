package xtask

import (
	"bytes"
	"strings"

	"github.com/juju/errgo"
)

const (
	errIsNameEmpty              = Error("name is empty")
	errStackIsEmpty             = Error("Stack is empty")
	errCyclicDependencyDetected = Error("task must not add itself as a dependency")
	errIsAlreadyRunning         = Error("tasker: already run")
	errIndexLowLink             = "w's index and lowlink differ, how!?"
)

var (
	ErrNoTasks = errgo.New("No tasks")
	Mask       = errgo.MaskFunc(IsErrNoTasks)
)

type Error string

func (e Error) Error() string { return string(e) }

func IsErrNoTasks(err error) bool {
	return errgo.Cause(err) == ErrNoTasks
}

// Errors is a type of []error
// This is used to pass multiple errors when using parallel or concurrent methods and yet subscribe to the error interface
type Errors []error

// Prints all errors from asynchronous tasks separated by space
func (e Errors) Error() string {
	b := bytes.NewBufferString(emptyStr)
	for _, err := range e {
		b.WriteString(err.Error())
		b.WriteString(" ")
	}
	return strings.TrimSpace(b.String())
}
