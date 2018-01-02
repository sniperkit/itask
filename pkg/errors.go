package xtask

import (
	"github.com/juju/errgo"
)

type Error string

func (e Error) Error() string { return string(e) }

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

func IsErrNoTasks(err error) bool {
	return errgo.Cause(err) == ErrNoTasks
}
