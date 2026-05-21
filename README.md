# context-audit

A standalone CLI that answers a single question:

> What is filling my Claude Code context, ranked by size?

**v0.1** answers this for a freshly started session only, via `--startup`: the
tool spawns a Claude session behind the scenes and audits the harness context
recorded in its JSONL.

Subsequent versions will extend to auditing any state of any session — live or
historical — via `--session=<id>`, alongside other work. Mid-timeline
reconstruction (folding deferred-tool deltas, defining "turn") is deliberately
out of v0.1 scope.

The built-in system prompt and built-in tool schemas (Read, Bash, etc.) are
not recorded in the JSONL and are reported as a single **baseline** row
computed by subtraction. MCP tools at startup appear only as names (their
schemas are deferred) and are broken down per server, so dropping an unused
MCP server is an actionable result.
See [`docs/adr/0001-session-jsonl-as-input.md`](docs/adr/0001-session-jsonl-as-input.md).
