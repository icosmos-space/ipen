package state

import (
	"strings"
	"testing"

	"github.com/icosmos-space/ipen/core/models"
)

func TestApplyRuntimeStateDelta_AppliesChapterLocalDelta(t *testing.T) {
	snapshot := RuntimeStateSnapshot{
		Manifest: models.StateManifest{
			SchemaVersion:      2,
			Language:           models.LanguageEN,
			LastAppliedChapter: 11,
			ProjectionVersion:  1,
			MigrationWarnings:  []string{},
		},
		CurrentState: models.CurrentStateState{
			Chapter: 11,
			Facts:   []models.CurrentStateFact{},
		},
		Hooks: models.HooksState{
			Hooks: []models.HookRecord{
				{
					HookID:              "mentor-debt",
					StartChapter:        1,
					Type:                "relationship",
					Status:              "open",
					LastAdvancedChapter: 11,
					ExpectedPayoff:      "Reveal the debt.",
					Notes:               "Still unresolved.",
				},
			},
		},
		ChapterSummaries: models.ChapterSummariesState{
			Rows: []models.ChapterSummaryRow{
				{
					Chapter:      11,
					Title:        "Old Ledger",
					Characters:   "Lin Yue",
					Events:       "Finds old ledger",
					StateChanges: "Debt trail tightens",
					HookActivity: "mentor-debt advanced",
					Mood:         "tense",
					ChapterType:  "mainline",
				},
			},
		},
	}

	currentGoal := "Trace the debt through the river-port ledger."
	delta := models.RuntimeStateDelta{
		Chapter: 12,
		CurrentStatePatch: &models.CurrentStatePatch{
			CurrentGoal: &currentGoal,
		},
		HookOps: models.HookOps{
			Upsert: []models.HookRecord{
				{
					HookID:              "mentor-debt",
					StartChapter:        1,
					Type:                "relationship",
					Status:              "progressing",
					LastAdvancedChapter: 12,
					ExpectedPayoff:      "Reveal the debt.",
					Notes:               "The river-port ledger sharpens the clue.",
				},
			},
			Resolve: []string{},
			Defer:   []string{},
		},
		ChapterSummary: &models.ChapterSummaryRow{
			Chapter:      12,
			Title:        "River-Port Ledger",
			Characters:   "Lin Yue",
			Events:       "Cross-checks the ledger",
			StateChanges: "Debt trail narrows",
			HookActivity: "mentor-debt advanced",
			Mood:         "tight",
			ChapterType:  "investigation",
		},
	}

	result, err := ApplyRuntimeStateDelta(snapshot, delta)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Manifest.LastAppliedChapter != 12 {
		t.Fatalf("expected last applied chapter 12, got %d", result.Manifest.LastAppliedChapter)
	}
	if result.CurrentState.Chapter != 12 {
		t.Fatalf("expected current state chapter 12, got %d", result.CurrentState.Chapter)
	}

	var foundGoal bool
	for _, fact := range result.CurrentState.Facts {
		if fact.Predicate == "Current Goal" && fact.Object == currentGoal && fact.SourceChapter == 12 {
			foundGoal = true
			break
		}
	}
	if !foundGoal {
		t.Fatalf("expected Current Goal fact to be patched")
	}

	if len(result.Hooks.Hooks) != 1 {
		t.Fatalf("expected 1 hook, got %d", len(result.Hooks.Hooks))
	}
	if result.Hooks.Hooks[0].Status != "progressing" {
		t.Fatalf("expected hook status progressing, got %s", result.Hooks.Hooks[0].Status)
	}
	if result.Hooks.Hooks[0].LastAdvancedChapter != 12 {
		t.Fatalf("expected hook lastAdvancedChapter 12, got %d", result.Hooks.Hooks[0].LastAdvancedChapter)
	}

	if len(result.ChapterSummaries.Rows) != 2 {
		t.Fatalf("expected 2 chapter summary rows, got %d", len(result.ChapterSummaries.Rows))
	}
	if result.ChapterSummaries.Rows[0].Chapter != 11 || result.ChapterSummaries.Rows[1].Chapter != 12 {
		t.Fatalf("expected summary chapters [11, 12], got [%d, %d]",
			result.ChapterSummaries.Rows[0].Chapter, result.ChapterSummaries.Rows[1].Chapter)
	}
}

func TestApplyRuntimeStateDelta_RejectsDuplicateSummaryRows(t *testing.T) {
	snapshot := RuntimeStateSnapshot{
		Manifest: models.StateManifest{
			SchemaVersion:      2,
			Language:           models.LanguageZH,
			LastAppliedChapter: 11,
			ProjectionVersion:  1,
			MigrationWarnings:  []string{},
		},
		CurrentState: models.CurrentStateState{
			Chapter: 11,
			Facts:   []models.CurrentStateFact{},
		},
		Hooks: models.HooksState{Hooks: []models.HookRecord{}},
		ChapterSummaries: models.ChapterSummariesState{
			Rows: []models.ChapterSummaryRow{
				{Chapter: 12, Title: "已有章节摘要"},
			},
		},
	}

	delta := models.RuntimeStateDelta{
		Chapter: 12,
		HookOps: models.HookOps{
			Upsert:  []models.HookRecord{},
			Resolve: []string{},
			Defer:   []string{},
		},
		ChapterSummary: &models.ChapterSummaryRow{Chapter: 12, Title: "重复章节摘要"},
	}

	_, err := ApplyRuntimeStateDelta(snapshot, delta)
	if err == nil {
		t.Fatalf("expected duplicate summary error, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "duplicate summary") {
		t.Fatalf("expected duplicate summary error, got %v", err)
	}
}

func TestApplyRuntimeStateDelta_IgnoresUnknownResolveDeferHooks(t *testing.T) {
	snapshot := RuntimeStateSnapshot{
		Manifest: models.StateManifest{
			SchemaVersion:      2,
			Language:           models.LanguageEN,
			LastAppliedChapter: 11,
			ProjectionVersion:  1,
			MigrationWarnings:  []string{},
		},
		CurrentState: models.CurrentStateState{
			Chapter: 11,
			Facts:   []models.CurrentStateFact{},
		},
		Hooks:            models.HooksState{Hooks: []models.HookRecord{}},
		ChapterSummaries: models.ChapterSummariesState{Rows: []models.ChapterSummaryRow{}},
	}

	delta := models.RuntimeStateDelta{
		Chapter: 12,
		HookOps: models.HookOps{
			Upsert:  []models.HookRecord{},
			Resolve: []string{"mentor-debt"},
			Defer:   []string{"mentor-debt-later"},
		},
	}

	result, err := ApplyRuntimeStateDelta(snapshot, delta)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Manifest.LastAppliedChapter != 12 {
		t.Fatalf("expected last applied chapter 12, got %d", result.Manifest.LastAppliedChapter)
	}
	if len(result.Hooks.Hooks) != 0 {
		t.Fatalf("expected no hooks, got %d", len(result.Hooks.Hooks))
	}
}
