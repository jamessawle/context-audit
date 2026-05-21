package probe

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Run spawns a probe Claude session and returns the path to its JSONL.
// home and cwd should be absolute paths; cwd MUST be symlink-resolved.
func Run(home, cwd string) (string, error) {
	sid := newSessionID()
	cmd := exec.Command("claude", buildClaudeArgs(sid)...)
	cmd.Dir = cwd
	cmd.Stdout = io.Discard // suppress claude's JSON result blob; stderr still flows through
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("claude probe failed: %w", err)
	}

	path := sessionJSONLPath(home, cwd, sid)
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("probe JSONL not found at %s: %w", path, err)
	}
	return path, nil
}

// buildClaudeArgs returns the CLI args for the probe Claude invocation.
// The probe sends prompt "exit" to a cheap model and asks for JSON output
// so the session terminates immediately and writes a JSONL transcript.
func buildClaudeArgs(sessionID string) []string {
	return []string{
		"--session-id", sessionID,
		"--model", "haiku",
		"-p", "exit",
		"--output-format", "json",
	}
}

// newSessionID returns an RFC 4122 version 4 UUID string. It falls back to
// the nil UUID only if the system CSPRNG fails, which should not happen in
// practice.
func newSessionID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "00000000-0000-0000-0000-000000000000"
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // RFC 4122 variant
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
