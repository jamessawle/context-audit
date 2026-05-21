package components

import (
	"strings"

	"github.com/jamessawle/context-audit/internal/jsonl"
)

// Component is one unit of startup context the report renders as a row.
//
// Kind classifies the component's source ("skill", "mcp_server", "hook",
// "claude_md"). Label is a short human-readable identifier (skill name,
// MCP server name, hook name, or CLAUDE.md path). Content is the raw
// text the harness loaded — exactly the bytes we want to attribute.
// Bytes mirrors len(Content) for convenience in sorting/aggregation.
// Action is an actionable hint the report can surface to the user.
//
// The shape of this struct is part of the contract with the report
// renderer (Task 6) and downstream tasks — fields must not be renamed.
type Component struct {
	Kind    string // "skill", "mcp_server", "hook", "claude_md"
	Label   string
	Content string
	Bytes   int
	Action  string
}

func newComponent(kind, label, content, action string) Component {
	return Component{
		Kind:    kind,
		Label:   label,
		Content: content,
		Bytes:   len(content),
		Action:  action,
	}
}

// Build turns a parsed JSONL session plus discovered CLAUDE.md files into a
// flat list of Components with Bytes populated.
//
// Mapping rules:
//   - skill_listing: each non-blank line becomes one "skill" component
//   - deferred_tools_delta: tool names matching mcp__<server>__* are grouped
//     by server into one "mcp_server" component per server; other names are
//     dropped (not actionable as a server)
//   - hook_success / hook_additional_context: each becomes one "hook"
//     component using the hook's stdout (or content as fallback)
//   - each ClaudeMdFile becomes one "claude_md" component
func Build(session *jsonl.Session, claudeMds []ClaudeMdFile) []Component {
	var out []Component

	for _, a := range session.Attachments {
		switch a.SubType {
		case "skill_listing":
			out = append(out, splitSkillListing(a.Content)...)
		case "deferred_tools_delta":
			out = append(out, groupMcpServers(a.Added)...)
		case "hook_success":
			body := a.Stdout
			if body == "" {
				body = a.Content
			}
			out = append(out, newComponent("hook", a.HookName, body, "disable this hook in settings.json"))
		case "hook_additional_context":
			out = append(out, newComponent("hook", a.HookName, a.Content, "disable this hook in settings.json"))
		}
	}

	for _, f := range claudeMds {
		out = append(out, newComponent("claude_md", f.Path, f.Content, "trim or remove this CLAUDE.md"))
	}
	return out
}

// splitSkillListing parses a skill_listing attachment's markdown content,
// where each line has the form "- <name>: <description>", into one
// component per skill. Blank lines are skipped. The full line is preserved
// as Content so byte attribution reflects what the harness actually loaded.
func splitSkillListing(content string) []Component {
	var out []Component
	// SplitAfter keeps each line's trailing "\n" attached, so the per-skill
	// Content (and hence Bytes) reflects the original source bytes.
	for _, line := range strings.SplitAfter(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		body := strings.TrimPrefix(trimmed, "- ")
		name := body
		if idx := strings.Index(body, ":"); idx > 0 {
			name = strings.TrimSpace(body[:idx])
		}
		out = append(out, newComponent("skill", name, line, "disable this skill / its plugin"))
	}
	return out
}

// groupMcpServers groups deferred tool names of the form mcp__<server>__<tool>
// by server, preserving first-seen server order. Names that do not match the
// MCP prefix are dropped: they cannot be disabled at the server level and so
// are not actionable here.
func groupMcpServers(names []string) []Component {
	byServer := map[string][]string{}
	var order []string
	for _, n := range names {
		if !strings.HasPrefix(n, "mcp__") {
			continue
		}
		rest := strings.TrimPrefix(n, "mcp__")
		idx := strings.Index(rest, "__")
		if idx < 0 {
			continue
		}
		server := rest[:idx]
		if _, seen := byServer[server]; !seen {
			order = append(order, server)
		}
		byServer[server] = append(byServer[server], n)
	}

	var out []Component
	for _, server := range order {
		out = append(out, newComponent("mcp_server", server, strings.Join(byServer[server], "\n"), "remove this MCP server"))
	}
	return out
}
