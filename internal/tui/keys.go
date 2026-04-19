package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Enter         key.Binding
	Back          key.Binding
	Quit          key.Binding
	GlobalPalette key.Binding
	HealthCheck   key.Binding
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
}
