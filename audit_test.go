package sshw

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunLogDir_EnvOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SSHW_RUN_LOG_DIR", dir)
	t.Setenv("XDG_STATE_HOME", "/should/be/ignored")
	got, err := RunLogDir()
	if err != nil {
		t.Fatal(err)
	}
	if got != dir {
		t.Errorf("RunLogDir() = %q, want %q", got, dir)
	}
}

func TestRunLogDir_XDG(t *testing.T) {
	t.Setenv("SSHW_RUN_LOG_DIR", "")
	xdg := t.TempDir()
	t.Setenv("XDG_STATE_HOME", xdg)
	got, err := RunLogDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(xdg, "sshw", "runs")
	if got != want {
		t.Errorf("RunLogDir() = %q, want %q", got, want)
	}
}

func TestSafeHostFilename(t *testing.T) {
	cases := map[string]string{
		"":              "host",
		"nas":           "nas",
		"my host.1":     "my_host.1",
		"a/b\\c":        "a_b_c",
		"...":           "host",
		"中文":            "host",
		"prod-web-01":   "prod-web-01",
		"router_2.5GHz": "router_2.5GHz",
	}
	for in, want := range cases {
		if got := safeHostFilename(in); got != want {
			t.Errorf("safeHostFilename(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestWriteRun_FilesAndIndex(t *testing.T) {
	base := t.TempDir()
	t.Setenv("SSHW_RUN_LOG_DIR", base)

	a := &Node{Name: "nas", Host: "192.168.31.30", User: "ehco"}
	b := &Node{Name: "my host", Host: "10.0.0.2", Port: 2222, User: "root"}

	started := time.Now().Add(-300 * time.Millisecond)
	finished := started.Add(300 * time.Millisecond)

	rec := RunRecord{
		Cmd:      "uptime",
		Targets:  []*Node{a, b},
		Started:  started,
		Finished: finished,
		Results: map[*Node]RunResult{
			a: {Stdout: []byte("up 12 days\n"), ExitCode: 0, Duration: 100 * time.Millisecond},
			b: {Stderr: []byte("connection refused"), ExitCode: -1, Duration: 50 * time.Millisecond, Err: errors.New("dial tcp: connection refused")},
		},
	}

	runID, logDir, err := WriteRun(rec)
	if err != nil {
		t.Fatalf("WriteRun: %v", err)
	}
	if runID == "" || logDir == "" {
		t.Fatalf("expected non-empty runID/logDir, got %q %q", runID, logDir)
	}
	if !strings.HasPrefix(logDir, base) {
		t.Fatalf("logDir %q not under base %q", logDir, base)
	}

	// Per-host files exist with sanitized names.
	nasPath := filepath.Join(logDir, "nas.log")
	hostPath := filepath.Join(logDir, "my_host.log")
	for _, p := range []string{nasPath, hostPath} {
		body, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		if !strings.Contains(string(body), "# sshw run "+runID) {
			t.Errorf("%s missing run header", p)
		}
	}

	// Index file appended with one well-formed JSON line.
	idx := filepath.Join(base, "runs.jsonl")
	f, err := os.Open(idx)
	if err != nil {
		t.Fatalf("open index: %v", err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	if !sc.Scan() {
		t.Fatal("expected one line in index")
	}
	var entry runIndexEntry
	if err := json.Unmarshal(sc.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal index entry: %v", err)
	}
	if entry.Cmd != "uptime" || entry.Total != 2 || entry.OK != 1 || entry.Fail != 1 {
		t.Errorf("unexpected index entry: %+v", entry)
	}
	if entry.RunID != runID || entry.LogDir != logDir {
		t.Errorf("run_id / log_dir mismatch in index: %+v", entry)
	}
}

func TestWriteRun_AppendsNotTruncate(t *testing.T) {
	base := t.TempDir()
	t.Setenv("SSHW_RUN_LOG_DIR", base)

	n := &Node{Name: "h", Host: "1.1.1.1", User: "u"}
	for i := 0; i < 3; i++ {
		_, _, err := WriteRun(RunRecord{
			Cmd:      "echo",
			Targets:  []*Node{n},
			Started:  time.Now(),
			Finished: time.Now(),
			Results:  map[*Node]RunResult{n: {ExitCode: 0}},
		})
		if err != nil {
			t.Fatalf("WriteRun #%d: %v", i, err)
		}
		// Bump clock granularity for runID uniqueness.
		time.Sleep(2 * time.Millisecond)
	}
	body, err := os.ReadFile(filepath.Join(base, "runs.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Count(string(body), "\n")
	if lines != 3 {
		t.Errorf("expected 3 lines in runs.jsonl, got %d", lines)
	}
}

func TestFormatHostLog_Privacy(t *testing.T) {
	n := &Node{
		Name: "secret", Host: "1.1.1.1", User: "u",
		Password: "hunter2", Passphrase: "topsecret",
	}
	out := formatHostLog("rid", time.Now(), n, "echo hi", RunResult{Stdout: []byte("hi\n")})
	for _, leak := range []string{"hunter2", "topsecret"} {
		if strings.Contains(out, leak) {
			t.Errorf("formatHostLog leaked secret %q in:\n%s", leak, out)
		}
	}
}
