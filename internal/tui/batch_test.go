package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/yinheli/sshw"
)

func TestFirstLine(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"hello", "hello"},
		{"hello\nworld", "hello"},
		{"\n\n  first real\nignored", "  first real"},
		{"trailing  \nx", "trailing"},
	}
	for _, tc := range cases {
		if got := firstLine(tc.in); got != tc.want {
			t.Errorf("firstLine(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestRenderResultBadge(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		res  sshw.RunResult
		want string
	}{
		{"ok", sshw.RunResult{ExitCode: 0}, "✓ exit=0"},
		{"nonzero", sshw.RunResult{ExitCode: 1}, "✗ exit=1"},
		{"timeout", sshw.RunResult{ExitCode: -1, Err: context.DeadlineExceeded}, "✗ timeout"},
		{"refused", sshw.RunResult{ExitCode: -1, Err: errors.New("dial tcp: connection refused")}, "✗ refused"},
		{"auth", sshw.RunResult{ExitCode: -1, Err: errors.New("ssh: unable to authenticate")}, "✗ auth"},
		{"other", sshw.RunResult{ExitCode: -1, Err: errors.New("kaboom")}, "✗ error"},
	}
	for _, tc := range cases {
		got := stripAnsi(renderResultBadge(tc.res))
		if got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestDetailTabContent(t *testing.T) {
	t.Parallel()
	r := sshw.RunResult{
		Stdout:   []byte("hello\n"),
		Stderr:   []byte("warn"),
		ExitCode: 2,
		Duration: 250 * time.Millisecond,
	}

	if got := stripAnsi(detailTabContent(0, r)); !strings.Contains(got, "hello") {
		t.Errorf("stdout tab missing 'hello': %q", got)
	}
	if got := stripAnsi(detailTabContent(1, r)); !strings.Contains(got, "warn") {
		t.Errorf("stderr tab missing 'warn': %q", got)
	}
	meta := stripAnsi(detailTabContent(2, r))
	for _, want := range []string{"exit=2", "duration=250ms", "stdout bytes=6", "stderr bytes=4"} {
		if !strings.Contains(meta, want) {
			t.Errorf("meta tab missing %q in:\n%s", want, meta)
		}
	}

	noOut := sshw.RunResult{ExitCode: 0, Duration: time.Millisecond}
	if got := stripAnsi(detailTabContent(0, noOut)); !strings.Contains(got, "(no stdout)") {
		t.Errorf("empty stdout tab should hint '(no stdout)'; got %q", got)
	}
	if got := stripAnsi(detailTabContent(1, noOut)); !strings.Contains(got, "(no stderr)") {
		t.Errorf("empty stderr tab should hint '(no stderr)'; got %q", got)
	}
}

func TestFailedTargetsAndFilter(t *testing.T) {
	t.Parallel()
	a := &sshw.Node{Name: "a", Host: "1.1.1.1"}
	b := &sshw.Node{Name: "b", Host: "1.1.1.2"}
	c := &sshw.Node{Name: "c", Host: "1.1.1.3"}

	m := newModel([]*sshw.Node{a, b, c})
	m.batch.targets = []*sshw.Node{a, b, c}
	m.batch.results = map[*sshw.Node]*batchTargetResult{
		a: {done: true, res: sshw.RunResult{ExitCode: 0}},
		b: {done: true, res: sshw.RunResult{ExitCode: 1}},
		c: {done: true, res: sshw.RunResult{ExitCode: -1, Err: errors.New("dial: timeout")}},
	}

	failed := m.failedTargets()
	if len(failed) != 2 || failed[0] != b || failed[1] != c {
		t.Fatalf("failedTargets wrong: %+v", failed)
	}

	// Default filter: all visible.
	if got := m.visibleResults(); len(got) != 3 {
		t.Fatalf("filter=0 visible should be 3, got %d", len(got))
	}

	// Failed-only filter.
	m.batch.resultsFilter = 1
	got := m.visibleResults()
	if len(got) != 2 || got[0] != b || got[1] != c {
		t.Errorf("filter=1 visible wrong: %+v", got)
	}
}

func TestBatchTargets(t *testing.T) {
	t.Parallel()
	a := &sshw.Node{Name: "a", Host: "1.1.1.1"}
	b := &sshw.Node{Name: "b", Host: "1.1.1.2"}
	c := &sshw.Node{Name: "c", Host: "1.1.1.3"}
	group := &sshw.Node{Name: "g", Children: []*sshw.Node{a, b, c}}

	m := newModel([]*sshw.Node{group})
	// Drill into the group so the visible list is [a, b, c].
	childItems := nodesToListItems(nodesToItems(group.Children))
	m.list.SetItems(childItems)

	if got := m.batchTargets(); len(got) != 3 {
		t.Fatalf("no marks: want 3 visible targets, got %d", len(got))
	}

	// Mark a subset.
	m.marks[a] = struct{}{}
	m.marks[c] = struct{}{}
	got := m.batchTargets()
	if len(got) != 2 || got[0] != a || got[1] != c {
		t.Fatalf("marked subset wrong: got %+v", got)
	}
}
