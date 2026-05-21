package components

import (
	"testing"

	"github.com/jamessawle/context-audit/internal/jsonl"
)

func TestBuild_SplitsSkillListingPerLineWithBytes(t *testing.T) {
	session := &jsonl.Session{
		Attachments: []jsonl.Attachment{
			{
				SubType: "skill_listing",
				Content: "- foo: does foo\n- bar: does bar\n",
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
	if skills[0].Bytes != len(skills[0].Content) || skills[0].Bytes == 0 {
		t.Fatalf("bytes not populated: %+v", skills[0])
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
