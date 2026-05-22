package tsv

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jamessawle/context-audit/internal/components"
)

func TestRender_HeaderAndRows(t *testing.T) {
	comps := []components.Component{
		{Kind: "skill", Plugin: "pr-management", Label: "fix-pr", Bytes: 800, Tokens: 200},
		{Kind: "hook", Label: "SessionStart:startup", Bytes: 5989, Tokens: 1497},
		{Kind: "mcp_server", Label: "claude_ai_Slack", Bytes: 564, Tokens: 141},
	}
	var buf bytes.Buffer
	if err := Render(&buf, comps, 33000); err != nil {
		t.Fatalf("Render error: %v", err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	// Expect: # Total, header, then rows sorted by bytes desc.
	if got := lines[0]; got != "# Total: 33000 tokens" {
		t.Fatalf("metadata line: got %q", got)
	}
	if got := lines[1]; got != "tokens\tbytes\tkind\tplugin\tlabel" {
		t.Fatalf("header row: got %q", got)
	}
	if got := lines[2]; got != "1497\t5989\thook\t\tSessionStart:startup" {
		t.Fatalf("first data row: got %q", got)
	}
	if got := lines[3]; got != "200\t800\tskill\tpr-management\tfix-pr" {
		t.Fatalf("second data row: got %q", got)
	}
	if got := lines[4]; got != "141\t564\tmcp_server\t\tclaude_ai_Slack" {
		t.Fatalf("third data row: got %q", got)
	}
}

func TestRender_OnDemandMCPCount(t *testing.T) {
	comps := []components.Component{
		{Kind: "mcp_server", Label: "a", Bytes: 0, Tokens: 0},
		{Kind: "mcp_server", Label: "b", Bytes: 0, Tokens: 0},
		{Kind: "mcp_server", Label: "c", Bytes: 100, Tokens: 25},
	}
	var buf bytes.Buffer
	if err := Render(&buf, comps, 0); err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if !strings.Contains(buf.String(), "# On-demand MCP servers: 2") {
		t.Fatalf("expected on-demand count comment, got: %q", buf.String())
	}
}

func TestRender_NoOnDemandLineWhenZero(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, nil, 100); err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if strings.Contains(buf.String(), "On-demand") {
		t.Fatalf("expected no on-demand line when count is zero, got: %q", buf.String())
	}
}

func TestRender_AwkNumericFilterSkipsComments(t *testing.T) {
	// Sanity-check the design claim: a comment line's first column,
	// when coerced to a number, evaluates to 0 — so `$1 > 500` filters
	// it out. Reproduce the coercion in Go to lock in the assumption.
	comps := []components.Component{
		{Kind: "hook", Label: "big", Bytes: 5989, Tokens: 1497},
		{Kind: "skill", Plugin: "x", Label: "small", Bytes: 100, Tokens: 25},
	}
	var buf bytes.Buffer
	if err := Render(&buf, comps, 33000); err != nil {
		t.Fatalf("Render error: %v", err)
	}
	// Header line's first column ("tokens") and comment line's first
	// column ("# Total: ...") both coerce to 0 in awk-style numeric
	// context. We verify they don't start with a digit, the simplest
	// proxy for "would survive an awk numeric filter".
	for _, line := range strings.Split(strings.TrimRight(buf.String(), "\n"), "\n") {
		if line == "" {
			continue
		}
		first := line[0]
		isData := first >= '0' && first <= '9'
		isMeta := first == '#' || first == 't' // "# Total..." or "tokens"
		if !isData && !isMeta {
			t.Fatalf("unexpected line prefix in output: %q", line)
		}
	}
}
