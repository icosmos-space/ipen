package pipeline

import (
	"reflect"
	"testing"

	"github.com/icosmos-space/ipen/core/agents"
	"github.com/icosmos-space/ipen/core/models"
)

var zeroUsage = ChapterReviewCycleUsage{
	PromptTokens:     0,
	CompletionTokens: 0,
	TotalTokens:      0,
}

func createAssessment(overrides func(*ChapterAssessment)) *ChapterAssessment {
	a := &ChapterAssessment{
		AuditResult: agents.AuditResult{
			Passed:  true,
			Issues:  []agents.AuditIssue{},
			Summary: "clean",
		},
		RepairIssues:   []agents.AuditIssue{},
		RepairDecision: ChapterRepairDecisionNone,
		AITellCount:    0,
		BlockingCount:  0,
		CriticalCount:  0,
	}
	if overrides != nil {
		overrides(a)
	}
	return a
}

func addUsage(left ChapterReviewCycleUsage, right *ChapterReviewCycleUsage) ChapterReviewCycleUsage {
	if right == nil {
		return left
	}
	return ChapterReviewCycleUsage{
		PromptTokens:     left.PromptTokens + right.PromptTokens,
		CompletionTokens: left.CompletionTokens + right.CompletionTokens,
		TotalTokens:      left.TotalTokens + right.TotalTokens,
	}
}

func TestRunChapterReviewCycle_AllowsInitialRewriteDecision(t *testing.T) {
	initialIssues := []agents.AuditIssue{{
		Severity:    "critical",
		Category:    "paragraph-shape",
		Description: "too fragmented",
		Suggestion:  "merge short fragments",
	}}

	assessCalls := []struct {
		Content string
		Options *AssessChapterOptions
	}{}
	assessSequence := []*ChapterAssessment{
		createAssessment(func(a *ChapterAssessment) {
			a.AuditResult.Passed = false
			a.AuditResult.Issues = cloneIssues(initialIssues)
			a.AuditResult.Summary = "rewrite directly"
			a.RepairIssues = cloneIssues(initialIssues)
			a.RepairDecision = ChapterRepairDecisionRewrite
			a.BlockingCount = 1
			a.CriticalCount = 1
		}),
		createAssessment(func(a *ChapterAssessment) {
			a.AuditResult.Passed = true
		}),
	}
	assessIdx := 0
	repairCalls := []struct {
		Content string
		Issues  []agents.AuditIssue
		Mode    ChapterRepairDecision
	}{}
	normalizeCalls := []string{}

	result, err := RunChapterReviewCycle(RunChapterReviewCycleParams{
		InitialOutput: struct {
			Content   string
			WordCount int
		}{Content: "raw draft", WordCount: 9},
		InitialRepairIssues: cloneIssues(initialIssues),
		InitialUsage:        zeroUsage,
		AssessChapter: func(chapterContent string, options *AssessChapterOptions) (*ChapterAssessment, error) {
			assessCalls = append(assessCalls, struct {
				Content string
				Options *AssessChapterOptions
			}{chapterContent, options})
			item := assessSequence[assessIdx]
			assessIdx++
			return item, nil
		},
		RepairChapter: func(chapterContent string, issues []agents.AuditIssue, mode ChapterRepairDecision) (*agents.ReviseOutput, error) {
			repairCalls = append(repairCalls, struct {
				Content string
				Issues  []agents.AuditIssue
				Mode    ChapterRepairDecision
			}{chapterContent, cloneIssues(issues), mode})
			return &agents.ReviseOutput{
				RevisedContent: "rewritten draft",
				TokenUsage:     &models.TokenUsage{},
			}, nil
		},
		NormalizeDraftLengthIfNeeded: func(chapterContent string) (*NormalizeDraftResult, error) {
			normalizeCalls = append(normalizeCalls, chapterContent)
			if chapterContent == "raw draft" {
				return &NormalizeDraftResult{Content: "raw draft", WordCount: 9, Applied: false, TokenUsage: &zeroUsage}, nil
			}
			return &NormalizeDraftResult{Content: "rewritten draft", WordCount: 10, Applied: false, TokenUsage: &zeroUsage}, nil
		},
		AssertChapterContentNotEmpty: func(content string, stage string) {},
		AddUsage:                     addUsage,
		RestoreAssessment: func(previous *ChapterAssessment, next *ChapterAssessment) *ChapterAssessment {
			return next
		},
	})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if len(repairCalls) != 1 || repairCalls[0].Mode != ChapterRepairDecisionRewrite {
		t.Fatalf("expected one rewrite repair call, got %#v", repairCalls)
	}
	if len(assessCalls) < 1 || assessCalls[0].Content != "raw draft" {
		t.Fatalf("unexpected assess calls: %#v", assessCalls)
	}
	if result.FinalContent != "rewritten draft" || !result.Revised {
		t.Fatalf("unexpected result: %#v", result)
	}
	if !reflect.DeepEqual(normalizeCalls, []string{"raw draft", "rewritten draft"}) {
		t.Fatalf("unexpected normalize calls: %#v", normalizeCalls)
	}
}

func TestRunChapterReviewCycle_RollsBackWhenAITellsIncrease(t *testing.T) {
	issues := []agents.AuditIssue{{
		Severity:    "warning",
		Category:    "continuity",
		Description: "broken continuity",
		Suggestion:  "fix it",
	}}

	assessSequence := []*ChapterAssessment{
		createAssessment(func(a *ChapterAssessment) {
			a.AuditResult.Passed = false
			a.AuditResult.Issues = cloneIssues(issues)
			a.RepairIssues = cloneIssues(issues)
			a.RepairDecision = ChapterRepairDecisionLocalFix
			a.BlockingCount = 1
			a.AITellCount = 0
		}),
		createAssessment(func(a *ChapterAssessment) {
			a.AuditResult.Passed = false
			a.AuditResult.Issues = cloneIssues(issues)
			a.RepairIssues = cloneIssues(issues)
			a.RepairDecision = ChapterRepairDecisionLocalFix
			a.BlockingCount = 1
			a.AITellCount = 1
		}),
		createAssessment(func(a *ChapterAssessment) {
			a.AuditResult.Passed = false
			a.AuditResult.Issues = cloneIssues(issues)
			a.RepairIssues = cloneIssues(issues)
			a.RepairDecision = ChapterRepairDecisionLocalFix
			a.BlockingCount = 1
			a.AITellCount = 0
		}),
	}
	assessIdx := 0
	assessContents := []string{}

	result, err := RunChapterReviewCycle(RunChapterReviewCycleParams{
		InitialOutput: struct {
			Content   string
			WordCount int
		}{Content: "original draft", WordCount: 13},
		InitialRepairIssues: []agents.AuditIssue{},
		InitialUsage:        zeroUsage,
		AssessChapter: func(chapterContent string, options *AssessChapterOptions) (*ChapterAssessment, error) {
			assessContents = append(assessContents, chapterContent)
			item := assessSequence[assessIdx]
			assessIdx++
			return item, nil
		},
		RepairChapter: func(chapterContent string, issues []agents.AuditIssue, mode ChapterRepairDecision) (*agents.ReviseOutput, error) {
			return &agents.ReviseOutput{
				RevisedContent: "rewritten draft",
				TokenUsage:     &models.TokenUsage{},
			}, nil
		},
		NormalizeDraftLengthIfNeeded: func(chapterContent string) (*NormalizeDraftResult, error) {
			if chapterContent == "original draft" {
				return &NormalizeDraftResult{Content: chapterContent, WordCount: 13, Applied: false, TokenUsage: &zeroUsage}, nil
			}
			return &NormalizeDraftResult{Content: chapterContent, WordCount: 15, Applied: false, TokenUsage: &zeroUsage}, nil
		},
		AssertChapterContentNotEmpty: func(content string, stage string) {},
		AddUsage:                     addUsage,
		RestoreAssessment: func(previous *ChapterAssessment, next *ChapterAssessment) *ChapterAssessment {
			return next
		},
	})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if result.FinalContent != "original draft" || result.Revised {
		t.Fatalf("expected rollback to original, got %#v", result)
	}
	expectedAssess := []string{"original draft", "rewritten draft", "original draft"}
	if !reflect.DeepEqual(assessContents, expectedAssess) {
		t.Fatalf("unexpected assess sequence: %#v", assessContents)
	}
}
