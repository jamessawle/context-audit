package jsonl

import "testing"

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
