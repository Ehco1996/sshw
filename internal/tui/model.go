package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yinheli/sshw"
)

const (
	chromeLines  = 4 // header + sep + body + sep + help
	maxBodyLines = 28 // cap list height on very tall terminals
)

type listState struct {
	items  []list.Item
	cursor int
	title  string
}

type model struct {
	list       list.Model
	cols       *columnWidths
	stack      []listState
	selected   *sshw.Node
	quitting   bool
	width      int
	height     int
	roots      []*sshw.Node
	globalMode bool
	preGlobal  *listState
}

func newModel(nodes []*sshw.Node) *model {
	items := nodesToListItems(nodesToItems(nodes))
	cols := computeColumns(items)

	delegate := compactDelegate{cols: &cols}
	l := list.New(items, delegate, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Filter = multiKeywordFilter
	l.SetShowHelp(false)
	l.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(colorPrimary)
	l.Styles.FilterCursor = lipgloss.NewStyle().Foreground(colorPrimary)

	l.KeyMap.GoToStart.SetKeys("left", "home", "g")
	l.KeyMap.GoToEnd.SetKeys("right", "end", "G")
	l.KeyMap.PrevPage.SetKeys("pgup")
	l.KeyMap.NextPage.SetKeys("pgdown")

	return &model{
		list:  l,
		cols:  &cols,
		roots: nodes,
	}
}

func globalPaletteFilter(term string, targets []string) []list.Rank {
	trimmed := strings.TrimSpace(term)
	if trimmed == "" {
		return multiKeywordFilter("", targets)
	}
	if strings.Contains(trimmed, " ") {
		return multiKeywordFilter(trimmed, targets)
	}
	return list.DefaultFilter(trimmed, targets)
}

func (m *model) syncLayout() {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	avail := m.height - chromeLines
	if avail < 1 {
		avail = 1
	}
	listH := max(3, min(maxBodyLines, avail))
	m.list.SetSize(m.width, listH)
}

func (m *model) setItems(items []list.Item, title string) tea.Cmd {
	cols := computeColumns(items)
	*m.cols = cols
	cmd := m.list.SetItems(items)
	m.syncLayout()
	return cmd
}

func (m *model) toggleGlobalPalette() tea.Cmd {
	if m.globalMode {
		return m.exitGlobalPalette()
	}
	m.preGlobal = &listState{
		items:  m.list.Items(),
		cursor: m.list.Index(),
		title:  m.list.Title,
	}
	hosts := FlattenLeaves(m.roots)
	items := make([]list.Item, len(hosts))
	for i := range hosts {
		items[i] = indexedLeafItem{idx: hosts[i]}
	}
	m.list.Filter = globalPaletteFilter
	m.globalMode = true
	cmd := m.setItems(items, "__global__")
	m.list.Select(0)
	m.list.ResetFilter()
	return cmd
}

func (m *model) exitGlobalPalette() tea.Cmd {
	if !m.globalMode || m.preGlobal == nil {
		return nil
	}
	prev := *m.preGlobal
	m.preGlobal = nil
	m.globalMode = false
	m.list.Filter = multiKeywordFilter
	cmd := m.setItems(prev.items, prev.title)
	m.list.Select(prev.cursor)
	m.list.ResetFilter()
	return cmd
}

func multiKeywordFilter(term string, targets []string) []list.Rank {
	var ranks []list.Rank
	for i, t := range targets {
		if matchMultiKeyword(term, t) {
			ranks = append(ranks, list.Rank{Index: i})
		}
	}
	return ranks
}

func matchMultiKeyword(input, content string) bool {
	input = strings.ToLower(input)
	content = strings.ToLower(content)
	if strings.Contains(input, " ") {
		for _, k := range strings.Split(input, " ") {
			k = strings.TrimSpace(k)
			if k != "" && !strings.Contains(content, k) {
				return false
			}
		}
		return true
	}
	return strings.Contains(content, input)
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.syncLayout()
		return m, nil

	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, keys.GlobalPalette):
			cmd := m.toggleGlobalPalette()
			return m, cmd

		case key.Matches(msg, keys.Enter):
			sel := m.list.SelectedItem()
			if sel == nil {
				break
			}
			switch v := sel.(type) {
			case indexedLeafItem:
				m.selected = v.idx.Node
				m.quitting = true
				return m, tea.Quit
			case item:
				node := v.node
				if len(node.Children) > 0 {
					m.stack = append(m.stack, listState{
						items:  m.list.Items(),
						cursor: m.list.Index(),
						title:  m.list.Title,
					})
					childItems := nodesToListItems(nodesToItems(node.Children))
					cmd := m.setItems(childItems, node.Name)
					m.list.Select(0)
					m.list.Title = node.Name
					return m, cmd
				}
				m.selected = node
				m.quitting = true
				return m, tea.Quit
			}

		case key.Matches(msg, keys.Back):
			if m.globalMode {
				cmd := m.exitGlobalPalette()
				return m, cmd
			}
			if len(m.stack) == 0 {
				m.quitting = true
				return m, tea.Quit
			}
			prev := m.stack[len(m.stack)-1]
			m.stack = m.stack[:len(m.stack)-1]
			cmd := m.setItems(prev.items, prev.title)
			m.list.Select(prev.cursor)
			m.list.Title = prev.title
			return m, cmd

		case key.Matches(msg, keys.Quit):
			m.quitting = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *model) View() string {
	if m.quitting {
		return ""
	}
	if m.width == 0 {
		return "Loading..."
	}

	header := m.renderHeader()
	topSep := separatorStyle.Render(strings.Repeat("─", m.width))
	body := m.list.View()
	botSep := separatorStyle.Render(strings.Repeat("─", m.width))
	help := m.renderHelp()

	return lipgloss.JoinVertical(lipgloss.Left, header, topSep, body, botSep, help)
}

func (m *model) renderHeader() string {
	parts := make([]string, 0, len(m.stack)+3)
	parts = append(parts, headerTitleStyle.Render("sshw"))
	if m.globalMode {
		parts = append(parts, headerPathStyle.Render("GLOBAL"))
		path := strings.Join(parts, headerSepStyle.Render(" ❯ "))
		count := headerCountStyle.Render(strings.Repeat(" ", max(0,
			m.width-lipgloss.Width(path)-12)))
		return " " + path + count
	}
	for _, s := range m.stack {
		parts = append(parts, headerPathStyle.Render(s.title))
	}
	if m.list.Title != "" && m.list.Title != "sshw" && m.list.Title != "__global__" {
		parts = append(parts, headerPathStyle.Render(m.list.Title))
	}

	path := strings.Join(parts, headerSepStyle.Render(" ❯ "))
	count := headerCountStyle.Render(strings.Repeat(" ", max(0,
		m.width-lipgloss.Width(path)-12)))

	return " " + path + count
}

func (m *model) renderHelp() string {
	bindings := []struct{ key, desc string }{
		{"↑↓", "nav"},
		{"enter", "select"},
		{"esc", "back"},
		{"/", "name or alias"},
		{"ctrl+k", "global"},
		{"q", "quit"},
	}

	var parts []string
	for _, b := range bindings {
		parts = append(parts, helpKeyStyle.Render(b.key)+" "+helpDescStyle.Render(b.desc))
	}
	return " " + strings.Join(parts, helpDescStyle.Render("  ·  "))
}

// Run starts the TUI and returns the selected node, or nil if the user quit.
func Run(nodes []*sshw.Node) (*sshw.Node, error) {
	m := newModel(nodes)
	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return nil, err
	}
	finalModel := result.(*model)
	return finalModel.selected, nil
}
