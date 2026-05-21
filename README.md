# context-audit

A standalone CLI that answers a single question:

> What is filling my Claude Code context, ranked by size?

In v1 it focuses on **start-up context** — the bytes the harness loads before
the user has typed anything — but the design is set up so the same tool can
later audit any turn of any session, including live ones.
