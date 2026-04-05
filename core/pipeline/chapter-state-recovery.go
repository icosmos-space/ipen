package pipeline

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/icosmos-space/ipen/core/agents"
	"github.com/icosmos-space/ipen/core/models"
)

// SettlementRetryInput 是passed to settlement callback for retry。
type SettlementRetryInput struct {
	Book               *models.BookConfig
	BookDir            string
	ChapterNumber      int
	Title              string
	Content            string
	ChapterIntent      string
	ContextPackage     *models.ContextPackage
	RuleStack          *models.RuleStack
	ValidationFeedback string
}

// SettlementRetryParams defines dependencies for retrying settlement after validation failure.
type SettlementRetryParams struct {
	SettleChapterState  func(input SettlementRetryInput) (*agents.WriteChapterOutput, error)
	Validate            func(chapterContent string, chapterNumber int, oldState string, newState string, oldHooks string, newHooks string, language string) (*agents.ValidationResult, error)
	Book                *models.BookConfig
	BookDir             string
	ChapterNumber       int
	Title               string
	Content             string
	ReducedControlInput *struct {
		ChapterIntent  string
		ContextPackage models.ContextPackage
		RuleStack      models.RuleStack
	}
	OldState           string
	OldHooks           string
	OriginalValidation *agents.ValidationResult
	Language           string
	LogWarn            func(message BiMessage)
	LoggerWarn         func(message string)
}

// SettlementRetryResult captures recovered or degraded retry outcome.
type SettlementRetryResult struct {
	Kind       string                     `json:"kind"`
	Output     *agents.WriteChapterOutput `json:"output,omitempty"`
	Validation *agents.ValidationResult   `json:"validation,omitempty"`
	Issues     []agents.AuditIssue        `json:"issues,omitempty"`
}

// RetrySettlementAfterValidationFailure retries settlement-only path with explicit validation feedback.
func RetrySettlementAfterValidationFailure(params SettlementRetryParams) (*SettlementRetryResult, error) {
	if params.LogWarn != nil {
		params.LogWarn(BiMessage{
			ZH: fmt.Sprintf("状态校验失败，正在仅重试结算层（第%d章）", params.ChapterNumber),
			EN: fmt.Sprintf("State validation failed; retrying settlement only for chapter %d", params.ChapterNumber),
		})
	}

	retryInput := SettlementRetryInput{
		Book:          params.Book,
		BookDir:       params.BookDir,
		ChapterNumber: params.ChapterNumber,
		Title:         params.Title,
		Content:       params.Content,
		ValidationFeedback: BuildStateValidationFeedback(
			nilSafeWarnings(params.OriginalValidation),
			params.Language,
		),
	}
	if params.ReducedControlInput != nil {
		retryInput.ChapterIntent = params.ReducedControlInput.ChapterIntent
		cp := params.ReducedControlInput.ContextPackage
		rs := params.ReducedControlInput.RuleStack
		retryInput.ContextPackage = &cp
		retryInput.RuleStack = &rs
	}

	retryOutput, err := params.SettleChapterState(retryInput)
	if err != nil {
		return nil, err
	}

	retryValidation, err := params.Validate(
		params.Content,
		params.ChapterNumber,
		params.OldState,
		retryOutput.UpdatedState,
		params.OldHooks,
		retryOutput.UpdatedHooks,
		params.Language,
	)
	if err != nil {
		return nil, fmt.Errorf("state validation retry failed for chapter %d: %w", params.ChapterNumber, err)
	}

	if len(retryValidation.Warnings) > 0 {
		if params.LogWarn != nil {
			params.LogWarn(BiMessage{
				ZH: fmt.Sprintf("状态校验重试后，第 %d 章仍有 %d 条警告", params.ChapterNumber, len(retryValidation.Warnings)),
				EN: fmt.Sprintf("State validation retry still reports %d warning(s) for chapter %d", len(retryValidation.Warnings), params.ChapterNumber),
			})
		}
		if params.LoggerWarn != nil {
			for _, warning := range retryValidation.Warnings {
				params.LoggerWarn(fmt.Sprintf("  [%s] %s", warning.Category, warning.Description))
			}
		}
	}

	if retryValidation.Passed {
		return &SettlementRetryResult{
			Kind:       "recovered",
			Output:     retryOutput,
			Validation: retryValidation,
		}, nil
	}

	return &SettlementRetryResult{
		Kind:   "degraded",
		Issues: BuildStateDegradedIssues(retryValidation.Warnings, params.Language),
	}, nil
}

// BuildStateValidationFeedback 格式化validator warnings into settlement retry instruction。
func BuildStateValidationFeedback(warnings []agents.ValidationWarning, language string) string {
	if len(warnings) == 0 {
		if strings.EqualFold(language, "en") {
			return "The previous settlement contradicted the chapter text. Reconcile truth files strictly to the body."
		}
		return "上一轮状态结算与正文矛盾。请严格以正文为准修正 truth files。"
	}

	if strings.EqualFold(language, "en") {
		lines := []string{"The previous settlement failed validation. Fix these contradictions against the chapter body:"}
		for _, warning := range warnings {
			lines = append(lines, fmt.Sprintf("- [%s] %s", warning.Category, warning.Description))
		}
		return strings.Join(lines, "\n")
	}

	lines := []string{"上一次状态结算未通过校验。请对照正文修正以下矛盾："}
	for _, warning := range warnings {
		lines = append(lines, fmt.Sprintf("- [%s] %s", warning.Category, warning.Description))
	}
	return strings.Join(lines, "\n")
}

// BuildStateDegradedIssues maps validation warnings to audit issues when retry cannot recover.
func BuildStateDegradedIssues(warnings []agents.ValidationWarning, language string) []agents.AuditIssue {
	if len(warnings) > 0 {
		result := make([]agents.AuditIssue, 0, len(warnings))
		for _, warning := range warnings {
			suggestion := "请先基于已保存正文修复本章 state，再继续后续章节。"
			if strings.EqualFold(language, "en") {
				suggestion = "Repair chapter state from the persisted body before continuing."
			}
			result = append(result, agents.AuditIssue{
				Severity:    "warning",
				Category:    "state-validation",
				Description: warning.Description,
				Suggestion:  suggestion,
			})
		}
		return result
	}

	description := "状态结算重试后仍未通过校验。"
	suggestion := "请先基于已保存正文修复本章 state，再继续后续章节。"
	if strings.EqualFold(language, "en") {
		description = "State validation still failed after settlement retry."
		suggestion = "Repair chapter state from the persisted body before continuing."
	}
	return []agents.AuditIssue{{
		Severity:    "warning",
		Category:    "state-validation",
		Description: description,
		Suggestion:  suggestion,
	}}
}

// BuildStateDegradedPersistenceOutput freezes persisted truth files to previous stable values.
func BuildStateDegradedPersistenceOutput(output *agents.WriteChapterOutput, oldState string, oldHooks string, oldLedger string) *agents.WriteChapterOutput {
	cloned := *output
	cloned.RuntimeStateDelta = nil
	cloned.RuntimeStateSnapshot = nil
	cloned.UpdatedState = oldState
	cloned.UpdatedLedger = oldLedger
	cloned.UpdatedHooks = oldHooks
	cloned.UpdatedChapterSummaries = ""
	return &cloned
}

// StateDegradedReviewNote 是serialized metadata for degraded chapters。
type StateDegradedReviewNote struct {
	Kind           string   `json:"kind"`
	BaseStatus     string   `json:"baseStatus"`
	InjectedIssues []string `json:"injectedIssues"`
}

// BuildStateDegradedReviewNote 构建JSON metadata for degraded chapter status。
func BuildStateDegradedReviewNote(baseStatus string, issues []agents.AuditIssue) string {
	note := StateDegradedReviewNote{
		Kind:           "state-degraded",
		BaseStatus:     baseStatus,
		InjectedIssues: make([]string, 0, len(issues)),
	}
	for _, issue := range issues {
		note.InjectedIssues = append(note.InjectedIssues, fmt.Sprintf("[%s] %s", issue.Severity, issue.Description))
	}
	payload, _ := json.Marshal(note)
	return string(payload)
}

// ParseStateDegradedReviewNote 解析degraded review metadata from JSON string。
func ParseStateDegradedReviewNote(reviewNote string) *StateDegradedReviewNote {
	if strings.TrimSpace(reviewNote) == "" {
		return nil
	}
	var parsed StateDegradedReviewNote
	if err := json.Unmarshal([]byte(reviewNote), &parsed); err != nil {
		return nil
	}
	if parsed.Kind != "state-degraded" {
		return nil
	}
	if parsed.BaseStatus != "ready-for-review" && parsed.BaseStatus != "audit-failed" {
		return nil
	}
	return &parsed
}

// ResolveStateDegradedBaseStatus 解析base chapter status for a degraded chapter。
func ResolveStateDegradedBaseStatus(chapter *models.ChapterMeta) string {
	if chapter == nil {
		return "ready-for-review"
	}
	if metadata := ParseStateDegradedReviewNote(chapter.ReviewNote); metadata != nil {
		return metadata.BaseStatus
	}
	for _, issue := range chapter.AuditIssues {
		if strings.Contains(strings.ToLower(issue), "[critical]") {
			return "audit-failed"
		}
	}
	return "ready-for-review"
}

func nilSafeWarnings(validation *agents.ValidationResult) []agents.ValidationWarning {
	if validation == nil {
		return []agents.ValidationWarning{}
	}
	return validation.Warnings
}
