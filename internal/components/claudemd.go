// Package components discovers and represents the inputs the Claude Code
// harness loads when starting a session (CLAUDE.md files, settings, etc.).
package components

import (
	"os"
	"path/filepath"
	"strings"
)

// ClaudeMdFile is a CLAUDE.md the harness would load, captured with its
// absolute path on disk and its raw textual content.
type ClaudeMdFile struct {
	Path    string
	Content string
}

// DiscoverClaudeMd returns every CLAUDE.md the harness would load, in the
// order the harness loads them:
//   - ~/.claude/CLAUDE.md (user-global)
//   - every ancestor-directory CLAUDE.md from home down to cwd inclusive
//
// Missing files are silently skipped, matching harness behaviour. If cwd is
// not under home, only cwd's own CLAUDE.md is considered (plus the user-global
// one) — no ancestor walk is attempted.
func DiscoverClaudeMd(home, cwd string) ([]ClaudeMdFile, error) {
	var out []ClaudeMdFile

	candidates := []string{filepath.Join(home, ".claude", "CLAUDE.md")}

	rel, err := filepath.Rel(home, cwd)
	if err == nil && !strings.HasPrefix(rel, "..") {
		parts := strings.Split(rel, string(filepath.Separator))
		cur := home
		for _, p := range parts {
			if p == "." || p == "" {
				continue
			}
			cur = filepath.Join(cur, p)
			candidates = append(candidates, filepath.Join(cur, "CLAUDE.md"))
		}
		// When cwd == home, the loop above adds nothing; include home's own CLAUDE.md.
		if rel == "." {
			candidates = append(candidates, filepath.Join(home, "CLAUDE.md"))
		}
	} else {
		candidates = append(candidates, filepath.Join(cwd, "CLAUDE.md"))
	}

	for _, p := range candidates {
		b, err := os.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		out = append(out, ClaudeMdFile{Path: p, Content: string(b)})
	}
	return out, nil
}
