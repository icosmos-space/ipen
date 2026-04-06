package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/icosmos-space/ipen/core/models"
)

// ========== LoadRuntimeStateSnapshot ==========

func TestLoadRuntimeStateSnapshot_LoadsValidSnapshot(t *testing.T) {
	bookDir := setupTestBookDir(t)

	// Create valid state files
	createValidStateFiles(t, bookDir)

	snapshot, err := LoadRuntimeStateSnapshot(bookDir)
	if err != nil {
		t.Fatalf("LoadRuntimeStateSnapshot failed: %v", err)
	}

	if snapshot.Manifest.SchemaVersion != 2 {
		t.Fatalf("expected schema version 2, got %d", snapshot.Manifest.SchemaVersion)
	}
}

// ========== SaveRuntimeStateSnapshot ==========

func TestSaveRuntimeStateSnapshot_CreatesFiles(t *testing.T) {
	bookDir := setupTestBookDir(t)

	snapshot := RuntimeStateSnapshot{
		Manifest: models.StateManifest{
			SchemaVersion:      2,
			Language:           models.LanguageEN,
			LastAppliedChapter: 5,
			ProjectionVersion:  1,
		},
		CurrentState: models.CurrentStateState{
			Chapter: 5,
			Facts: []models.CurrentStateFact{
				{Subject: "protagonist", Predicate: "location", Object: "Village", ValidFromChapter: 5, SourceChapter: 5},
			},
		},
		Hooks: models.HooksState{
			Hooks: []models.HookRecord{
				{HookID: "h1", StartChapter: 1, Status: models.HookStatusOpenRT},
			},
		},
		ChapterSummaries: models.ChapterSummariesState{
			Rows: []models.ChapterSummaryRow{
				{Chapter: 1, Title: "Ch1"},
			},
		},
	}

	err := SaveRuntimeStateSnapshot(bookDir, snapshot)
	if err != nil {
		t.Fatalf("SaveRuntimeStateSnapshot failed: %v", err)
	}

	// Verify files exist
	stateDir := filepath.Join(bookDir, "story", "state")
	expectedFiles := []string{"manifest.json", "current_state.json", "hooks.json", "chapter_summaries.json"}
	for _, f := range expectedFiles {
		path := filepath.Join(stateDir, f)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected file %s to exist", f)
		}
	}
}

func TestSaveRuntimeStateSnapshot_PersistsAndReloads(t *testing.T) {
	bookDir := setupTestBookDir(t)

	snapshot := RuntimeStateSnapshot{
		Manifest: models.StateManifest{
			SchemaVersion:      2,
			Language:           models.LanguageEN,
			LastAppliedChapter: 3,
			ProjectionVersion:  2,
		},
		CurrentState: models.CurrentStateState{Chapter: 3},
		Hooks:        models.HooksState{},
		ChapterSummaries: models.ChapterSummariesState{
			Rows: []models.ChapterSummaryRow{{Chapter: 1}, {Chapter: 2}, {Chapter: 3}},
		},
	}

	err := SaveRuntimeStateSnapshot(bookDir, snapshot)
	if err != nil {
		t.Fatalf("SaveRuntimeStateSnapshot failed: %v", err)
	}

	loaded, err := LoadRuntimeStateSnapshot(bookDir)
	if err != nil {
		t.Fatalf("LoadRuntimeStateSnapshot failed: %v", err)
	}

	if loaded.Manifest.ProjectionVersion != 2 {
		t.Fatalf("expected projection version 2, got %d", loaded.Manifest.ProjectionVersion)
	}
	if len(loaded.ChapterSummaries.Rows) != 3 {
		t.Fatalf("expected 3 summary rows, got %d", len(loaded.ChapterSummaries.Rows))
	}
}

// ========== BuildRuntimeStateArtifacts ==========

func TestBuildRuntimeStateArtifacts_AppliesDelta(t *testing.T) {
	bookDir := setupTestBookDir(t)
	createValidStateFiles(t, bookDir)

	delta := models.RuntimeStateDelta{
		Chapter: 6,
		ChapterSummary: &models.ChapterSummaryRow{
			Chapter:      6,
			Title:        "Chapter 6",
			Characters:   "Alice",
			Events:       "New events",
			StateChanges: "State changed",
			HookActivity: "hook advanced",
			Mood:         "tense",
			ChapterType:  "mainline",
		},
	}

	artifacts, err := BuildRuntimeStateArtifacts(bookDir, delta, "en")
	if err != nil {
		t.Fatalf("BuildRuntimeStateArtifacts failed: %v", err)
	}

	if artifacts.Snapshot.CurrentState.Chapter != 6 {
		t.Fatalf("expected chapter 6, got %d", artifacts.Snapshot.CurrentState.Chapter)
	}
	if !stringsContains(artifacts.CurrentStateMarkdown, "Current Chapter") || !stringsContains(artifacts.CurrentStateMarkdown, "| 6 |") {
		t.Fatalf("expected chapter 6 in markdown, got:\n%s", artifacts.CurrentStateMarkdown)
	}
}

// ========== LoadNarrativeMemorySeed ==========

func TestLoadNarrativeMemorySeed_LoadsSummariesAndHooks(t *testing.T) {
	bookDir := setupTestBookDir(t)
	createValidStateFiles(t, bookDir)

	seed, err := LoadNarrativeMemorySeed(bookDir)
	if err != nil {
		t.Fatalf("LoadNarrativeMemorySeed failed: %v", err)
	}

	if len(seed.Summaries) != 2 {
		t.Fatalf("expected 2 summaries, got %d", len(seed.Summaries))
	}
	if len(seed.Hooks) != 1 {
		t.Fatalf("expected 1 hook, got %d", len(seed.Hooks))
	}
}

func TestLoadNarrativeMemorySeed_PayoffTiming(t *testing.T) {
	bookDir := setupTestBookDir(t)

	timing := models.TimingNearTerm
	stateDir := filepath.Join(bookDir, "story", "state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatalf("failed to create state dir: %v", err)
	}

	manifest := models.StateManifest{
		SchemaVersion:      2,
		Language:           models.LanguageEN,
		LastAppliedChapter: 0,
	}
	writeTestJSON(t, filepath.Join(stateDir, "manifest.json"), manifest)
	writeTestJSON(t, filepath.Join(stateDir, "current_state.json"), models.CurrentStateState{Chapter: 0})
	writeTestJSON(t, filepath.Join(stateDir, "hooks.json"), models.HooksState{
		Hooks: []models.HookRecord{
			{HookID: "h1", StartChapter: 1, LastAdvancedChapter: 1, PayoffTiming: &timing},
		},
	})
	writeTestJSON(t, filepath.Join(stateDir, "chapter_summaries.json"), models.ChapterSummariesState{})

	seed, err := LoadNarrativeMemorySeed(bookDir)
	if err != nil {
		t.Fatalf("LoadNarrativeMemorySeed failed: %v", err)
	}

	if seed.Hooks[0].PayoffTiming != string(timing) {
		t.Fatalf("expected payoff timing %s, got %s", timing, seed.Hooks[0].PayoffTiming)
	}
}

func TestLoadNarrativeMemorySeed_NilPayoffTiming(t *testing.T) {
	bookDir := setupTestBookDir(t)

	stateDir := filepath.Join(bookDir, "story", "state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		t.Fatalf("failed to create state dir: %v", err)
	}

	manifest := models.StateManifest{
		SchemaVersion:      2,
		Language:           models.LanguageEN,
		LastAppliedChapter: 0,
	}
	writeTestJSON(t, filepath.Join(stateDir, "manifest.json"), manifest)
	writeTestJSON(t, filepath.Join(stateDir, "current_state.json"), models.CurrentStateState{Chapter: 0})
	writeTestJSON(t, filepath.Join(stateDir, "hooks.json"), models.HooksState{
		Hooks: []models.HookRecord{
			{HookID: "h1", StartChapter: 1, LastAdvancedChapter: 1, PayoffTiming: nil},
		},
	})
	writeTestJSON(t, filepath.Join(stateDir, "chapter_summaries.json"), models.ChapterSummariesState{})

	seed, err := LoadNarrativeMemorySeed(bookDir)
	if err != nil {
		t.Fatalf("LoadNarrativeMemorySeed failed: %v", err)
	}

	if seed.Hooks[0].PayoffTiming != "" {
		t.Fatalf("expected empty payoff timing, got %s", seed.Hooks[0].PayoffTiming)
	}
}

// ========== LoadSnapshotCurrentStateFacts ==========

func TestLoadSnapshotCurrentStateFacts_LoadsStructuredState(t *testing.T) {
	bookDir := setupTestBookDir(t)

	// Create snapshot with structured state
	snapshotDir := filepath.Join(bookDir, "story", "snapshots", "5", "state")
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		t.Fatalf("failed to create snapshot state dir: %v", err)
	}

	structuredState := models.CurrentStateState{
		Chapter: 5,
		Facts: []models.CurrentStateFact{
			{Subject: "Alice", Predicate: "location", Object: "Village", ValidFromChapter: 5, SourceChapter: 5},
		},
	}
	writeTestJSON(t, filepath.Join(snapshotDir, "current_state.json"), structuredState)

	facts, err := LoadSnapshotCurrentStateFacts(bookDir, 5)
	if err != nil {
		t.Fatalf("LoadSnapshotCurrentStateFacts failed: %v", err)
	}

	if len(facts) != 1 {
		t.Fatalf("expected 1 fact, got %d", len(facts))
	}
	if facts[0].Object != "Village" {
		t.Fatalf("expected object 'Village', got %s", facts[0].Object)
	}
}

func TestLoadSnapshotCurrentStateFacts_FallbackToMarkdown(t *testing.T) {
	bookDir := setupTestBookDir(t)

	// Create snapshot with markdown only
	snapshotDir := filepath.Join(bookDir, "story", "snapshots", "3")
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		t.Fatalf("failed to create snapshot dir: %v", err)
	}

	markdown := `| Field | Value |
| --- | --- |
| Current Chapter | 3 |
| location | Forest |
`
	if err := os.WriteFile(filepath.Join(snapshotDir, "current_state.md"), []byte(markdown), 0644); err != nil {
		t.Fatalf("failed to write markdown: %v", err)
	}

	facts, err := LoadSnapshotCurrentStateFacts(bookDir, 3)
	if err != nil {
		t.Fatalf("LoadSnapshotCurrentStateFacts failed: %v", err)
	}

	if len(facts) != 1 {
		t.Fatalf("expected 1 fact, got %d", len(facts))
	}
	if facts[0].Object != "Forest" {
		t.Fatalf("expected object 'Forest', got %s", facts[0].Object)
	}
}

// ========== Helper functions ==========

func setupTestBookDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bookDir := filepath.Join(dir, "books", "test-book")
	if err := os.MkdirAll(filepath.Join(bookDir, "story", "state"), 0755); err != nil {
		t.Fatalf("failed to create book dir: %v", err)
	}
	// Create book.json
	bookConfig := map[string]string{"language": "en"}
	writeTestJSON(t, filepath.Join(bookDir, "book.json"), bookConfig)
	return bookDir
}

func createValidStateFiles(t *testing.T, bookDir string) {
	t.Helper()
	stateDir := filepath.Join(bookDir, "story", "state")

	manifest := models.StateManifest{
		SchemaVersion:      2,
		Language:           models.LanguageEN,
		LastAppliedChapter: 5,
		ProjectionVersion:  1,
	}
	writeTestJSON(t, filepath.Join(stateDir, "manifest.json"), manifest)

	currentState := models.CurrentStateState{
		Chapter: 5,
		Facts: []models.CurrentStateFact{
			{Subject: "protagonist", Predicate: "location", Object: "Village", ValidFromChapter: 5, SourceChapter: 5},
		},
	}
	writeTestJSON(t, filepath.Join(stateDir, "current_state.json"), currentState)

	hooks := models.HooksState{
		Hooks: []models.HookRecord{
			{HookID: "mystery-1", StartChapter: 1, Status: models.HookStatusOpenRT, LastAdvancedChapter: 3},
		},
	}
	writeTestJSON(t, filepath.Join(stateDir, "hooks.json"), hooks)

	summaries := models.ChapterSummariesState{
		Rows: []models.ChapterSummaryRow{
			{Chapter: 1, Title: "Ch1", Characters: "Alice", Events: "Intro"},
			{Chapter: 2, Title: "Ch2", Characters: "Bob", Events: "Development"},
		},
	}
	writeTestJSON(t, filepath.Join(stateDir, "chapter_summaries.json"), summaries)
}

func writeTestJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("failed to write JSON file: %v", err)
	}
}

func stringsContains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (len(s) >= len(substr)) && (s == substr || len(s) > len(substr) && searchSubstring(s, substr))
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
