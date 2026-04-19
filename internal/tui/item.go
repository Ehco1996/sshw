package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yinheli/sshw"
)

// item wraps a *sshw.Node to satisfy the bubbles/list.Item interface.
type item struct {
	node *sshw.Node
}

func (i item) FilterValue() string {
	return fmt.Sprintf("%s %s %s %s %d", i.node.Name, i.node.Alias, i.node.User, i.node.Host, i.node.Port)
}

// indexedLeafItem is a flattened connectable host for the global palette.
type indexedLeafItem struct {
	idx IndexedHost
}

func (i indexedLeafItem) FilterValue() string {
	return filterValueForIndexed(i.idx)
}

func filterValueForIndexed(idx IndexedHost) string {
	n := idx.Node
	user := n.User
	if user == "" {
		user = "root"
	}
	port := ""
	if n.Port > 0 {
		port = fmt.Sprintf("%d", n.Port)
	}
	parts := []string{idx.Breadcrumb, n.Name, n.Alias, n.Host, user, port}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func nodesToItems(nodes []*sshw.Node) []item {
	items := make([]item, len(nodes))
	for i, n := range nodes {
		items[i] = item{node: n}
	}
	return items
}

func nodesToListItems(items []item) []list.Item {
	listItems := make([]list.Item, len(items))
	for i, it := range items {
		listItems[i] = it
	}
	return listItems
}

// columnWidths holds computed widths for tabular alignment.
type columnWidths struct {
	name  int
	alias int
	host  int
	user  int
}

func computeColumns(items []list.Item) columnWidths {
	cols := columnWidths{name: 8, host: 8, user: 4}
	for _, li := range items {
		var n *sshw.Node
		switch v := li.(type) {
		case item:
			n = v.node
		case indexedLeafItem:
			n = v.idx.Node
		default:
			continue
		}
		if w := lipgloss.Width(n.Name); w > cols.name {
			cols.name = w
		}
		if n.Alias != "" {
			w := lipgloss.Width("@" + n.Alias)
			if w > cols.alias {
				cols.alias = w
			}
		}
		if w := lipgloss.Width(n.Host); w > cols.host {
			cols.host = w
		}
		u := n.User
		if u == "" {
			u = "root"
		}
		if w := lipgloss.Width(u); w > cols.user {
			cols.user = w
		}
	}
	// Cap column widths to avoid overflow
	if cols.name > 30 {
		cols.name = 30
	}
	if cols.alias > 12 {
		cols.alias = 12
	}
	if cols.host > 25 {
		cols.host = 25
	}
	if cols.user > 12 {
		cols.user = 12
	}
	return cols
}

// compactDelegate renders each item as a single compact line with tabular alignment.
type compactDelegate struct {
	cols   *columnWidths
	health *healthState
}

func (d compactDelegate) Height() int  { return 1 }
func (d compactDelegate) Spacing() int { return 0 }
func (d compactDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

func (d compactDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	sel := index == m.Index()
	switch it := listItem.(type) {
	case item:
		node := it.node
		if len(node.Children) > 0 {
			d.renderDir(w, node, sel, m.Width())
		} else {
			d.renderHost(w, node, sel, m.Width())
		}
	case indexedLeafItem:
		d.renderIndexedLeaf(w, it.idx, sel, m.Width())
	default:
		return
	}
}

// countLeafHosts counts all leaf (non-directory) nodes recursively.
func countLeafHosts(nodes []*sshw.Node) int {
	count := 0
	for _, n := range nodes {
		if len(n.Children) > 0 {
			count += countLeafHosts(n.Children)
		} else {
			count++
		}
	}
	return count
}

// childPreview returns a comma-separated preview of child names, truncated to fit width.
func childPreview(nodes []*sshw.Node, maxWidth int) string {
	if len(nodes) == 0 {
		return ""
	}
	var parts []string
	totalWidth := 0
	for _, n := range nodes {
		name := n.Name
		// +2 for ", " separator
		needed := len(name) + 2
		if totalWidth+needed > maxWidth && len(parts) > 0 {
			remaining := len(nodes) - len(parts)
			if remaining > 0 {
				parts = append(parts, fmt.Sprintf("+%d", remaining))
			}
			break
		}
		parts = append(parts, name)
		totalWidth += needed
	}
	return strings.Join(parts, ", ")
}

func (d compactDelegate) renderDir(w io.Writer, node *sshw.Node, sel bool, width int) {
	name := node.Name
	totalHosts := countLeafHosts(node.Children)
	badge := fmt.Sprintf(" %d hosts ›", totalHosts)

	// Compute available space for preview
	cursor := " ▸ "
	if !sel {
		cursor = "   "
	}
	nameWidth := lipgloss.Width(name)
	badgeWidth := lipgloss.Width(badge)
	separatorWidth := 5 // " · " with padding
	previewMaxWidth := width - 3 - nameWidth - badgeWidth - separatorWidth
	preview := ""
	if previewMaxWidth > 10 {
		preview = childPreview(node.Children, previewMaxWidth)
	}

	var line string
	if sel {
		nameStr := selNameStyle.Render(name)
		var mid string
		if preview != "" {
			mid = selDirPreviewStyle.Render(" · " + preview)
		}
		badgeStr := selDirBadgeStyle.Render(badge)
		usedWidth := 3 + lipgloss.Width(nameStr) + lipgloss.Width(mid) + lipgloss.Width(badgeStr)
		gap := max(0, width-usedWidth)
		line = cursorStyle.Render(cursor) + nameStr + mid + strings.Repeat(" ", gap) + badgeStr
	} else {
		nameStr := norNameStyle.Render(name)
		var mid string
		if preview != "" {
			mid = norDirPreviewStyle.Render(" · " + preview)
		}
		badgeStr := norDirBadgeStyle.Render(badge)
		usedWidth := 3 + lipgloss.Width(nameStr) + lipgloss.Width(mid) + lipgloss.Width(badgeStr)
		gap := max(0, width-usedWidth)
		line = cursor + nameStr + mid + strings.Repeat(" ", gap) + badgeStr
	}
	fmt.Fprint(w, applyRowHighlight(line, sel, width))
}

func applyRowHighlight(line string, sel bool, termWidth int) string {
	if termWidth > 0 && lipgloss.Width(line) > termWidth {
		line = truncateWithWidth(line, termWidth)
	}
	if !sel || termWidth <= 0 {
		return line
	}
	return selRowStyle.Width(termWidth).Render(line)
}

func truncateWithWidth(s string, maxW int) string {
	w := lipgloss.Width(s)
	if w <= maxW {
		return s
	}
	// Truncate rune by rune
	result := []rune(s)
	for lipgloss.Width(string(result)) > maxW-1 && len(result) > 0 {
		result = result[:len(result)-1]
	}
	return string(result) + "…"
}

func (d compactDelegate) renderHost(w io.Writer, node *sshw.Node, sel bool, termWidth int) {
	cols := d.cols

	// Cap name column to avoid overflow on narrow terminals
	nameW := cols.name + 2
	maxNameW := termWidth/3 - 3
	if nameW > maxNameW && maxNameW > 8 {
		nameW = maxNameW
	}

	name := node.Name
	if lipgloss.Width(name) > nameW-2 {
		name = truncateWithWidth(name, nameW-2)
	}
	host := node.Host
	user := node.User
	if user == "" {
		user = "root"
	}
	port := ""
	if node.Port > 0 && node.Port != 22 {
		port = fmt.Sprintf(":%d", node.Port)
	} else {
		port = ":22"
	}

	var jump string
	if len(node.Jump) > 0 {
		jName := node.Jump[0].Name
		if jName == "" {
			jName = node.Jump[0].Host
		}
		jump = " → " + jName
	}

	nameCol := lipgloss.NewStyle().Width(nameW)
	hostCol := lipgloss.NewStyle().Width(cols.host + 2)
	userCol := lipgloss.NewStyle().Width(cols.user + 2)

	aliasW := cols.alias
	aliasCell := ""
	if aliasW > 0 {
		at := ""
		if node.Alias != "" {
			at = "@" + node.Alias
			if lipgloss.Width(at) > aliasW {
				at = truncateWithWidth(at, aliasW)
			}
		}
		aliasCol := lipgloss.NewStyle().Width(aliasW + 1)
		if sel {
			aliasCell = aliasCol.Render(selAliasStyle.Render(at))
		} else {
			aliasCell = aliasCol.Render(norAliasStyle.Render(at))
		}
	}

	var line string
	if sel {
		line = fmt.Sprintf("%s%s%s%s%s%s%s",
			cursorStyle.Render(" ▸ "),
			nameCol.Render(selNameStyle.Render(name)),
			aliasCell,
			hostCol.Render(selHostStyle.Render(host)),
			userCol.Render(selUserStyle.Render(user)),
			selPortStyle.Render(port),
			selJumpStyle.Render(jump),
		)
	} else {
		line = fmt.Sprintf("   %s%s%s%s%s%s",
			nameCol.Render(norNameStyle.Render(name)),
			aliasCell,
			hostCol.Render(norHostStyle.Render(host)),
			userCol.Render(norUserStyle.Render(user)),
			norPortStyle.Render(port),
			norJumpStyle.Render(jump),
		)
	}

	if d.health != nil {
		if r, ok := d.health.results[node]; ok {
			line += renderHealthIndicator(r, d.health.spinner)
		}
	}

	fmt.Fprint(w, applyRowHighlight(line, sel, termWidth))
}

func (d compactDelegate) renderIndexedLeaf(w io.Writer, idx IndexedHost, sel bool, termWidth int) {
	n := idx.Node
	cols := d.cols

	bc := idx.Breadcrumb
	bcMax := max(0, termWidth/3)
	if bc != "" && lipgloss.Width(bc) > bcMax {
		bc = truncateWithWidth(bc, bcMax)
	}

	nameW := cols.name + 2
	maxNameW := termWidth/4 - 3
	if nameW > maxNameW && maxNameW > 8 {
		nameW = maxNameW
	}
	name := n.Name
	if lipgloss.Width(name) > nameW-2 {
		name = truncateWithWidth(name, nameW-2)
	}
	host := n.Host
	user := n.User
	if user == "" {
		user = "root"
	}
	port := ""
	if n.Port > 0 && n.Port != 22 {
		port = fmt.Sprintf(":%d", n.Port)
	} else {
		port = ":22"
	}
	var jump string
	if len(n.Jump) > 0 {
		jName := n.Jump[0].Name
		if jName == "" {
			jName = n.Jump[0].Host
		}
		jump = " → " + jName
	}

	bcCol := lipgloss.NewStyle().Width(min(lipgloss.Width(bc)+2, bcMax+2))
	nameCol := lipgloss.NewStyle().Width(nameW)
	hostCol := lipgloss.NewStyle().Width(cols.host + 2)
	userCol := lipgloss.NewStyle().Width(cols.user + 2)

	aliasW := cols.alias
	aliasCell := ""
	if aliasW > 0 {
		at := ""
		if n.Alias != "" {
			at = "@" + n.Alias
			if lipgloss.Width(at) > aliasW {
				at = truncateWithWidth(at, aliasW)
			}
		}
		aliasCol := lipgloss.NewStyle().Width(aliasW + 1)
		if sel {
			aliasCell = aliasCol.Render(selAliasStyle.Render(at))
		} else {
			aliasCell = aliasCol.Render(norAliasStyle.Render(at))
		}
	}

	bcPrefix := ""
	if bc != "" {
		bcPrefix = bcCol.Render(breadcrumbStyle.Render(bc)) + " "
	}

	var line string
	if sel {
		line = fmt.Sprintf("%s%s%s%s%s%s%s%s",
			cursorStyle.Render(" ▸ "),
			bcPrefix,
			nameCol.Render(selNameStyle.Render(name)),
			aliasCell,
			hostCol.Render(selHostStyle.Render(host)),
			userCol.Render(selUserStyle.Render(user)),
			selPortStyle.Render(port),
			selJumpStyle.Render(jump),
		)
	} else {
		line = fmt.Sprintf("   %s%s%s%s%s%s%s",
			bcPrefix,
			nameCol.Render(norNameStyle.Render(name)),
			aliasCell,
			hostCol.Render(norHostStyle.Render(host)),
			userCol.Render(norUserStyle.Render(user)),
			norPortStyle.Render(port),
			norJumpStyle.Render(jump),
		)
	}

	if d.health != nil {
		if r, ok := d.health.results[n]; ok {
			line += renderHealthIndicator(r, d.health.spinner)
		}
	}

	fmt.Fprint(w, applyRowHighlight(line, sel, termWidth))
}
