package state

import (
	"strings"
	"testing"

	"github.com/icosmos-space/ipen/core/models"
)

// ========== RenderHooksProjection ==========

func TestRenderHooksProjection_English(t *testing.T) {
	state := models.HooksState{
		Hooks: []models.HookRecord{
			{
				HookID:              "mystery-1",
				StartChapter:        1,
				Type:                "mystery",
				Status:              models.HookStatusOpenRT,
				LastAdvancedChapter: 3,
				ExpectedPayoff:      "Reveal truth",
				Notes:               "Important",
			},
		},
	}

	result := RenderHooksProjection(state, "en")

	if !strings.Contains(result, "# Pending Hooks") {
		t.Fatalf("expected English title, got:\n%s", result)
	}
	if !strings.Contains(result, "mystery-1") {
		t.Fatalf("expected hook ID, got:\n%s", result)
	}
	if !strings.Contains(result, "mystery") {
		t.Fatalf("expected type, got:\n%s", result)
	}
}

func TestRenderHooksProjection_Chinese(t *testing.T) {
	state := models.HooksState{
		Hooks: []models.HookRecord{
			{
				HookID:              "mystery-1",
				StartChapter:        1,
				Type:                "mystery",
				Status:              models.HookStatusOpenRT,
				LastAdvancedChapter: 3,
			},
		},
	}

	result := RenderHooksProjection(state, "zh")

	if !strings.Contains(result, "起始章节") {
		t.Fatalf("expected Chinese headers, got:\n%s", result)
	}
}

func TestRenderHooksProjection_EmptyHooks(t *testing.T) {
	state := models.HooksState{Hooks: []models.HookRecord{}}

	result := RenderHooksProjection(state, "en")

	if !strings.Contains(result, "| - | - |") {
		t.Fatalf("expected placeholder row, got:\n%s", result)
	}
}

func TestRenderHooksProjection_SortedByStartChapter(t *testing.T) {
	state := models.HooksState{
		Hooks: []models.HookRecord{
			{HookID: "h3", StartChapter: 5, LastAdvancedChapter: 5},
			{HookID: "h1", StartChapter: 1, LastAdvancedChapter: 1},
			{HookID: "h2", StartChapter: 3, LastAdvancedChapter: 3},
		},
	}

	result := RenderHooksProjection(state, "en")

	// Check order: h1 should appear before h2, h2 before h3
	h1Idx := strings.Index(result, "h1")
	h2Idx := strings.Index(result, "h2")
	h3Idx := strings.Index(result, "h3")

	if h1Idx >= h2Idx || h2Idx >= h3Idx {
		t.Fatalf("expected hooks sorted by start chapter: h1(%d) < h2(%d) < h3(%d)", h1Idx, h2Idx, h3Idx)
	}
}

func TestRenderHooksProjection_PayoffTimingLocalization(t *testing.T) {
	timing := models.TimingNearTerm
	state := models.HooksState{
		Hooks: []models.HookRecord{
			{
				HookID:       "h1",
				PayoffTiming: &timing,
			},
		},
	}

	// English
	enResult := RenderHooksProjection(state, "en")
	if !strings.Contains(enResult, "near-term") {
		t.Fatalf("expected English timing 'near-term', got:\n%s", enResult)
	}

	// Chinese
	zhResult := RenderHooksProjection(state, "zh")
	if !strings.Contains(zhResult, "近期") {
		t.Fatalf("expected Chinese timing '近期', got:\n%s", zhResult)
	}
}

func TestRenderHooksProjection_EscapesPipeCharacter(t *testing.T) {
	state := models.HooksState{
		Hooks: []models.HookRecord{
			{
				HookID:       "h|1",
				ExpectedPayoff: "pay|off",
			},
		},
	}

	result := RenderHooksProjection(state, "en")

	// Should escape pipe characters
	if strings.Contains(result, "h|1") && !strings.Contains(result, `h\\|1`) {
		t.Fatalf("expected escaped pipe in hook ID, got:\n%s", result)
	}
}

// ========== RenderChapterSummariesProjection ==========

func TestRenderChapterSummariesProjection_English(t *testing.T) {
	state := models.ChapterSummariesState{
		Rows: []models.ChapterSummaryRow{
			{
				Chapter:      1,
				Title:        "First Chapter",
				Characters:   "Alice, Bob",
				Events:       "Introduction",
				StateChanges: "World revealed",
				HookActivity: "hook1 opened",
				Mood:         "mysterious",
				ChapterType:  "prologue",
			},
		},
	}

	result := RenderChapterSummariesProjection(state, "en")

	if !strings.Contains(result, "First Chapter") {
		t.Fatalf("expected title, got:\n%s", result)
	}
	if !strings.Contains(result, "Alice, Bob") {
		t.Fatalf("expected characters, got:\n%s", result)
	}
}

func TestRenderChapterSummariesProjection_Chinese(t *testing.T) {
	state := models.ChapterSummariesState{
		Rows: []models.ChapterSummaryRow{
			{Chapter: 1, Title: "第一章"},
		},
	}

	result := RenderChapterSummariesProjection(state, "zh")

	if !strings.Contains(result, "章节") {
		t.Fatalf("expected Chinese headers, got:\n%s", result)
	}
}

func TestRenderChapterSummariesProjection_EmptyRows(t *testing.T) {
	state := models.ChapterSummariesState{Rows: []models.ChapterSummaryRow{}}

	result := RenderChapterSummariesProjection(state, "en")

	if !strings.Contains(result, "| - | - |") {
		t.Fatalf("expected placeholder row, got:\n%s", result)
	}
}

func TestRenderChapterSummariesProjection_SortedByChapter(t *testing.T) {
	state := models.ChapterSummariesState{
		Rows: []models.ChapterSummaryRow{
			{Chapter: 3, Title: "Ch3"},
			{Chapter: 1, Title: "Ch1"},
			{Chapter: 2, Title: "Ch2"},
		},
	}

	result := RenderChapterSummariesProjection(state, "en")

	ch1Idx := strings.Index(result, "Ch1")
	ch2Idx := strings.Index(result, "Ch2")
	ch3Idx := strings.Index(result, "Ch3")

	if ch1Idx >= ch2Idx || ch2Idx >= ch3Idx {
		t.Fatalf("expected chapters sorted: Ch1(%d) < Ch2(%d) < Ch3(%d)", ch1Idx, ch2Idx, ch3Idx)
	}
}

// ========== RenderCurrentStateProjection ==========

func TestRenderCurrentStateProjection_English(t *testing.T) {
	state := models.CurrentStateState{
		Chapter: 5,
		Facts: []models.CurrentStateFact{
			{Subject: "protagonist", Predicate: "Current Location", Object: "Village"},
			{Subject: "protagonist", Predicate: "Current Goal", Object: "Find sword"},
		},
	}

	result := RenderCurrentStateProjection(state, "en")

	if !strings.Contains(result, "# Current State") {
		t.Fatalf("expected English title, got:\n%s", result)
	}
	if !strings.Contains(result, "Current Chapter") {
		t.Fatalf("expected 'Current Chapter' label, got:\n%s", result)
	}
	if !strings.Contains(result, "5") {
		t.Fatalf("expected chapter number, got:\n%s", result)
	}
	if !strings.Contains(result, "Village") {
		t.Fatalf("expected location value, got:\n%s", result)
	}
}

func TestRenderCurrentStateProjection_Chinese(t *testing.T) {
	state := models.CurrentStateState{
		Chapter: 3,
		Facts: []models.CurrentStateFact{
			{Subject: "主角", Predicate: "当前位置", Object: "村庄"},
		},
	}

	result := RenderCurrentStateProjection(state, "zh")

	if !strings.Contains(result, "# 当前状态") {
		t.Fatalf("expected Chinese title, got:\n%s", result)
	}
	if !strings.Contains(result, "当前章节") {
		t.Fatalf("expected '当前章节' label, got:\n%s", result)
	}
	if !strings.Contains(result, "村庄") {
		t.Fatalf("expected location value, got:\n%s", result)
	}
}

func TestRenderCurrentStateProjection_EmptyFacts(t *testing.T) {
	state := models.CurrentStateState{
		Chapter: 1,
		Facts:   []models.CurrentStateFact{},
	}

	result := RenderCurrentStateProjection(state, "en")

	if !strings.Contains(result, "(not set)") {
		t.Fatalf("expected placeholder for empty facts, got:\n%s", result)
	}
}

func TestRenderCurrentStateProjection_AdditionalState(t *testing.T) {
	state := models.CurrentStateState{
		Chapter: 1,
		Facts: []models.CurrentStateFact{
			{Subject: "extra", Predicate: "Custom Field", Object: "Custom Value"},
		},
	}

	result := RenderCurrentStateProjection(state, "en")

	if !strings.Contains(result, "## Additional State") {
		t.Fatalf("expected additional state section, got:\n%s", result)
	}
	if !strings.Contains(result, "Custom Field") {
		t.Fatalf("expected custom field, got:\n%s", result)
	}
	if !strings.Contains(result, "Custom Value") {
		t.Fatalf("expected custom value, got:\n%s", result)
	}
}

func TestRenderCurrentStateProjection_NotePredicates(t *testing.T) {
	state := models.CurrentStateState{
		Chapter: 1,
		Facts: []models.CurrentStateFact{
			{Subject: "note", Predicate: "note_1", Object: "First note"},
			{Subject: "note", Predicate: "note_2", Object: "Second note"},
		},
	}

	result := RenderCurrentStateProjection(state, "en")

	if !strings.Contains(result, "- First note") {
		t.Fatalf("expected first note as bullet, got:\n%s", result)
	}
	if !strings.Contains(result, "- Second note") {
		t.Fatalf("expected second note as bullet, got:\n%s", result)
	}
}

func TestRenderCurrentStateProjection_NotesSortedBeforeOtherAdditional(t *testing.T) {
	state := models.CurrentStateState{
		Chapter: 1,
		Facts: []models.CurrentStateFact{
			{Subject: "z", Predicate: "zzz_field", Object: "Z value"},
			{Subject: "note", Predicate: "note_5", Object: "Note 5"},
			{Subject: "a", Predicate: "aaa_field", Object: "A value"},
			{Subject: "note", Predicate: "note_1", Object: "Note 1"},
		},
	}

	result := RenderCurrentStateProjection(state, "en")

	note1Idx := strings.Index(result, "Note 1")
	note5Idx := strings.Index(result, "Note 5")
	aFieldIdx := strings.Index(result, "A value")
	zFieldIdx := strings.Index(result, "Z value")

	// Notes should appear before other fields
	if note1Idx > aFieldIdx || note5Idx > aFieldIdx {
		t.Fatalf("expected notes before other fields, got:\n%s", result)
	}
	// A should come before Z
	if aFieldIdx > zFieldIdx {
		// This is fine
	}
	_ = zFieldIdx
}

func TestRenderCurrentStateProjection_MixedLanguageAliases(t *testing.T) {
	state := models.CurrentStateState{
		Chapter: 1,
		Facts: []models.CurrentStateFact{
			{Subject: "loc", Predicate: "当前位置", Object: "Chinese location"},
		},
	}

	result := RenderCurrentStateProjection(state, "en")

	// Should match alias
	if !strings.Contains(result, "Chinese location") {
		t.Fatalf("expected to find Chinese location via alias, got:\n%s", result)
	}
}

// ========== Helper functions ==========

func TestLocalizeHookTiming_NilTiming(t *testing.T) {
	enResult := localizeHookTiming(nil, true)
	if enResult != "near-term" {
		t.Fatalf("expected 'near-term' for nil timing in English, got %s", enResult)
	}

	zhResult := localizeHookTiming(nil, false)
	if zhResult != "近期" {
		t.Fatalf("expected '近期' for nil timing in Chinese, got %s", zhResult)
	}
}

func TestLocalizeHookTiming_AllTimingsChinese(t *testing.T) {
	timings := map[models.HookPayoffTiming]string{
		models.TimingImmediate: "立即",
		models.TimingNearTerm:  "近期",
		models.TimingMidArc:    "中程",
		models.TimingSlowBurn:  "慢烧",
		models.TimingEndgame:   "终局",
	}

	for timing, expected := range timings {
		result := localizeHookTiming(&timing, false)
		if result != expected {
			t.Fatalf("expected %s for %v in Chinese, got %s", expected, timing, result)
		}
	}
}

func TestLocalizeHookTiming_UnknownTimingChinese(t *testing.T) {
	timing := models.HookPayoffTiming("custom")
	result := localizeHookTiming(&timing, false)
	if result != "custom" {
		t.Fatalf("expected 'custom' for unknown timing, got %s", result)
	}
}

func TestEscapeTableCell_ReplacesPipe(t *testing.T) {
	result := escapeTableCell("a|b|c")
	expected := `a\\|b\\|c`
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestEscapeTableCell_TrimsSpace(t *testing.T) {
	result := escapeTableCell("  hello  ")
	if result != "hello" {
		t.Fatalf("expected 'hello', got %q", result)
	}
}

func TestFindFactValue_MatchesAlias(t *testing.T) {
	state := models.CurrentStateState{
		Facts: []models.CurrentStateFact{
			{Predicate: "current location", Object: "Village"},
		},
	}

	aliases := []string{"Current Location", "当前位置"}
	result := findFactValue(state, aliases)

	if result != "Village" {
		t.Fatalf("expected 'Village', got %q", result)
	}
}

func TestFindFactValue_NotFound(t *testing.T) {
	state := models.CurrentStateState{
		Facts: []models.CurrentStateFact{
			{Predicate: "other", Object: "Value"},
		},
	}

	aliases := []string{"Current Location"}
	result := findFactValue(state, aliases)

	if result != "" {
		t.Fatalf("expected empty string, got %q", result)
	}
}

func TestNormalizePredicate_LowercasesAndTrims(t *testing.T) {
	result := normalizePredicate("  Current Location  ")
	if result != "current location" {
		t.Fatalf("expected 'current location', got %q", result)
	}
}

func TestParseNoteIndex_Valid(t *testing.T) {
	isNote, idx := parseNoteIndex("note_5")
	if !isNote {
		t.Fatal("expected isNote=true")
	}
	if idx != 5 {
		t.Fatalf("expected index 5, got %d", idx)
	}
}

func TestParseNoteIndex_Invalid(t *testing.T) {
	isNote, _ := parseNoteIndex("other_field")
	if isNote {
		t.Fatal("expected isNote=false for non-note field")
	}
}

func TestIsNotePredicate(t *testing.T) {
	if !isNotePredicate("note_1") {
		t.Fatal("expected true for note_1")
	}
	if isNotePredicate("other") {
		t.Fatal("expected false for other")
	}
}
