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
}
