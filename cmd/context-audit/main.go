package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/mattn/go-isatty"

	"github.com/jamessawle/context-audit/internal/components"
	"github.com/jamessawle/context-audit/internal/jsonl"
	"github.com/jamessawle/context-audit/internal/probe"
	"github.com/jamessawle/context-audit/internal/tsv"
	"github.com/jamessawle/context-audit/internal/tui"
)

// version is the binary's reported version. Bumped per release; we don't
// currently inject via ldflags because the project is small enough that a
// hardcoded const is easier to reason about than build-time substitution.
const version = "0.1.1"

func main() {
	startup := flag.Bool("startup", false, "Audit harness context at a fresh session start")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}
	if !*startup {
		fmt.Fprintln(os.Stderr, "context-audit requires --startup (or --version)")
		os.Exit(2)
	}
	if err := runStartup(); err != nil {
		fmt.Fprintf(os.Stderr, "context-audit: %v\n", err)
		os.Exit(1)
	}
}

func runStartup() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}

	rawCwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cwd: %w", err)
	}
	cwd, err := filepath.EvalSymlinks(rawCwd)
	if err != nil {
		return fmt.Errorf("resolve cwd: %w", err)
	}

	if _, err := exec.LookPath("claude"); err != nil {
		return fmt.Errorf("claude CLI not found on PATH; install Claude Code (https://claude.com/claude-code) before running context-audit")
	}

	stdoutIsTTY := isatty.IsTerminal(os.Stdout.Fd())

	// The progress message is for humans; suppress it when piping so it
	// doesn't appear above stdout data in the user's terminal.
	if stdoutIsTTY {
		fmt.Fprintln(os.Stderr, "Probing harness context — spawning a short Claude session (~10s, costs a few cents)...")
	}
	jsonlPath, err := probe.Run(home, cwd)
	if err != nil {
		return err
	}
	if stdoutIsTTY {
		fmt.Fprintln(os.Stderr) // blank line between status output and the TUI
	}

	session, err := jsonl.ParseFile(jsonlPath)
	if err != nil {
		return fmt.Errorf("parse jsonl: %w", err)
	}

	claudeMds, err := components.DiscoverClaudeMd(home, cwd)
	if err != nil {
		return fmt.Errorf("discover CLAUDE.md: %w", err)
	}

	for _, w := range session.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", w)
	}

	comps := components.Build(session, claudeMds)
	if servers, err := probe.ListMCPServers(); err == nil {
		// Dedup against MCP servers already in comps from the JSONL
		// (deferred-tool delta). The two sources name the same server
		// differently (e.g. "claude_ai_Atlassian" vs "claude.ai Atlassian"),
		// so DedupMCPServers normalises both before comparing.
		comps = components.DedupMCPServers(comps, servers)
	}
	totalTokens := session.InputTokens + session.CacheCreationInputTokens + session.CacheReadInputTokens

	// If stdout is a real terminal, launch the interactive TUI. When the
	// caller pipes output (e.g. `context-audit --startup > out.txt` or
	// `| awk ...`), emit raw TSV with `#`-prefixed metadata comments
	// so downstream tools can consume it cleanly.
	if stdoutIsTTY {
		return tui.Run(comps, totalTokens)
	}
	return tsv.Render(os.Stdout, comps, totalTokens)
}
