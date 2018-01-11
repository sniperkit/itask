package github

import (
	"sync"
	"time"

	"github.com/google/go-github/github"
)

var (
	tokens        []string       = []string{}
	clientManager *ClientManager = NewManager(tokens)
)

type GHClient struct {
	Client     *github.Client
	Manager    *ClientManager
	rateLimits [categories]Rate
	timer      *time.Timer
	rateMu     sync.Mutex
}

// newClients create a client list based on tokens.
func newClients(tokens []string) []*Github {
	defer funcTrack(time.Now())

	var clients []*Github

	for _, t := range tokens {
		client, err := newClient(t)
		if err != nil {
			continue
		}

		clients = append(clients, client)
	}

	return clients
}

// ClientManager used to manage the valid client.
type ClientManager struct {
	Dispatch chan *Github
	reclaim  chan *Github
	shutdown chan struct{}
}

// start start reclaim and dispatch the client.
func (cm *ClientManager) start() {
	defer funcTrack(time.Now())

	for {
		select {
		case v := <-cm.reclaim:
			cm.Dispatch <- v
		case <-cm.shutdown:
			close(cm.Dispatch)
			close(cm.reclaim)
			return
		}
	}
}

// NewManager create a new client manager based on tokens.
func NewManager(tokens []string) *ClientManager {
	defer funcTrack(time.Now())

	var cm *ClientManager = &ClientManager{
		reclaim:  make(chan *Github),
		Dispatch: make(chan *Github, len(tokens)),
		shutdown: make(chan struct{}),
	}
	clients := newClients(tokens)
	go cm.start()
	go func() {
		for _, c := range clients {
			if !c.isLimited() {
				c.manager = cm
				cm.reclaim <- c
			}
		}
	}()
	return cm
}

// Fetch fetch a valid client.
func (cm *ClientManager) Fetch() *Github {
	defer funcTrack(time.Now())

	return <-cm.Dispatch
}

// Reclaim reclaim client while the client is valid.
// resp: The response returned when calling the client.
func Reclaim(client *Github, resp *github.Response) {
	defer funcTrack(time.Now())

	client.initTimer(resp)

	select {
	case <-client.timer.C:
		client.manager.reclaim <- client
	}
}

// Shutdown shutdown the client manager.
func (cm *ClientManager) Shutdown() {
	defer funcTrack(time.Now())

	close(cm.shutdown)
}
