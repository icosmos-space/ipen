package agents

import (
	"path/filepath"
	"testing"
)

func TestParseWriterOutput_UsesTaggedSections(t *testing.T) {
	agent := &WriterAgent{}
	raw := `=== CHAPTER_TITLE ===
A Decision

=== CHAPTER_CONTENT ===
Line one.
Line two.

=== PRE_WRITE_CHECK ===
- ok`

	title, content := agent.parseWriterOutput(12, "en", raw)
	if title != "A Decision" {
		t.Fatalf("expected title A Decision, got %q", title)
	}
	if content != "Line one.\nLine two." {
		t.Fatalf("unexpected parsed content: %q", content)
	}
}

func TestParseWriterOutput_FallbacksToHeading(t *testing.T) {
	agent := &WriterAgent{}
	raw := "# Hidden Oath\nBody line"

	title, content := agent.parseWriterOutput(3, "zh", raw)
	if title != "Hidden Oath" {
		t.Fatalf("expected heading title, got %q", title)
	}
	if content != "Body line" {
		t.Fatalf("expected body content, got %q", content)
	}
}

func TestParseWriterOutput_UsesDefaultTitleWhenMissing(t *testing.T) {
	agent := &WriterAgent{}

	title, content := agent.parseWriterOutput(3, "en", "Plain text output")
	if title != "Chapter 3" {
		t.Fatalf("expected default title Chapter 3, got %q", title)
	}
	if content != "Plain text output" {
		t.Fatalf("expected raw content fallback, got %q", content)
	}
}

func TestReadFileOrDefault_MissingFileUsesPlaceholder(t *testing.T) {
	agent := &WriterAgent{}
	missingPath := filepath.Join(t.TempDir(), "missing.md")

	content, err := agent.readFileOrDefault(missingPath)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if content != missingFilePlaceholder {
		t.Fatalf("expected missing placeholder, got %q", content)
	}
}
