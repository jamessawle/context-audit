package probe

import (
	"path/filepath"
	"strings"
)

// sessionJSONLPath returns the path where the harness writes the JSONL for a
// session started in resolvedCwd. Caller must resolve symlinks beforehand.
// Both '/' and '.' are replaced with '-' (discovered during fixture capture).
func sessionJSONLPath(home, resolvedCwd, sessionID string) string {
	encoded := strings.NewReplacer("/", "-", ".", "-").Replace(resolvedCwd)
	return filepath.Join(home, ".claude", "projects", encoded, sessionID+".jsonl")
}
