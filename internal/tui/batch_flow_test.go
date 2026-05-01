package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yinheli/sshw"
)

// TestBatchFlow_Rendering drives the model through every mode of the batch
// flow and asserts View() returns non-empty output without panicking. This is
// the headless equivalent of "boot the TUI and tab through the whole flow".
func TestBatchFlow_Rendering(t *testing.T) {
	t.Parallel()
	a := &sshw.Node{Name: "a", Host: "10.0.0.1", User: "u"}
	b := &sshw.Node{Name: "b", Host: "10.0.0.2", User: "u"}
	c := &sshw.Node{Name: "c", Host: "10.0.0.3", User: "u"}
	group := &sshw.Node{Name: "g", Children: []*sshw.Node{a, b, c}}

	m := newModel([]*sshw.Node{group})
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	// Drill into the group so the visible list is the three hosts.
	childItems := nodesToListItems(nodesToItems(group.Children))
	m.list.SetItems(childItems)

	// modeList — should show the help footer and the list rows.
	if v := m.View(); !strings.Contains(v, "ctrl+x") {
		t.Errorf("modeList View should mention ctrl+x in help; got:\n%s", v)
	}

	// Mark two hosts; verify the marked count appears in the header.
	m.marks[a] = struct{}{}
	m.marks[c] = struct{}{}
	if v := m.View(); !strings.Contains(stripAnsi(v), "[2 marked]") {
		t.Errorf("expected [2 marked] in header; got:\n%s", stripAnsi(v))
	}

	// Manually transition through batch states (avoiding tea.KeyMsg parsing).
	targets := m.batchTargets()
	if len(targets) != 2 {
		t.Fatalf("expected 2 marked targets, got %d", len(targets))
	}
	m.batch.targets = targets
	m.mode = modeBatchPrompt
	m.syncBatchLayout()
	if v := m.View(); !strings.Contains(stripAnsi(v), "running on 2 host(s)") {
		t.Errorf("modeBatchPrompt should advertise target count; got:\n%s", stripAnsi(v))
	}

	// Confirm view.
	m.batch.cmdLine = "ls /"
	m.mode = modeBatchConfirm
	if v := m.View(); !strings.Contains(stripAnsi(v), "Run on 2 host(s)?") {
		t.Errorf("modeBatchConfirm should ask for confirmation; got:\n%s", stripAnsi(v))
	}

	// Running view: simulate one done, one pending.
	m.mode = modeBatchRunning
	m.batch.active = true
	m.batch.results = map[*sshw.Node]*batchTargetResult{
		a: {done: true, res: sshw.RunResult{ExitCode: 0, Stdout: []byte("ok\n"), Duration: 50 * time.Millisecond}},
		c: {done: false},
	}
	if v := m.View(); !strings.Contains(stripAnsi(v), "exit=0") || !strings.Contains(stripAnsi(v), "running") {
		t.Errorf("modeBatchRunning should show one done and one running; got:\n%s", stripAnsi(v))
	}

	// Results view: complete second host with non-zero exit.
	m.batch.results[c] = &batchTargetResult{
		done: true,
		res: sshw.RunResult{
			ExitCode: 2,
			Stderr:   []byte("permission denied\n"),
			Duration: 80 * time.Millisecond,
		},
	}
	m.batch.active = false
	m.mode = modeBatchResults
	v := stripAnsi(m.View())
	for _, want := range []string{"exit=0", "exit=2", "✓ 1", "✗ 1", "ls /"} {
		if !strings.Contains(v, want) {
			t.Errorf("modeBatchResults missing %q in:\n%s", want, v)
		}
	}

	// Detail view: drill into host c. Default tab is stdout but host c
	// has no stdout — assert the tab bar is rendered and stdout hint shows.
	m.openDetail(c, nil, m.batch.results[c].res)
	v = stripAnsi(m.View())
	for _, want := range []string{"stdout", "stderr", "meta", "(no stdout)"} {
		if !strings.Contains(v, want) {
			t.Errorf("modeBatchDetail (stdout tab) missing %q in:\n%s", want, v)
		}
	}
	// Switch to stderr tab → should show the error body.
	m.setDetailTab(1)
	v = stripAnsi(m.View())
	if !strings.Contains(v, "permission denied") {
		t.Errorf("stderr tab missing 'permission denied' in:\n%s", v)
	}
	// Switch to meta → exit code present.
	m.setDetailTab(2)
	v = stripAnsi(m.View())
	if !strings.Contains(v, "exit=2") {
		t.Errorf("meta tab missing 'exit=2' in:\n%s", v)
	}

	// Group view: toggle on, expect a single bucket holding host a (only ✓).
	m.batch.detailNode = nil
	m.mode = modeBatchResults
	m.batch.groupView = true
	v = stripAnsi(m.View())
	for _, want := range []string{"[grouped]", "[1 host]", "ok"} {
		if !strings.Contains(v, want) {
			t.Errorf("grouped view missing %q in:\n%s", want, v)
		}
	}

	// Drill into the bucket — header lists hosts.
	m.openDetail(a, []*sshw.Node{a}, m.batch.results[a].res)
	v = stripAnsi(m.View())
	if !strings.Contains(v, "ok") {
		t.Errorf("bucket detail should show stdout; got:\n%s", v)
	}
}

// TestBatchConfirm_Danger ensures the danger banner renders when the
// command matches a destructive pattern.
func TestBatchConfirm_Danger(t *testing.T) {
	t.Parallel()
	a := &sshw.Node{Name: "a", Host: "10.0.0.1", User: "u"}
	m := newModel([]*sshw.Node{a})
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	m.batch.targets = []*sshw.Node{a}
	m.batch.cmdLine = "rm -rf /var/log"
	m.batch.dangerous = "rm -rf"
	m.mode = modeBatchConfirm
	m.syncBatchLayout()
	m.batch.confirmInput.Focus()

	v := stripAnsi(m.View())
	for _, want := range []string{"destructive pattern", "rm -rf", "yes I am sure"} {
		if !strings.Contains(v, want) {
			t.Errorf("danger confirm missing %q in:\n%s", want, v)
		}
	}
}
