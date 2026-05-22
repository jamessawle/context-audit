package probe

import (
	"os/exec"
	"strings"
)

// ListMCPServers enumerates MCP servers configured for the user via
// `claude mcp list` and returns their names. The probe runs `claude -p`
// in non-interactive mode which does NOT load MCP servers, so this is
// the only way to surface configured-but-not-loaded servers in the audit.
//
// On any error (claude not installed, no servers configured, non-zero
// exit), returns an empty slice and a nil error: failure to enumerate
// MCP servers should never fail the whole audit.
func ListMCPServers() ([]string, error) {
	out, err := exec.Command("claude", "mcp", "list").Output()
	if err != nil {
		return nil, nil
	}
	return parseMCPList(string(out)), nil
}

// parseMCPList parses the textual output of `claude mcp list`. Each
// configured server appears as a line of the form:
//
//	<name>: <url> - <status>
//
// e.g. "claude.ai Slack: https://mcp.slack.com/mcp - ✓ Connected".
// Server names may contain spaces and even ".", so we split on the
// first ": " (colon followed by space) to isolate the name.
//
// Lines that don't match the expected shape (banners, blank lines,
// "No MCP servers configured.") are skipped.
func parseMCPList(s string) []string {
	var names []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.Index(line, ": ")
		if idx <= 0 {
			continue
		}
		rest := line[idx+2:]
		// Require the rest to look like "<url> - <status>". A bare URL
		// scheme is sufficient to disambiguate from prose lines.
		if !(strings.Contains(rest, "://") && strings.Contains(rest, " - ")) {
			continue
		}
		names = append(names, line[:idx])
	}
	return names
}
