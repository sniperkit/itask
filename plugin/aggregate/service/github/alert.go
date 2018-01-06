package github

import (
	"log"
	"time"
)

func (g *Github) notifyAttempts(err error, i time.Duration) {
	log.Println(err.Error())
}
