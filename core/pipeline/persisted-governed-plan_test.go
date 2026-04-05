package pipeline

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadPersistedPlan_ParsesIntentMarkdown(t *testing.T) {
	bookDir := t.TempDir()
	runtimeDir := filepath.Join(bookDir, "story", "runtime")
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	runtimePath := filepath.Join(runtimeDir, "chapter-0007.intent.md")
	content := stringsJoinLines(
		"# Chapter Intent",
		"",
		"## Goal",
		"Bring the focus back to the mentor oath conflict.",
		"",
		"## Outline Node",
		"Track the mentor oath fallout.",
		"",
		"## Must Keep",
		"- Lin Yue keeps the oath token hidden.",
		"- Mentor debt stays unresolved.",
		"",
		"## Must Avoid",
		"- Open a new guild-route mystery.",
		"",
		"## Style Emphasis",
		"- restrained prose",
		"",
		"## Conflicts",
		"- duty: repay the oath without exposing the token",
		"- trust: keep the mentor debt personal",
		"",
	)
	if err := os.WriteFile(runtimePath, []byte(content), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	plan, err := LoadPersistedPlan(bookDir, 7)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if plan == nil {
		t.Fatalf("expected plan, got nil")
	}
	if plan.RuntimePath != runtimePath {
		t.Fatalf("unexpected runtimePath: %s", plan.RuntimePath)
	}
	if plan.Intent.Goal != "Bring the focus back to the mentor oath conflict." {
		t.Fatalf("unexpected goal: %q", plan.Intent.Goal)
	}
	if plan.Intent.OutlineNode == nil || *plan.Intent.OutlineNode != "Track the mentor oath fallout." {
		t.Fatalf("unexpected outlineNode: %#v", plan.Intent.OutlineNode)
	}
	if !reflect.DeepEqual(plan.Intent.MustKeep, []string{
		"Lin Yue keeps the oath token hidden.",
		"Mentor debt stays unresolved.",
	}) {
		t.Fatalf("unexpected mustKeep: %#v", plan.Intent.MustKeep)
	}
	if !reflect.DeepEqual(plan.Intent.MustAvoid, []string{"Open a new guild-route mystery."}) {
		t.Fatalf("unexpected mustAvoid: %#v", plan.Intent.MustAvoid)
	}
	if !reflect.DeepEqual(plan.Intent.StyleEmphasis, []string{"restrained prose"}) {
		t.Fatalf("unexpected styleEmphasis: %#v", plan.Intent.StyleEmphasis)
	}
	if len(plan.Intent.Conflicts) != 2 {
		t.Fatalf("expected 2 conflicts, got %d", len(plan.Intent.Conflicts))
	}
	if plan.Intent.Conflicts[0].Type != "duty" || plan.Intent.Conflicts[0].Resolution != "repay the oath without exposing the token" {
		t.Fatalf("unexpected first conflict: %#v", plan.Intent.Conflicts[0])
	}
	if plan.Intent.Conflicts[1].Type != "trust" || plan.Intent.Conflicts[1].Resolution != "keep the mentor debt personal" {
		t.Fatalf("unexpected second conflict: %#v", plan.Intent.Conflicts[1])
	}
	if !reflect.DeepEqual(plan.PlannerInputs, []string{runtimePath}) {
		t.Fatalf("unexpected planner inputs: %#v", plan.PlannerInputs)
	}
}

func TestLoadPersistedPlan_RejectsPlaceholderGoal(t *testing.T) {
	bookDir := t.TempDir()
	runtimeDir := filepath.Join(bookDir, "story", "runtime")
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	path := filepath.Join(runtimeDir, "chapter-0003.intent.md")
	content := stringsJoinLines(
		"# Chapter Intent",
		"",
		"## Goal",
		"(describe the goal here)",
		"",
	)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	plan, err := LoadPersistedPlan(bookDir, 3)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if plan != nil {
		t.Fatalf("expected nil plan for placeholder goal, got %#v", plan)
	}
}

func TestRelativeToBookDir_NormalizesPath(t *testing.T) {
	result := RelativeToBookDir(
		filepath.FromSlash("/tmp/book"),
		filepath.FromSlash("/tmp/book/story/runtime/chapter-0001.intent.md"),
	)
	if result != "story/runtime/chapter-0001.intent.md" {
		t.Fatalf("unexpected relative path: %s", result)
	}
}

func stringsJoinLines(lines ...string) string {
	return strings.Join(lines, "\n")
}
