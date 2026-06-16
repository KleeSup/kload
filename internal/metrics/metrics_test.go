package metrics

import (
	"errors"
	"testing"
	"time"
)

func ms(n int) time.Duration {
	return time.Duration(n) * time.Millisecond
}

// helper to build a successful result
func ok(d time.Duration) Result {
	return Result{Duration: d, StatusCode: 200}
}

func TestCompute_EmptyResults(t *testing.T) {
	s := Compute(nil, time.Second)
	if s.Total != 0 {
		t.Errorf("expected Total 0, got %d", s.Total)
	}
	if s.Successful != 0 || s.Failed != 0 {
		t.Errorf("expected no successes or failures, got %d/%d", s.Successful, s.Failed)
	}
}

func TestCompute_AllSuccessful(t *testing.T) {
	results := []Result{ok(ms(10)), ok(ms(20)), ok(ms(30)), ok(ms(40))}
	s := Compute(results, time.Second)

	if s.Total != 4 {
		t.Errorf("expected Total 4, got %d", s.Total)
	}
	if s.Successful != 4 {
		t.Errorf("expected 4 successful, got %d", s.Successful)
	}
	if s.Failed != 0 {
		t.Errorf("expected 0 failed, got %d", s.Failed)
	}
	if s.Min != ms(10) {
		t.Errorf("expected Min 10ms, got %v", s.Min)
	}
	if s.Max != ms(40) {
		t.Errorf("expected Max 40ms, got %v", s.Max)
	}
	if s.Mean != ms(25) {
		t.Errorf("expected Mean 25ms, got %v", s.Mean)
	}
}

func TestCompute_WarmupExcluded(t *testing.T) {
	results := []Result{
		{Duration: ms(5), StatusCode: 200, Warmup: true}, // should be ignored
		{Duration: ms(5), StatusCode: 200, Warmup: true}, // should be ignored
		ok(ms(10)),
		ok(ms(20)),
	}
	s := Compute(results, time.Second)

	if s.Total != 2 {
		t.Errorf("expected Total 2 (warmup excluded), got %d", s.Total)
	}
	if s.Min != ms(10) {
		t.Errorf("warmup leaked into Min: got %v", s.Min)
	}
}

func TestCompute_FailuresCounted(t *testing.T) {
	results := []Result{
		ok(ms(10)),
		{Duration: ms(5), Err: errors.New("connection refused")}, // network error
		{Duration: ms(8), StatusCode: 500},                       // server error
		{Duration: ms(9), StatusCode: 503},                       // server error
	}
	s := Compute(results, time.Second)

	if s.Total != 4 {
		t.Errorf("expected Total 4, got %d", s.Total)
	}
	if s.Successful != 1 {
		t.Errorf("expected 1 successful, got %d", s.Successful)
	}
	if s.Failed != 3 {
		t.Errorf("expected 3 failed, got %d", s.Failed)
	}
	if s.Errors != 1 {
		t.Errorf("expected 1 network error, got %d", s.Errors)
	}
}

func TestCompute_4xxCountsAsSuccess(t *testing.T) {
	// 4xx is a valid response (client error), not a server/network failure
	results := []Result{
		ok(ms(10)),
		{Duration: ms(12), StatusCode: 404},
		{Duration: ms(14), StatusCode: 429},
	}
	s := Compute(results, time.Second)

	if s.Failed != 0 {
		t.Errorf("4xx should not count as failure, got %d failed", s.Failed)
	}
	if s.Successful != 3 {
		t.Errorf("expected 3 successful (4xx included), got %d", s.Successful)
	}
}

func TestCompute_Percentiles(t *testing.T) {
	// 100 requests: 1ms, 2ms, ... 100ms
	var results []Result
	for i := 1; i <= 100; i++ {
		results = append(results, ok(ms(i)))
	}
	s := Compute(results, time.Second)

	// percentile uses index = (len-1) * p/100
	// p50 → index 49 → 50ms ; p90 → index 89 → 90ms ; p99 → index 98 → 99ms
	tests := []struct {
		name string
		got  time.Duration
		want time.Duration
	}{
		{"p50", s.P50, ms(50)},
		{"p90", s.P90, ms(90)},
		{"p95", s.P95, ms(95)},
		{"p99", s.P99, ms(99)},
	}
	for _, tc := range tests {
		if tc.got != tc.want {
			t.Errorf("%s: expected %v, got %v", tc.name, tc.want, tc.got)
		}
	}
}

func TestCompute_StatusCodeBreakdown(t *testing.T) {
	results := []Result{
		ok(ms(10)),
		ok(ms(11)),
		{Duration: ms(12), StatusCode: 404},
		{Duration: ms(13), StatusCode: 500},
	}
	s := Compute(results, time.Second)

	if s.StatusCodes[200] != 2 {
		t.Errorf("expected two 200s, got %d", s.StatusCodes[200])
	}
	if s.StatusCodes[404] != 1 {
		t.Errorf("expected one 404, got %d", s.StatusCodes[404])
	}
	if s.StatusCodes[500] != 1 {
		t.Errorf("expected one 500, got %d", s.StatusCodes[500])
	}
}

func TestCompute_ReqPerSec(t *testing.T) {
	results := []Result{ok(ms(10)), ok(ms(10)), ok(ms(10)), ok(ms(10))}
	// 4 requests in 2 seconds → 2 req/s
	s := Compute(results, 2*time.Second)

	if s.ReqPerSec != 2.0 {
		t.Errorf("expected 2.0 req/s, got %.2f", s.ReqPerSec)
	}
}

func TestCompute_SingleResult(t *testing.T) {
	s := Compute([]Result{ok(ms(42))}, time.Second)

	if s.Min != ms(42) || s.Max != ms(42) || s.Mean != ms(42) {
		t.Errorf("single result: min/max/mean should all be 42ms, got %v/%v/%v", s.Min, s.Max, s.Mean)
	}
	if s.P50 != ms(42) || s.P99 != ms(42) {
		t.Errorf("single result: percentiles should be 42ms, got p50=%v p99=%v", s.P50, s.P99)
	}
}
