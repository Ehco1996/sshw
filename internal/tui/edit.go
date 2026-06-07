package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yinheli/sshw"
)

type editState struct {
	inputs   []textinput.Model
	fieldIdx int
	creating bool
	target   *sshw.Node
	errMsg   string
	flash    string

	// delete confirmation
	deleteTarget *sshw.Node
}

func newEditState() *editState {
	inputs := make([]textinput.Model, sshw.EditFieldCount)
	for i := range inputs {
		in := textinput.New()
		in.CharLimit = 512
		inputs[i] = in
	}
	inputs[sshw.EditFieldPassphrase].EchoMode = textinput.EchoPassword
	inputs[sshw.EditFieldPassword].EchoMode = textinput.EchoPassword
	inputs[sshw.EditFieldKeyPath].Placeholder = "~/.ssh/id_rsa"
	inputs[sshw.EditFieldAgentPath].Placeholder = "optional ssh-agent socket"
	return &editState{inputs: inputs}
}

func (e *editState) reset() {
	e.fieldIdx = 0
	e.creating = false
	e.target = nil
	e.errMsg = ""
	e.flash = ""
	e.deleteTarget = nil
	for i := range e.inputs {
		e.inputs[i].Reset()
		e.inputs[i].Blur()
	}
}

func (e *editState) syncLayout(width int) {
	w := max(20, width-12)
	for i := range e.inputs {
		e.inputs[i].Width = w
	}
}

func (e *editState) focusField(idx sshw.EditField) tea.Cmd {
	if idx < 0 || idx >= sshw.EditFieldCount {
		return nil
	}
	e.fieldIdx = int(idx)
	var cmds []tea.Cmd
	for i := range e.inputs {
		e.inputs[i].Blur()
	}
	e.inputs[idx].Focus()
	cmds = append(cmds, textinput.Blink)
	return tea.Batch(cmds...)
}

func (e *editState) beginCreate() tea.Cmd {
	e.reset()
	e.creating = true
	e.inputs[sshw.EditFieldName].Placeholder = "required"
	e.inputs[sshw.EditFieldHost].Placeholder = "leave empty for a group"
	return e.focusField(sshw.EditFieldName)
}

func (e *editState) beginEdit(n *sshw.Node) tea.Cmd {
	e.reset()
	e.creating = false
	e.target = n
	v := sshw.NodeToEditFormValues(n)
	e.inputs[sshw.EditFieldName].SetValue(v.Name)
	e.inputs[sshw.EditFieldHost].SetValue(v.Host)
	e.inputs[sshw.EditFieldUser].SetValue(v.User)
	e.inputs[sshw.EditFieldPort].SetValue(v.Port)
	e.inputs[sshw.EditFieldAlias].SetValue(v.Alias)
	e.inputs[sshw.EditFieldKeyPath].SetValue(v.KeyPath)
	e.inputs[sshw.EditFieldAgentPath].SetValue(v.AgentPath)
	e.inputs[sshw.EditFieldPassphrase].SetValue(v.Passphrase)
	e.inputs[sshw.EditFieldPassword].SetValue(v.Password)
	if len(n.Children) > 0 {
		e.inputs[sshw.EditFieldHost].Placeholder = "(group — host ignored)"
	}
	return e.focusField(sshw.EditFieldName)
}

func (e *editState) beginDelete(n *sshw.Node) {
	e.reset()
	e.deleteTarget = n
}

func (e *editState) formValues() sshw.EditFormValues {
	return sshw.EditFormValues{
		Name:       e.inputs[sshw.EditFieldName].Value(),
		Host:       e.inputs[sshw.EditFieldHost].Value(),
		User:       e.inputs[sshw.EditFieldUser].Value(),
		Port:       e.inputs[sshw.EditFieldPort].Value(),
		Alias:      e.inputs[sshw.EditFieldAlias].Value(),
		KeyPath:    e.inputs[sshw.EditFieldKeyPath].Value(),
		AgentPath:  e.inputs[sshw.EditFieldAgentPath].Value(),
		Passphrase: e.inputs[sshw.EditFieldPassphrase].Value(),
		Password:   e.inputs[sshw.EditFieldPassword].Value(),
	}
}

func (e *editState) isGroupForm() bool {
	return sshw.IsGroupForm(e.target, e.inputs[sshw.EditFieldHost].Value(), e.creating)
}

func (e *editState) visibleFields() []sshw.EditField {
	return sshw.VisibleEditFields(e.isGroupForm())
}

func (e *editState) nextField(delta int) tea.Cmd {
	fields := e.visibleFields()
	cur := 0
	for i, f := range fields {
		if int(f) == e.fieldIdx {
			cur = i
			break
		}
	}
	next := (cur + delta + len(fields)) % len(fields)
	return e.focusField(fields[next])
}

func (e *editState) applyForm() (*sshw.Node, error) {
	return sshw.ApplyEditForm(e.formValues(), e.target, e.creating, e.isGroupForm())
}

func (m *model) renderEditForm() string {
	e := m.edit
	title := batchSectionStyle.Render("Edit node")
	if e.creating {
		title = batchSectionStyle.Render("Add node")
	}
	var rows []string
	for _, f := range e.visibleFields() {
		label := sshw.EditFieldLabel(f)
		prefix := "  "
		if int(f) == e.fieldIdx {
			prefix = batchPromptStyle.Render("▸ ")
		}
		line := prefix + batchHintStyle.Render(label+": ") + e.inputs[f].View()
		rows = append(rows, line)
	}

	parts := []string{title, "", strings.Join(rows, "\n")}
	if e.errMsg != "" {
		parts = append(parts, "", batchExitFailStyle.Render(e.errMsg))
	} else if e.flash != "" {
		parts = append(parts, "", batchHintStyle.Render(e.flash))
	}
	body := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return m.frame(body)
}

func (m *model) renderEditDeleteConfirm() string {
	n := m.edit.deleteTarget
	if n == nil {
		return m.frame("")
	}
	label := n.Name
	if len(n.Children) > 0 {
		label = fmt.Sprintf("%s (%d hosts inside)", n.Name, countLeafHosts(n.Children))
	}
	msg := batchPromptStyle.Render("Delete ") +
		batchCmdStyle.Render(label) +
		batchPromptStyle.Render("? ") +
		batchHintStyle.Render("[y/enter to delete · n/esc to cancel]")
	return m.frame(msg)
}

func (m *model) editActiveKeys() modeKeys {
	switch m.mode {
	case modeEditDeleteConfirm:
		return modeKeys{
			short: []key.Binding{
				key.NewBinding(key.WithKeys("y", "enter"), key.WithHelp("y/enter", "delete")),
				key.NewBinding(key.WithKeys("n", "esc"), key.WithHelp("n/esc", "cancel")),
			},
		}
	default:
		return modeKeys{
			short: []key.Binding{
				key.NewBinding(key.WithKeys("tab", "shift+tab"), key.WithHelp("tab/shift+tab", "field")),
				key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "save")),
				key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
			},
		}
	}
}

func (m *model) updateEditForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	e := m.edit
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "esc":
			e.reset()
			m.mode = modeList
			return m, nil
		case "tab":
			return m, e.nextField(1)
		case "shift+tab":
			return m, e.nextField(-1)
		case "enter":
			node, err := e.applyForm()
			if err != nil {
				e.errMsg = err.Error()
				return m, nil
			}
			siblings := m.currentChildren()
			if e.creating {
				*siblings = append(*siblings, node)
			}
			sshw.SetConfig(m.roots)
			if err := sshw.SaveConfig(); err != nil {
				e.errMsg = "save failed: " + err.Error()
				if e.creating {
					*siblings = (*siblings)[:len(*siblings)-1]
				}
				return m, nil
			}
			e.reset()
			m.mode = modeList
			return m, m.refreshCurrentList()
		}
	}
	var cmd tea.Cmd
	e.inputs[e.fieldIdx], cmd = e.inputs[e.fieldIdx].Update(msg)
	e.errMsg = ""
	return m, cmd
}

func (m *model) updateEditDeleteConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	e := m.edit
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch strings.ToLower(km.String()) {
	case "y", "enter":
		if e.deleteTarget == nil {
			m.mode = modeList
			return m, nil
		}
		if !m.removeNode(e.deleteTarget) {
			e.flash = "node not found"
			m.mode = modeList
			e.reset()
			return m, nil
		}
		sshw.SetConfig(m.roots)
		if err := sshw.SaveConfig(); err != nil {
			e.reset()
			m.mode = modeList
			m.flash = "save failed: " + err.Error()
			return m, nil
		}
		e.reset()
		m.mode = modeList
		return m, m.refreshCurrentList()
	case "n", "esc":
		e.reset()
		m.mode = modeList
		return m, nil
	}
	return m, nil
}

func (m *model) copySelectedNode() tea.Cmd {
	n := nodeForListItem(m.list.SelectedItem())
	if n == nil {
		return nil
	}
	dup := sshw.CloneNode(n)
	siblings := m.currentChildren()
	dup.Name = sshw.UniqueCopyName(n.Name, *siblings)
	*siblings = append(*siblings, dup)
	sshw.SetConfig(m.roots)
	if err := sshw.SaveConfig(); err != nil {
		*siblings = (*siblings)[:len(*siblings)-1]
		m.flash = "save failed: " + err.Error()
		return nil
	}
	return m.refreshCurrentList()
}

func (m *model) currentChildren() *[]*sshw.Node {
	if len(m.navStack) == 0 {
		return &m.roots
	}
	parent := m.navStack[len(m.navStack)-1]
	return &parent.Children
}

func (m *model) removeNode(target *sshw.Node) bool {
	siblings := m.currentChildren()
	for i, n := range *siblings {
		if n == target {
			*siblings = append((*siblings)[:i], (*siblings)[i+1:]...)
			return true
		}
	}
	return removeNodeRecursive(m.roots, target)
}

func removeNodeRecursive(nodes []*sshw.Node, target *sshw.Node) bool {
	for _, n := range nodes {
		for i, c := range n.Children {
			if c == target {
				n.Children = append(n.Children[:i], n.Children[i+1:]...)
				return true
			}
		}
		if removeNodeRecursive(n.Children, target) {
			return true
		}
	}
	return false
}

func (m *model) refreshCurrentList() tea.Cmd {
	items := nodesToListItems(nodesToItems(*m.currentChildren()))
	idx := m.list.Index()
	if idx >= len(items) {
		idx = max(0, len(items)-1)
	}
	cmd := m.setItems(items, m.list.Title)
	m.list.Select(idx)
	return cmd
}

func (m *model) syncEditLayout() {
	if m.width <= 0 {
		return
	}
	m.edit.syncLayout(m.width)
}

func (m *model) canEditNodes() bool {
	return m.editable && !m.globalMode && m.mode == modeList
}
