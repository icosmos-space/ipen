package pipeline

import (
	"strings"
	"testing"

	"github.com/icosmos-space/ipen/core/agents"
)

func TestValidateChapterTruthPersistence_UsesRecoveredOutput(t *testing.T) {
	validatorCalls := 0
	var writerInput SettlementRetryInput
	loggerWarnings := []string{}

	result, err := ValidateChapterTruthPersistence(TruthValidationParams{
		SettleChapterState: func(input SettlementRetryInput) (*agents.WriteChapterOutput, error) {
			writerInput = input
			return createWriterOutput(func(out *agents.WriteChapterOutput) {
				out.UpdatedState = "fixed state"
				out.UpdatedHooks = "fixed hooks"
				out.UpdatedLedger = "fixed ledger"
			}), nil
		},
		Validate: func(chapterContent string, chapterNumber int, oldState string, newState string, oldHooks string, newHooks string, language string) (*agents.ValidationResult, error) {
			validatorCalls++
			if validatorCalls == 1 {
				return &agents.ValidationResult{
					Passed: false,
					Warnings: []agents.ValidationWarning{{
						Category:    "unsupported_change",
						Description: "正文写铜牌在怀里，但 state 说未携带。",
					}},
				}, nil
			}
			return &agents.ValidationResult{
				Passed:   true,
				Warnings: []agents.ValidationWarning{},
			}, nil
		},
		Book:          testBookConfig(),
		BookDir:       "/tmp/book",
		ChapterNumber: 3,
		Title:         "Test Chapter",
		Content:       "Healthy chapter body with the copper token in his coat.",
		PersistenceOutput: createWriterOutput(func(out *agents.WriteChapterOutput) {
			out.UpdatedState = "broken state"
			out.UpdatedHooks = "broken hooks"
			out.UpdatedLedger = "broken ledger"
		}),
		AuditResult: agents.AuditResult{
			Passed:  true,
			Issues:  []agents.AuditIssue{},
			Summary: "clean",
		},
		PreviousTruth: struct {
			OldState  string
			OldHooks  string
			OldLedger string
		}{
			OldState:  "stable state",
			OldHooks:  "stable hooks",
			OldLedger: "stable ledger",
		},
		Language: "zh",
		LoggerWarn: func(message string) {
			loggerWarnings = append(loggerWarnings, message)
		},
	})
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if result.ChapterStatus != "" {
		t.Fatalf("expected no degraded status, got %q", result.ChapterStatus)
	}
	if result.PersistenceOutput.UpdatedState != "fixed state" || result.PersistenceOutput.UpdatedHooks != "fixed hooks" {
		t.Fatalf("expected recovered output, got %#v", result.PersistenceOutput)
	}
	if len(result.AuditResult.Issues) != 0 {
		t.Fatalf("expected no injected issues, got %#v", result.AuditResult.Issues)
	}
	if !strings.Contains(writerInput.ValidationFeedback, "铜牌在怀里") {
		t.Fatalf("expected validation feedback to contain warning detail: %q", writerInput.ValidationFeedback)
	}
	if len(loggerWarnings) == 0 || loggerWarnings[0] != "  [unsupported_change] 正文写铜牌在怀里，但 state 说未携带。" {
		t.Fatalf("unexpected logger warnings: %#v", loggerWarnings)
	}
}

func TestValidateChapterTruthPersistence_DegradesWhenRetryFails(t *testing.T) {
	validatorCalls := 0
	baseIssue := agents.AuditIssue{
		Severity:    "warning",
		Category:    "title-dedup",
		Description: "title adjusted",
		Suggestion:  "check title",
	}

	result, err := ValidateChapterTruthPersistence(TruthValidationParams{
		SettleChapterState: func(input SettlementRetryInput) (*agents.WriteChapterOutput, error) {
			return createWriterOutput(func(out *agents.WriteChapterOutput) {
				out.UpdatedState = "still broken state"
				out.UpdatedHooks = "still broken hooks"
				out.UpdatedLedger = "still broken ledger"
			}), nil
		},
		Validate: func(chapterContent string, chapterNumber int, oldState string, newState string, oldHooks string, newHooks string, language string) (*agents.ValidationResult, error) {
			validatorCalls++
			if validatorCalls == 1 {
				return &agents.ValidationResult{
					Passed: false,
					Warnings: []agents.ValidationWarning{{
						Category:    "unsupported_change",
						Description: "第一次校验失败。",
					}},
				}, nil
			}
			return &agents.ValidationResult{
				Passed: false,
				Warnings: []agents.ValidationWarning{{
					Category:    "unsupported_change",
					Description: "重试后仍然失败。",
				}},
			}, nil
		},
		Book:          testBookConfig(),
		BookDir:       "/tmp/book",
		ChapterNumber: 4,
		Title:         "Test Chapter",
		Content:       "Healthy chapter body with the copper token in his coat.",
		PersistenceOutput: createWriterOutput(func(out *agents.WriteChapterOutput) {
			out.UpdatedState = "broken state"
			out.UpdatedHooks = "broken hooks"
			out.UpdatedLedger = "broken ledger"
		}),
		AuditResult: agents.AuditResult{
			Passed:  true,
			Issues:  []agents.AuditIssue{baseIssue},
			Summary: "clean",
		},
		PreviousTruth: struct {
			OldState  string
			OldHooks  string
			OldLedger string
		}{
			OldState:  "stable state",
			OldHooks:  "stable hooks",
			OldLedger: "stable ledger",
		},
		Language: "zh",
	})
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if result.ChapterStatus != "state-degraded" {
		t.Fatalf("expected state-degraded, got %q", result.ChapterStatus)
	}
	if len(result.DegradedIssues) != 1 || result.DegradedIssues[0].Category != "state-validation" || result.DegradedIssues[0].Description != "重试后仍然失败。" {
		t.Fatalf("unexpected degraded issues: %#v", result.DegradedIssues)
	}
	if result.PersistenceOutput.UpdatedState != "stable state" || result.PersistenceOutput.UpdatedHooks != "stable hooks" || result.PersistenceOutput.UpdatedLedger != "stable ledger" {
		t.Fatalf("expected persistence output frozen to old truth, got %#v", result.PersistenceOutput)
	}
	if len(result.AuditResult.Issues) != 2 {
		t.Fatalf("expected 2 audit issues, got %#v", result.AuditResult.Issues)
	}
	if result.AuditResult.Issues[1].Category != "state-validation" || result.AuditResult.Issues[1].Description != "重试后仍然失败。" {
		t.Fatalf("unexpected injected audit issue: %#v", result.AuditResult.Issues[1])
	}
}
