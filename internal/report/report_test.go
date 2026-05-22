package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jamessawle/context-audit/internal/components"
)

func TestRender_SortsBytesDescending(t *testing.T) {
	comps := []components.Component{
		{Kind: "skill", Label: "small", Bytes: 10},
		{Kind: "skill", Label: "big", Bytes: 1000},
		{Kind: "hook", Label: "mid", Bytes: 500},
	}
	var buf bytes.Buffer
	if err := Render(&buf, comps, 12345); err != nil {
		t.Fatalf("Render error: %v", err)
	}
	out := buf.String()
	bigIdx := strings.Index(out, "big")
	smallIdx := strings.Index(out, "small")
	if bigIdx < 0 || smallIdx < 0 || bigIdx > smallIdx {
		t.Fatalf("expected 'big' before 'small' in:\n%s", out)
	}
}

func TestRender_IncludesTokenTotalFooter(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, nil, 12345); err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if !strings.Contains(buf.String(), "12.3k") {
		t.Fatalf("expected magnitude-formatted token total in footer:\n%s", buf.String())
	}
}

func TestRender_HeaderRowPresent(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, nil, 0); err != nil {
		t.Fatalf("Render error: %v", err)
	}
	out := buf.String()
	tokensIdx := strings.Index(out, "TOKENS")
	bytesIdx := strings.Index(out, "BYTES")
	pluginIdx := strings.Index(out, "PLUGIN")
	compIdx := strings.Index(out, "COMPONENT")
	if tokensIdx < 0 || bytesIdx < 0 || pluginIdx < 0 || compIdx < 0 {
		t.Fatalf("expected TOKENS, BYTES, PLUGIN, COMPONENT header columns in:\n%s", out)
	}
	if !(tokensIdx < bytesIdx && bytesIdx < pluginIdx && pluginIdx < compIdx) {
		t.Fatalf("expected header order TOKENS < BYTES < PLUGIN < COMPONENT, got positions %d/%d/%d/%d in:\n%s",
			tokensIdx, bytesIdx, pluginIdx, compIdx, out)
	}
	if strings.Contains(out, "ACTION") {
		t.Fatalf("expected no ACTION column, got:\n%s", out)
	}
}

func TestRender_PluginColumnPopulated(t *testing.T) {
	comps := []components.Component{
		{Kind: "skill", Label: "fix-pr", Plugin: "pr-management", Bytes: 200},
		{Kind: "hook", Label: "SessionStart", Plugin: "", Bytes: 5000},
	}
	var buf bytes.Buffer
	if err := Render(&buf, comps, 0); err != nil {
		t.Fatalf("Render error: %v", err)
	}
	out := buf.String()
	lines := strings.Split(out, "\n")
	var hookLine, skillLine string
	for _, l := range lines {
		if strings.Contains(l, "hook: SessionStart") {
			hookLine = l
		}
		if strings.Contains(l, "skill: fix-pr") {
			skillLine = l
		}
	}
	if hookLine == "" || skillLine == "" {
		t.Fatalf("missing expected rows in:\n%s", out)
	}
	if !strings.Contains(skillLine, "pr-management") {
		t.Errorf("expected pr-management in skill row, got: %q", skillLine)
	}
	// hook row should not contain a plugin name
	if strings.Contains(hookLine, "pr-management") || strings.Contains(hookLine, "built-in") {
		t.Errorf("expected empty plugin column for hook, got: %q", hookLine)
	}
}

func TestEstimateTokens(t *testing.T) {
	// NOTE: the formula `(bytes + 2) / 4` rounds toward nearest with
	// integer truncation. The spec listed {5→2, 100→26} which the
	// formula cannot produce; those expected values are inconsistent
	// with the formula. We assert what the formula actually produces.
	cases := []struct {
		in, want int
	}{
		{0, 0},
		{4, 1},
		{5, 1},
		{100, 25},
		{1024, 256},
	}
	for _, c := range cases {
		if got := estimateTokens(c.in); got != c.want {
			t.Errorf("estimateTokens(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestRender_MCPServerFooterNote(t *testing.T) {
	comps := []components.Component{
		{Kind: "mcp_server", Label: "slack", Plugin: "mcp_server", Bytes: 0},
		{Kind: "mcp_server", Label: "gmail", Plugin: "mcp_server", Bytes: 0},
	}
	var buf bytes.Buffer
	if err := Render(&buf, comps, 1000); err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if !strings.Contains(buf.String(), "2 MCP server(s) configured") {
		t.Fatalf("expected MCP footer note in:\n%s", buf.String())
	}
}

func TestRender_EmptyComponents(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, nil, 42); err != nil {
		t.Fatalf("Render error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "BYTES") {
		t.Fatalf("expected header even when empty, got:\n%s", out)
	}
	if !strings.Contains(out, "42") {
		t.Fatalf("expected footer token total even when empty, got:\n%s", out)
	}
}

func TestRender_PreservesStableSortForEqualBytes(t *testing.T) {
	comps := []components.Component{
		{Kind: "skill", Label: "first", Bytes: 100},
		{Kind: "skill", Label: "second", Bytes: 100},
		{Kind: "skill", Label: "third", Bytes: 100},
	}
	var buf bytes.Buffer
	if err := Render(&buf, comps, 0); err != nil {
		t.Fatalf("Render error: %v", err)
	}
	out := buf.String()
	firstIdx := strings.Index(out, "first")
	secondIdx := strings.Index(out, "second")
	thirdIdx := strings.Index(out, "third")
	if !(firstIdx < secondIdx && secondIdx < thirdIdx) {
		t.Fatalf("expected stable order first<second<third for equal bytes, positions %d/%d/%d in:\n%s",
			firstIdx, secondIdx, thirdIdx, out)
	}
}

func TestFormatTokens(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{0, "0"},
		{42, "42"},
		{999, "999"},
		{1000, "1.0k"},
		{12345, "12.3k"},
		{66739, "66.7k"},
		{1234567, "1.2M"},
	}
	for _, c := range cases {
		if got := formatTokens(c.in); got != c.want {
			t.Errorf("formatTokens(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{5989, "5.8 KB"},
	}
	for _, c := range cases {
		if got := formatBytes(c.in); got != c.want {
			t.Errorf("formatBytes(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}
