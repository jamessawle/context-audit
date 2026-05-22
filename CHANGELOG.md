# Changelog

All notable changes to `context-audit` are documented here. Format follows [Keep a Changelog](https://keepachangelog.com/); versioning follows [SemVer](https://semver.org/) once we're past 0.x.

## [0.1.1] — 2026-05-22

### Added
- Interactive TUI (bubbletea + bubbles) that launches automatically when stdout is a TTY. Navigate rows with `↑/↓`, preview the highlighted component's loaded content in a side pane, sort with `b`/`t`/`n`, live-filter with `/`, quit with `q`.
- TSV output mode used automatically when stdout is piped or redirected. Raw integer columns for tokens and bytes; `#`-prefixed comment lines carry `# Total: …` and `# On-demand MCP servers: …` metadata. Standard `awk` numeric filters skip the comments without extra effort.
- `--version` flag.

### Changed
- The static lipgloss table (previously the default non-TUI output) has been removed. TTY users get the TUI; piped users get TSV. No middle ground.
- `Component.Tokens` is now populated at build time via `components.EstimateTokens` rather than re-derived on every render call. The estimation function moved out of the presentation layer.
- The "Probing harness context…" status line is suppressed when stdout is piped, so it no longer interleaves with TSV output in the user's terminal.

### Fixed
- Multi-line skill listings (e.g. `claude-api`, whose description spans `TRIGGER when:` and `SKIP:` continuation lines) reported stale token counts. `splitSkillListing` was updating `Bytes` when folding continuation lines but not the cached `Tokens`. Fixed.

## [0.1.0] — 2026-05-21

Initial release.

### Added
- `context-audit --startup` spawns a short Claude probe session (using the user's default model), parses its JSONL, and prints harness-context components ranked by byte size.
- Surfaces hooks, skills (grouped by plugin in a dedicated column), CLAUDE.md files, and MCP servers (the latter via `deferred_tools_delta` from the JSONL plus a `claude mcp list` enumeration with normalised-name dedup).
- Five-column table: `TOKENS (≈)`, `BYTES`, `TYPE`, `PLUGIN`, `COMPONENT`.
- Compact footer: `Total: X tokens · N MCP on-demand`.
- Stdlib-only Go (lipgloss is the only external dep, added solely for table rendering).
- No API key required.

### Known limitations
- MCP server tool schemas are not captured at startup: `claude -p` mode skips MCP server initialisation, so the deferred-tools attachment is partial. Configured servers without schemas appear as zero-sized rows. Tracked in [#2](https://github.com/jamessawle/context-audit/issues/2).
- Custom agents (shown by claude's `/context`) are not surfaced. Tracked in [#3](https://github.com/jamessawle/context-audit/issues/3).
