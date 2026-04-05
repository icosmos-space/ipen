package pipeline

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/icosmos-space/ipen/core/agents"
	"github.com/icosmos-space/ipen/core/models"
	"github.com/icosmos-space/ipen/core/state"
)

func testBookConfig() *models.BookConfig {
	now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	return &models.BookConfig{
		ID:               "test-book",
		Title:            "Test Book",
		Platform:         models.PlatformTomato,
		Genre:            models.Genre("xuanhuan"),
		Status:           models.StatusActive,
		TargetChapters:   10,
		ChapterWordCount: 3000,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func createValidationResult(passed bool, warnings []agents.ValidationWarning) *agents.ValidationResult {
	return &agents.ValidationResult{
		Passed:   passed,
		Warnings: warnings,
	}
}

func createWriterOutput(overrides func(*agents.WriteChapterOutput)) *agents.WriteChapterOutput {
	out := &agents.WriteChapterOutput{
		ChapterNumber:          3,
		Title:                  "第三章",
		Content:                "铜牌贴在胸口。",
		WordCount:              len([]rune("铜牌贴在胸口。")),
		PreWriteCheck:          "ok",
		PostSettlement:         "ok",
		UpdatedState:           "new state",
		UpdatedLedger:          "new ledger",
		UpdatedHooks:           "new hooks",
		ChapterSummary:         "| 3 | 第三章 |",
		UpdatedSubplots:        "new subplots",
		UpdatedEmotionalArcs:   "new emotional arcs",
		UpdatedCharacterMatrix: "new character matrix",
	}
	if overrides != nil {
		overrides(out)
	}
	return out
}

func TestRetrySettlementAfterValidationFailure_RecoversWithFeedback(t *testing.T) {
	capturedFeedback := ""
	logs := []BiMessage{}
	warns := []string{}

	result, err := RetrySettlementAfterValidationFailure(SettlementRetryParams{
		SettleChapterState: func(input SettlementRetryInput) (*agents.WriteChapterOutput, error) {
			capturedFeedback = input.ValidationFeedback
			return createWriterOutput(func(out *agents.WriteChapterOutput) {
				out.UpdatedState = "fixed state"
				out.UpdatedHooks = "fixed hooks"
			}), nil
		},
		Validate: func(chapterContent string, chapterNumber int, oldState string, newState string, oldHooks string, newHooks string, language string) (*agents.ValidationResult, error) {
			return createValidationResult(true, []agents.ValidationWarning{}), nil
		},
		Book:          testBookConfig(),
		BookDir:       "/tmp/test-book",
		ChapterNumber: 3,
		Title:         "第三章",
		Content:       "铜牌贴在胸口。",
		OldState:      "old state",
		OldHooks:      "old hooks",
		OriginalValidation: createValidationResult(false, []agents.ValidationWarning{{
			Category:    "current-state",
			Description: "铜牌位置与正文矛盾",
		}}),
		Language: "zh",
		LogWarn:  func(message BiMessage) { logs = append(logs, message) },
		LoggerWarn: func(message string) {
			warns = append(warns, message)
		},
	})
	if err != nil {
		t.Fatalf("retry failed: %v", err)
	}
	if result.Kind != "recovered" {
		t.Fatalf("expected recovered, got %s", result.Kind)
	}
	if !strings.Contains(capturedFeedback, "上一次状态结算未通过校验") {
		t.Fatalf("expected zh feedback header, got %q", capturedFeedback)
	}
	if !strings.Contains(capturedFeedback, "铜牌位置与正文矛盾") {
		t.Fatalf("expected warning detail in feedback, got %q", capturedFeedback)
	}
	if len(logs) == 0 || !strings.Contains(logs[0].ZH, "仅重试结算层") {
		t.Fatalf("expected retry log, got %#v", logs)
	}
	if len(warns) != 0 {
		t.Fatalf("expected no logger warns on recovered path, got %#v", warns)
	}
}

func TestRetrySettlementAfterValidationFailure_ReturnsDegradedIssues(t *testing.T) {
	result, err := RetrySettlementAfterValidationFailure(SettlementRetryParams{
		SettleChapterState: func(input SettlementRetryInput) (*agents.WriteChapterOutput, error) {
			return createWriterOutput(nil), nil
		},
		Validate: func(chapterContent string, chapterNumber int, oldState string, newState string, oldHooks string, newHooks string, language string) (*agents.ValidationResult, error) {
			return createValidationResult(false, []agents.ValidationWarning{{
				Category:    "current-state",
				Description: "挂坠状态仍与正文冲突",
			}}), nil
		},
		Book:          testBookConfig(),
		BookDir:       "/tmp/test-book",
		ChapterNumber: 3,
		Title:         "第三章",
		Content:       "铜牌贴在胸口。",
		OldState:      "old state",
		OldHooks:      "old hooks",
		OriginalValidation: createValidationResult(false, []agents.ValidationWarning{{
			Category:    "current-state",
			Description: "挂坠状态仍与正文冲突",
		}}),
		Language: "zh",
	})
	if err != nil {
		t.Fatalf("retry failed: %v", err)
	}
	if result.Kind != "degraded" {
		t.Fatalf("expected degraded, got %s", result.Kind)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	issue := result.Issues[0]
	if issue.Category != "state-validation" || issue.Description != "挂坠状态仍与正文冲突" {
		t.Fatalf("unexpected issue: %#v", issue)
	}
	if issue.Suggestion != "请先基于已保存正文修复本章 state，再继续后续章节。" {
		t.Fatalf("unexpected suggestion: %q", issue.Suggestion)
	}
}

func TestBuildStateDegradedPersistenceOutput_FreezesTruthFiles(t *testing.T) {
	output := createWriterOutput(func(out *agents.WriteChapterOutput) {
		out.RuntimeStateDelta = &models.RuntimeStateDelta{Chapter: 3}
		out.RuntimeStateSnapshot = &state.RuntimeStateSnapshot{}
		out.UpdatedChapterSummaries = "| 3 | 新摘要 |"
	})

	degraded := BuildStateDegradedPersistenceOutput(output, "stable state", "stable hooks", "stable ledger")
	if degraded.UpdatedState != "stable state" || degraded.UpdatedHooks != "stable hooks" || degraded.UpdatedLedger != "stable ledger" {
		t.Fatalf("truth freeze failed: %#v", degraded)
	}
	if degraded.RuntimeStateDelta != nil || degraded.RuntimeStateSnapshot != nil {
		t.Fatalf("expected runtime delta/snapshot cleared")
	}
	if degraded.UpdatedChapterSummaries != "" {
		t.Fatalf("expected updated chapter summaries cleared")
	}
}

func TestStateDegradedReviewMetadata_Roundtrip(t *testing.T) {
	issues := []agents.AuditIssue{{
		Severity:    "warning",
		Category:    "state-validation",
		Description: "状态结算重试后仍未通过校验。",
		Suggestion:  "请先基于已保存正文修复本章 state，再继续后续章节。",
	}}
	note := BuildStateDegradedReviewNote("audit-failed", issues)
	parsed := ParseStateDegradedReviewNote(note)
	if parsed == nil {
		t.Fatalf("expected parsed metadata")
	}
	if parsed.Kind != "state-degraded" || parsed.BaseStatus != "audit-failed" {
		t.Fatalf("unexpected parsed note: %#v", parsed)
	}
	if !reflect.DeepEqual(parsed.InjectedIssues, []string{"[warning] 状态结算重试后仍未通过校验。"}) {
		t.Fatalf("unexpected injected issues: %#v", parsed.InjectedIssues)
	}

	now := time.Now()
	chapter := &models.ChapterMeta{
		Number:         3,
		Title:          "第三章",
		Status:         models.StatusStateDegraded,
		WordCount:      1200,
		CreatedAt:      now,
		UpdatedAt:      now,
		AuditIssues:    []string{},
		LengthWarnings: []string{},
		ReviewNote:     note,
	}
	if ResolveStateDegradedBaseStatus(chapter) != "audit-failed" {
		t.Fatalf("expected audit-failed from metadata")
	}

	chapter.ReviewNote = "{bad json"
	chapter.AuditIssues = []string{"[critical] still broken"}
	if ResolveStateDegradedBaseStatus(chapter) != "audit-failed" {
		t.Fatalf("expected critical fallback")
	}
	chapter.AuditIssues = []string{"[warning] needs review"}
	if ResolveStateDegradedBaseStatus(chapter) != "ready-for-review" {
		t.Fatalf("expected warning fallback")
	}
}
