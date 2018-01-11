package github

import (
	"math/rand"
	"time"
)

func isStatus2XX(status int) bool {
	return status > 199 && status < 300
}

func randIntMapKey(m map[int]bool) int {
	defer funcTrack(time.Now())

	i := rand.Intn(len(m))
	for k, v := range m {
		if !v {
			if i == 0 {
				return k
			}
			i--
		}
	}
	return randIntMapKey(m)
}

func random(min, max int) int {
	defer funcTrack(time.Now())

	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}
