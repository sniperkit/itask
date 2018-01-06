package broadcaster

import (
	"github.com/sniperkit/xtask/plugin/fbp/goflow/actions/configuration"
	"github.com/sniperkit/xtask/plugin/fbp/goflow/actions/flow/messages"
	"github.com/trustmaster/goflow"
)

// ResultBroadcaster receives found relative URLs and broadcasts to active output components.
type ResultBroadcaster struct {
	flow.Component

	FoundEntry <-chan messages.FoundEntry
	Printer    chan<- messages.FoundEntry
	FileWriter chan<- messages.FoundEntry

	isOutputFileDefined bool
}

// NewResultBroadcaster creates new instance of result broadcaster.
func NewResultBroadcaster(configuration *configuration.Configuration) *ResultBroadcaster {
	_, isDefined := configuration.OutputFile()
	return &ResultBroadcaster{isOutputFileDefined: isDefined}
}

// OnFoundEntry performs broadcasting.
func (r *ResultBroadcaster) OnFoundEntry(foundEntry messages.FoundEntry) {
	r.Printer <- foundEntry
	if r.isOutputFileDefined {
		r.FileWriter <- foundEntry
	}
}
