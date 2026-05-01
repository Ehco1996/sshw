package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/yinheli/sshw"
)

// firstLine returns the first non-empty line of s, with trailing whitespace trimmed.
func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimRight(line, " \t\r")
		if line != "" {
			return line
		}
	}
	return ""
}

// renderHostLabel formats a node like "name  user@host:port".
func renderHostLabel(n *sshw.Node) string {
	port := ""
	if n.Port > 0 && n.Port != 22 {
		port = fmt.Sprintf(":%d", n.Port)
	} else {
		port = ":22"
	}
	return fmt.Sprintf("%s  %s@%s%s", n.Name, n.EffectiveUser(), n.Host, port)
}

// renderBatchPrompt is shown while the user types the command.
func (m *model) renderBatchPrompt() string {
	count := len(m.batch.targets)
	hint := batchHintStyle.Render(fmt.Sprintf("running on %d host(s) — enter to confirm, esc to cancel", count))
	prompt := batchPromptStyle.Render("$ ") + m.batch.input.View()
	body := lipgloss.JoinVertical(lipgloss.Left, hint, "", prompt)
	return m.frame(body)
}

// renderBatchConfirm shows the command, target list, and yes/no prompt.
// When the command was flagged dangerous, it switches to a typed-override
// screen highlighting the matched substring.
func (m *model) renderBatchConfirm() string {
	cmdLine := batchPromptStyle.Render("$ ") + renderCmdHighlighted(m.batch.cmdLine, m.batch.dangerous)
	count := len(m.batch.targets)

	maxList := 10
	var rows []string
	for i, n := range m.batch.targets {
		if i >= maxList {
			rows = append(rows, batchMetaStyle.Render(fmt.Sprintf("  … and %d more", count-maxList)))
			break
		}
		rows = append(rows, "  "+renderHostLabel(n))
	}

	var confirm string
	if m.batch.dangerous != "" {
		warn := batchExitFailStyle.Render("⚠ ") +
			batchCmdStyle.Render("This command matches a destructive pattern (") +
			batchExitFailStyle.Render(m.batch.dangerous) +
			batchCmdStyle.Render(").")
		instruction := batchPromptStyle.Render("Type ") +
			batchExitFailStyle.Render(`"`+dangerConfirmPhrase+`"`) +
			batchPromptStyle.Render(" to proceed (esc to edit):")
		input := batchPromptStyle.Render("> ") + m.batch.confirmInput.View()
		mismatch := ""
		if m.batch.confirmFailed {
			mismatch = batchExitFailStyle.Render("did not match — try again or esc")
		}
		confirm = lipgloss.JoinVertical(lipgloss.Left, warn, "", instruction, input, mismatch)
	} else {
		confirm = batchPromptStyle.Render("Run on ") +
			batchCmdStyle.Render(fmt.Sprintf("%d", count)) +
			batchPromptStyle.Render(" host(s)? ") +
			batchHintStyle.Render("[y/N]")
	}

	header := fmt.Sprintf("about to run on %d host(s):", count)
	body := lipgloss.JoinVertical(lipgloss.Left,
		cmdLine,
		"",
		batchHintStyle.Render(header),
		strings.Join(rows, "\n"),
		"",
		confirm,
	)
	return m.frame(body)
}

// renderCmdHighlighted renders cmd with the dangerous substring (if any)
// underlined + colored red. Plain text when no danger.
func renderCmdHighlighted(cmd, danger string) string {
	if danger == "" {
		return batchCmdStyle.Render(cmd)
	}
	idx := strings.Index(cmd, danger)
	if idx < 0 {
		return batchCmdStyle.Render(cmd)
	}
	return batchCmdStyle.Render(cmd[:idx]) +
		dangerHighlightStyle.Render(danger) +
		batchCmdStyle.Render(cmd[idx+len(danger):])
}

// renderBatchRunning renders the per-host progress while commands are running.
func (m *model) renderBatchRunning() string {
	cmdLine := batchPromptStyle.Render("$ ") + batchCmdStyle.Render(m.batch.cmdLine)

	rows := make([]string, 0, len(m.batch.targets))
	for _, n := range m.batch.targets {
		r := m.batch.results[n]
		var status string
		if r == nil || !r.done {
			status = m.batch.spinner.View() + " " + batchHintStyle.Render("running")
		} else {
			status = renderResultBadge(r.res)
		}
		rows = append(rows, fmt.Sprintf("%s  %s", status, renderHostLabel(n)))
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		cmdLine,
		"",
		strings.Join(rows, "\n"),
	)
	return m.frame(body)
}

// renderResultBadge returns "✓ exit=0" / "✗ exit=N" / "✗ err".
func renderResultBadge(r sshw.RunResult) string {
	if r.Err != nil && r.ExitCode == -1 {
		// reuse the healthcheck reason classifier for connection errors
		msg := r.Err.Error()
		reason := "error"
		switch {
		case strings.Contains(msg, "context deadline exceeded") || strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline"):
			reason = "timeout"
		case strings.Contains(msg, "refused"):
			reason = "refused"
		case strings.Contains(msg, "no route"):
			reason = "no route"
		case strings.Contains(msg, "auth") || strings.Contains(msg, "unable to authenticate"):
			reason = "auth"
		}
		return batchExitFailStyle.Render("✗ " + reason)
	}
	if r.ExitCode == 0 {
		return batchExitOKStyle.Render("✓ exit=0")
	}
	return batchExitFailStyle.Render(fmt.Sprintf("✗ exit=%d", r.ExitCode))
}

// renderBatchResults shows the summary list with a movable cursor.
// When groupView is on, switches to a bucket-grouped layout (dshbak-mode).
func (m *model) renderBatchResults() string {
	cmdLine := batchPromptStyle.Render("$ ") + batchCmdStyle.Render(m.batch.cmdLine)
	total, _, ok, fail := m.batch.counts()
	summary := batchHintStyle.Render(fmt.Sprintf("%d host(s) · ", total)) +
		batchExitOKStyle.Render(fmt.Sprintf("✓ %d", ok)) +
		batchHintStyle.Render(" · ") +
		batchExitFailStyle.Render(fmt.Sprintf("✗ %d", fail))
	if m.batch.resultsFilter == 1 {
		summary += "  " + batchExitFailStyle.Render("[failed only]")
	}
	if m.batch.groupView {
		summary += "  " + batchSectionStyle.Render("[grouped]")
	}

	if m.batch.groupView {
		return m.renderBatchResultsGrouped(cmdLine, summary)
	}

	visible := m.visibleResults()
	rows := make([]string, 0, len(visible))
	width := m.width
	if width <= 0 {
		width = 80
	}
	if len(visible) == 0 && m.batch.resultsFilter == 1 {
		rows = append(rows, batchHintStyle.Render("  no failed hosts — press f to clear filter"))
	}
	for i, n := range visible {
		r := m.batch.results[n]
		var badge, preview, dur string
		if r != nil && r.done {
			badge = renderResultBadge(r.res)
			d := r.res.Duration.Round(time.Millisecond)
			dur = batchMetaStyle.Render(d.String())
			preview = firstLine(string(r.res.Stdout))
			if preview == "" && r.res.Err != nil {
				preview = r.res.Err.Error()
			} else if preview == "" {
				preview = firstLine(string(r.res.Stderr))
			}
		} else {
			badge = batchHintStyle.Render("…")
			preview = ""
		}

		label := renderHostLabel(n)
		// Available width for preview: total - cursor(2) - badge - label - dur - separators
		used := 2 + lipgloss.Width(badge) + 2 + lipgloss.Width(label) + 2 + lipgloss.Width(dur) + 2
		previewMax := width - used
		if previewMax > 4 && preview != "" {
			if lipgloss.Width(preview) > previewMax {
				preview = truncateWithWidth(preview, previewMax)
			}
		} else {
			preview = ""
		}

		cursor := "  "
		if i == m.batch.resultIdx {
			cursor = cursorStyle.Render("▸ ")
		}
		line := cursor + badge + "  " + label
		if preview != "" {
			line += "  " + batchMetaStyle.Render(preview)
		}
		if dur != "" {
			line += "  " + dur
		}
		if i == m.batch.resultIdx && width > 0 {
			line = applyRowHighlight(line, true, width)
		}
		rows = append(rows, line)
	}

	parts := []string{cmdLine, summary}
	if hint := m.batch.logHint(); hint != "" {
		parts = append(parts, hint)
	}
	if m.batch.flash != "" {
		parts = append(parts, batchExitFailStyle.Render(m.batch.flash))
	}
	parts = append(parts, "", strings.Join(rows, "\n"))
	body := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return m.frame(body)
}

// renderBatchResultsGrouped renders the dshbak-style bucket list. Each
// bucket gets one row showing host count + class badge + first-line preview
// + duration of the exemplar host. Largest bucket first.
func (m *model) renderBatchResultsGrouped(cmdLine, summary string) string {
	buckets := m.visibleBuckets()
	width := m.width
	if width <= 0 {
		width = 80
	}

	rows := make([]string, 0, len(buckets))
	if len(buckets) == 0 {
		rows = append(rows, batchHintStyle.Render("  no completed results"))
	}
	for i, bk := range buckets {
		badge := renderResultBadge(bk.exemplar)
		count := fmt.Sprintf("[%d %s]", len(bk.hosts), pluralize("host", len(bk.hosts)))
		dur := batchMetaStyle.Render(bk.exemplar.Duration.Round(time.Millisecond).String())
		preview := firstLine(stripAnsi(string(bk.exemplar.Stdout)))
		if preview == "" && bk.exemplar.Err != nil {
			preview = bk.exemplar.Err.Error()
		} else if preview == "" {
			preview = firstLine(stripAnsi(string(bk.exemplar.Stderr)))
		}

		used := 2 + lipgloss.Width(count) + 2 + lipgloss.Width(badge) + 2 + lipgloss.Width(dur) + 2
		previewMax := width - used
		if previewMax > 4 && preview != "" && lipgloss.Width(preview) > previewMax {
			preview = truncateWithWidth(preview, previewMax)
		} else if previewMax <= 4 {
			preview = ""
		}

		cursor := "  "
		if i == m.batch.bucketIdx {
			cursor = cursorStyle.Render("▸ ")
		}
		line := cursor + batchSectionStyle.Render(count) + "  " + badge
		if preview != "" {
			line += "  " + batchMetaStyle.Render(preview)
		}
		if dur != "" {
			line += "  " + dur
		}
		if i == m.batch.bucketIdx {
			line = applyRowHighlight(line, true, width)
		}
		rows = append(rows, line)
	}

	parts := []string{cmdLine, summary}
	if hint := m.batch.logHint(); hint != "" {
		parts = append(parts, hint)
	}
	if m.batch.flash != "" {
		parts = append(parts, batchExitFailStyle.Render(m.batch.flash))
	}
	parts = append(parts, "", strings.Join(rows, "\n"))
	return m.frame(lipgloss.JoinVertical(lipgloss.Left, parts...))
}

func pluralize(word string, n int) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

// renderBatchDetail shows a single host's full output via the viewport.
// If the user drilled in from a grouped bucket, the header lists every
// host in the bucket so they can see who shares this output.
func (m *model) renderBatchDetail() string {
	n := m.batch.detailNode
	if n == nil {
		return m.frame(batchHintStyle.Render("no host selected"))
	}
	cmdLine := batchPromptStyle.Render("$ ") + batchCmdStyle.Render(m.batch.cmdLine)

	var headerLines []string
	if len(m.batch.bucketHosts) > 1 {
		headerLines = append(headerLines,
			batchSectionStyle.Render(fmt.Sprintf("%d hosts in this bucket", len(m.batch.bucketHosts))),
			batchMetaStyle.Render(hostListLabel(m.batch.bucketHosts, 12)),
		)
	} else {
		headerLines = append(headerLines, batchSectionStyle.Render(renderHostLabel(n)))
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		append(append([]string{}, headerLines...), cmdLine, "", m.batch.detail.View())...,
	)
	return m.frame(body)
}

// detailContent assembles the viewport content for a single host's result.
func detailContent(cmd string, r sshw.RunResult) string {
	var b strings.Builder
	if len(r.Stdout) > 0 {
		b.WriteString(batchSectionStyle.Render("--- stdout ---") + "\n")
		b.WriteString(string(r.Stdout))
		if !strings.HasSuffix(string(r.Stdout), "\n") {
			b.WriteString("\n")
		}
	}
	if len(r.Stderr) > 0 {
		b.WriteString(batchSectionStyle.Render("--- stderr ---") + "\n")
		b.WriteString(string(r.Stderr))
		if !strings.HasSuffix(string(r.Stderr), "\n") {
			b.WriteString("\n")
		}
	}
	b.WriteString(batchSectionStyle.Render("--- meta ---") + "\n")
	b.WriteString(fmt.Sprintf("exit=%d  duration=%s",
		r.ExitCode, r.Duration.Round(time.Millisecond)))
	if r.Err != nil {
		b.WriteString("\nerr=" + r.Err.Error())
	}
	if len(r.Stdout) == 0 && len(r.Stderr) == 0 {
		b.WriteString("\n(no output)")
	}
	_ = cmd // header already prints the command above the viewport
	return b.String()
}

// frame wraps a body with the standard header / separators / batch-help footer.
func (m *model) frame(body string) string {
	header := m.renderHeader()
	topSep := separatorStyle.Render(strings.Repeat("─", m.width))
	botSep := separatorStyle.Render(strings.Repeat("─", m.width))
	help := m.renderBatchHelp()
	return lipgloss.JoinVertical(lipgloss.Left, header, topSep, body, botSep, help)
}

func (m *model) renderBatchHelp() string {
	var bindings []struct{ key, desc string }
	switch m.mode {
	case modeBatchPrompt:
		bindings = []struct{ key, desc string }{
			{"enter", "confirm"},
			{"esc", "cancel"},
		}
	case modeBatchConfirm:
		if m.batch.dangerous != "" {
			bindings = []struct{ key, desc string }{
				{"type phrase", "confirm"},
				{"esc", "edit cmd"},
			}
		} else {
			bindings = []struct{ key, desc string }{
				{"y", "run"},
				{"n/esc", "edit cmd"},
			}
		}
	case modeBatchRunning:
		bindings = []struct{ key, desc string }{
			{"esc", "cancel"},
		}
	case modeBatchResults:
		bindings = []struct{ key, desc string }{
			{"↑↓", "nav"},
			{"enter", "detail"},
			{"g", "group"},
			{"f", "filter ✗"},
			{"r/R", "rerun/failed"},
			{"esc", "back"},
		}
	case modeBatchDetail:
		bindings = []struct{ key, desc string }{
			{"↑↓/pgup/pgdn", "scroll"},
			{"esc", "back"},
		}
	default:
		return m.renderHelp()
	}
	parts := make([]string, 0, len(bindings))
	for _, b := range bindings {
		parts = append(parts, helpKeyStyle.Render(b.key)+" "+helpDescStyle.Render(b.desc))
	}
	return " " + strings.Join(parts, helpDescStyle.Render("  ·  "))
}
