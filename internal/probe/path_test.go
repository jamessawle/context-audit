package probe

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSessionJSONLPath_EncodesResolvedCwd(t *testing.T) {
	home := t.TempDir()
	cwd := "/Users/alice/projects/foo"
	got := sessionJSONLPath(home, cwd, "abc-123")
	want := filepath.Join(home, ".claude", "projects", "-Users-alice-projects-foo", "abc-123.jsonl")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestSessionJSONLPath_EncodesDotsInPath(t *testing.T) {
	// Discovered during Task 1: dots in path components are also replaced with dashes.
	home := t.TempDir()
	cwd := "/Users/james.sawle/proj"
	got := sessionJSONLPath(home, cwd, "s")
	want := filepath.Join(home, ".claude", "projects", "-Users-james-sawle-proj", "s.jsonl")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestSessionJSONLPath_DoesNotResolveSymlinksItself(t *testing.T) {
	// Contract: callers resolve symlinks (EvalSymlinks) before calling.
	// The function encodes the input verbatim (modulo / and . substitution).
	got := sessionJSONLPath("/h", "/tmp/foo", "s")
	if !strings.Contains(got, "-tmp-foo") {
		t.Fatalf("expected literal encoding of input cwd, got %q", got)
	}
}
