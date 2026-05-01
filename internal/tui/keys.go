package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Enter            key.Binding
	Back             key.Binding
	Quit             key.Binding
	GlobalPalette    key.Binding
	HealthCheck      key.Binding
	Select           key.Binding
	BatchRun         key.Binding
	BatchRerun       key.Binding
	BatchRerunFailed key.Binding
	BatchFilterFail  key.Binding
	BatchGroup       key.Binding
	Help             key.Binding

	// Movement bindings, only surfaced in help; the actual nav keys are
	// hard-wired in updateBatchResults / list.Model.
	Up   key.Binding
	Down key.Binding
}

var keys = keyMap{
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc", "backspace"),
		key.WithHelp("esc", "back"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "quit"),
	),
	GlobalPalette: key.NewBinding(
		key.WithKeys("ctrl+k"),
		key.WithHelp("ctrl+k", "global"),
	),
	HealthCheck: key.NewBinding(
		key.WithKeys("ctrl+h"),
		key.WithHelp("ctrl+h", "check"),
	),
	Select: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "mark"),
	),
	BatchRun: key.NewBinding(
		key.WithKeys("ctrl+x"),
		key.WithHelp("ctrl+x", "batch run"),
	),
	BatchRerun: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "rerun"),
	),
	BatchRerunFailed: key.NewBinding(
		key.WithKeys("R"),
		key.WithHelp("R", "rerun failed"),
	),
	BatchFilterFail: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "filter ✗"),
	),
	BatchGroup: key.NewBinding(
		key.WithKeys("g"),
		key.WithHelp("g", "group"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
}

// modeKeys is a small adapter that satisfies help.KeyMap. We build one
// at render time per active mode so the help bubble shows the right keys
// in short and full views.
type modeKeys struct {
	short []key.Binding
	full  [][]key.Binding
}

func (k modeKeys) ShortHelp() []key.Binding  { return k.short }
func (k modeKeys) FullHelp() [][]key.Binding { return k.full }
