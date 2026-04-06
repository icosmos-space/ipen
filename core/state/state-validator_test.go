package state

import (
	"testing"

	"github.com/icosmos-space/ipen/core/models"
)

func TestValidateRuntimeState_ValidState(t *testing.T) {
	snapshot := RuntimeStateSnapshot{
		Manifest: models.StateManifest{
			SchemaVersion:      2,
			Language:           models.LanguageEN,
			LastAppliedChapter: 5,
			ProjectionVersion:  1,
		},
		CurrentState: models.CurrentStateState{
			Chapter: 5,
			Facts:   []models.CurrentStateFact{},
		},
		Hooks: models.HooksState{
			Hooks: []models.HookRecord{
				{
					HookID:              "hook1",
					StartChapter:        1,
					Type:                "mystery",
					Status:              models.HookStatusOpenRT,
					LastAdvancedChapter: 3,
				},
			},
		},
		ChapterSummaries: models.ChapterSummariesState{
			Rows: []models.ChapterSummaryRow{
				{Chapter: 1, Title: "Ch1"},
				{Chapter: 2, Title: "Ch2"},
			},
		},
	}

	issues := ValidateRuntimeState(snapshot)
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %d: %+v", len(issues), issues)
	}
}

func TestValidateRuntimeState_InvalidSchemaVersion(t *testing.T) {
	snapshot := RuntimeStateSnapshot{
		Manifest: models.StateManifest{
			SchemaVersion:      1,
			Language:           models.LanguageEN,
			LastAppliedChapter: 0,
		},
		CurrentState:     models.CurrentStateState{Chapter: 0},
		Hooks:            models.HooksState{},
		ChapterSummaries: models.ChapterSummariesState{},
	}

	issues := ValidateRuntimeState(snapshot)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].Code != "INVALID_SCHEMA_VERSION" {
		t.Fatalf("expected INVALID_SCHEMA_VERSION, got %s", issues[0].Code)
	}
	if issues[0].Severity != "error" {
		t.Fatalf("expected error severity, got %s", issues[0].Severity)
	}
}

func TestValidateRuntimeState_InvalidLanguage(t *testing.T) {
	snapshot := RuntimeStateSnapshot{
		Manifest: models.StateManifest{
			SchemaVersion:      2,
			Language:           "fr",
			LastAppliedChapter: 0,
		},
		CurrentState:     models.CurrentStateState{Chapter: 0},
		Hooks:            models.HooksState{},
		ChapterSummaries: models.ChapterSummariesState{},
	}

	issues := ValidateRuntimeState(snapshot)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].Code != "INVALID_LANGUAGE" {
		t.Fatalf("expected INVALID_LANGUAGE, got %s", issues[0].Code)
	}
}

func TestValidateRuntimeState_InvalidLastAppliedChapter(t *testing.T) {
	snapshot := RuntimeStateSnapshot{
		Manifest: models.StateManifest{
			SchemaVersion:      2,
			Language:           models.LanguageEN,
			LastAppliedChapter: -1,
		},
		CurrentState:     models.CurrentStateState{Chapter: -1},
		Hooks:            models.HooksState{},
		ChapterSummaries: models.ChapterSummariesState{},
	}

	issues := ValidateRuntimeState(snapshot)
	// Should have at least INVALID_LAST_APPLIED_CHAPTER and CHAPTER_MISMATCH
	found := false
	for _, issue := range issues {
		if issue.Code == "INVALID_LAST_APPLIED_CHAPTER" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected INVALID_LAST_APPLIED_CHAPTER issue, got: %v", issues)
	}
}

func TestValidateRuntimeState_ChapterMismatch(t *testing.T) {
	snapshot := RuntimeStateSnapshot{
		Manifest: models.StateManifest{
			SchemaVersion:      2,
			Language:           models.LanguageEN,
			LastAppliedChapter: 5,
		},
		CurrentState:     models.CurrentStateState{Chapter: 3},
		Hooks:            models.HooksState{},
		ChapterSummaries: models.ChapterSummariesState{},
	}

	issues := ValidateRuntimeState(snapshot)
	found := false
	for _, issue := range issues {
		if issue.Code == "CHAPTER_MISMATCH" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected CHAPTER_MISMATCH issue, got: %v", issues)
	}
}

func TestValidateRuntimeState_DuplicateHook(t *testing.T) {
	snapshot := RuntimeStateSnapshot{
		Manifest: models.StateManifest{
			SchemaVersion:      2,
			Language:           models.LanguageEN,
			LastAppliedChapter: 0,
		},
		CurrentState: models.CurrentStateState{Chapter: 0},
		Hooks: models.HooksState{
			Hooks: []models.HookRecord{
				{HookID: "h1", StartChapter: 1, LastAdvancedChapter: 1},
				{HookID: "h1", StartChapter: 2, LastAdvancedChapter: 2},
			},
		},
		ChapterSummaries: models.ChapterSummariesState{},
	}

	issues := ValidateRuntimeState(snapshot)
	found := false
	for _, issue := range issues {
		if issue.Code == "DUPLICATE_HOOK" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected DUPLICATE_HOOK issue, got: %v", issues)
	}
}

func TestValidateRuntimeState_InvalidHookStartChapter(t *testing.T) {
	snapshot := RuntimeStateSnapshot{
		Manifest: models.StateManifest{
			SchemaVersion:      2,
			Language:           models.LanguageEN,
			LastAppliedChapter: 0,
		},
		CurrentState: models.CurrentStateState{Chapter: 0},
		Hooks: models.HooksState{
			Hooks: []models.HookRecord{
				{HookID: "h1", StartChapter: -1, LastAdvancedChapter: 0},
			},
		},
		ChapterSummaries: models.ChapterSummariesState{},
	}

	issues := ValidateRuntimeState(snapshot)
	found := false
	for _, issue := range issues {
		if issue.Code == "INVALID_HOOK_START_CHAPTER" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected INVALID_HOOK_START_CHAPTER issue, got: %v", issues)
	}
}

func TestValidateRuntimeState_InvalidHookLastAdvanced(t *testing.T) {
	snapshot := RuntimeStateSnapshot{
		Manifest: models.StateManifest{
			SchemaVersion:      2,
			Language:           models.LanguageEN,
			LastAppliedChapter: 0,
		},
		CurrentState: models.CurrentStateState{Chapter: 0},
		Hooks: models.HooksState{
			Hooks: []models.HookRecord{
				{HookID: "h1", StartChapter: 5, LastAdvancedChapter: 2},
			},
		},
		ChapterSummaries: models.ChapterSummariesState{},
	}

	issues := ValidateRuntimeState(snapshot)
	found := false
	for _, issue := range issues {
		if issue.Code == "INVALID_HOOK_LAST_ADVANCED" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected INVALID_HOOK_LAST_ADVANCED issue, got: %v", issues)
	}
}

func TestValidateRuntimeState_DuplicateSummary(t *testing.T) {
	snapshot := RuntimeStateSnapshot{
		Manifest: models.StateManifest{
			SchemaVersion:      2,
			Language:           models.LanguageEN,
			LastAppliedChapter: 0,
		},
		CurrentState: models.CurrentStateState{Chapter: 0},
		Hooks:        models.HooksState{},
		ChapterSummaries: models.ChapterSummariesState{
			Rows: []models.ChapterSummaryRow{
				{Chapter: 1},
				{Chapter: 1},
			},
		},
	}

	issues := ValidateRuntimeState(snapshot)
	found := false
	for _, issue := range issues {
		if issue.Code == "DUPLICATE_SUMMARY" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected DUPLICATE_SUMMARY issue, got: %v", issues)
	}
}

func TestValidateRuntimeState_InvalidSummaryChapter(t *testing.T) {
	snapshot := RuntimeStateSnapshot{
		Manifest: models.StateManifest{
			SchemaVersion:      2,
			Language:           models.LanguageEN,
			LastAppliedChapter: 0,
		},
		CurrentState: models.CurrentStateState{Chapter: 0},
		Hooks:        models.HooksState{},
		ChapterSummaries: models.ChapterSummariesState{
			Rows: []models.ChapterSummaryRow{
				{Chapter: 0},
			},
		},
	}

	issues := ValidateRuntimeState(snapshot)
	found := false
	for _, issue := range issues {
		if issue.Code == "INVALID_SUMMARY_CHAPTER" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected INVALID_SUMMARY_CHAPTER issue, got: %v", issues)
	}
}

func TestValidateRuntimeState_MultipleIssues(t *testing.T) {
	snapshot := RuntimeStateSnapshot{
		Manifest: models.StateManifest{
			SchemaVersion:      1,
			Language:           "fr",
			LastAppliedChapter: -1,
		},
		CurrentState:     models.CurrentStateState{Chapter: 5},
		Hooks:            models.HooksState{},
		ChapterSummaries: models.ChapterSummariesState{},
	}

	issues := ValidateRuntimeState(snapshot)
	// Should have at least 3 issues
	if len(issues) < 3 {
		t.Fatalf("expected at least 3 issues, got %d", len(issues))
	}

	codes := make(map[string]bool)
	for _, issue := range issues {
		codes[issue.Code] = true
	}

	expected := []string{"INVALID_SCHEMA_VERSION", "INVALID_LANGUAGE", "INVALID_LAST_APPLIED_CHAPTER", "CHAPTER_MISMATCH"}
	for _, code := range expected {
		if !codes[code] {
			t.Fatalf("expected issue code %s, got: %v", code, issues)
		}
	}
}

func TestValidateRuntimeState_ZhLanguage(t *testing.T) {
	snapshot := RuntimeStateSnapshot{
		Manifest: models.StateManifest{
			SchemaVersion:      2,
			Language:           models.LanguageZH,
			LastAppliedChapter: 0,
		},
		CurrentState:     models.CurrentStateState{Chapter: 0},
		Hooks:            models.HooksState{},
		ChapterSummaries: models.ChapterSummariesState{},
	}

	issues := ValidateRuntimeState(snapshot)
	if len(issues) != 0 {
		t.Fatalf("expected no issues for valid zh language, got %d", len(issues))
	}
}
