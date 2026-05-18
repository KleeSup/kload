package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

// Headers is a custom flag type that allows -H to be repeated
type Headers []string

func (h *Headers) String() string { return strings.Join(*h, ", ") }
func (h *Headers) Set(v string) error {
	*h = append(*h, v)
	return nil
}

// Config holds all parsed CLI options
type Config struct {
	// Target
	URL      string
	Method   string
	Headers  Headers
	Body     string
	BodyFile string

	// Load shape
	Requests    int
	Concurrency int
	Duration    time.Duration
	RPS         int
	Warmup      int

	// Timeouts and retries
	Timeout    time.Duration
	Retries    int
	NoRedirect bool

	// Output
	OutputFile string
	Format     string
	NoProgress bool
	Verbose    bool

	// TLS / connection
	Insecure  bool
	HTTP2     bool
	KeepAlive bool
}

func Parse() (*Config, error) {
	cfg := &Config{}

	flag.StringVar(&cfg.URL, "u", "", "Target URL (required)")
	flag.StringVar(&cfg.URL, "url", "", "Target URL (required)")

	flag.StringVar(&cfg.Method, "m", "GET", "HTTP method (GET, POST, PUT, PATCH, DELETE)")
	flag.StringVar(&cfg.Method, "method", "GET", "HTTP method")

	flag.Var(&cfg.Headers, "H", "Request header, repeatable: -H \"Key: Value\"")
	flag.Var(&cfg.Headers, "header", "Request header, repeatable")

	flag.StringVar(&cfg.Body, "b", "", "Request body as raw string")
	flag.StringVar(&cfg.Body, "body", "", "Request body as raw string")

	flag.StringVar(&cfg.BodyFile, "body-file", "", "Read request body from file")

	flag.IntVar(&cfg.Requests, "n", 100, "Total number of requests")
	flag.IntVar(&cfg.Requests, "requests", 100, "Total number of requests")

	flag.IntVar(&cfg.Concurrency, "c", 10, "Number of concurrent workers")
	flag.IntVar(&cfg.Concurrency, "concurrency", 10, "Number of concurrent workers")

	flag.DurationVar(&cfg.Duration, "d", 0, "Run for a fixed duration instead of request count (e.g. 30s, 2m)")
	flag.DurationVar(&cfg.Duration, "duration", 0, "Run for a fixed duration")

	flag.IntVar(&cfg.RPS, "rps", 0, "Max requests per second (0 = unlimited)")

	flag.IntVar(&cfg.Warmup, "warmup", 0, "Warmup requests to send before measuring")

	flag.DurationVar(&cfg.Timeout, "t", 10*time.Second, "Per-request timeout (e.g. 5s, 500ms)")
	flag.DurationVar(&cfg.Timeout, "timeout", 10*time.Second, "Per-request timeout")

	flag.IntVar(&cfg.Retries, "retries", 0, "Retries on timeout or 5xx before marking failed")

	flag.BoolVar(&cfg.NoRedirect, "no-redirect", false, "Disable following HTTP redirects")

	flag.StringVar(&cfg.OutputFile, "o", "", "Write results to file (.json or .csv)")
	flag.StringVar(&cfg.OutputFile, "output", "", "Write results to file")

	flag.StringVar(&cfg.Format, "format", "table", "Terminal output style: table, json, csv, silent")

	flag.BoolVar(&cfg.NoProgress, "no-progress", false, "Hide live progress bar")

	flag.BoolVar(&cfg.Verbose, "v", false, "Print each result as it completes")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "Print each result as it completes")

	flag.BoolVar(&cfg.Insecure, "insecure", false, "Skip TLS certificate verification")

	flag.BoolVar(&cfg.HTTP2, "http2", false, "Force HTTP/2")

	flag.BoolVar(&cfg.KeepAlive, "keep-alive", true, "Reuse TCP connections between requests")

	flag.Usage = usage
	flag.Parse()

	return cfg, validate(cfg)
}

func validate(cfg *Config) error {
	if cfg.URL == "" {
		return fmt.Errorf("target URL is required (-u)")
	}
	if !strings.HasPrefix(cfg.URL, "http://") && !strings.HasPrefix(cfg.URL, "https://") {
		return fmt.Errorf("URL must start with http:// or https://")
	}
	if cfg.Concurrency < 1 {
		return fmt.Errorf("concurrency must be at least 1")
	}
	if cfg.Requests < 1 && cfg.Duration == 0 {
		return fmt.Errorf("requests must be at least 1 (or use -d for duration mode)")
	}
	if cfg.Duration > 0 && cfg.Requests != 100 {
		// both set: duration wins, warn user
		_, err := fmt.Fprintln(os.Stderr, "warning: both -n and -d set; -d takes precedence")
		if err != nil {
			return err
		}
	}
	cfg.Method = strings.ToUpper(cfg.Method)
	return nil
}

func usage() {
	_, err := fmt.Fprintf(os.Stderr, `kload ~ HTTP load testing tool

Usage:
  kload -u <url> [flags]

Examples:
  kload -u https://api.example.com/health -n 500 -c 20
  kload -u https://api.example.com/login -m POST -H "Content-Type: application/json" -b '{"user":"test"}' -d 30s --rps 100
  kload -u https://api.example.com -n 1000 -c 50 --no-progress -o results.json

Flags:
  -u,  --url           Target URL (required)
  -m,  --method        HTTP method (default: GET)
  -H,  --header        Request header, repeatable
  -b,  --body          Request body as raw string
       --body-file     Read body from file
  -n,  --requests      Total requests (default: 100)
  -c,  --concurrency   Concurrent workers (default: 10)
  -d,  --duration      Run duration, e.g. 30s, 2m (overrides -n)
       --rps           Max requests per second (default: unlimited)
       --warmup        Warmup requests before measuring
  -t,  --timeout       Per-request timeout (default: 10s)
       --retries       Retries on timeout/5xx (default: 0)
       --no-redirect   Disable redirect following
  -o,  --output        Output file (.json or .csv)
       --format        Terminal format: table, json, csv, silent
       --no-progress   Hide progress bar
  -v,  --verbose       Print each result live
       --insecure      Skip TLS verification
       --http2         Force HTTP/2
       --keep-alive    Reuse connections (default: true)

`)
	if err != nil {
		return
	}
}
