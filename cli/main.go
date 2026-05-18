package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"../internal/config"
	"../internal/metrics"
	"../internal/reporter"
	"../internal/runner"
)

func main() {
	cfg, err := config.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "kload: %v\n", err)
		os.Exit(1)
	}

	var body []byte
	if cfg.BodyFile != "" {
		body, err = os.ReadFile(cfg.BodyFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "kload: could not read body file: %v\n", err)
			os.Exit(1)
		}
	} else if cfg.Body != "" {
		body = []byte(cfg.Body)
	}

	if cfg.Format != "silent" && cfg.Format != "json" && cfg.Format != "csv" {
		printHeader(cfg)
	}

	total := cfg.Requests + cfg.Warmup
	if cfg.Duration > 0 {
		total = 0
	}
	progress := reporter.NewProgress(total, cfg.NoProgress || cfg.Verbose || cfg.Format == "silent")

	onResult := func(r metrics.Result, done, _ int) {
		if cfg.Verbose && !r.Warmup {
			progress.Clear()
			reporter.PrintVerbose(r, done, total)
		} else {
			progress.Update(done)
		}
	}

	results, wallTime := runner.Run(cfg, body, onResult)
	progress.Clear()

	summary := metrics.Compute(results, wallTime)

	switch cfg.Format {
	case "silent":
	case "json":
		b, _ := json.MarshalIndent(summaryMap(summary), "", "  ")
		fmt.Println(string(b))
	case "csv":
		fmt.Println("total,successful,failed,req_per_sec,p50_ms,p95_ms,p99_ms")
		fmt.Printf("%d,%d,%d,%.2f,%.2f,%.2f,%.2f\n",
			summary.Total, summary.Successful, summary.Failed, summary.ReqPerSec,
			msf(summary.P50), msf(summary.P95), msf(summary.P99))
	default:
		reporter.PrintTable(summary)
	}

	if cfg.OutputFile != "" {
		ext := strings.ToLower(filepath.Ext(cfg.OutputFile))
		switch ext {
		case ".json":
			if err := reporter.WriteJSON(cfg.OutputFile, summary); err != nil {
				fmt.Fprintf(os.Stderr, "kload: could not write JSON: %v\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "  results written to %s\n", cfg.OutputFile)
			}
		case ".csv":
			if err := reporter.WriteCSV(cfg.OutputFile, results); err != nil {
				fmt.Fprintf(os.Stderr, "kload: could not write CSV: %v\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "  results written to %s\n", cfg.OutputFile)
			}
		default:
			fmt.Fprintf(os.Stderr, "kload: unknown output extension %q — use .json or .csv\n", ext)
		}
	}

	if summary.Total > 0 && float64(summary.Failed)/float64(summary.Total) > 0.10 {
		os.Exit(1)
	}
}

func printHeader(cfg *config.Config) {
	mode := fmt.Sprintf("%d requests · %d workers", cfg.Requests, cfg.Concurrency)
	if cfg.Duration > 0 {
		mode = fmt.Sprintf("%s · %d workers", cfg.Duration, cfg.Concurrency)
	}
	if cfg.Warmup > 0 {
		mode += fmt.Sprintf(" · %d warmup", cfg.Warmup)
	}
	if cfg.RPS > 0 {
		mode += fmt.Sprintf(" · max %d rps", cfg.RPS)
	}
	fmt.Printf("\n  \033[1mkload\033[0m  %s  →  %s\n\n", mode, cfg.URL)
}

func msf(d interface{ Microseconds() int64 }) float64 {
	return float64(d.Microseconds()) / 1000.0
}

func summaryMap(s metrics.Summary) map[string]interface{} {
	return map[string]interface{}{
		"total":            s.Total,
		"successful":       s.Successful,
		"failed":           s.Failed,
		"req_per_sec":      s.ReqPerSec,
		"total_duration_s": s.TotalDuration.Seconds(),
		"latency": map[string]float64{
			"min_ms":  msf(s.Min),
			"mean_ms": msf(s.Mean),
			"p50_ms":  msf(s.P50),
			"p90_ms":  msf(s.P90),
			"p95_ms":  msf(s.P95),
			"p99_ms":  msf(s.P99),
			"max_ms":  msf(s.Max),
		},
		"status_codes": s.StatusCodes,
	}
}
