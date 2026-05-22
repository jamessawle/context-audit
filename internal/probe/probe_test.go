package probe

import (
	"strings"
	"testing"
)

func TestBuildClaudeArgs_IncludesExitAndNoModel(t *testing.T) {
	args := buildClaudeArgs("11111111-2222-3333-4444-555555555555")
	joined := strings.Join(args, " ")
	for _, w := range []string{
		"--session-id", "11111111-2222-3333-4444-555555555555",
		"-p", "exit",
		"--output-format", "json",
	} {
		if !strings.Contains(joined, w) {
			t.Fatalf("missing %q in args: %v", w, args)
		}
	}
	// Probe must NOT pin --model — see buildClaudeArgs doc comment.
	if strings.Contains(joined, "--model") {
		t.Fatalf("probe args should not pin --model, got: %v", args)
	}
}

func TestNewSessionID_IsUUIDLike(t *testing.T) {
	id := newSessionID()
	if len(id) != 36 || id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
		t.Fatalf("not uuid-shaped: %q", id)
	}
}

func TestNewSessionID_VersionAndVariantBits(t *testing.T) {
	id := newSessionID()
	// version 4 nibble lives at index 14 (after three dashes + 14 hex chars)
	if id[14] != '4' {
		t.Fatalf("expected version 4 nibble at index 14, got %q in %q", id[14], id)
	}
	// RFC 4122 variant: first nibble of clock_seq_hi_and_reserved is 8, 9, a, or b
	v := id[19]
	if v != '8' && v != '9' && v != 'a' && v != 'b' {
		t.Fatalf("expected RFC4122 variant nibble (8/9/a/b) at index 19, got %q in %q", v, id)
	}
}

func TestNewSessionID_Unique(t *testing.T) {
	const n = 1000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		id := newSessionID()
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate UUID generated after %d calls: %q", i, id)
		}
		seen[id] = struct{}{}
	}
}
