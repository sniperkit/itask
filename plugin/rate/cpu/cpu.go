// Package cpulimit implements a simple CPU usage instrument that
// communicates whether the CPU is running "cold" or "hot" via two
// distinct channels. It is intended to be used for self-restraining
// resource-hungry programs.
//
// The author is well aware that all modern operating systems provide
// means of providing the task scheduler with priority hints
// (e.g. nice(), SetPriorityClass(). This package is intended to help
// with limiting actual CPU usage, e.g. to work around system
// administrators who are scared by high CPU load even if the load is
// predominantly caused by low-priority processes that do actual work.
package cpu

import (
	"time"

	"github.com/shirou/gopsutil/cpu"
)

type Limiter struct {
	// MaxCPUUsage is the CPU usage threshold, in percent, above which
	// the CPU is considered as "hot". This value can be set while the
	// measurement process is running. Default: 50.0
	MaxCPUUsage float64
	// SwitchPeriod determines how often the Limiter decides whether
	// the CPU is "hot" or "cold". Default: 1 second
	SwitchPeriod time.Duration
	// MeasurePeriod determines how often CPU usage is
	// measured. Default: SwitchPeriod / 4 (if MeasurePeriod >=
	// SwitchPeriod/2)
	MeasurePeriod time.Duration
	// For deciding whether the CPU is "hot" or "cold", a rolling
	// average over MeasureDuration is taken.  Default: 10 *
	// SwitchPeriod (if MeasureDuration <= SwitchPeriod)
	MeasureDuration time.Duration
	// The rolling average and the current CPU usage will be sent to H
	// ("hot") whenever average CPU usage over the Measurements period
	// is equal to or higher than MaxCPUUsage.
	h chan info
	// The rolling average and the current CPU usage will be sent to C
	// ("cold") whenever average CPU usage over the Measurements
	// period is below MaxCPUUsage.
	c chan info

	stop chan interface{}
}

type info struct {
	current, average float64
}

func New(c *Limiter) (*Limiter, error) {

	l := &Limiter{}

	if c.MaxCPUUsage == 0.0 {
		l.MaxCPUUsage = 50.0
	}
	if c.SwitchPeriod == 0 {
		l.SwitchPeriod = 1 * time.Second
	}
	if c.MeasurePeriod == 0 || c.MeasurePeriod >= l.SwitchPeriod/2 {
		l.MeasurePeriod = l.SwitchPeriod / 4
	}
	// A longer MeasureDuration may contribute to dampening flapping
	// between "hot" and "cold" states.
	if c.MeasureDuration <= l.SwitchPeriod {
		l.MeasureDuration = 10 * l.SwitchPeriod
	}

	l.h = make(chan info)
	l.c = make(chan info)
	l.stop = make(chan interface{})

	return l, nil
}

// Start initializes and starts the measurement process using the
// pre-set values.
func (l *Limiter) Start() {
	if l.MaxCPUUsage == 0.0 {
		l.MaxCPUUsage = 50.0
	}
	if l.SwitchPeriod == 0 {
		l.SwitchPeriod = 1 * time.Second
	}
	if l.MeasurePeriod == 0 || l.MeasurePeriod >= l.SwitchPeriod/2 {
		l.MeasurePeriod = l.SwitchPeriod / 4
	}
	// A longer MeasureDuration may contribute to dampening flapping
	// between "hot" and "cold" states.
	if l.MeasureDuration <= l.SwitchPeriod {
		l.MeasureDuration = 10 * l.SwitchPeriod
	}
	if l.h == nil {
		l.h = make(chan info)
	}
	if l.c == nil {
		l.c = make(chan info)
	}
	if l.stop == nil {
		l.stop = make(chan interface{})
	}
	go l.run(
		time.NewTicker(l.MeasurePeriod),
		time.NewTicker(l.SwitchPeriod),
		int(l.MeasureDuration/l.MeasurePeriod),
	)
}

// lifted from gopsutil/cpu
func getAllBusy(t cpu.TimesStat) (float64, float64) {
	busy := t.User + t.System + t.Nice + t.Iowait + t.Irq +
		t.Softirq + t.Steal + t.Guest + t.GuestNice + t.Stolen
	return busy + t.Idle, busy
}

func calculatePercent(ts1, ts0 cpu.TimesStat) float64 {
	all1, busy1 := getAllBusy(ts1)
	all0, busy0 := getAllBusy(ts0)
	return (busy1 - busy0) / (all1 - all0) * 100
}

func (l *Limiter) run(measure, decide *time.Ticker, n int) {
	output := l.c
	measurements := make([]float64, n)
	ts0, err := cpu.Times(false)
	if err != nil {
		panic(err)
	}
	var i int
	var a float64
	for {
		select {
		case <-measure.C:
			ts1, err := cpu.Times(false)
			if err != nil {
				panic(err)
			}
			m := calculatePercent(ts1[0], ts0[0])
			ts0 = ts1
			i = (i + 1) % len(measurements)
			measurements[i] = m
		case <-decide.C:
			if a = average(measurements); a >= l.MaxCPUUsage {
				output = l.h
			} else {
				output = l.c
			}
		case output <- info{measurements[i], a}:
		case <-l.stop:
			measure.Stop()
			decide.Stop()
			close(l.h)
			close(l.c)
			l.h = nil
			l.c = nil
			return
		}
	}
}

func average(data []float64) (rv float64) {
	for _, f := range data {
		rv += f
	}
	rv /= float64(len(data))
	return
}

// Stop halts the measurement process. Both H and C channels are
// closed so that any waiting goroutines are unblocked.
func (l *Limiter) Stop() {
	if l.stop == nil {
		return
	}
	close(l.stop)
	l.stop = nil
}
