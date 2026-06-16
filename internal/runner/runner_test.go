package runner

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"kload/internal/config"
)

// newTestServer returns a local HTTP server that counts requests
func newTestServer(handler http.HandlerFunc) (*httptest.Server, *int64) {
	var count int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&count, 1)
		handler(w, r)
	}))
	return srv, &count
}

func TestRun_CountMode(t *testing.T) {
	srv, count := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	cfg := &config.Config{
		URL:         srv.URL,
		Method:      "GET",
		Requests:    50,
		Concurrency: 5,
		Timeout:     5 * time.Second,
		KeepAlive:   true,
	}

	results, wall := Run(cfg, nil, nil)

	if len(results) != 50 {
		t.Errorf("expected 50 results, got %d", len(results))
	}
	if atomic.LoadInt64(count) != 50 {
		t.Errorf("server received %d requests, expected 50", *count)
	}
	if wall <= 0 {
		t.Errorf("wall time should be positive, got %v", wall)
	}
	for _, r := range results {
		if r.StatusCode != 200 {
			t.Errorf("expected all 200s, got %d", r.StatusCode)
		}
	}
}

func TestRun_WarmupExcludedFromCount(t *testing.T) {
	srv, count := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	cfg := &config.Config{
		URL:         srv.URL,
		Method:      "GET",
		Requests:    20,
		Warmup:      10,
		Concurrency: 4,
		Timeout:     5 * time.Second,
		KeepAlive:   true,
	}

	results, _ := Run(cfg, nil, nil)

	// Server should see warmup + real = 30 requests
	if atomic.LoadInt64(count) != 30 {
		t.Errorf("server should see 30 requests (20 + 10 warmup), got %d", *count)
	}

	// But only 20 should be non-warmup
	nonWarmup := 0
	for _, r := range results {
		if !r.Warmup {
			nonWarmup++
		}
	}
	if nonWarmup != 20 {
		t.Errorf("expected 20 non-warmup results, got %d", nonWarmup)
	}
}

func TestRun_HandlesServerErrors(t *testing.T) {
	srv, _ := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer srv.Close()

	cfg := &config.Config{
		URL:         srv.URL,
		Method:      "GET",
		Requests:    10,
		Concurrency: 2,
		Timeout:     5 * time.Second,
		KeepAlive:   true,
	}

	results, _ := Run(cfg, nil, nil)

	if len(results) != 10 {
		t.Errorf("expected 10 results, got %d", len(results))
	}
	for _, r := range results {
		if r.StatusCode != 500 {
			t.Errorf("expected 500, got %d", r.StatusCode)
		}
	}
}

func TestRun_RateLimit(t *testing.T) {
	srv, _ := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	defer srv.Close()

	cfg := &config.Config{
		URL:         srv.URL,
		Method:      "GET",
		Requests:    20,
		Concurrency: 10,
		RPS:         20, // cap at 20 req/s → 20 requests should take ~1s
		Timeout:     5 * time.Second,
		KeepAlive:   true,
	}

	_, wall := Run(cfg, nil, nil)

	// With a 20 rps cap, 20 requests should take at least ~0.9s
	if wall < 800*time.Millisecond {
		t.Errorf("rate limiting too fast: 20 reqs at 20rps took only %v", wall)
	}
}

func TestRun_POSTWithBody(t *testing.T) {
	var gotBody string
	srv, _ := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		r.Body.Read(buf)
		gotBody = string(buf)
		w.WriteHeader(http.StatusCreated)
	})
	defer srv.Close()

	cfg := &config.Config{
		URL:         srv.URL,
		Method:      "POST",
		Requests:    1,
		Concurrency: 1,
		Timeout:     5 * time.Second,
		KeepAlive:   true,
	}

	body := []byte(`{"hello":"world"}`)
	results, _ := Run(cfg, body, nil)

	if results[0].StatusCode != 201 {
		t.Errorf("expected 201, got %d", results[0].StatusCode)
	}
	if gotBody != `{"hello":"world"}` {
		t.Errorf("server got wrong body: %q", gotBody)
	}
}
