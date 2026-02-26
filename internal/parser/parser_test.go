package parser

import (
	"testing"
)

func TestParse_FrontmatterAndBody(t *testing.T) {
	input := []byte("---\ntitle: Hello\ntags:\n  - go\n  - kenaz\n---\n# Hello\nBody text.\n")
	r, err := Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Title != "Hello" {
		t.Errorf("title = %q, want %q", r.Title, "Hello")
	}
	if len(r.Tags) < 2 || r.Tags[0] != "go" || r.Tags[1] != "kenaz" {
		t.Errorf("tags = %v, want [go kenaz]", r.Tags)
	}
	if r.Body != "# Hello\nBody text.\n" {
		t.Errorf("body = %q", r.Body)
	}
}

func TestParse_NoFrontmatter(t *testing.T) {
	input := []byte("# Just a heading\nSome text.\n")
	r, err := Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Frontmatter != nil {
		t.Errorf("expected nil frontmatter, got %v", r.Frontmatter)
	}
	if r.Title != "Just a heading" {
		t.Errorf("title = %q, want %q", r.Title, "Just a heading")
	}
}

func TestParse_InvalidYAMLFallback(t *testing.T) {
	input := []byte("---\n: invalid: yaml: {{{\n---\nBody\n")
	r, err := Parse(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Invalid YAML falls back to treating everything as body.
	if r.Frontmatter != nil {
		t.Errorf("expected nil frontmatter on invalid YAML")
	}
}

func TestExtractLinks_Basic(t *testing.T) {
	body := "See [[Note A]] and [[Note B|alias]].\nAlso [[Note A]] again."
	links := extractLinks(body)
	if len(links) != 2 {
		t.Fatalf("len(links) = %d, want 2", len(links))
	}
	if links[0] != "Note A" || links[1] != "Note B" {
		t.Errorf("links = %v", links)
	}
}

func TestExtractLinks_EmptyTarget(t *testing.T) {
	links := extractLinks("see [[ ]] and [[|alias]]")
	if len(links) != 0 {
		t.Errorf("expected no links, got %v", links)
	}
}

func TestExtractTags_InlineAndFrontmatter(t *testing.T) {
	fm := map[string]any{
		"tags": []any{"alpha"},
	}
	body := "Some text #beta and #alpha again."
	tags := extractTags(body, fm)
	// alpha from FM, beta from body; alpha not duplicated.
	if len(tags) != 2 || tags[0] != "alpha" || tags[1] != "beta" {
		t.Errorf("tags = %v, want [alpha beta]", tags)
	}
}

func TestDeriveTitle_FrontmatterOverH1(t *testing.T) {
	fm := map[string]any{"title": "FM Title"}
	body := "# H1 Title\ntext"
	title := deriveTitle(fm, body)
	if title != "FM Title" {
		t.Errorf("title = %q, want %q", title, "FM Title")
	}
}

func TestDeriveTitle_H1Fallback(t *testing.T) {
	title := deriveTitle(nil, "some text\n# My Heading\nmore")
	if title != "My Heading" {
		t.Errorf("title = %q, want %q", title, "My Heading")
	}
}
