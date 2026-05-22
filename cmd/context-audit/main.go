package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/jamessawle/context-audit/internal/components"
	"github.com/jamessawle/context-audit/internal/jsonl"
	"github.com/jamessawle/context-audit/internal/probe"
	"github.com/jamessawle/context-audit/internal/report"
)

func main() {
	startup := flag.Bool("startup", false, "Audit harness context at a fresh session start")
	flag.Parse()

	if !*startup {
		fmt.Fprintln(os.Stderr, "context-audit v0.1 requires --startup")
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

	fmt.Fprintln(os.Stderr, "Spawning probe session (this costs a small amount on your Claude account)...")
	jsonlPath, err := probe.Run(home, cwd)
	if err != nil {
		return err
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
	return report.Render(os.Stdout, comps, totalTokens)
}
