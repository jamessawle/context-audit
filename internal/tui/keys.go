package tui

import "github.com/charmbracelet/bubbles/key"

// keyMap collects every key.Binding the TUI listens for so the help bar
// can render them consistently with the bindings that Update actually
// checks against.
type keyMap struct {
	Up        key.Binding
	Down      key.Binding
	PageUp    key.Binding
	PageDown  key.Binding
	SortBytes key.Binding
	SortTok   key.Binding
	SortName  key.Binding
	Filter    key.Binding
	Esc       key.Binding
	Enter     key.Binding
	Quit      key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("pgup"),
		key.WithHelp("PgUp", "scroll preview"),
	),
	PageDown: key.NewBinding(
		key.WithKeys("pgdown"),
		key.WithHelp("PgDn", "scroll preview"),
	),
	SortBytes: key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "sort bytes"),
	),
	SortTok: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "sort tokens"),
	),
	SortName: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "sort name"),
	),
	Filter: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "filter"),
	),
	Esc: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "clear/exit"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "commit filter"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
