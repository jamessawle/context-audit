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

// Render writes a two-column table (BYTES, COMPONENT) to w, sorted by
// Bytes descending with a stable order preserved for ties. The BYTES
// column is rendered as a human-readable size (e.g. "5.8 KB"). The
// COMPONENT column is formatted as "<kind>: <label>".
//
// After the table, Render prints a one-line footer with totalTokens —
// the sum of input_tokens + cache_creation_input_tokens +
// cache_read_input_tokens reported by the harness for the session-start
// turn. The footer is informational only: it uses a different unit
// (tokens vs bytes) and includes the unmeasured baseline (built-in
// system prompt plus tool schemas), so it will not equal the sum of the
// BYTES column.
func Render(w io.Writer, comps []components.Component, totalTokens int) error {
	sorted := make([]components.Component, len(comps))
	copy(sorted, comps)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Bytes > sorted[j].Bytes
	})

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "BYTES\tCOMPONENT")
	for _, c := range sorted {
		fmt.Fprintf(tw, "%s\t%s: %s\n", formatBytes(c.Bytes), c.Kind, c.Label)
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	_, err := fmt.Fprintf(w, "\nHarness recorded %s input tokens for the session-start turn (includes built-in system prompt + tool schemas, not measured here).\n", formatTokens(totalTokens))
	return err
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
