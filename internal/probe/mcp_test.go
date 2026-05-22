package probe

import (
	"reflect"
	"testing"
)

func TestParseMCPList(t *testing.T) {
	in := `Checking MCP server health...

claude.ai Slack: https://mcp.slack.com/mcp - ✓ Connected
claude.ai Atlassian: https://mcp.atlassian.com/v1/sse - ✓ Connected
plugin slack slack: https://example.com/mcp - ✗ Failed to connect
`
	got := parseMCPList(in)
	want := []string{
		"claude.ai Slack",
		"claude.ai Atlassian",
		"plugin slack slack",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseMCPList: got %#v, want %#v", got, want)
	}
}

func TestParseMCPList_Empty(t *testing.T) {
	if got := parseMCPList(""); len(got) != 0 {
		t.Fatalf("expected empty, got %#v", got)
	}
	if got := parseMCPList("No MCP servers configured.\n"); len(got) != 0 {
		t.Fatalf("expected empty for none-configured banner, got %#v", got)
	}
}
