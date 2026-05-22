# Session JSONL as input, not disk-based prediction

`context-audit` reads a Claude Code session JSONL to determine harness context, rather than predicting harness context from on-disk configuration (settings, plugins, skill directories, hook configs).

The JSONL is the **only captured artefact** the tool needs. `CLAUDE.md` files are read directly from disk to size them — that is configuration, not a probe artefact.

## Why

The JSONL records what the harness *actually loaded*, including hook output verbatim. Disk-based prediction can enumerate configuration but cannot know a hook's output size without executing the hook — and executing hooks for measurement has side effects. JSONL is also portable across machines: a colleague can send their JSONL and we can audit it without their config.

We also verified that no supplementary capture is needed:
- Total input tokens for the session-start turn live in the JSONL itself (the assistant message's `usage.input_tokens + cache_creation_input_tokens + cache_read_input_tokens`).
- Every named attachment (skill listing, deferred-tool delta, hook outputs, command permissions, task reminders) is in the JSONL.
- Tool schemas are not in the JSONL, but MCP tools are deferred at startup — only their names appear, and those names are in the JSONL. Built-in tool schemas (Read, Bash, Edit, etc.) and the harness's built-in system prompt are not measured per-component; they contribute to the harness-supplied total but cannot be broken out as rows.

## Trade-offs accepted

- The user must have run a session for there to be anything to audit. v0.1 closes this with `--startup`, which spawns a fresh session whose JSONL is then read.
- The JSONL does **not** record the built-in system prompt or built-in tool schemas (Read, Bash, Edit, etc.). These are real components of harness context but are invisible to this input. The tool reports them implicitly through the footer's total-token figure (which the harness supplies and includes them) without attempting to derive a "baseline" row by subtraction — per-component sizes are byte-derived estimates, and the harness-supplied total is in a different (exact) unit, so the two numbers should not be reconciled arithmetically.
- The JSONL records `deferred_tools_delta` as additions rather than full state. For v0.1 (start snapshot) this is moot; for later versions auditing mid-session state, deltas must be folded to reconstruct state.
- `claude -p` mode (the probe shape) does not initialise MCP servers, so deferred-tool entries are only written for servers that the harness still loads eagerly under that mode. Configured servers that don't appear in the JSONL are enumerated separately via `claude mcp list` and shown as zero-sized rows. Full per-server schema sizing under interactive-mode probing is tracked as a follow-up.
