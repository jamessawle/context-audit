# context-audit

A standalone CLI that answers a single question:

> What is filling my Claude Code context, ranked by size?

Today this answers the question for a freshly started session only, via
`--startup`: the tool spawns a Claude probe session behind the scenes and
audits the harness context recorded in its JSONL.

Future versions will extend to auditing any state of any session — live or
historical — via `--session=<id>`. Mid-timeline reconstruction (folding
deferred-tool deltas, defining "turn") is deliberately out of scope today.

The built-in system prompt and built-in tool schemas (Read, Bash, etc.) are
not recorded per-component in the JSONL. They contribute to the
harness-supplied total token figure shown in the footer but cannot be
broken out as individual rows. MCP servers without loaded schemas under
`-p` mode appear as zero-sized rows so you can still see what's wired up.
See [`docs/adr/0001-session-jsonl-as-input.md`](docs/adr/0001-session-jsonl-as-input.md)
and [CHANGELOG.md](CHANGELOG.md).

## Usage

```sh
go install github.com/jamessawle/context-audit/cmd/context-audit@latest
context-audit --startup
```

Run from the directory you want to audit. The tool spawns a single Claude session whose only purpose is to record the harness's start-up context, then opens an interactive TUI for inspecting components ranked by size.

**Cost:** one short probe call per run, using whichever model your `claude` CLI is configured to default to (so the measurement matches what you'd see in `/context` for a real interactive session — different models receive different harness content). Cold-cache cost on Opus is roughly $0.60; subsequent warm-cache runs within ~5 minutes are an order of magnitude cheaper. No API key needed beyond a working `claude` CLI.

**Output, by stdout type:**

- **Terminal (TTY):** an interactive TUI with a sortable table on top and a content-preview pane underneath. See key bindings below.
- **Pipe / redirect:** raw TSV on stdout. Metadata (`# Total: …`, `# On-demand MCP servers: …`) is emitted as `#`-prefixed comment lines at the top of the same stream, followed by a header row (`tokens\tbytes\tkind\tplugin\tlabel`) and the data rows. Standard `awk` numeric filters skip the comment lines naturally. The "Probing…" status line is suppressed when piping.

Example pipelines:

```sh
# Rows with > 500 tokens
context-audit --startup | awk -F'\t' 'NR > 1 && $1 > 500'

# Drop into a spreadsheet
context-audit --startup > audit.tsv

# Sum bytes by kind
context-audit --startup | awk -F'\t' 'NR > 1 {b[$3] += $2} END {for (k in b) print k, b[k]}'
```

## Interactive use

In a TTY the tool opens an interactive TUI with a sortable table on top and a preview pane underneath showing the loaded content of the highlighted component. Useful for "why is this hook so big?" without having to grep through JSONL by hand.

Key bindings:

- `↑` / `k`, `↓` / `j` — move selection
- `PgUp` / `PgDn` — scroll the preview pane when the content overflows
- `b` — sort by bytes (default)
- `t` — sort by tokens (same order as bytes; the estimate is byte-derived)
- `n` — sort alphabetically by component name
- `/` — enter filter mode; type a substring (matched case-insensitively against type/plugin/component) to live-filter the table. `Enter` commits, `Esc` exits filter mode without clearing
- `Esc` (outside filter mode) — clear the active filter
- `q` / `Ctrl+C` — quit


**Limitation:** MCP server tool *schemas* are not captured at startup. `claude -p` mode skips MCP server initialisation, so the deferred-tools attachment is never written to the probe's JSONL. As a partial mitigation, configured MCP servers are enumerated via `claude mcp list` and listed as zero-sized rows so you can see what's wired up — full per-server schema sizes are tracked in [#2](https://github.com/jamessawle/context-audit/issues/2).
