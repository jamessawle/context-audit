// Package tui renders the startup-context audit as an interactive
// terminal UI built on bubbletea. The TUI is launched by
// cmd/context-audit when stdout is a TTY; when the output is piped,
// the caller falls back to the static report.Render path instead.
package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jamessawle/context-audit/internal/components"
)

// sortKey identifies which Component field the visible slice is sorted on.
// "bytes" is the default and matches the static report; "tokens" sorts on
// the same underlying byte count (tokens are byte-derived) but is named
// for the column the user thinks they're sorting; "name" is alphabetical
// by Label.
type sortKey string

const (
	sortBytes  sortKey = "bytes"
	sortTokens sortKey = "tokens"
	sortName   sortKey = "name"
)

// Model is the bubbletea state for the interactive audit view.
//
// allComps is the original, unfiltered/unsorted slice as built by
// components.Build. visible is the projection currently shown in the
// table — applyFilter then applySort produce it. The two are kept
// separate so toggling filter/sort never mutates the source list.
type Model struct {
	allComps    []components.Component
	visible     []components.Component
	totalTokens int

	table   table.Model
	preview viewport.Model
	filter  textinput.Model

	filterActive bool
	filterText   string
	sortMode     sortKey

	width, height int
	ready         bool
}

// New constructs a Model with comps sorted by bytes desc and filter inactive.
func New(comps []components.Component, totalTokens int) Model {
	ti := textinput.New()
	ti.Prompt = "/"
	ti.Placeholder = "filter"
	ti.CharLimit = 80

	m := Model{
		allComps:    comps,
		totalTokens: totalTokens,
		sortMode:    sortBytes,
		filter:      ti,
	}
	m.visible = applySort(applyFilter(comps, ""), sortBytes)

	// Initial table with zero size — WindowSizeMsg will resize.
	cols := []table.Column{
		{Title: "TOKENS (≈)", Width: 11},
		{Title: "BYTES", Width: 9},
		{Title: "TYPE", Width: 10},
		{Title: "PLUGIN", Width: 14},
		{Title: "COMPONENT", Width: 30},
	}
	t := table.New(
		table.WithColumns(cols),
		table.WithRows(buildRows(m.visible)),
		table.WithFocused(true),
	)
	st := table.DefaultStyles()
	st.Header = st.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(borderColor).
		BorderBottom(true).
		Bold(true)
	st.Selected = st.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(st)
	m.table = t

	m.preview = viewport.New(0, 0)
	m.refreshPreview()
	return m
}

// Run constructs a Model and runs it as a bubbletea program.
// Returns the final model or an error if the program failed to start.
func Run(comps []components.Component, totalTokens int) error {
	m := New(comps, totalTokens)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m Model) Init() tea.Cmd { return textinput.Blink }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout()
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		if m.filterActive {
			return m.updateFilterMode(msg)
		}
		return m.updateNavMode(msg)
	}

	// Forward other messages to inner components (e.g. mouse).
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	cmds = append(cmds, cmd)
	m.preview, cmd = m.preview.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m Model) updateFilterMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Enter):
		m.filterActive = false
		m.filter.Blur()
		return m, nil
	case key.Matches(msg, keys.Esc):
		// Exit filter mode but preserve text.
		m.filterActive = false
		m.filter.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	m.filterText = m.filter.Value()
	m.recomputeVisible()
	return m, cmd
}

func (m Model) updateNavMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, keys.Filter):
		m.filterActive = true
		m.filter.Focus()
		return m, textinput.Blink
	case key.Matches(msg, keys.Esc):
		if m.filterText != "" {
			m.filterText = ""
			m.filter.SetValue("")
			m.recomputeVisible()
		}
		return m, nil
	case key.Matches(msg, keys.SortBytes):
		m.sortMode = sortBytes
		m.recomputeVisible()
		return m, nil
	case key.Matches(msg, keys.SortTok):
		m.sortMode = sortTokens
		m.recomputeVisible()
		return m, nil
	case key.Matches(msg, keys.SortName):
		m.sortMode = sortName
		m.recomputeVisible()
		return m, nil
	case key.Matches(msg, keys.PageUp):
		m.preview.HalfPageUp()
		return m, nil
	case key.Matches(msg, keys.PageDown):
		m.preview.HalfPageDown()
		return m, nil
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	m.refreshPreview()
	return m, cmd
}

func (m Model) View() string {
	if !m.ready {
		return ""
	}

	helpLine := m.helpView()
	previewBlock := previewBorderStyle.
		Width(m.width - 2).
		Height(m.preview.Height).
		Render(previewHeaderStyle.Render(m.previewHeader()) + "\n" + m.preview.View())

	return lipgloss.JoinVertical(
		lipgloss.Left,
		tableBorderStyle.Render(m.table.View()),
		previewBlock,
		helpLine,
	)
}

// --- helpers ---

// recomputeVisible re-derives the visible slice from filter + sort, rebuilds
// the table rows, clamps the table cursor, and refreshes the preview pane.
// Called whenever filter text or sort mode changes.
func (m *Model) recomputeVisible() {
	m.visible = applySort(applyFilter(m.allComps, m.filterText), m.sortMode)
	rows := buildRows(m.visible)
	m.table.SetRows(rows)
	if c := m.table.Cursor(); c >= len(rows) {
		if len(rows) == 0 {
			m.table.SetCursor(0)
		} else {
			m.table.SetCursor(len(rows) - 1)
		}
	}
	m.refreshPreview()
}

// refreshPreview reloads the viewport content from the row currently
// selected in the table. Called after navigation and after recomputeVisible.
func (m *Model) refreshPreview() {
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.visible) {
		m.preview.SetContent("")
		return
	}
	m.preview.SetContent(m.visible[idx].Content)
	m.preview.GotoTop()
}

// layout proportions the three regions (table ~60%, preview ~30%, help ~10%)
// against the current terminal size and resizes the inner widgets.
func (m *Model) layout() {
	w := m.width
	h := m.height
	if w < 20 {
		w = 20
	}
	if h < 10 {
		h = 10
	}
	tableH := h * 6 / 10
	previewH := h * 3 / 10
	if tableH < 5 {
		tableH = 5
	}
	if previewH < 3 {
		previewH = 3
	}

	// Account for table borders + header (~3 lines) and viewport
	// borders+header (~3 lines). SetHeight on table.Model is the body row count.
	bodyRows := tableH - 4
	if bodyRows < 1 {
		bodyRows = 1
	}
	m.table.SetHeight(bodyRows)
	m.table.SetWidth(w - 2)

	// Resize columns: divide remaining width across the 5 cols. Fixed widths
	// for numeric cols; the rest goes to PLUGIN + COMPONENT.
	fixed := 11 + 9 + 10 // tokens, bytes, type
	rem := (w - 2) - fixed - 12 /* padding */
	if rem < 20 {
		rem = 20
	}
	pluginW := rem / 3
	compW := rem - pluginW
	if pluginW < 8 {
		pluginW = 8
	}
	if compW < 12 {
		compW = 12
	}
	m.table.SetColumns([]table.Column{
		{Title: "TOKENS (≈)", Width: 11},
		{Title: "BYTES", Width: 9},
		{Title: "TYPE", Width: 10},
		{Title: "PLUGIN", Width: pluginW},
		{Title: "COMPONENT", Width: compW},
	})

	m.preview.Width = w - 4
	m.preview.Height = previewH - 4
	if m.preview.Height < 1 {
		m.preview.Height = 1
	}
	m.refreshPreview()
}

func (m Model) previewHeader() string {
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.visible) {
		return "(no selection)"
	}
	c := m.visible[idx]
	plugin := c.Plugin
	if plugin == "" {
		return fmt.Sprintf("%s: %s (%s)", c.Kind, c.Label, formatBytes(c.Bytes))
	}
	return fmt.Sprintf("%s: %s/%s (%s)", c.Kind, plugin, c.Label, formatBytes(c.Bytes))
}

func (m Model) helpView() string {
	if m.filterActive {
		return filterPromptStyle.Render(m.filter.View())
	}
	parts := []string{
		"↑/↓ navigate",
		"/ filter",
		"t/b/n sort",
		"PgUp/PgDn preview",
		"q quit",
	}
	suffix := fmt.Sprintf("  sort:%s", m.sortMode)
	if m.filterText != "" {
		suffix += fmt.Sprintf("  filter:%q", m.filterText)
	}
	suffix += fmt.Sprintf("  total:%s", formatTokens(m.totalTokens))
	return helpStyle.Render(strings.Join(parts, "  ") + suffix)
}

// --- pure functions (testable without bubbletea) ---

// applyFilter returns a new slice containing only components whose
// "Kind Plugin Label" (joined with spaces) contains q (case-insensitive).
// Empty q returns a copy of comps unchanged.
func applyFilter(comps []components.Component, q string) []components.Component {
	if strings.TrimSpace(q) == "" {
		out := make([]components.Component, len(comps))
		copy(out, comps)
		return out
	}
	needle := strings.ToLower(q)
	out := make([]components.Component, 0, len(comps))
	for _, c := range comps {
		hay := strings.ToLower(c.Kind + " " + c.Plugin + " " + c.Label)
		if strings.Contains(hay, needle) {
			out = append(out, c)
		}
	}
	return out
}

// applySort returns a new slice sorted per mode. Ties preserve input order
// (sort.SliceStable). "tokens" sorts on Bytes since our token estimate is
// monotonic in Bytes; treating them identically keeps ordering consistent
// between the two columns.
func applySort(comps []components.Component, mode sortKey) []components.Component {
	out := make([]components.Component, len(comps))
	copy(out, comps)
	switch mode {
	case sortName:
		sort.SliceStable(out, func(i, j int) bool {
			return strings.ToLower(out[i].Label) < strings.ToLower(out[j].Label)
		})
	default: // bytes & tokens both sort on Bytes desc
		sort.SliceStable(out, func(i, j int) bool {
			return out[i].Bytes > out[j].Bytes
		})
	}
	return out
}

// buildRows converts visible components into the [][]string shape the
// bubbles table expects, mirroring the columns of the static report.
func buildRows(comps []components.Component) []table.Row {
	rows := make([]table.Row, 0, len(comps))
	for _, c := range comps {
		rows = append(rows, table.Row{
			formatTokens(estimateTokens(c.Bytes)),
			formatBytes(c.Bytes),
			c.Kind,
			c.Plugin,
			c.Label,
		})
	}
	return rows
}

// estimateTokens, formatBytes, formatTokens duplicate the helpers in
// internal/report so the TUI doesn't need to import that package
// (which would invert the dependency direction). The 4 chars/token
// heuristic, 1024-based byte units, and lowercase k/M magnitude suffix
// match report.go exactly.
func estimateTokens(b int) int { return (b + 2) / 4 }

func formatBytes(n int) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := int64(n) / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

func formatTokens(n int) string {
	const unit = 1000
	if n < unit {
		return fmt.Sprintf("%d", n)
	}
	div, exp := int64(unit), 0
	for v := int64(n) / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%c", float64(n)/float64(div), "kMBT"[exp])
}
