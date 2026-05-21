# context-audit

A CLI that audits what fills a Claude Code session's context, ranked by size, so the user can make informed decisions about which hooks, skills, plugins, MCP servers, or `CLAUDE.md` content are paying their way.

## Language

**Harness context**:
The bytes the Claude Code harness contributes to the model's input at a given turn, independent of user or conversation content. The thing `context-audit` audits.
_Avoid_: start-up context, first-turn context, baseline context, system context.

**Component**:
A part of harness context — for example a hook output, the skill listing, the deferred-tool list, a `CLAUDE.md` file, an MCP server's tool schemas, or the built-in system prompt. Each component has a size and a kind; some kinds the harness names individually (a specific hook, a specific skill), others it does not (the built-in system prompt). How components are grouped or rendered in output is an implementation concern, not a domain one.
_Avoid_: contributor (implies agency), entry, source, attachment, fragment.

## Example dialogue

> **User:** My sessions feel bloated. What's eating my context window?
>
> **Dev:** Run `context-audit` — it'll show you the components of your harness context ranked by size.
>
> **User:** "Harness context" meaning the whole thing?
>
> **Dev:** Just the part the harness loads — system prompt, your `CLAUDE.md` files, hook outputs, the skill listing, MCP schemas. Your own messages and tool results aren't in there; that's conversation, not harness.
>
> **User:** So if a hook fires every turn with a 3KB payload, that shows up as a component?
>
> **Dev:** Yes, it'd be a named component — the JSONL labels it by hook name. The built-in system prompt is also a component, but the harness doesn't name it individually, so v1 may render it grouped with other un-named components under a single row.
