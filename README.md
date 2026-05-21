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

## Usage

```sh
go install github.com/jamessawle/context-audit/cmd/context-audit@latest
context-audit --startup
```

Run from the directory you want to audit. The tool spawns a single Claude session whose only purpose is to record the harness's start-up context, then prints components ranked by **byte size**.

**Cost:** one short probe call per run, using whichever model your `claude` CLI is configured to default to (so the measurement matches what you'd see in `/context` for a real interactive session — different models receive different harness content). Cold-cache cost on Opus is roughly $0.60; subsequent warm-cache runs within ~5 minutes are an order of magnitude cheaper. No API key needed beyond a working `claude` CLI.

**Output:** a table of `BYTES  COMPONENT`, sorted descending. Below the table, a footer reports the harness's recorded input-token total for orientation — this includes the built-in system prompt and built-in tool schemas, which are not broken down per row (they aren't actionable individually).

**Why bytes, not tokens?** Token-accurate per-component sizing would require an API key and an HTTP call per component. Ranking by size is invariant under unit choice — bytes are good enough to tell you which thing is the biggest, and that's the actionable question.

**Limitation:** MCP server tool listings are not currently captured. `claude -p` mode skips MCP server initialisation, so the deferred-tools attachment is never written to the probe's JSONL. Tracked in [#2](https://github.com/jamessawle/context-audit/issues/2) — coming in v0.2.
