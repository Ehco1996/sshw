package tui

import (
	"context"
	"errors"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yinheli/sshw"
)

// errBatchCancelled is the synthetic error stamped on hosts that were
// pending when the user soft-cancelled a batch run. Distinct from the
// transport-level "context deadline exceeded" so renderResultBadge can
// label these hosts "cancelled" instead of "timeout".
var errBatchCancelled = errors.New("cancelled by user")

const (
	batchParallelism = 8
	batchTimeout     = 30 * time.Second
)

type uiMode int

const (
	modeList uiMode = iota
	modeBatchPrompt
	modeBatchConfirm
	modeBatchRunning
	modeBatchResults
	modeBatchDetail
)

type batchTargetResult struct {
	done bool
	res  sshw.RunResult
}

type batchState struct {
	cmdLine    string
	targets    []*sshw.Node
	results    map[*sshw.Node]*batchTargetResult
	generation int
	active     bool
	resultIdx  int
	spinner    spinner.Model
	progress   progress.Model
	input      textinput.Model
	detail     viewport.Model
	detailNode *sshw.Node
	sem        chan struct{}

	// Audit log tracking. Set when this run started; resolved when results
	// are written to disk on transition to modeBatchResults.
	runStarted time.Time
	runID      string // empty if WriteRun has not run yet (or failed)
	logDir     string // absolute path to this run's directory; empty on failure
	logErr     error  // last audit error, surfaced as a footer hint

	// Dangerous-command flow: when cmdLine matches a dangerPattern,
	// `dangerous` holds the matched substring (so it can be highlighted),
	// and confirmInput is the textinput that captures the typed override.
	dangerous     string
	confirmInput  textinput.Model
	confirmFailed bool // true after a mismatch attempt; cleared on next keystroke

	// Results-view state. resultsFilter: 0 = all, 1 = failed only.
	resultsFilter int
	flash         string // transient footer message (e.g. "no failed hosts")

	// Grouped (dshbak-style) results view state.
	groupView   bool
	bucketIdx   int
	bucketHosts []*sshw.Node // hosts of the bucket currently drilled into; non-nil only when in detail-from-bucket

	// Detail-tab state. detailTab indexes detailTabs (stdout=0, stderr=1, meta=2).
	// detailRes is the result currently being inspected (so tab switches don't
	// need to re-resolve it from results map). Set on entry to modeBatchDetail.
	detailTab int
	detailRes sshw.RunResult
}

// detailTabs is the ordered list of section labels rendered in the detail tab bar.
var detailTabs = []string{"stdout", "stderr", "meta"}

func newBatchState() *batchState {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorPrimary)

	in := textinput.New()
	in.Placeholder = "remote command, e.g. uptime"
	in.CharLimit = 4096
	// Width is set dynamically when entering prompt mode.

	confirm := textinput.New()
	confirm.Placeholder = dangerConfirmPhrase
	confirm.CharLimit = 64

	pb := progress.New(progress.WithDefaultGradient(), progress.WithoutPercentage())
	pb.Width = 40 // resized at runtime in syncBatchLayout

	return &batchState{
		results:      make(map[*sshw.Node]*batchTargetResult),
		spinner:      s,
		progress:     pb,
		input:        in,
		confirmInput: confirm,
		sem:          make(chan struct{}, batchParallelism),
	}
}

func (b *batchState) reset() {
	b.cmdLine = ""
	b.targets = nil
	b.results = make(map[*sshw.Node]*batchTargetResult)
	b.active = false
	b.resultIdx = 0
	b.detailNode = nil
	b.runStarted = time.Time{}
	b.runID = ""
	b.logDir = ""
	b.logErr = nil
	b.dangerous = ""
	b.confirmInput.Reset()
	b.confirmInput.Blur()
	b.confirmFailed = false
	b.resultsFilter = 0
	b.flash = ""
	b.groupView = false
	b.bucketIdx = 0
	b.bucketHosts = nil
	b.detailTab = 0
	b.detailRes = sshw.RunResult{}
}

// isFailed reports whether r counts as a failure for filter / rerun purposes.
func isFailed(r *batchTargetResult) bool {
	return r != nil && r.done && (r.res.Err != nil || r.res.ExitCode != 0)
}

// logHint formats the audit-log path or error for display in the results footer.
func (b *batchState) logHint() string {
	if b.logErr != nil {
		return batchHintStyle.Render("audit log failed: " + b.logErr.Error())
	}
	if b.logDir == "" {
		return ""
	}
	return batchHintStyle.Render("logged: " + b.logDir)
}

// counts returns total + completed (finished, regardless of outcome) + failed.
func (b *batchState) counts() (total, done, ok, fail int) {
	total = len(b.targets)
	for _, r := range b.results {
		if !r.done {
			continue
		}
		done++
		if r.res.Err == nil && r.res.ExitCode == 0 {
			ok++
		} else {
			fail++
		}
	}
	return
}

// batchResultMsg is delivered when a single host's command finishes.
type batchResultMsg struct {
	node       *sshw.Node
	generation int
	res        sshw.RunResult
}

// runCommandCmd opens an SSH session, runs cmd, and returns the result through
// the bubbletea message loop. The semaphore caps how many connections are
// in-flight at once.
func runCommandCmd(node *sshw.Node, cmd string, gen int, sem chan struct{}) tea.Cmd {
	return func() tea.Msg {
		sem <- struct{}{}
		defer func() { <-sem }()

		ctx, cancel := context.WithTimeout(context.Background(), batchTimeout)
		defer cancel()

		runner := sshw.NewRunner(node)
		return batchResultMsg{
			node:       node,
			generation: gen,
			res:        runner.RunCommand(ctx, cmd),
		}
	}
}
