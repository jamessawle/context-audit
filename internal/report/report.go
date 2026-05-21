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

// Render writes a three-column table (BYTES, COMPONENT, ACTION) to w,
// sorted by Bytes descending with a stable order preserved for ties.
// The COMPONENT column is formatted as "<kind>: <label>".
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
	fmt.Fprintln(tw, "BYTES\tCOMPONENT\tACTION")
	for _, c := range sorted {
		fmt.Fprintf(tw, "%d\t%s: %s\t%s\n", c.Bytes, c.Kind, c.Label, c.Action)
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	_, err := fmt.Fprintf(w, "\nHarness recorded %d input tokens for the session-start turn (includes built-in system prompt + tool schemas, not measured here).\n", totalTokens)
	return err
}
