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
	errTypeNotFunction          = Error("argument type not function")
	errInArgsMissMatch          = Error("input arguments count not match")
	errOutCntMissMatch          = Error("output parameter count not match")
	errExecuteTimeout           = Error("parallel execute timeout")
	errIndexLowLink             = "w's index and lowlink differ, how!?"
)

var (
	errNoTasks = errgo.New("No tasks")
	mask       = errgo.MaskFunc(isErrNoTasks)
)

type Error string

func (e Error) Error() string { return string(e) }

func isErrNoTasks(err error) bool {
	return errgo.Cause(err) == errNoTasks
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
