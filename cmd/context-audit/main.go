package main

import (
	"flag"
	"fmt"
	"os"
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

	comps := components.Build(session, claudeMds)
	if err := report.Render(os.Stdout, comps, session.CacheCreationInputTokens); err != nil {
		return err
	}
	for _, w := range session.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", w)
	}
	return nil
}
