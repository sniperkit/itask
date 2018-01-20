package github

import (
	"time"
	// "github.com/google/go-github/github"
)

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

// Fetch fetch a valid client.
func (cm *ClientManager) Fetch() *Github {
	defer funcTrack(time.Now())

	return <-cm.Dispatch
}

// Reclaim reclaim client while the client is valid.
// resp: The response returned when calling the client.
/*
func Reclaim2(g *Github, resetAt time.Time) {
	defer funcTrack(time.Now())

	g.initTimer(resetAt)

	select {
	case <-g.timer.C:
		g.manager.reclaim <- g
	}
}
*/

func Reclaim(g *Github, resetAt time.Time) {
	defer funcTrack(time.Now())

	g.startTimer(resetAt)

	select {
	case <-g.timer.C:
		g.manager.reclaim <- g
	}
}

// Reclaim reclaim client while the client is valid.
// resp: The response returned when calling the client.
func (g *Github) Reclaim(resetAt time.Time) { //resp *github.Response) {
	defer funcTrack(time.Now())

	g.startTimer(resetAt)

	select {
	case <-g.timer.C:
		g.manager.reclaim <- g
	}
}

// Shutdown shutdown the client manager.
func (cm *ClientManager) Shutdown() {
	defer funcTrack(time.Now())

	close(cm.shutdown)
}
