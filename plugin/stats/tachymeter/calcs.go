package tachymeter

import (
	"fmt"
	"math"
	"sort"
	"sync/atomic"
	"time"
)

// Calc summarizes Tachymeter sample data and returns it in the form of a *Metrics.
func (m *Tachymeter) Calc() *Metrics {
	metrics := &Metrics{}
	if atomic.LoadUint64(&m.Count) == 0 {
		return metrics
	}

	m.Lock()

	metrics.Samples = int(math.Min(float64(atomic.LoadUint64(&m.Count)), float64(m.Size)))
	metrics.Count = int(atomic.LoadUint64(&m.Count))
	metrics.Wall = m.WallTime

	ranks := make(timeRank, metrics.Samples)
	copy(ranks, m.Ranks[:metrics.Samples])

	// GO 1.8 or above:
	// sort.Slice(ranks)
	// sort.Sort(ranks)

	sort.Slice(ranks, func(i, j int) bool { return int64(ranks[i].duration) < int64(ranks[j].duration) })

	metrics.Rank.Cumulative = ranks.cumulative()
	var rateTime float64
	if m.WallTime != 0 {
		rateTime = float64(metrics.Count) / float64(m.WallTime)
	} else {
		rateTime = float64(metrics.Samples) / float64(metrics.Time.Cumulative)
	}

	metrics.Rate.Second = rateTime * 1e9

	m.Unlock()

	metrics.Rank.Avg = ranks.avg()
	metrics.Rank.HMean = ranks.hMean()
	metrics.Rank.P50 = ranks.p50() // ranks.p(0.50) //[ranks.Len()/2]
	metrics.Rank.P75 = ranks.p(0.75)
	metrics.Rank.P95 = ranks.p(0.95)
	metrics.Rank.P99 = ranks.p(0.99)
	metrics.Rank.P999 = ranks.p(0.999)
	metrics.Rank.Long5p = ranks.long5p()
	metrics.Rank.Short5p = ranks.short5p()
	metrics.Rank.Max = ranks.maxStr()
	metrics.Rank.Min = ranks.minStr()
	metrics.Rank.Range = ranks.srange()
	metrics.Histogram, metrics.HistogramBucketSize = ranks.hgram(m.HBuckets)

	return metrics
}

// These should be self-explanatory:
func (tr timeRank) hMean() time.Duration {
	var total float64
	for _, t := range tr {
		total += (1 / float64(t.duration))
	}
	return time.Duration(float64(tr.Len()) / total)
}

func (tr timeRank) cumulative() time.Duration {
	var total time.Duration
	for _, t := range tr {
		total += t.duration
	}
	return total
}

func (tr timeRank) avg() time.Duration {
	var total time.Duration
	for _, t := range tr {
		total += t.duration
	}
	return time.Duration(int(total) / tr.Len())
}

func (tr timeRank) p(p float64) time.Duration {
	return tr[int(float64(tr.Len())*p+0.5)-1].duration
}

func (tr timeRank) long5p() time.Duration {
	set := tr[int(float64(tr.Len())*0.95+0.5):]
	if len(set) <= 1 {
		return tr[tr.Len()-1].duration
	}
	var t time.Duration
	var i int
	for _, n := range set {
		t += n.duration
		i++
	}

	return time.Duration(int(t) / i)
}

func (tr timeRank) short5p() time.Duration {
	set := tr[:int(float64(tr.Len())*0.05+0.5)]
	if len(set) <= 1 {
		return tr[0].duration
	}
	var t time.Duration
	var i int
	for _, n := range set {
		t += n.duration
		i++
	}
	return time.Duration(int(t) / i)
}

func (tr timeRank) srange() time.Duration {
	return tr.max() - tr.min()
}

func (tr timeRank) p50() time.Duration {
	k := tr.Len() / 2
	return tr[k].duration
}

func (tr timeRank) min() time.Duration {
	return tr[0].duration
}

func (tr timeRank) minStr() string {
	return fmt.Sprintf("label=%s, duration=%s, len=%d", tr[0].label, tr[0].duration, len(tr))
}

func (tr timeRank) max() time.Duration {
	k := tr.Len() - 1
	return tr[k].duration
}

func (tr timeRank) maxStr() string {
	k := tr.Len() - 1
	return fmt.Sprintf("label=%s, duration=%s, len=%d", tr[k].label, tr[k].duration, len(tr))
}

// hgram returns a histogram of event durations in b buckets, along with the bucket size.
func (tr timeRank) hgram(b int) (*Histogram, time.Duration) {
	res := time.Duration(1000)
	// Interval is the time range / n buckets.
	interval := time.Duration(int64(tr.srange()) / int64(b))
	high := tr.min() + interval
	low := tr.min()
	max := tr.max()
	hgram := &Histogram{}
	pos := 1 // Bucket position.

	bstring := fmt.Sprintf("%s - %s", low/res*res, high/res*res)
	bucket := map[string]uint64{}

	for _, v := range tr {
		// If v fits in the current bucket,
		// increment the bucket count.
		if v.duration <= high {
			bucket[bstring]++
		} else {
			// If not, prepare the next bucket.
			*hgram = append(*hgram, bucket)
			bucket = map[string]uint64{}

			// Update the high/low range values.
			low = high + time.Nanosecond

			high += interval
			// if we're going into the
			// last bucket, set high to max.
			if pos == b-1 {
				high = max
			}

			bstring = fmt.Sprintf("%s - %s", low/res*res, high/res*res)

			// The value didn't fit in the previous
			// bucket, so the new bucket count should
			// be incremented.
			bucket[bstring]++

			pos++
		}
	}

	*hgram = append(*hgram, bucket)

	return hgram, interval
}

// hgram returns a histogram of event durations in b buckets, along with the bucket size.
func (ts timeSlice) hgram(b int) (*Histogram, time.Duration) {
	res := time.Duration(1000)
	// Interval is the time range / n buckets.
	interval := time.Duration(int64(ts.srange()) / int64(b))
	high := ts.min() + interval
	low := ts.min()
	max := ts.max()
	hgram := &Histogram{}
	pos := 1 // Bucket position.

	bstring := fmt.Sprintf("%s - %s", low/res*res, high/res*res)
	bucket := map[string]uint64{}

	for _, v := range ts {
		// If v fits in the current bucket,
		// increment the bucket count.
		if v <= high {
			bucket[bstring]++
		} else {
			// If not, prepare the next bucket.
			*hgram = append(*hgram, bucket)
			bucket = map[string]uint64{}

			// Update the high/low range values.
			low = high + time.Nanosecond

			high += interval
			// if we're going into the
			// last bucket, set high to max.
			if pos == b-1 {
				high = max
			}

			bstring = fmt.Sprintf("%s - %s", low/res*res, high/res*res)

			// The value didn't fit in the previous
			// bucket, so the new bucket count should
			// be incremented.
			bucket[bstring]++

			pos++
		}
	}

	*hgram = append(*hgram, bucket)

	return hgram, interval
}

// These should be self-explanatory:
func (ts timeSlice) hMean() time.Duration {
	var total float64

	for _, t := range ts {
		total += (1 / float64(t))
	}

	return time.Duration(float64(ts.Len()) / total)
}

func (ts timeSlice) cumulative() time.Duration {
	var total time.Duration
	for _, t := range ts {
		total += t
	}

	return total
}

func (ts timeSlice) avg() time.Duration {
	var total time.Duration
	for _, t := range ts {
		total += t
	}
	return time.Duration(int(total) / ts.Len())
}

func (ts timeSlice) p(p float64) time.Duration {
	return ts[int(float64(ts.Len())*p+0.5)-1]
}

func (ts timeSlice) long5p() time.Duration {
	set := ts[int(float64(ts.Len())*0.95+0.5):]

	if len(set) <= 1 {
		return ts[ts.Len()-1]
	}

	var t time.Duration
	var i int
	for _, n := range set {
		t += n
		i++
	}

	return time.Duration(int(t) / i)
}

func (ts timeSlice) short5p() time.Duration {
	set := ts[:int(float64(ts.Len())*0.05+0.5)]

	if len(set) <= 1 {
		return ts[0]
	}

	var t time.Duration
	var i int
	for _, n := range set {
		t += n
		i++
	}

	return time.Duration(int(t) / i)
}

func (ts timeSlice) srange() time.Duration {
	return ts.max() - ts.min()
}

func (ts timeSlice) min() time.Duration {
	return ts[0]
}

func (ts timeSlice) max() time.Duration {
	return ts[ts.Len()-1]
}
