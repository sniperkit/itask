package xtask_test

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/sniperkit/xtask/pkg"
	"github.com/sniperkit/xtask/plugin/tachymeter"
)

type Thing struct {
	value int
}

const (
	items   = 1024
	readers = 128
	reads   = 50000
)

func TestNewCollider(t *testing.T) {
	fmt.Println("==================================TestNewCollider=================================")

	c, err := Collider(items)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for i := 0; i < items; i++ {
		c.Add(&Thing{value: 0})
	}

	tm := tachymeter.New(&tachymeter.Config{Size: readers * reads})
	wg := sync.WaitGroup{}
	wg.Add(readers)

	start := time.Now()
	for i := 0; i < readers; i++ {
		go reader(c, tm, &wg)
	}
	tm.SetWallTime(time.Since(start))

	wg.Wait()
	tm.Calc().Dump()
}

func reader(c *Ring, tm *tachymeter.Tachymeter, wg *sync.WaitGroup) {
	for i := 0; i < reads; i++ {
		start := time.Now()
		_ = c.Get().(*Thing).value
		tm.AddTime(time.Since(start))
	}

	wg.Done()
}
