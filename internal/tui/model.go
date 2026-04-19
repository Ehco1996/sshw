package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yinheli/sshw"
)

type listState struct {
	items  []list.Item
	cursor int
	title  string
}

type model struct {
	list     list.Model
	cols     *columnWidths
	stack    []listState
	selected *sshw.Node
	quitting bool
	width    int
	height   int
}

func newModel(nodes []*sshw.Node) model {
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

	// Remap left/right to jump to start/end (matching promptui behavior)
	l.KeyMap.GoToStart.SetKeys("left", "home", "g")
	l.KeyMap.GoToEnd.SetKeys("right", "end", "G")
	l.KeyMap.PrevPage.SetKeys("pgup")
	l.KeyMap.NextPage.SetKeys("pgdown")

	return model{
		list: l,
		cols: &cols,
	}
}

func (m *model) setItems(items []list.Item, title string) tea.Cmd {
	cols := computeColumns(items)
	*m.cols = cols
	cmd := m.list.SetItems(items)
	// Adjust list height for inline mode
	listHeight := min(len(items)+4, 24)
	m.list.SetSize(m.width, listHeight)
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

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Inline mode: cap list height to show at most 20 items
		listHeight := min(len(m.list.Items())+4, 24)
		m.list.SetSize(m.width, listHeight)
		return m, nil

	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, keys.Enter):
			selected, ok := m.list.SelectedItem().(item)
			if !ok {
				break
			}
			node := selected.node

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

		case key.Matches(msg, keys.Back):
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

func (m model) View() string {
	if m.quitting {
		return ""
	}
	if m.width == 0 {
		return "Loading..."
	}

	header := m.renderHeader()
	sep := separatorStyle.Render(strings.Repeat("─", m.width))
	body := m.list.View()
	help := m.renderHelp()

	return lipgloss.JoinVertical(lipgloss.Left, header, sep, body, sep, help)
}

func (m model) renderHeader() string {
	// Build breadcrumb path
	parts := make([]string, 0, len(m.stack)+1)
	parts = append(parts, headerTitleStyle.Render("sshw"))
	for _, s := range m.stack {
		parts = append(parts, headerPathStyle.Render(s.title))
	}
	if m.list.Title != "" && m.list.Title != "sshw" {
		parts = append(parts, headerPathStyle.Render(m.list.Title))
	}

	path := strings.Join(parts, headerSepStyle.Render(" ❯ "))

	// Item count on the right
	count := headerCountStyle.Render(strings.Repeat(" ", max(0,
		m.width-lipgloss.Width(path)-12)))

	return " " + path + count
}

func (m model) renderHelp() string {
	bindings := []struct{ key, desc string }{
		{"↑↓", "nav"},
		{"enter", "select"},
		{"esc", "back"},
		{"/", "filter"},
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
	finalModel := result.(model)
	return finalModel.selected, nil
}
