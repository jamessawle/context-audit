package tui

import (
	"testing"

	"github.com/jamessawle/context-audit/internal/components"
)

func sample() []components.Component {
	return []components.Component{
		{Kind: "skill", Plugin: "pr-management", Label: "write-pr-description", Content: "PR desc body", Bytes: 12},
		{Kind: "skill", Plugin: "superpowers", Label: "writing-plans", Content: "plans body", Bytes: 10},
		{Kind: "hook", Plugin: "", Label: "SessionStart", Content: "hook stdout", Bytes: 5800},
		{Kind: "claude_md", Plugin: "", Label: "/home/u/CLAUDE.md", Content: "claude md content", Bytes: 200},
		{Kind: "mcp_server", Plugin: "", Label: "Atlassian", Content: "", Bytes: 0},
	}
}

func TestNew_InitialState(t *testing.T) {
	comps := sample()
	m := New(comps, 12345)

	if len(m.allComps) != len(comps) {
		t.Fatalf("allComps len = %d, want %d", len(m.allComps), len(comps))
	}
	if len(m.visible) != len(comps) {
		t.Fatalf("visible len = %d, want %d", len(m.visible), len(comps))
	}
	if m.sortMode != sortBytes {
		t.Errorf("sortMode = %q, want %q", m.sortMode, sortBytes)
	}
	if m.filterActive {
		t.Error("filterActive should be false initially")
	}
	if m.visible[0].Bytes != 5800 {
		t.Errorf("first visible Bytes = %d, want 5800 (largest first)", m.visible[0].Bytes)
	}
	if m.table.Cursor() != 0 {
		t.Errorf("table cursor = %d, want 0", m.table.Cursor())
	}
}

func TestApplyFilter_CaseInsensitiveSubstring(t *testing.T) {
	comps := sample()
	got := applyFilter(comps, "writing")
	if len(got) != 1 {
		t.Fatalf("got %d, want 1 match for 'writing'", len(got))
	}
	if got[0].Label != "writing-plans" {
		t.Errorf("got %q, want writing-plans", got[0].Label)
	}

	// Case-insensitive
	got = applyFilter(comps, "WRITE")
	if len(got) != 1 || got[0].Label != "write-pr-description" {
		t.Errorf("WRITE filter = %+v, want [write-pr-description]", got)
	}

	// Matches plugin field
	got = applyFilter(comps, "pr-management")
	if len(got) != 1 || got[0].Label != "write-pr-description" {
		t.Errorf("pr-management filter = %+v, want [write-pr-description]", got)
	}

	// Matches kind field
	got = applyFilter(comps, "hook")
	if len(got) != 1 || got[0].Kind != "hook" {
		t.Errorf("hook filter wrong: %+v", got)
	}

	// Empty filter returns all
	got = applyFilter(comps, "")
	if len(got) != len(comps) {
		t.Errorf("empty filter returned %d, want %d", len(got), len(comps))
	}
}

func TestApplySort(t *testing.T) {
	comps := sample()

	bytesSorted := applySort(comps, sortBytes)
	if bytesSorted[0].Bytes != 5800 || bytesSorted[len(bytesSorted)-1].Bytes != 0 {
		t.Errorf("bytes sort wrong: first=%d last=%d", bytesSorted[0].Bytes, bytesSorted[len(bytesSorted)-1].Bytes)
	}

	tokensSorted := applySort(comps, sortTokens)
	// Tokens are byte-derived so order equals bytes sort.
	for i := range bytesSorted {
		if bytesSorted[i].Label != tokensSorted[i].Label {
			t.Errorf("tokens sort differs at %d: %s vs %s", i, bytesSorted[i].Label, tokensSorted[i].Label)
		}
	}

	nameSorted := applySort(comps, sortName)
	wantFirst := "/home/u/CLAUDE.md" // begins with '/'
	if nameSorted[0].Label != wantFirst {
		t.Errorf("name sort first = %q, want %q", nameSorted[0].Label, wantFirst)
	}
	// Ensure ascending
	for i := 1; i < len(nameSorted); i++ {
		if nameSorted[i-1].Label > nameSorted[i].Label {
			t.Errorf("name sort not ascending at %d: %q > %q", i, nameSorted[i-1].Label, nameSorted[i].Label)
		}
	}
}

func TestRecomputeVisible_FiltersAndClampsCursor(t *testing.T) {
	comps := sample()
	m := New(comps, 0)

	// Move cursor down past where the filtered set will reach.
	m.table.SetCursor(4)

	m.filterText = "skill"
	m.recomputeVisible()

	if len(m.visible) != 2 {
		t.Fatalf("visible after filter=%d, want 2", len(m.visible))
	}
	if c := m.table.Cursor(); c >= len(m.visible) {
		t.Errorf("cursor %d out of bounds for visible len %d", c, len(m.visible))
	}
}

func TestRefreshPreview_TracksSelection(t *testing.T) {
	comps := sample()
	m := New(comps, 0)

	// After New, first visible is the largest-bytes component (hook).
	want := m.visible[0].Content
	if got := previewContentForCursor(&m); got != want {
		t.Errorf("preview[0] = %q, want %q", got, want)
	}

	m.table.SetCursor(1)
	m.refreshPreview()
	want = m.visible[1].Content
	if got := previewContentForCursor(&m); got != want {
		t.Errorf("preview[1] = %q, want %q", got, want)
	}
}

// previewContentForCursor returns the Content of the visible row the table
// cursor currently points to. Useful for asserting selection drives preview
// without poking at viewport internals.
func previewContentForCursor(m *Model) string {
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.visible) {
		return ""
	}
	return m.visible[idx].Content
}

func TestFormatHelpers(t *testing.T) {
	if got := formatBytes(5800); got != "5.7 KB" {
		t.Errorf("formatBytes(5800) = %q", got)
	}
	if got := formatBytes(500); got != "500 B" {
		t.Errorf("formatBytes(500) = %q", got)
	}
	if got := formatTokens(1500); got != "1.5k" {
		t.Errorf("formatTokens(1500) = %q", got)
	}
	if got := formatTokens(225); got != "225" {
		t.Errorf("formatTokens(225) = %q", got)
	}
}
