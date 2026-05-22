// Package report renders the audited components as a human-readable table
// for terminal output. It is intentionally minimal: a header row, one line
// per component sorted by byte cost descending, then a single-line footer
// echoing the token total the harness reported.
package report

import (
	"fmt"
	"io"
	"sort"
	"text/tabwriter"

	"github.com/jamessawle/context-audit/internal/components"
)

// Render writes a five-column table (TOKENS (≈), BYTES, TYPE, PLUGIN,
// COMPONENT) to w, sorted by Bytes descending with a stable order preserved
// for ties. TOKENS is a heuristic estimate from byte length (4 chars/token),
// useful for ranking but not for exact comparison with /context. BYTES is
// the raw loaded byte count. TYPE is the component kind ("skill", "hook",
// "mcp_server", "claude_md"). PLUGIN is the plugin source (e.g.
// "pr-management", "built-in"); empty for hooks, claude_md, and MCP servers.
// COMPONENT is the label.
//
// After the table, Render prints a footer with totalTokens — the sum of
// input_tokens + cache_creation_input_tokens + cache_read_input_tokens
// reported by the harness for the session-start turn. That total is exact
// (harness-supplied) and includes the unmeasured baseline (built-in system
// prompt plus tool schemas), so it will not equal the sum of the TOKENS
// (≈) column.
//
// If any of the supplied components is an MCP server with zero Bytes
// (configured but loaded on-demand, surfaced via `claude mcp list`), a
// note is appended to the footer naming how many were detected.
func Render(w io.Writer, comps []components.Component, totalTokens int) error {
	sorted := make([]components.Component, len(comps))
	copy(sorted, comps)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Bytes > sorted[j].Bytes
	})

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "TOKENS (≈)\tBYTES\tTYPE\tPLUGIN\tCOMPONENT")
	mcpCount := 0
	for _, c := range sorted {
		if c.Kind == "mcp_server" && c.Bytes == 0 {
			mcpCount++
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			formatTokens(estimateTokens(c.Bytes)),
			formatBytes(c.Bytes),
			c.Kind,
			c.Plugin,
			c.Label,
		)
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "\nHarness recorded %s input tokens for the session-start turn (includes built-in system prompt + tool schemas, not measured here).\n", formatTokens(totalTokens)); err != nil {
		return err
	}
	if mcpCount > 0 {
		if _, err := fmt.Fprintf(w, "Note: %d MCP server(s) configured but loaded on-demand (size shown as zero).\n", mcpCount); err != nil {
			return err
		}
	}
	return nil
}

// estimateTokens approximates the token count from byte length.
// Uses a fixed 4 chars/token heuristic. Accurate to within ~30% for
// English/code; suitable for ranking but not for exact comparison with
// /context. The footer total comes from the harness's own count and is
// exact.
func estimateTokens(bytes int) int {
	return (bytes + 2) / 4 // round to nearest
}

// formatBytes returns "5.8 KB", "1.2 MB", "133 B" etc.
// Uses 1024-based units (KiB/MiB internally, displayed as KB/MB for readability).
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
	// "KMGTPE" — KB, MB, GB, ...
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

// formatTokens renders a token count with a magnitude suffix, e.g.
// 66739 → "66.7k", 1234567 → "1.2M". Below 1000 the raw integer is
// returned. Lowercase k/M/B are used (rather than KB/MB) because the
// values are counts, not bytes.
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
	// "kMBT" — thousands, millions, billions, trillions.
	return fmt.Sprintf("%.1f%c", float64(n)/float64(div), "kMBT"[exp])
}
