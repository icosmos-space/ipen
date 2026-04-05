package pipeline

import (
	"fmt"

	"github.com/icosmos-space/ipen/core/agents"
	"github.com/icosmos-space/ipen/core/models"
)

// TruthValidationParams 分组dependencies for chapter truth-file validation。
type TruthValidationParams struct {
	SettleChapterState func(input SettlementRetryInput) (*agents.WriteChapterOutput, error)
	Validate           func(chapterContent string, chapterNumber int, oldState string, newState string, oldHooks string, newHooks string, language string) (*agents.ValidationResult, error)
	Book               *models.BookConfig
	BookDir            string
	ChapterNumber      int
	Title              string
	Content            string
	PersistenceOutput  *agents.WriteChapterOutput
	AuditResult        agents.AuditResult
	PreviousTruth      struct {
		OldState  string
		OldHooks  string
		OldLedger string
	}
	ReducedControlInput *struct {
		ChapterIntent  string
		ContextPackage models.ContextPackage
		RuleStack      models.RuleStack
	}
	Language   string
	LogWarn    func(message BiMessage)
	LoggerWarn func(message string)
}

// TruthValidationOutcome 是the return shape for ValidateChapterTruthPersistence。
type TruthValidationOutcome struct {
	Validation        *agents.ValidationResult   `json:"validation"`
	ChapterStatus     string                     `json:"chapterStatus,omitempty"`
	DegradedIssues    []agents.AuditIssue        `json:"degradedIssues"`
	PersistenceOutput *agents.WriteChapterOutput `json:"persistenceOutput"`
	AuditResult       agents.AuditResult         `json:"auditResult"`
}

// ValidateChapterTruthPersistence 校验persisted truth updates and performs settlement retry when needed。
func ValidateChapterTruthPersistence(params TruthValidationParams) (*TruthValidationOutcome, error) {
	validation, err := params.Validate(
		params.Content,
		params.ChapterNumber,
		params.PreviousTruth.OldState,
		params.PersistenceOutput.UpdatedState,
		params.PreviousTruth.OldHooks,
		params.PersistenceOutput.UpdatedHooks,
		params.Language,
	)
	if err != nil {
		return nil, fmt.Errorf("state validation failed for chapter %d: %w", params.ChapterNumber, err)
	}

	if len(validation.Warnings) > 0 {
		if params.LogWarn != nil {
			params.LogWarn(BiMessage{
				ZH: fmt.Sprintf("状态校验：第 %d 章发现 %d 条警告", params.ChapterNumber, len(validation.Warnings)),
				EN: fmt.Sprintf("State validation: %d warning(s) for chapter %d", len(validation.Warnings), params.ChapterNumber),
			})
		}
		if params.LoggerWarn != nil {
			for _, warning := range validation.Warnings {
				params.LoggerWarn(fmt.Sprintf("  [%s] %s", warning.Category, warning.Description))
			}
		}
	}

	chapterStatus := ""
	degradedIssues := []agents.AuditIssue{}
	persistenceOutput := params.PersistenceOutput
	auditResult := params.AuditResult

	if !validation.Passed {
		recovery, err := RetrySettlementAfterValidationFailure(SettlementRetryParams{
			SettleChapterState:  params.SettleChapterState,
			Validate:            params.Validate,
			Book:                params.Book,
			BookDir:             params.BookDir,
			ChapterNumber:       params.ChapterNumber,
			Title:               params.Title,
			Content:             params.Content,
			ReducedControlInput: params.ReducedControlInput,
			OldState:            params.PreviousTruth.OldState,
			OldHooks:            params.PreviousTruth.OldHooks,
			OriginalValidation:  validation,
			Language:            params.Language,
			LogWarn:             params.LogWarn,
			LoggerWarn:          params.LoggerWarn,
		})
		if err != nil {
			return nil, err
		}

		if recovery.Kind == "recovered" {
			persistenceOutput = recovery.Output
			validation = recovery.Validation
		} else {
			chapterStatus = "state-degraded"
			degradedIssues = recovery.Issues
			persistenceOutput = BuildStateDegradedPersistenceOutput(
				persistenceOutput,
				params.PreviousTruth.OldState,
				params.PreviousTruth.OldHooks,
				params.PreviousTruth.OldLedger,
			)
			auditResult.Issues = append(append([]agents.AuditIssue{}, auditResult.Issues...), recovery.Issues...)
		}
	}

	return &TruthValidationOutcome{
		Validation:        validation,
		ChapterStatus:     chapterStatus,
		DegradedIssues:    degradedIssues,
		PersistenceOutput: persistenceOutput,
		AuditResult:       auditResult,
	}, nil
}
