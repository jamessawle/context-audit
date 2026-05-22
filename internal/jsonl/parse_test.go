package jsonl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParse_FixtureHasTotalTokens(t *testing.T) {
	session, err := ParseFile("../../testdata/sample-startup.jsonl")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if session.CacheCreationInputTokens <= 0 {
		t.Fatalf("expected positive cache_creation_input_tokens, got %d", session.CacheCreationInputTokens)
	}
}

func TestParse_FixtureHasSkillListing(t *testing.T) {
	session, err := ParseFile("../../testdata/sample-startup.jsonl")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var hasSkillListing bool
	for _, a := range session.Attachments {
		if a.SubType == "skill_listing" {
			hasSkillListing = true
			break
		}
	}
	if !hasSkillListing {
		t.Fatalf("expected a skill_listing attachment in fixture")
	}
}

// writeJSONL writes the given content to a temp file and returns its path.
func writeJSONL(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "transcript.jsonl")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp jsonl: %v", err)
	}
	return path
}

func TestParseFile_StringContent(t *testing.T) {
	jsonl := `{"type":"attachment","attachment":{"type":"skill_listing","content":"hello skills"}}` + "\n"
	session, err := ParseFile(writeJSONL(t, jsonl))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(session.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(session.Attachments))
	}
	if got := session.Attachments[0].Content; got != "hello skills" {
		t.Fatalf("Content = %q, want %q", got, "hello skills")
	}
	if len(session.Warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", session.Warnings)
	}
}

func TestParseFile_ArrayContent(t *testing.T) {
	jsonl := `{"type":"attachment","attachment":{"type":"hook_additional_context","hookName":"h","content":["a","b"]}}` + "\n"
	session, err := ParseFile(writeJSONL(t, jsonl))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(session.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(session.Attachments))
	}
	if got := session.Attachments[0].Content; got != "a\nb" {
		t.Fatalf("Content = %q, want %q", got, "a\nb")
	}
}

func TestParseFile_HookSuccessFields(t *testing.T) {
	jsonl := `{"type":"attachment","attachment":{"type":"hook_success","hookName":"my-hook","stdout":"raw output"}}` + "\n"
	session, err := ParseFile(writeJSONL(t, jsonl))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(session.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(session.Attachments))
	}
	a := session.Attachments[0]
	if a.HookName != "my-hook" {
		t.Errorf("HookName = %q, want %q", a.HookName, "my-hook")
	}
	if a.Stdout != "raw output" {
		t.Errorf("Stdout = %q, want %q", a.Stdout, "raw output")
	}
}

func TestParseFile_DeferredToolsDelta(t *testing.T) {
	jsonl := `{"type":"attachment","attachment":{"type":"deferred_tools_delta","addedNames":["mcp__foo__bar"]}}` + "\n"
	session, err := ParseFile(writeJSONL(t, jsonl))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(session.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(session.Attachments))
	}
	got := session.Attachments[0].Added
	if len(got) != 1 || got[0] != "mcp__foo__bar" {
		t.Fatalf("Added = %v, want [mcp__foo__bar]", got)
	}
}

func TestParseFile_BlankLineIgnored(t *testing.T) {
	jsonl := `{"type":"attachment","attachment":{"type":"skill_listing","content":"a"}}` + "\n" +
		"\n" +
		`{"type":"attachment","attachment":{"type":"skill_listing","content":"b"}}` + "\n"
	session, err := ParseFile(writeJSONL(t, jsonl))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(session.Attachments) != 2 {
		t.Fatalf("expected 2 attachments, got %d", len(session.Attachments))
	}
	if len(session.Warnings) != 0 {
		t.Fatalf("unexpected warnings on blank line: %v", session.Warnings)
	}
}

func TestParseFile_MalformedLineRecordsWarning(t *testing.T) {
	jsonl := `{"type":"attachment","attachment":{"type":"skill_listing","content":"first"}}` + "\n" +
		"this is not json" + "\n" +
		`{"type":"attachment","attachment":{"type":"skill_listing","content":"third"}}` + "\n"
	session, err := ParseFile(writeJSONL(t, jsonl))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(session.Attachments) != 2 {
		t.Fatalf("expected 2 attachments around the bad line, got %d", len(session.Attachments))
	}
	if len(session.Warnings) == 0 {
		t.Fatalf("expected a warning for the malformed line")
	}
	found := false
	for _, w := range session.Warnings {
		if strings.Contains(w, "line 2") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected warning referencing line 2, got: %v", session.Warnings)
	}
}

func TestParseFile_UnknownContentShapeRecordsWarning(t *testing.T) {
	jsonl := `{"type":"attachment","attachment":{"type":"weird","content":{"foo":"bar"}}}` + "\n"
	session, err := ParseFile(writeJSONL(t, jsonl))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(session.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(session.Attachments))
	}
	got := session.Attachments[0].Content
	if !strings.Contains(got, `"foo"`) || !strings.Contains(got, `"bar"`) {
		t.Fatalf("expected raw JSON in Content, got %q", got)
	}
	if len(session.Warnings) == 0 {
		t.Fatalf("expected a warning for unknown content shape")
	}
}

func TestParseFile_CapturesAllTokenFieldsFromFirstAssistant(t *testing.T) {
	jsonl := `{"type":"assistant","message":{"usage":{"input_tokens":11,"cache_creation_input_tokens":22,"cache_read_input_tokens":33}}}` + "\n" +
		`{"type":"assistant","message":{"usage":{"input_tokens":99,"cache_creation_input_tokens":99,"cache_read_input_tokens":99}}}` + "\n"
	session, err := ParseFile(writeJSONL(t, jsonl))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if session.InputTokens != 11 {
		t.Errorf("InputTokens = %d, want 11", session.InputTokens)
	}
	if session.CacheCreationInputTokens != 22 {
		t.Errorf("CacheCreationInputTokens = %d, want 22", session.CacheCreationInputTokens)
	}
	if session.CacheReadInputTokens != 33 {
		t.Errorf("CacheReadInputTokens = %d, want 33", session.CacheReadInputTokens)
	}
}

func TestParseFile_FirstAssistantTokensUsedEvenIfZero(t *testing.T) {
	jsonl := `{"type":"assistant","message":{"usage":{"cache_creation_input_tokens":0}}}` + "\n" +
		`{"type":"assistant","message":{"usage":{"cache_creation_input_tokens":100}}}` + "\n"
	session, err := ParseFile(writeJSONL(t, jsonl))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if session.CacheCreationInputTokens != 0 {
		t.Fatalf("CacheCreationInputTokens = %d, want 0 (from first assistant)", session.CacheCreationInputTokens)
	}
}
