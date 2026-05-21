package components

import (
	"strings"
	"testing"

	"github.com/jamessawle/context-audit/internal/jsonl"
)

func TestBuild_SplitsSkillListingPerLineWithBytes(t *testing.T) {
	input := "- foo: does foo\n- bar: does bar\n"
	session := &jsonl.Session{
		Attachments: []jsonl.Attachment{
			{
				SubType: "skill_listing",
				Content: input,
			},
		},
	}
	comps := Build(session, nil)

	var skills []Component
	for _, c := range comps {
		if c.Kind == "skill" {
			skills = append(skills, c)
		}
	}
	if len(skills) != 2 {
		t.Fatalf("want 2 skills, got %d: %+v", len(skills), skills)
	}
	if skills[0].Label != "foo" || skills[1].Label != "bar" {
		t.Fatalf("unexpected labels: %+v", skills)
	}
	if skills[0].Content != "- foo: does foo\n" {
		t.Fatalf("expected trailing newline preserved in Content, got %q", skills[0].Content)
	}
	if skills[0].Bytes != len(skills[0].Content) || skills[0].Bytes == 0 {
		t.Fatalf("bytes not populated: %+v", skills[0])
	}
	sum := 0
	for _, s := range skills {
		sum += s.Bytes
	}
	if sum != len(input) {
		t.Fatalf("sum of skill bytes %d != len(input) %d", sum, len(input))
	}
}

func TestBuild_SkillListingBytesSumApproximatesInput(t *testing.T) {
	input := "- alpha: first\n- beta: second\n- gamma: third\n"
	session := &jsonl.Session{
		Attachments: []jsonl.Attachment{
			{SubType: "skill_listing", Content: input},
		},
	}
	comps := Build(session, nil)

	sum := 0
	count := 0
	for _, c := range comps {
		if c.Kind == "skill" {
			sum += c.Bytes
			count++
		}
	}
	if count != 3 {
		t.Fatalf("want 3 skills, got %d", count)
	}
	if sum != len(input) {
		t.Fatalf("sum of skill bytes %d != len(input) %d", sum, len(input))
	}
}

func TestBuild_GroupsDeferredToolsByServer(t *testing.T) {
	session := &jsonl.Session{
		Attachments: []jsonl.Attachment{
			{
				SubType: "deferred_tools_delta",
				Added:   []string{"mcp__slack__send", "mcp__slack__read", "mcp__gmail__list", "WebFetch"},
			},
		},
	}
	comps := Build(session, nil)

	servers := map[string]Component{}
	for _, c := range comps {
		if c.Kind == "mcp_server" {
			servers[c.Label] = c
		}
	}
	if len(servers) != 2 {
		t.Fatalf("want 2 servers, got %d: %+v", len(servers), servers)
	}
	if _, ok := servers["slack"]; !ok {
		t.Fatalf("expected slack server, got %+v", servers)
	}
	if _, ok := servers["gmail"]; !ok {
		t.Fatalf("expected gmail server")
	}
	if servers["slack"].Bytes == 0 {
		t.Fatalf("slack bytes not populated")
	}
}

func TestBuild_HookAttachments(t *testing.T) {
	session := &jsonl.Session{
		Attachments: []jsonl.Attachment{
			{SubType: "hook_success", HookName: "SessionStart:startup", Stdout: `{"context":"hi"}`},
			{SubType: "hook_additional_context", HookName: "SessionStart", Content: "extra"},
		},
	}
	comps := Build(session, nil)

	var hooks []Component
	for _, c := range comps {
		if c.Kind == "hook" {
			hooks = append(hooks, c)
		}
	}
	if len(hooks) != 2 {
		t.Fatalf("want 2 hooks, got %d", len(hooks))
	}
	if hooks[0].Bytes == 0 || hooks[1].Bytes == 0 {
		t.Fatalf("hook bytes not populated")
	}
}

func TestBuild_ClaudeMdFiles(t *testing.T) {
	files := []ClaudeMdFile{
		{Path: "/h/.claude/CLAUDE.md", Content: "global"},
		{Path: "/h/p/CLAUDE.md", Content: "project content longer"},
	}
	comps := Build(&jsonl.Session{}, files)

	var mds []Component
	for _, c := range comps {
		if c.Kind == "claude_md" {
			mds = append(mds, c)
		}
	}
	if len(mds) != 2 {
		t.Fatalf("want 2 claude_md, got %d", len(mds))
	}
	if mds[1].Bytes <= mds[0].Bytes {
		t.Fatalf("expected longer file to have more bytes")
	}
}

func TestBuild_SkillListingLabelsHandleColonsInName(t *testing.T) {
	input := "- pr-management:fix-pr: fix a PR\n- handoff: compact\n- no-desc-line\n"
	session := &jsonl.Session{
		Attachments: []jsonl.Attachment{
			{SubType: "skill_listing", Content: input},
		},
	}
	comps := Build(session, nil)

	var skills []Component
	for _, c := range comps {
		if c.Kind == "skill" {
			skills = append(skills, c)
		}
	}
	if len(skills) != 3 {
		t.Fatalf("want 3 skills, got %d: %+v", len(skills), skills)
	}
	if skills[0].Label != "pr-management:fix-pr" {
		t.Errorf("skill[0] Label = %q, want %q", skills[0].Label, "pr-management:fix-pr")
	}
	if skills[1].Label != "handoff" {
		t.Errorf("skill[1] Label = %q, want %q", skills[1].Label, "handoff")
	}
	if skills[2].Label != "no-desc-line" {
		t.Errorf("skill[2] Label = %q, want %q", skills[2].Label, "no-desc-line")
	}
}

func TestBuild_SkillListingFoldsContinuationLines(t *testing.T) {
	input := "- foo: desc line 1\n  continuation\nTRIGGER when: x\n- bar: desc\n"
	session := &jsonl.Session{
		Attachments: []jsonl.Attachment{
			{SubType: "skill_listing", Content: input},
		},
	}
	comps := Build(session, nil)

	var skills []Component
	for _, c := range comps {
		if c.Kind == "skill" {
			skills = append(skills, c)
		}
	}
	if len(skills) != 2 {
		t.Fatalf("want 2 skills (continuation lines folded), got %d: %+v", len(skills), skills)
	}
	if skills[0].Label != "foo" || skills[1].Label != "bar" {
		t.Fatalf("unexpected labels: %+v", skills)
	}
	if !strings.Contains(skills[0].Content, "continuation") || !strings.Contains(skills[0].Content, "TRIGGER when") {
		t.Errorf("expected continuation lines folded into foo Content, got %q", skills[0].Content)
	}
	sum := 0
	for _, s := range skills {
		sum += s.Bytes
	}
	if sum != len(input) {
		t.Fatalf("sum of skill bytes %d != len(input) %d (bytes invariant broken)", sum, len(input))
	}
}

func TestBuild_SkillListingLeadingContinuationDropped(t *testing.T) {
	input := "stray prefix\n- foo: desc\n"
	session := &jsonl.Session{
		Attachments: []jsonl.Attachment{
			{SubType: "skill_listing", Content: input},
		},
	}
	comps := Build(session, nil)
	var skills []Component
	for _, c := range comps {
		if c.Kind == "skill" {
			skills = append(skills, c)
		}
	}
	if len(skills) != 1 {
		t.Fatalf("want 1 skill, got %d: %+v", len(skills), skills)
	}
	if skills[0].Label != "foo" {
		t.Errorf("Label = %q, want foo", skills[0].Label)
	}
}

func TestBuild_RealFixture(t *testing.T) {
	session, err := jsonl.ParseFile("../../testdata/sample-startup.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	comps := Build(session, nil)
	if len(comps) == 0 {
		t.Fatalf("expected components from real fixture, got 0")
	}
	var hasSkill bool
	for _, c := range comps {
		if c.Kind == "skill" {
			hasSkill = true
			break
		}
	}
	if !hasSkill {
		t.Fatalf("expected skill components from real fixture")
	}
}
