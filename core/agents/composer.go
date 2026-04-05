package agents

import (
	"context"
	"fmt"

	"github.com/icosmos-space/ipen/core/models"
)

// ComposeChapterInput 表示compose chapter input。
type ComposeChapterInput struct {
	Book          *models.BookConfig
	BookDir       string
	ChapterNumber int
	Plan          PlanChapterOutput
}

// ComposeChapterOutput 表示compose chapter output。
type ComposeChapterOutput struct {
	ContextPackage models.ContextPackage
	RuleStack      models.RuleStack
	Trace          models.ChapterTrace
	ContextPath    string
	RuleStackPath  string
	TracePath      string
}

// ComposerAgent 表示the composer agent。
type ComposerAgent struct {
	*BaseAgent
}

// NewComposerAgent 创建新的composer agent。
func NewComposerAgent(ctx AgentContext) *ComposerAgent {
	return &ComposerAgent{
		BaseAgent: NewBaseAgent(ctx),
	}
}

// Name 返回the agent name。
func (c *ComposerAgent) Name() string {
	return "composer"
}

// ComposeChapter composes a chapter context
func (c *ComposerAgent) ComposeChapter(ctx context.Context, input ComposeChapterInput) (*ComposeChapterOutput, error) {
	// Collect context sources
	contextPackage := c.buildContextPackage(input)
	ruleStack := c.buildRuleStack(input)
	trace := c.buildTrace(input, &contextPackage)

	return &ComposeChapterOutput{
		ContextPackage: contextPackage,
		RuleStack:      ruleStack,
		Trace:          trace,
		ContextPath:    fmt.Sprintf("runtime/chapter-%04d.context.json", input.ChapterNumber),
		RuleStackPath:  fmt.Sprintf("runtime/chapter-%04d.rule-stack.yaml", input.ChapterNumber),
		TracePath:      fmt.Sprintf("runtime/chapter-%04d.trace.json", input.ChapterNumber),
	}, nil
}

func (c *ComposerAgent) buildContextPackage(input ComposeChapterInput) models.ContextPackage {
	// Build context package from various sources
	return models.ContextPackage{
		Chapter: input.ChapterNumber,
		SelectedContext: []models.ContextSource{
			{
				Source: "story/current_focus.md",
				Reason: "Current task focus",
			},
			{
				Source: "story/story_bible.md",
				Reason: "Canon constraints",
			},
		},
	}
}

func (c *ComposerAgent) buildRuleStack(input ComposeChapterInput) models.RuleStack {
	return models.RuleStack{
		Layers: []models.RuleLayer{
			{ID: "L1", Name: "hard_facts", Precedence: 100, Scope: models.ScopeGlobal},
			{ID: "L2", Name: "author_intent", Precedence: 80, Scope: models.ScopeBook},
			{ID: "L3", Name: "planning", Precedence: 60, Scope: models.ScopeArc},
			{ID: "L4", Name: "current_task", Precedence: 70, Scope: models.ScopeLocal},
		},
		Sections: models.RuleStackSections{
			Hard:       []string{"story_bible", "current_state", "book_rules"},
			Soft:       []string{"author_intent", "current_focus", "volume_outline"},
			Diagnostic: []string{"anti_ai_checks", "continuity_audit"},
		},
	}
}

func (c *ComposerAgent) buildTrace(input ComposeChapterInput, contextPackage *models.ContextPackage) models.ChapterTrace {
	return models.ChapterTrace{
		Chapter:        input.ChapterNumber,
		PlannerInputs:  input.Plan.PlannerInputs,
		ComposerInputs: []string{input.Plan.RuntimePath},
		SelectedSources: func() []string {
			sources := make([]string, len(contextPackage.SelectedContext))
			for i, ctx := range contextPackage.SelectedContext {
				sources[i] = ctx.Source
			}
			return sources
		}(),
	}
}
