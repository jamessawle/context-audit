# Session JSONL as input, not disk-based prediction

`context-audit` reads a Claude Code session JSONL to determine harness context, rather than predicting harness context from on-disk configuration (settings, plugins, skill directories, hook configs).

The JSONL is the **only captured artefact** the tool needs. `CLAUDE.md` files are read directly from disk to size them — that is configuration, not a probe artefact.

## Why

The JSONL records what the harness *actually loaded*, including hook output verbatim. Disk-based prediction can enumerate configuration but cannot know a hook's output size without executing the hook — and executing hooks for measurement has side effects. JSONL is also portable across machines: a colleague can send their JSONL and we can audit it without their config.

We also verified that no supplementary capture is needed:
- Total input tokens for the session-start turn live in the JSONL itself (the assistant message's `usage.cache_creation_input_tokens`).
- Every named attachment (skill listing, deferred-tool delta, hook outputs, command permissions, task reminders) is in the JSONL.
- Tool schemas are not in the JSONL, but MCP tools are deferred at startup — only their names appear, and those names are in the JSONL. Built-in tool schemas are part of the unmeasured baseline (see below).

## Trade-offs accepted

- The user must have run a session for there to be anything to audit. v0.1 closes this with `--startup`, which spawns a fresh session whose JSONL is then read.
- The JSONL does **not** record the built-in system prompt or built-in tool schemas (Read, Bash, Edit, etc.). These are real components of harness context but are invisible to this input. They are reported as a single labelled **baseline** row, computed by subtracting the sum of measured attachments from the total input tokens. This is acceptable because the baseline is dominated by non-user-actionable content (you cannot disable Bash or the Claude Code system prompt).
- The JSONL records `deferred_tools_delta` as additions rather than full state. For v0.1 (start snapshot) this is moot; for later versions auditing mid-session state, deltas must be folded to reconstruct state.
- If a setup has few enough tools that the harness loads MCP schemas eagerly rather than deferring them, those schemas shift into the unmeasured baseline. This is a known partial blind spot for unusual configurations.
