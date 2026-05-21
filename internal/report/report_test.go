package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jamessawle/context-audit/internal/components"
)

func TestRender_SortsBytesDescending(t *testing.T) {
	comps := []components.Component{
		{Kind: "skill", Label: "small", Bytes: 10},
		{Kind: "skill", Label: "big", Bytes: 1000},
		{Kind: "hook", Label: "mid", Bytes: 500},
	}
	var buf bytes.Buffer
	if err := Render(&buf, comps, 12345); err != nil {
		t.Fatalf("Render error: %v", err)
	}
	out := buf.String()
	bigIdx := strings.Index(out, "big")
	smallIdx := strings.Index(out, "small")
	if bigIdx < 0 || smallIdx < 0 || bigIdx > smallIdx {
		t.Fatalf("expected 'big' before 'small' in:\n%s", out)
	}
}

func TestRender_IncludesTokenTotalFooter(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, nil, 12345); err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if !strings.Contains(buf.String(), "12345") {
		t.Fatalf("expected token total in footer:\n%s", buf.String())
	}
}

func TestRender_HeaderRowPresent(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, nil, 0); err != nil {
		t.Fatalf("Render error: %v", err)
	}
	out := buf.String()
	bytesIdx := strings.Index(out, "BYTES")
	compIdx := strings.Index(out, "COMPONENT")
	if bytesIdx < 0 || compIdx < 0 {
		t.Fatalf("expected BYTES and COMPONENT header columns in:\n%s", out)
	}
	if !(bytesIdx < compIdx) {
		t.Fatalf("expected header order BYTES < COMPONENT, got positions %d/%d in:\n%s",
			bytesIdx, compIdx, out)
	}
	if strings.Contains(out, "ACTION") {
		t.Fatalf("expected no ACTION column, got:\n%s", out)
	}
}

func TestRender_EmptyComponents(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, nil, 42); err != nil {
		t.Fatalf("Render error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "BYTES") {
		t.Fatalf("expected header even when empty, got:\n%s", out)
	}
	if !strings.Contains(out, "42") {
		t.Fatalf("expected footer token total even when empty, got:\n%s", out)
	}
}

func TestRender_PreservesStableSortForEqualBytes(t *testing.T) {
	comps := []components.Component{
		{Kind: "skill", Label: "first", Bytes: 100},
		{Kind: "skill", Label: "second", Bytes: 100},
		{Kind: "skill", Label: "third", Bytes: 100},
	}
	var buf bytes.Buffer
	if err := Render(&buf, comps, 0); err != nil {
		t.Fatalf("Render error: %v", err)
	}
	out := buf.String()
	firstIdx := strings.Index(out, "first")
	secondIdx := strings.Index(out, "second")
	thirdIdx := strings.Index(out, "third")
	if !(firstIdx < secondIdx && secondIdx < thirdIdx) {
		t.Fatalf("expected stable order first<second<third for equal bytes, positions %d/%d/%d in:\n%s",
			firstIdx, secondIdx, thirdIdx, out)
	}
}

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		in   int
		want string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{5989, "5.8 KB"},
	}
	for _, c := range cases {
		if got := formatBytes(c.in); got != c.want {
			t.Errorf("formatBytes(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}
