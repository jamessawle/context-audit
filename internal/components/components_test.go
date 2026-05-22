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
	if skills[0].Plugin != "pr-management" || skills[0].Label != "fix-pr" {
		t.Errorf("skill[0] = (%q, %q), want (pr-management, fix-pr)", skills[0].Plugin, skills[0].Label)
	}
	if skills[1].Plugin != "built-in" || skills[1].Label != "handoff" {
		t.Errorf("skill[1] = (%q, %q), want (built-in, handoff)", skills[1].Plugin, skills[1].Label)
	}
	if skills[2].Plugin != "built-in" || skills[2].Label != "no-desc-line" {
		t.Errorf("skill[2] = (%q, %q), want (built-in, no-desc-line)", skills[2].Plugin, skills[2].Label)
	}
}

func TestBuild_SkillPluginExtraction(t *testing.T) {
	input := "- pr-management:fix-pr: fix\n- superpowers:writing-plans: plan\n- slack:standup: stand\n- init: init\n- handoff: handoff\n"
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
	cases := []struct {
		plugin, label string
	}{
		{"pr-management", "fix-pr"},
		{"superpowers", "writing-plans"},
		{"slack", "standup"},
		{"built-in", "init"},
		{"built-in", "handoff"},
	}
	if len(skills) != len(cases) {
		t.Fatalf("want %d skills, got %d: %+v", len(cases), len(skills), skills)
	}
	for i, want := range cases {
		if skills[i].Plugin != want.plugin || skills[i].Label != want.label {
			t.Errorf("skill[%d] = (%q, %q), want (%q, %q)", i, skills[i].Plugin, skills[i].Label, want.plugin, want.label)
		}
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

func TestNormalizeMCPName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"claude_ai_Atlassian", "claudeaiatlassian"},
		{"claude.ai Atlassian", "claudeaiatlassian"},
		{"plugin:slack:slack", "plugin:slack:slack"},
		{"claude.ai Slack", "claudeaislack"},
		{"claude_ai_Slack", "claudeaislack"},
	}
	for _, c := range cases {
		if got := normalizeMCPName(c.in); got != c.want {
			t.Errorf("normalizeMCPName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestDedupMCPServers_SkipsDuplicates(t *testing.T) {
	comps := []Component{
		{Kind: "skill", Label: "fix-pr", Plugin: "pr-management", Bytes: 200},
		{Kind: "mcp_server", Label: "claude_ai_Atlassian", Bytes: 1500},
		{Kind: "mcp_server", Label: "plugin_slack_slack", Bytes: 800},
	}
	names := []string{
		"claude.ai Atlassian", // duplicate of claude_ai_Atlassian — skip
		"claude.ai Gmail",     // new
		"plugin slack slack",  // duplicate of plugin_slack_slack — skip
	}
	got := DedupMCPServers(comps, names)
	if len(got) != len(comps)+1 {
		t.Fatalf("want %d components after dedup, got %d: %+v", len(comps)+1, len(got), got)
	}
	// Find Gmail row — order in input is preserved among appended entries.
	var gotGmail bool
	for _, c := range got {
		if c.Label == "claude.ai Gmail" {
			if c.Kind != "mcp_server" || c.Bytes != 0 {
				t.Errorf("unexpected Gmail row: %+v", c)
			}
			gotGmail = true
		}
	}
	if !gotGmail {
		t.Errorf("expected appended Gmail row, got: %+v", got)
	}
	// Original Atlassian row must keep its bytes.
	for _, c := range got {
		if c.Label == "claude_ai_Atlassian" && c.Bytes != 1500 {
			t.Errorf("expected JSONL-sourced Atlassian row preserved with 1500 bytes, got %+v", c)
		}
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

func TestEstimateTokens(t *testing.T) {
	// The formula (bytes + 2) / 4 rounds toward nearest with integer
	// truncation; we assert what it actually produces.
	cases := []struct {
		in, want int
	}{
		{0, 0},
		{4, 1},
		{5, 1},
		{100, 25},
		{1024, 256},
	}
	for _, c := range cases {
		if got := EstimateTokens(c.in); got != c.want {
			t.Errorf("EstimateTokens(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestNewComponent_PopulatesTokensFromBytes(t *testing.T) {
	// Components built via newComponent should have Tokens populated at
	// construction time so renderers don't re-derive on every render.
	// We exercise this indirectly through Build, which is the only
	// caller of newComponent.
	files := []ClaudeMdFile{{Path: "/tmp/CLAUDE.md", Content: "abcd"}} // 4 bytes
	comps := Build(&jsonl.Session{}, files)
	if len(comps) != 1 {
		t.Fatalf("expected 1 component, got %d", len(comps))
	}
	got := comps[0]
	if got.Bytes != 4 {
		t.Fatalf("Bytes: got %d want 4", got.Bytes)
	}
	if got.Tokens != EstimateTokens(4) {
		t.Fatalf("Tokens: got %d want EstimateTokens(4)=%d", got.Tokens, EstimateTokens(4))
	}
}
