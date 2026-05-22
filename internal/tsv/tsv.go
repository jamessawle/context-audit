// Package tsv renders audited components as tab-separated values for
// non-interactive use (pipes, scripts, redirects).
//
// The format is deliberately raw: integer columns for tokens and bytes
// (no "1.5k" / "5.8 KB" formatting), one tab-separated header row, then
// one row per component in the order callers supply. Awk, jq-via-jc,
// cut, and friends all consume it without any unboxing step.
package tsv

import (
	"fmt"
	"io"
	"sort"

	"github.com/jamessawle/context-audit/internal/components"
)

// Render writes a TSV table of components to w, sorted by Bytes
// descending with a stable order preserved for ties. The body is:
//
//	# Total: <N> tokens
//	# On-demand MCP servers: <M>        (only when M > 0)
//	tokens<TAB>bytes<TAB>kind<TAB>plugin<TAB>label
//	<N>\t<N>\t<kind>\t<plugin>\t<label>
//	...
//
// Each data row carries raw integer columns for tokens and bytes and
// verbatim strings for the rest. Metadata lines are emitted as `#`
// comments at the top so they live in the same stream as the data and
// don't interleave visually with downstream tools. Standard `awk`
// filters like `NR > 1 && $1 > 500` skip them naturally (the comment
// prefix coerces to 0 in numeric context).
func Render(w io.Writer, comps []components.Component, totalTokens int) error {
	sorted := make([]components.Component, len(comps))
	copy(sorted, comps)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Bytes > sorted[j].Bytes
	})

	mcpCount := 0
	for _, c := range sorted {
		if c.Kind == "mcp_server" && c.Bytes == 0 {
			mcpCount++
		}
	}

	if _, err := fmt.Fprintf(w, "# Total: %d tokens\n", totalTokens); err != nil {
		return err
	}
	if mcpCount > 0 {
		if _, err := fmt.Fprintf(w, "# On-demand MCP servers: %d\n", mcpCount); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, "tokens\tbytes\tkind\tplugin\tlabel"); err != nil {
		return err
	}
	for _, c := range sorted {
		if _, err := fmt.Fprintf(w, "%d\t%d\t%s\t%s\t%s\n", c.Tokens, c.Bytes, c.Kind, c.Plugin, c.Label); err != nil {
			return err
		}
	}
	return nil
}
