package pipeline

import (
	"fmt"

	"github.com/icosmos-space/ipen/core/agents"
	"github.com/icosmos-space/ipen/core/models"
)

// ChapterReviewCycleUsage tracks token usage inside review cycle.
type ChapterReviewCycleUsage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}

// ChapterRepairDecision controls auto-repair strategy.
type ChapterRepairDecision string

const (
	ChapterRepairDecisionNone     ChapterRepairDecision = "none"
	ChapterRepairDecisionLocalFix ChapterRepairDecision = "local-fix"
	ChapterRepairDecisionRewrite  ChapterRepairDecision = "rewrite"
)

// ChapterAssessment 是a single audit+repair recommendation round。
type ChapterAssessment struct {
	AuditResult    agents.AuditResult    `json:"auditResult"`
	RepairIssues   []agents.AuditIssue   `json:"repairIssues"`
	RepairDecision ChapterRepairDecision `json:"repairDecision"`
	AITellCount    int                   `json:"aiTellCount"`
	BlockingCount  int                   `json:"blockingCount"`
	CriticalCount  int                   `json:"criticalCount"`
}

// ChapterReviewCycleResult 是the final output of review cycle。
type ChapterReviewCycleResult struct {
	FinalContent                string                  `json:"finalContent"`
	FinalWordCount              int                     `json:"finalWordCount"`
	PreAuditNormalizedWordCount int                     `json:"preAuditNormalizedWordCount"`
	Revised                     bool                    `json:"revised"`
	AuditResult                 agents.AuditResult      `json:"auditResult"`
	TotalUsage                  ChapterReviewCycleUsage `json:"totalUsage"`
	PostReviseCount             int                     `json:"postReviseCount"`
	NormalizeApplied            bool                    `json:"normalizeApplied"`
}

// AssessChapterOptions are optional knobs for one assess round.
type AssessChapterOptions struct {
	Temperature         *float64
	InitialRepairIssues []agents.AuditIssue
}

// NormalizeDraftResult 是returned by length normalization callback。
type NormalizeDraftResult struct {
	Content    string                   `json:"content"`
	WordCount  int                      `json:"wordCount"`
	Applied    bool                     `json:"applied"`
	TokenUsage *ChapterReviewCycleUsage `json:"tokenUsage,omitempty"`
}

// BiMessage carries zh/en log message variants.
type BiMessage struct {
	ZH string
	EN string
}

// RunChapterReviewCycleParams 分组dependencies for review cycle orchestration。
type RunChapterReviewCycleParams struct {
	InitialOutput struct {
		Content   string
		WordCount int
	}
	InitialRepairIssues []agents.AuditIssue
	LengthSpec          models.LengthSpec
	InitialUsage        ChapterReviewCycleUsage

	AssessChapter                func(chapterContent string, options *AssessChapterOptions) (*ChapterAssessment, error)
	RepairChapter                func(chapterContent string, issues []agents.AuditIssue, mode ChapterRepairDecision) (*agents.ReviseOutput, error)
	NormalizeDraftLengthIfNeeded func(chapterContent string) (*NormalizeDraftResult, error)
	AssertChapterContentNotEmpty func(content string, stage string)
	AddUsage                     func(left ChapterReviewCycleUsage, right *ChapterReviewCycleUsage) ChapterReviewCycleUsage
	RestoreAssessment            func(previous *ChapterAssessment, next *ChapterAssessment) *ChapterAssessment
	LogWarn                      func(message BiMessage)
	LogStage                     func(message BiMessage)
}

// RunChapterReviewCycle executes audit -> repair loop with deterministic rollback safeguards.
func RunChapterReviewCycle(params RunChapterReviewCycleParams) (*ChapterReviewCycleResult, error) {
	assess := func(chapterContent string, options *AssessChapterOptions, totalUsage *ChapterReviewCycleUsage) (*ChapterAssessment, error) {
		assessment, err := params.AssessChapter(chapterContent, options)
		if err != nil {
			return nil, err
		}
		*totalUsage = params.AddUsage(*totalUsage, usageFromTokenUsage(assessment.AuditResult.TokenUsage))
		return assessment, nil
	}

	totalUsage := params.InitialUsage
	postReviseCount := 0
	normalizeApplied := false
	finalContent := params.InitialOutput.Content
	finalWordCount := params.InitialOutput.WordCount
	revised := false

	normalizedBeforeAudit, err := params.NormalizeDraftLengthIfNeeded(finalContent)
	if err != nil {
		return nil, err
	}
	totalUsage = params.AddUsage(totalUsage, normalizedBeforeAudit.TokenUsage)
	finalContent = normalizedBeforeAudit.Content
	finalWordCount = normalizedBeforeAudit.WordCount
	normalizeApplied = normalizeApplied || normalizedBeforeAudit.Applied
	if params.AssertChapterContentNotEmpty != nil {
		params.AssertChapterContentNotEmpty(finalContent, "draft generation")
	}

	if len(params.InitialRepairIssues) > 0 && params.LogWarn != nil {
		params.LogWarn(BiMessage{
			ZH: fmt.Sprintf("首轮评审接收了 %d 条预检修复问题", len(params.InitialRepairIssues)),
			EN: fmt.Sprintf("%d preflight repair issues were fed into the first assessment", len(params.InitialRepairIssues)),
		})
	}

	if params.LogStage != nil {
		params.LogStage(BiMessage{ZH: "审计草稿", EN: "auditing draft"})
	}
	assessment, err := assess(finalContent, &AssessChapterOptions{
		InitialRepairIssues: cloneIssues(params.InitialRepairIssues),
	}, &totalUsage)
	if err != nil {
		return nil, err
	}

	for assessment.RepairDecision != ChapterRepairDecisionNone && len(assessment.RepairIssues) > 0 {
		repairMode := assessment.RepairDecision
		if params.LogStage != nil {
			if repairMode == ChapterRepairDecisionLocalFix {
				params.LogStage(BiMessage{
					ZH: "自动修复当前章的局部问题",
					EN: "auto-fixing local issues in the current chapter",
				})
			} else {
				params.LogStage(BiMessage{
					ZH: "当前章局部修复未通过，升级为整章改写",
					EN: "local repair still failed, escalating to full chapter rewrite",
				})
			}
		}

		reviseOutput, err := params.RepairChapter(finalContent, cloneIssues(assessment.RepairIssues), repairMode)
		if err != nil {
			return nil, err
		}
		totalUsage = params.AddUsage(totalUsage, usageFromTokenUsage(reviseOutput.TokenUsage))

		if reviseOutput.RevisedContent == "" || reviseOutput.RevisedContent == finalContent {
			if repairMode == ChapterRepairDecisionRewrite {
				break
			}
			continue
		}

		normalizedRevision, err := params.NormalizeDraftLengthIfNeeded(reviseOutput.RevisedContent)
		if err != nil {
			return nil, err
		}
		totalUsage = params.AddUsage(totalUsage, normalizedRevision.TokenUsage)
		postReviseCount = normalizedRevision.WordCount
		normalizeApplied = normalizeApplied || normalizedRevision.Applied

		previousAssessment := cloneAssessment(assessment)
		previousContent := finalContent

		temperatureZero := 0.0
		nextAssessmentRaw, err := assess(normalizedRevision.Content, &AssessChapterOptions{
			Temperature: &temperatureZero,
		}, &totalUsage)
		if err != nil {
			return nil, err
		}
		nextAssessment := params.RestoreAssessment(previousAssessment, nextAssessmentRaw)

		if nextAssessment.AITellCount > previousAssessment.AITellCount {
			rollbackAssessmentRaw, err := assess(previousContent, &AssessChapterOptions{
				Temperature: &temperatureZero,
			}, &totalUsage)
			if err != nil {
				return nil, err
			}
			assessment = params.RestoreAssessment(previousAssessment, rollbackAssessmentRaw)
			break
		}

		finalContent = normalizedRevision.Content
		finalWordCount = normalizedRevision.WordCount
		revised = true
		if params.AssertChapterContentNotEmpty != nil {
			stage := "revision"
			if repairMode == ChapterRepairDecisionRewrite {
				stage = "rewrite"
			}
			params.AssertChapterContentNotEmpty(finalContent, stage)
		}
		assessment = nextAssessment
	}

	return &ChapterReviewCycleResult{
		FinalContent:                finalContent,
		FinalWordCount:              finalWordCount,
		PreAuditNormalizedWordCount: normalizedBeforeAudit.WordCount,
		Revised:                     revised,
		AuditResult:                 assessment.AuditResult,
		TotalUsage:                  totalUsage,
		PostReviseCount:             postReviseCount,
		NormalizeApplied:            normalizeApplied,
	}, nil
}

func usageFromTokenUsage(usage *models.TokenUsage) *ChapterReviewCycleUsage {
	if usage == nil {
		return nil
	}
	return &ChapterReviewCycleUsage{
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalTokens:      usage.TotalTokens,
	}
}

func cloneIssues(issues []agents.AuditIssue) []agents.AuditIssue {
	if len(issues) == 0 {
		return []agents.AuditIssue{}
	}
	out := make([]agents.AuditIssue, len(issues))
	copy(out, issues)
	return out
}

func cloneAssessment(in *ChapterAssessment) *ChapterAssessment {
	if in == nil {
		return nil
	}
	clone := *in
	clone.AuditResult = agents.AuditResult{
		Passed:  in.AuditResult.Passed,
		Issues:  cloneIssues(in.AuditResult.Issues),
		Summary: in.AuditResult.Summary,
	}
	if in.AuditResult.TokenUsage != nil {
		u := *in.AuditResult.TokenUsage
		clone.AuditResult.TokenUsage = &u
	}
	clone.RepairIssues = cloneIssues(in.RepairIssues)
	return &clone
}
