package metrics

import (
	"sort"
	"time"
)

// Result holds the outcome of a single HTTP request
type Result struct {
	Duration   time.Duration
	StatusCode int
	Err        error
	Warmup     bool // excluded from final stats
}

// Summary holds aggregated statistics across all results
type Summary struct {
	Total      int
	Successful int
	Failed     int
	Errors     int // network errors (no status code)

	TotalDuration time.Duration

	Min  time.Duration
	Max  time.Duration
	Mean time.Duration
	P50  time.Duration
	P90  time.Duration
	P95  time.Duration
	P99  time.Duration

	ReqPerSec float64

	// Status code breakdown
	StatusCodes map[int]int
}

// Compute builds a Summary from a slice of Results
// Warmup results are excluded automatically
func Compute(results []Result, wallTime time.Duration) Summary {
	var filtered []Result
	codes := make(map[int]int)

	for _, r := range results {
		if r.Warmup {
			continue
		}
		filtered = append(filtered, r)
		if r.Err == nil {
			codes[r.StatusCode]++
		}
	}

	s := Summary{
		Total:         len(filtered),
		TotalDuration: wallTime,
		StatusCodes:   codes,
	}

	if s.Total == 0 {
		return s
	}

	// Separate successful from failed
	var durations []time.Duration
	for _, r := range filtered {
		if r.Err == nil && r.StatusCode < 500 {
			s.Successful++
			durations = append(durations, r.Duration)
		} else if r.Err != nil {
			s.Errors++
			s.Failed++
		} else {
			s.Failed++
		}
	}

	if len(durations) == 0 {
		return s
	}

	// Sort for percentiles
	sort.Slice(durations, func(i, j int) bool {
		return durations[i] < durations[j]
	})

	s.Min = durations[0]
	s.Max = durations[len(durations)-1]
	s.Mean = mean(durations)
	s.P50 = percentile(durations, 50)
	s.P90 = percentile(durations, 90)
	s.P95 = percentile(durations, 95)
	s.P99 = percentile(durations, 99)

	if wallTime.Seconds() > 0 {
		s.ReqPerSec = float64(s.Total) / wallTime.Seconds()
	}

	return s
}

func mean(d []time.Duration) time.Duration {
	if len(d) == 0 {
		return 0
	}
	var total time.Duration
	for _, v := range d {
		total += v
	}
	return total / time.Duration(len(d))
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * p / 100.0)
	return sorted[idx]
}
