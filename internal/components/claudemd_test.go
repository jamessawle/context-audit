package components

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverClaudeMd_FindsUserAncestorAndProject(t *testing.T) {
	home := t.TempDir()
	project := filepath.Join(home, "work", "proj")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(p, s string) {
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(filepath.Join(home, ".claude", "CLAUDE.md"), "global")
	write(filepath.Join(home, "work", "CLAUDE.md"), "ancestor")
	write(filepath.Join(project, "CLAUDE.md"), "project")

	files, err := DiscoverClaudeMd(home, project)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 {
		t.Fatalf("DiscoverClaudeMd(%q, %q): want 3 files, got %d: %+v", home, project, len(files), files)
	}
	for _, f := range files {
		if f.Path == "" || f.Content == "" {
			t.Fatalf("empty field in %+v", f)
		}
	}
}

func TestDiscoverClaudeMd_MissingFilesOmitted(t *testing.T) {
	home := t.TempDir()
	project := filepath.Join(home, "p")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	files, err := DiscoverClaudeMd(home, project)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("DiscoverClaudeMd(%q, %q): want 0 files, got %d", home, project, len(files))
	}
}

func TestDiscoverClaudeMd_CwdEqualsHome(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".claude", "CLAUDE.md"), []byte("global"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "CLAUDE.md"), []byte("home"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := DiscoverClaudeMd(home, home)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("DiscoverClaudeMd(%q, %q): want 2 files, got %d: %+v", home, home, len(files), files)
	}
}
