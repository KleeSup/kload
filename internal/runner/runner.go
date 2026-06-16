package runner

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"kload/internal/config"
	"kload/internal/metrics"
)

// OnResult is called after each request completes (then used for live verbose output)
type OnResult func(r metrics.Result, done, total int)

// Run executes the load test and returns all results
func Run(cfg *config.Config, body []byte, onResult OnResult) ([]metrics.Result, time.Duration) {
	transport := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: cfg.Insecure},
		DisableKeepAlives:   !cfg.KeepAlive,
		ForceAttemptHTTP2:   cfg.HTTP2,
		MaxIdleConnsPerHost: cfg.Concurrency + 10,
	}

	client := &http.Client{
		Timeout:   cfg.Timeout,
		Transport: transport,
	}

	// Disable redirect following if requested
	if cfg.NoRedirect {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	// Determine total jobs
	durationMode := cfg.Duration > 0
	totalJobs := cfg.Requests + cfg.Warmup

	// Rate limiter (token bucket via ticker)
	var rateTicker <-chan time.Time
	if cfg.RPS > 0 {
		interval := time.Second / time.Duration(cfg.RPS)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		rateTicker = ticker.C
	}

	// Result collection
	resultsCh := make(chan metrics.Result, totalJobs+1024)
	var results []metrics.Result

	// Job channel
	jobsCh := make(chan bool, cfg.Concurrency*2) // bool = isWarmup

	var wg sync.WaitGroup
	var done int64

	//  Launch workers
	for i := 0; i < cfg.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for isWarmup := range jobsCh {
				// Rate limit: wait for token if RPS cap set
				if rateTicker != nil {
					<-rateTicker
				}

				result := doRequest(client, cfg, body, isWarmup)
				resultsCh <- result

				n := int(atomic.AddInt64(&done, 1))
				if onResult != nil {
					total := totalJobs
					if durationMode {
						total = 0 // unknown in duration mode
					}
					onResult(result, n, total)
				}
			}
		}()
	}

	// Feed jobs
	start := time.Now()

	go func() {
		defer close(jobsCh)

		if durationMode {
			// Duration mode: send jobs until time runs out
			ctx, cancel := context.WithTimeout(context.Background(), cfg.Duration)
			defer cancel()

			// Warmup first
			for i := 0; i < cfg.Warmup; i++ {
				select {
				case <-ctx.Done():
					return
				case jobsCh <- true:
				}
			}

			// Then real requests
			for {
				select {
				case <-ctx.Done():
					return
				case jobsCh <- false:
				}
			}
		} else {
			// Count mode: send warmup + n requests
			for i := 0; i < cfg.Warmup; i++ {
				jobsCh <- true
			}
			for i := 0; i < cfg.Requests; i++ {
				jobsCh <- false
			}
		}
	}()

	// Wait for workers then close results
	wg.Wait()
	wallTime := time.Since(start)
	close(resultsCh)

	for r := range resultsCh {
		results = append(results, r)
	}

	return results, wallTime
}

// doRequest performs a single HTTP request and returns its result
func doRequest(client *http.Client, cfg *config.Config, body []byte, isWarmup bool) metrics.Result {
	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = strings.NewReader(string(body))
	}

	req, err := http.NewRequest(cfg.Method, cfg.URL, bodyReader)
	if err != nil {
		return metrics.Result{Err: fmt.Errorf("failed to build request: %w", err), Warmup: isWarmup}
	}

	// Apply headers
	for _, h := range cfg.Headers {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		return metrics.Result{Duration: elapsed, Err: err, Warmup: isWarmup}
	}
	defer resp.Body.Close()
	// Drain body so connection can be reused
	io.Copy(io.Discard, resp.Body)

	return metrics.Result{
		Duration:   elapsed,
		StatusCode: resp.StatusCode,
		Warmup:     isWarmup,
	}
}
