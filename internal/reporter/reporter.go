package reporter

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"../metrics"
)

// ANSI colour codes (! degrade gracefully on terminals that don't support them !)
const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	dim    = "\033[2m"
	green  = "\033[32m"
	red    = "\033[31m"
	yellow = "\033[33m"
	cyan   = "\033[36m"
	white  = "\033[97m"
)

// Progress tracks live progress bar state
type Progress struct {
	mu      sync.Mutex
	total   int
	current int
	width   int
	hidden  bool
}

func NewProgress(total int, hidden bool) *Progress {
	return &Progress{total: total, width: 40, hidden: hidden}
}

func (p *Progress) Update(done int) {
	if p.hidden {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current = done

	var pct float64
	var filled int
	if p.total > 0 {
		pct = float64(done) / float64(p.total) * 100
		filled = int(float64(p.width) * float64(done) / float64(p.total))
	} else {
		// Duration mode — no total known, just spin
		filled = (done / 5) % p.width
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", p.width-filled)

	if p.total > 0 {
		fmt.Fprintf(os.Stderr, "\r  %s%s%s  %d / %d  (%.0f%%)",
			cyan, bar, reset, done, p.total, pct)
	} else {
		fmt.Fprintf(os.Stderr, "\r  %s%s%s  %d requests sent",
			cyan, bar, reset, done)
	}
}

func (p *Progress) Clear() {
	if p.hidden {
		return
	}
	fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", p.width+40))
}

// PrintTable renders the final summary table to stdout
func PrintTable(s metrics.Summary) {
	successRate := 0.0
	if s.Total > 0 {
		successRate = float64(s.Successful) / float64(s.Total) * 100
	}

	successColor := green
	if successRate < 95 {
		successColor = yellow
	}
	if successRate < 80 {
		successColor = red
	}

	fmt.Println()
	fmt.Printf("  %s%s── Results %s%s\n", bold, white, strings.Repeat("─", 30), reset)
	fmt.Printf("  %-22s %s%d%s\n", "Total Requests", bold, s.Total, reset)
	fmt.Printf("  %-22s %s%s%d%s%s  (%.1f%%)\n",
		"Successful", successColor, bold, s.Successful, reset, successColor, successRate)
	fmt.Printf("  %-22s %s%d%s\n", "Failed", red, s.Failed, reset)
	fmt.Printf("  %-22s %s%.1fs%s\n", "Total Duration", dim, s.TotalDuration.Seconds(), reset)
	fmt.Printf("  %-22s %s%.1f%s\n", "Req / sec", bold, s.ReqPerSec, reset)

	fmt.Println()
	fmt.Printf("  %s%s── Latency %s%s\n", bold, white, strings.Repeat("─", 30), reset)
	fmt.Printf("  %-22s %s\n", "Min", fmtDur(s.Min))
	fmt.Printf("  %-22s %s\n", "Mean", fmtDur(s.Mean))
	fmt.Printf("  %-22s %s\n", "p50", fmtDur(s.P50))
	fmt.Printf("  %-22s %s\n", "p90", fmtDur(s.P90))
	fmt.Printf("  %-22s %s\n", "p95", fmtDur(s.P95))
	fmt.Printf("  %-22s %s\n", "p99", fmtDur(s.P99))
	fmt.Printf("  %-22s %s\n", "Max", fmtDur(s.Max))

	if len(s.StatusCodes) > 0 {
		fmt.Println()
		fmt.Printf("  %s%s── Status Codes %s%s\n", bold, white, strings.Repeat("─", 25), reset)
		for code, count := range s.StatusCodes {
			color := green
			if code >= 400 {
				color = yellow
			}
			if code >= 500 {
				color = red
			}
			fmt.Printf("  %s%-6d%s  %d\n", color, code, reset, count)
		}
	}
	fmt.Println()
}

// PrintVerbose prints a single result line
func PrintVerbose(r metrics.Result, done, total int) {
	if r.Err != nil {
		fmt.Printf("  [%s%d%s] %sERR%s  %s\n",
			dim, done, reset, red, reset, r.Err.Error())
		return
	}
	color := green
	if r.StatusCode >= 400 {
		color = yellow
	}
	if r.StatusCode >= 500 {
		color = red
	}
	fmt.Printf("  [%s%d%s] %s%d%s  %s\n",
		dim, done, reset, color, r.StatusCode, reset, fmtDur(r.Duration))
}

// WriteJSON writes the summary as JSON to a file
func WriteJSON(path string, s metrics.Summary) error {
	type jsonOut struct {
		Total         int               `json:"total"`
		Successful    int               `json:"successful"`
		Failed        int               `json:"failed"`
		ReqPerSec     float64           `json:"req_per_sec"`
		TotalDuration float64           `json:"total_duration_s"`
		Latency       map[string]string `json:"latency"`
		StatusCodes   map[int]int       `json:"status_codes"`
	}
	out := jsonOut{
		Total:         s.Total,
		Successful:    s.Successful,
		Failed:        s.Failed,
		ReqPerSec:     s.ReqPerSec,
		TotalDuration: s.TotalDuration.Seconds(),
		Latency: map[string]string{
			"min":  fmtDur(s.Min),
			"mean": fmtDur(s.Mean),
			"p50":  fmtDur(s.P50),
			"p90":  fmtDur(s.P90),
			"p95":  fmtDur(s.P95),
			"p99":  fmtDur(s.P99),
			"max":  fmtDur(s.Max),
		},
		StatusCodes: s.StatusCodes,
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// WriteCSV writes per-request results as CSV to a file
func WriteCSV(path string, results []metrics.Result) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	w.Write([]string{"status_code", "duration_ms", "error"})
	for _, r := range results {
		if r.Warmup {
			continue
		}
		errStr := ""
		if r.Err != nil {
			errStr = r.Err.Error()
		}
		w.Write([]string{
			fmt.Sprintf("%d", r.StatusCode),
			fmt.Sprintf("%.2f", float64(r.Duration.Microseconds())/1000.0),
			errStr,
		})
	}
	w.Flush()
	return w.Error()
}

func fmtDur(d time.Duration) string {
	if d == 0 {
		return "—"
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%sµs%s", bold, reset) // shouldn't happen in practice
	}
	ms := float64(d.Microseconds()) / 1000.0
	if ms < 1000 {
		return fmt.Sprintf("%s%.1fms%s", bold, ms, reset)
	}
	return fmt.Sprintf("%s%.2fs%s", bold, ms/1000.0, reset)
}
