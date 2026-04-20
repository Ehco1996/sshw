package tui

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yinheli/sshw"
)

type healthResult struct {
	done    bool
	ok      bool
	latency time.Duration
	err     error
}

type healthState struct {
	results    map[*sshw.Node]*healthResult
	active     bool
	generation int
	spinner    spinner.Model
}

func newHealthState() *healthState {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorPrimary)
	return &healthState{
		results: make(map[*sshw.Node]*healthResult),
		spinner: s,
	}
}

func (h *healthState) reset() {
	h.results = make(map[*sshw.Node]*healthResult)
	h.active = false
}

func (h *healthState) counts() (total, ok, fail int) {
	for _, r := range h.results {
		total++
		if r.done {
			if r.ok {
				ok++
			} else {
				fail++
			}
		}
	}
	return
}

type healthCheckResultMsg struct {
	node       *sshw.Node
	generation int
	err        error
	latency    time.Duration
}

func checkHostCmd(node *sshw.Node, gen int) tea.Cmd {
	return func() tea.Msg {
		addr := net.JoinHostPort(node.Host, fmt.Sprintf("%d", node.SSHPort()))
		start := time.Now()
		conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
		latency := time.Since(start)
		if err != nil {
			return healthCheckResultMsg{node: node, generation: gen, err: err, latency: latency}
		}
		conn.Close()
		return healthCheckResultMsg{node: node, generation: gen, latency: latency}
	}
}

func renderHealthIndicator(r *healthResult, sp spinner.Model) string {
	if r == nil {
		return ""
	}
	if !r.done {
		return "  " + sp.View()
	}
	if r.ok {
		return "  " + healthOKStyle.Render("✓ "+r.latency.Round(time.Millisecond).String())
	}
	reason := "fail"
	if r.err != nil {
		msg := r.err.Error()
		switch {
		case strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline"):
			reason = "timeout"
		case strings.Contains(msg, "refused"):
			reason = "refused"
		}
	}
	return "  " + healthFailStyle.Render("✗ "+reason)
}
