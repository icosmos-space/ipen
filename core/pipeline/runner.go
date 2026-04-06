package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/icosmos-space/ipen/core/agents"
	"github.com/icosmos-space/ipen/core/llm"
	"github.com/icosmos-space/ipen/core/models"
	"github.com/icosmos-space/ipen/core/notify"
	"github.com/icosmos-space/ipen/core/state"
	"github.com/icosmos-space/ipen/core/utils"
)

// PipelineConfig 表示pipeline configuration。
type PipelineConfig struct {
	Client              *llm.LLMClient
	Model               string
	ProjectRoot         string
	DefaultLLMConfig    *models.LLMConfig
	NotifyChannels      []models.NotifyChannel
	ModelOverrides      map[string]any
	InputGovernanceMode models.InputGovernanceMode
	Logger              utils.Logger
	OnStreamChunk       llm.OnStreamChunk
	OnStreamProgress    llm.OnStreamProgress
}

// TokenUsageSummary 表示token usage summary。
type TokenUsageSummary struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}

type ChapterPipelineStatus string

const (
	ChapterPipelineStatusReadyForReview ChapterPipelineStatus = "ready-for-review"
	ChapterPipelineStatusAuditFailed    ChapterPipelineStatus = "audit-failed"
	ChapterPipelineStatusStateDegraded  ChapterPipelineStatus = "state-degraded"
)

// ChapterPipelineResult 表示chapter pipeline result。
type ChapterPipelineResult struct {
	ChapterNumber   int                     `json:"chapterNumber"`
	Title           string                  `json:"title"`
	WordCount       int                     `json:"wordCount"`
	AuditResult     agents.AuditResult      `json:"auditResult"`
	Revised         bool                    `json:"revised"`
	Status          string                  `json:"status" validate:"oneof=ready-for-review audit-failed state-degraded"` // "ready-for-review", "audit-failed", "state-degraded"
	LengthWarnings  []string                `json:"lengthWarnings"`
	LengthTelemetry *models.LengthTelemetry `json:"lengthTelemetry,omitempty"`
	TokenUsage      *TokenUsageSummary      `json:"tokenUsage,omitempty"`
}

// DraftResult 原子操作结果
type DraftResult struct {
	ChapterNumber   int                     `json:"chapterNumber"`
	Title           string                  `json:"title"`
	Content         string                  `json:"content"` // 待定
	WordCount       int                     `json:"wordCount"`
	FilePath        string                  `json:"filePath"`
	LengthWarnings  []string                `json:"lengthWarnings"`
	LengthTelemetry *models.LengthTelemetry `json:"lengthTelemetry,omitempty"`
	TokenUsage      *TokenUsageSummary      `json:"tokenUsage,omitempty"`
}

// PlanChapterResult 表示plan chapter result。
type PlanChapterResult struct {
	BookID        string   `json:"bookId"`
	ChapterNumber int      `json:"chapterNumber"`
	IntentPath    string   `json:"intentPath"`
	Goal          string   `json:"goal"`
	Conflicts     []string `json:"conflicts"`
}

// ComposeChapterResult 表示compose chapter result。
type ComposeChapterResult struct {
	ContextPackage *models.ContextPackage `json:"contextPackage"` // 待定
	RuleStack      *models.RuleStack      `json:"ruleStack"`      // 待定
	Trace          *models.ChapterTrace   `json:"trace"`          // 待定
	ContextPath    string                 `json:"contextPath"`
	RuleStackPath  string                 `json:"ruleStackPath"`
	TracePath      string                 `json:"tracePath"`
}

type ReviseStatus string

const (
	ReviseStatusUnchanged      ReviseStatus = "unchanged"
	ReviseStatusReadyForReview ReviseStatus = "ready-for-review"
	ReviseStatusAuditFailed    ReviseStatus = "audit-failed"
)

// ReviseResult 表示revise result。
type ReviseResult struct {
	ChapterNumber   int                     `json:"chapterNumber"`
	WordCount       int                     `json:"wordCount"`
	FixedIssues     []string                `json:"fixedIssues"`
	Status          ChapterPipelineStatus   `json:"status"`
	SkippedReason   string                  `json:"skippedReason"`
	LengthWarnings  []string                `json:"lengthWarnings"`
	LengthTelemetry *models.LengthTelemetry `json:"lengthTelemetry,omitempty"`
	Applied         bool                    `json:"applied"`
	Title           string                  `json:"title"`
	Content         string                  `json:"content"` // 待定
}

// TruthFiles 表示truth files。
type TruthFiles struct {
	CurrentState   string `json:"currentState"`
	ParticleLedger string `json:"particleLedger"`
	PendingHooks   string `json:"pendingHooks"`
	StoryBible     string `json:"storyBible"`
	VolumeOutline  string `json:"volumeOutline"`
	BookRules      string `json:"bookRules"`
}

// BookStatusInfo 表示book status info。
type BookStatusInfo struct {
	BookID         string `json:"bookId"`
	Title          string `json:"title"`
	Genre          string `json:"genre"`
	Platform       string `json:"platform"`
	Status         string `json:"status"`
	ChatperWritten int    `json:"chatperWritten"`
	TotalWords     int    `json:"totalWords"`
	NextChapter    int    `json:"nextChapter"`
	Chapters       int    `json:"chapters"`
}

type MergedAuditEvaluation struct {
	AuditResult    agents.AuditResult    `json:"auditResult"`
	AiTellCount    int                   `json:"aiTellCount"`
	BlockingCount  int                   `json:"blockingCount"`
	CriticalCount  int                   `json:"criticalCount"`
	RepairIssues   []agents.AuditIssue   `json:"repairIssues"`
	RepairDecision ChapterRepairDecision `json:"repairDecision"`
}

type chapter struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type ImportChaptersInput struct {
	BookID     string    `json:"bookId"`
	Chapters   []chapter `json:"chapters"`
	ResumeFrom int       `json:"resumeFrom"`
}

// ImportChaptersResult 表示import chapters result。
type ImportChaptersResult struct {
	BookID        string `json:"bookId"`
	ImportedCount int    `json:"importedCount"`
	TotalWords    int    `json:"totalWords"`
	NextChapter   int    `json:"nextChapter"`
}

// PipelineRunner 表示the pipeline runner。
type PipelineRunner struct {
	config       PipelineConfig
	stateManager *state.StateManager
	logger       utils.Logger
	tokenUsage   TokenUsageSummary
}

// NewPipelineRunner 创建新的pipeline runner。
func NewPipelineRunner(config PipelineConfig) *PipelineRunner {
	return &PipelineRunner{
		config:       config,
		stateManager: state.NewStateManager(config.ProjectRoot),
		logger:       config.Logger,
	}
}

// RunChapterPipeline runs the chapter pipeline
func (pr *PipelineRunner) RunChapterPipeline(ctx context.Context, bookID string, chapterNumber int) (*ChapterPipelineResult, error) {
	startTime := time.Now()
	pr.logger.Info("开始运行章节管道", map[string]any{
		"bookId":        bookID,
		"chapterNumber": chapterNumber,
	})

	// Load book config
	bookConfig, err := pr.stateManager.LoadBookConfig(bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to load book config: %w", err)
	}

	bookDir := pr.stateManager.BookDir(bookID)

	// Phase 1: Plan
	pr.logger.Info("Phase 1: Planning chapter", nil)
	planResult, err := pr.planChapter(ctx, bookConfig, bookDir, chapterNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to plan chapter: %w", err)
	}

	// Phase 2: Compose
	pr.logger.Info("Phase 2: Composing chapter", nil)
	composeResult, err := pr.composeChapter(ctx, bookConfig, bookDir, chapterNumber, planResult)
	if err != nil {
		return nil, fmt.Errorf("failed to compose chapter: %w", err)
	}

	// Phase 3: Write
	pr.logger.Info("Phase 3: Writing chapter", nil)
	writeResult, err := pr.writeChapter(ctx, bookConfig, bookDir, chapterNumber, planResult, composeResult)
	if err != nil {
		return nil, fmt.Errorf("failed to write chapter: %w", err)
	}

	// Phase 4: Settle state
	pr.logger.Info("Phase 4: Settling state", nil)
	_, err = pr.settleChapterState(ctx, bookConfig, bookDir, chapterNumber, writeResult.Title, writeResult.Content, planResult, composeResult)
	if err != nil {
		return nil, fmt.Errorf("failed to settle state: %w", err)
	}

	// Calculate word count
	countingMode := utils.ResolveLengthCountingMode(utils.LengthLanguage(bookConfig.Language))
	wordCount := utils.CountChapterLength(writeResult.Content, countingMode)

	elapsed := time.Since(startTime)
	pr.logger.Info("Chapter pipeline completed", map[string]any{
		"elapsedMs": elapsed.Milliseconds(),
		"wordCount": wordCount,
	})

	return &ChapterPipelineResult{
		ChapterNumber: chapterNumber,
		Title:         writeResult.Title,
		WordCount:     wordCount,
		Revised:       false,
		Status:        "ready-for-review",
		TokenUsage:    &pr.tokenUsage,
	}, nil
}

func (pr *PipelineRunner) planChapter(ctx context.Context, bookConfig *models.BookConfig, bookDir string, chapterNumber int) (*PlanChapterResult, error) {
	// agentCtx := agents.AgentContext{
	// 	Client:           pr.config.Client,
	// 	Model:            pr.config.Model,
	// 	ProjectRoot:      pr.config.ProjectRoot,
	// 	BookID:           bookConfig.ID,
	// 	Logger:           pr.logger,
	// 	OnStreamChunk:    pr.config.OnStreamChunk,
	// 	OnStreamProgress: pr.config.OnStreamProgress,
	// }

	// planner := agents.NewPlannerAgent(agentCtx)
	// input := agents.PlanChapterInput{
	// 	Book:          bookConfig,
	// 	BookDir:       bookDir,
	// 	ChapterNumber: chapterNumber,
	// }

	// output, err := planner.PlanChapter(ctx, input)
	// if err != nil {
	// 	return nil, err
	// }

	return &PlanChapterResult{
		// Intent:         &output.Intent,
		// IntentMarkdown: output.IntentMarkdown,
		// RuntimePath:    output.RuntimePath,
	}, nil
}

func (pr *PipelineRunner) composeChapter(ctx context.Context, bookConfig *models.BookConfig, bookDir string, chapterNumber int, planResult *PlanChapterResult) (*ComposeChapterResult, error) {
	agentCtx := agents.AgentContext{
		Client:           pr.config.Client,
		Model:            pr.config.Model,
		ProjectRoot:      pr.config.ProjectRoot,
		BookID:           bookConfig.ID,
		Logger:           pr.logger,
		OnStreamChunk:    pr.config.OnStreamChunk,
		OnStreamProgress: pr.config.OnStreamProgress,
	}

	composer := agents.NewComposerAgent(agentCtx)
	input := agents.ComposeChapterInput{
		Book:          bookConfig,
		BookDir:       bookDir,
		ChapterNumber: chapterNumber,
		Plan:          agents.PlanChapterOutput{
			// Intent:         planResult.Intent,
			// IntentMarkdown: planResult.IntentMarkdown,
			// RuntimePath:    planResult.RuntimePath,
		},
	}

	output, err := composer.ComposeChapter(ctx, input)
	if err != nil {
		return nil, err
	}

	return &ComposeChapterResult{
		ContextPackage: &output.ContextPackage,
		RuleStack:      &output.RuleStack,
		Trace:          &output.Trace,
		ContextPath:    output.ContextPath,
		RuleStackPath:  output.RuleStackPath,
		TracePath:      output.TracePath,
	}, nil
}

func (pr *PipelineRunner) writeChapter(ctx context.Context, bookConfig *models.BookConfig, bookDir string, chapterNumber int, planResult *PlanChapterResult, composeResult *ComposeChapterResult) (*DraftResult, error) {
	agentCtx := agents.AgentContext{
		Client:           pr.config.Client,
		Model:            pr.config.Model,
		ProjectRoot:      pr.config.ProjectRoot,
		BookID:           bookConfig.ID,
		Logger:           pr.logger,
		OnStreamChunk:    pr.config.OnStreamChunk,
		OnStreamProgress: pr.config.OnStreamProgress,
	}

	writer := agents.NewWriterAgent(agentCtx)
	input := agents.WriteChapterInput{
		Book:          bookConfig,
		BookDir:       bookDir,
		ChapterNumber: chapterNumber,
	}

	output, err := writer.WriteChapter(ctx, input)
	if err != nil {
		return nil, err
	}

	// Track token usage
	pr.tokenUsage = TokenUsageSummary{
		PromptTokens:     output.TokenUsage.PromptTokens,
		CompletionTokens: output.TokenUsage.CompletionTokens,
		TotalTokens:      output.TokenUsage.TotalTokens,
	}

	return &DraftResult{
		ChapterNumber: chapterNumber,
		Title:         output.Title,
		Content:       output.Content,
	}, nil
}

func (pr *PipelineRunner) settleChapterState(ctx context.Context, bookConfig *models.BookConfig, bookDir string, chapterNumber int, title string, content string, planResult *PlanChapterResult, composeResult *ComposeChapterResult) (*state.RuntimeStateSnapshot, error) {
	_ = ctx
	_ = title
	_ = content
	_ = planResult
	_ = composeResult

	if err := pr.stateManager.SnapshotStateAt(bookDir, chapterNumber); err != nil {
		return nil, err
	}

	language := models.LanguageZH
	if bookConfig != nil && bookConfig.Language == string(models.LanguageEN) {
		language = models.LanguageEN
	}

	snapshot := &state.RuntimeStateSnapshot{
		Manifest: models.StateManifest{
			SchemaVersion:      2,
			Language:           models.RuntimeStateLanguage(language),
			LastAppliedChapter: chapterNumber,
			ProjectionVersion:  1,
			MigrationWarnings:  []string{},
		},
		CurrentState: models.CurrentStateState{
			Chapter: chapterNumber,
			Facts:   []models.CurrentStateFact{},
		},
		Hooks: models.HooksState{
			Hooks: []models.HookRecord{},
		},
		ChapterSummaries: models.ChapterSummariesState{
			Rows: []models.ChapterSummaryRow{},
		},
	}

	return snapshot, nil
}

// SendNotification sends a notification
func (pr *PipelineRunner) SendNotification(ctx context.Context, message notify.NotifyMessage) error {
	return notify.DispatchNotification(ctx, pr.config.NotifyChannels, message)
}

// GetStateManager 返回the state manager。
func (pr *PipelineRunner) GetStateManager() *state.StateManager {
	return pr.stateManager
}

// GetTokenUsage 返回current token usage。
func (pr *PipelineRunner) GetTokenUsage() TokenUsageSummary {
	return pr.tokenUsage
}
