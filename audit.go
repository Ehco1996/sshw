package sshw

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// RunRecord is the input to WriteRun: everything the audit layer needs
// to persist a batch run. Targets gives stable host ordering; Results
// is keyed by *Node so callers can pass the live in-flight map.
type RunRecord struct {
	Cmd      string
	Targets  []*Node
	Results  map[*Node]RunResult
	Started  time.Time
	Finished time.Time
}

// hostNameSafe matches characters disallowed in cross-platform filenames.
// Anything outside ASCII alphanumerics, dot, dash, underscore is replaced
// with '_' to keep per-host log files portable.
var hostNameSafe = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// RunLogDir resolves the base directory for run artifacts.
// Precedence: SSHW_RUN_LOG_DIR > $XDG_STATE_HOME/sshw/runs > ~/.local/state/sshw/runs.
func RunLogDir() (string, error) {
	if env := os.Getenv("SSHW_RUN_LOG_DIR"); env != "" {
		return env, nil
	}
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		return filepath.Join(xdg, "sshw", "runs"), nil
	}
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(u.HomeDir, ".local", "state", "sshw", "runs"), nil
}

// newRunID returns "<RFC3339-compact>-<6-byte hex>", e.g.
// 20260501T113000+0800-3f9a2c. RFC3339 prefix sorts chronologically;
// the hex suffix avoids collisions on rapid reruns within the same second.
func newRunID(t time.Time) string {
	stamp := t.Format("20060102T150405-0700")
	var buf [3]byte
	if _, err := rand.Read(buf[:]); err != nil {
		// Fall back to nanoseconds — collision-resistant enough for our use.
		return fmt.Sprintf("%s-%06x", stamp, t.UnixNano()&0xffffff)
	}
	return stamp + "-" + hex.EncodeToString(buf[:])
}

// safeHostFilename strips characters not safe across filesystems.
func safeHostFilename(name string) string {
	if name == "" {
		return "host"
	}
	cleaned := hostNameSafe.ReplaceAllString(name, "_")
	cleaned = strings.Trim(cleaned, "._-")
	if cleaned == "" {
		return "host"
	}
	return cleaned
}

// formatHostLog renders the per-host log file body. Plain text, no ANSI,
// laid out so it's recognizable from the in-TUI detail viewport.
func formatHostLog(runID string, ts time.Time, n *Node, cmd string, r RunResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# sshw run %s\n", runID)
	fmt.Fprintf(&b, "# ts: %s\n", ts.Format(time.RFC3339))
	fmt.Fprintf(&b, "# host: %s (%s@%s:%d)\n", n.Name, n.EffectiveUser(), n.Host, n.SSHPort())
	fmt.Fprintf(&b, "# cmd: %s\n", cmd)
	fmt.Fprintf(&b, "# exit: %d  duration: %s\n", r.ExitCode, r.Duration.Round(time.Millisecond))
	if r.Err != nil {
		fmt.Fprintf(&b, "# err: %s\n", r.Err.Error())
	}
	b.WriteString("\n--- stdout ---\n")
	b.Write(r.Stdout)
	if len(r.Stdout) > 0 && !strings.HasSuffix(string(r.Stdout), "\n") {
		b.WriteByte('\n')
	}
	b.WriteString("--- stderr ---\n")
	b.Write(r.Stderr)
	if len(r.Stderr) > 0 && !strings.HasSuffix(string(r.Stderr), "\n") {
		b.WriteByte('\n')
	}
	return b.String()
}

// runIndexEntry is the JSONL shape appended to runs.jsonl.
type runIndexEntry struct {
	TS         string   `json:"ts"`
	RunID      string   `json:"run_id"`
	Cmd        string   `json:"cmd"`
	Hosts      []string `json:"hosts"`
	OK         int      `json:"ok"`
	Fail       int      `json:"fail"`
	Total      int      `json:"total"`
	DurationMs int64    `json:"duration_ms"`
	LogDir     string   `json:"log_dir"`
}

// WriteRun persists rec to disk: appends one line to runs.jsonl and writes
// one log file per target. Auditing is best-effort — failures are returned
// for logging by the caller but should not abort the surrounding flow.
func WriteRun(rec RunRecord) (runID, logDir string, err error) {
	base, err := RunLogDir()
	if err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", "", err
	}

	ts := rec.Finished
	if ts.IsZero() {
		ts = time.Now()
	}
	runID = newRunID(ts)
	logDir = filepath.Join(base, runID)
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return runID, logDir, err
	}

	hosts := make([]string, 0, len(rec.Targets))
	used := make(map[string]int, len(rec.Targets))
	var ok, fail int
	for _, n := range rec.Targets {
		hosts = append(hosts, n.Name)

		base := safeHostFilename(n.Name)
		fname := base + ".log"
		// Disambiguate when two hosts share a Name (or sanitize collides).
		if used[fname] > 0 {
			fname = fmt.Sprintf("%s.%d.log", base, used[fname])
		}
		used[base]++

		r, hasResult := rec.Results[n]
		if !hasResult {
			r = RunResult{ExitCode: -1, Err: fmt.Errorf("no result recorded")}
		}
		body := formatHostLog(runID, ts, n, rec.Cmd, r)
		_ = os.WriteFile(filepath.Join(logDir, fname), []byte(body), 0o644)

		if r.Err == nil && r.ExitCode == 0 {
			ok++
		} else {
			fail++
		}
	}

	entry := runIndexEntry{
		TS:         ts.Format(time.RFC3339),
		RunID:      runID,
		Cmd:        rec.Cmd,
		Hosts:      hosts,
		OK:         ok,
		Fail:       fail,
		Total:      len(rec.Targets),
		DurationMs: rec.Finished.Sub(rec.Started).Milliseconds(),
		LogDir:     logDir,
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return runID, logDir, err
	}

	idxPath := filepath.Join(base, "runs.jsonl")
	f, err := os.OpenFile(idxPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return runID, logDir, err
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return runID, logDir, err
	}
	return runID, logDir, nil
}
