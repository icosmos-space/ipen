package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/icosmos-space/ipen/core/llm"
	"github.com/icosmos-space/ipen/core/models"
)

// PlanChapterInput 表示plan chapter input。
type PlanChapterInput struct {
	Book            *models.BookConfig
	BookDir         string
	ChapterNumber   int
	ExternalContext string
}

// PlanChapterOutput 表示plan chapter output。
type PlanChapterOutput struct {
	Intent         models.ChapterIntent
	IntentMarkdown string
	PlannerInputs  []string
	RuntimePath    string
}

// PlannerAgent 表示the planner agent。
type PlannerAgent struct {
	*BaseAgent
}

// NewPlannerAgent 创建新的planner agent。
func NewPlannerAgent(ctx AgentContext) *PlannerAgent {
	return &PlannerAgent{
		BaseAgent: NewBaseAgent(ctx),
	}
}

// Name 返回the agent name。
func (p *PlannerAgent) Name() string {
	return "planner"
}

// PlanChapter plans a chapter
func (p *PlannerAgent) PlanChapter(ctx context.Context, input PlanChapterInput) (*PlanChapterOutput, error) {
	if input.Book == nil {
		return nil, fmt.Errorf("book config is required")
	}

	bookDir := input.BookDir

	authorIntent, err := p.readFileOrDefault(filepath.Join(bookDir, "story/author_intent.md"))
	if err != nil {
		return nil, fmt.Errorf("read author_intent.md failed: %w", err)
	}
	currentFocus, err := p.readFileOrDefault(filepath.Join(bookDir, "story/current_focus.md"))
	if err != nil {
		return nil, fmt.Errorf("read current_focus.md failed: %w", err)
	}
	storyBible, err := p.readFileOrDefault(filepath.Join(bookDir, "story/story_bible.md"))
	if err != nil {
		return nil, fmt.Errorf("read story_bible.md failed: %w", err)
	}
	volumeOutline, err := p.readFileOrDefault(filepath.Join(bookDir, "story/volume_outline.md"))
	if err != nil {
		return nil, fmt.Errorf("read volume_outline.md failed: %w", err)
	}
	chapterSummaries, err := p.readFileOrDefault(filepath.Join(bookDir, "story/chapter_summaries.md"))
	if err != nil {
		return nil, fmt.Errorf("read chapter_summaries.md failed: %w", err)
	}
	currentState, err := p.readFileOrDefault(filepath.Join(bookDir, "story/current_state.md"))
	if err != nil {
		return nil, fmt.Errorf("read current_state.md failed: %w", err)
	}

	systemPrompt := p.buildPlannerSystemPrompt(input.Book.Language)
	userPrompt := p.buildPlannerUserPrompt(
		input.ChapterNumber,
		authorIntent,
		currentFocus,
		storyBible,
		volumeOutline,
		chapterSummaries,
		currentState,
		input.ExternalContext,
	)

	messages := []llm.LLMMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	response, err := p.Chat(ctx, messages, &llm.ChatOptions{
		Temperature: 0.7,
		MaxTokens:   4096,
	})
	if err != nil {
		return nil, fmt.Errorf("chat completion failed: %w", err)
	}

	intent, intentMarkdown := p.parsePlannerOutput(response.Content)
	if intent.Chapter == 0 {
		intent.Chapter = input.ChapterNumber
	}
	if strings.TrimSpace(intent.Goal) == "" {
		intent.Goal = fmt.Sprintf("Advance chapter %d with clear narrative focus.", input.ChapterNumber)
	}

	return &PlanChapterOutput{
		Intent:         intent,
		IntentMarkdown: intentMarkdown,
		PlannerInputs: []string{
			authorIntent,
			currentFocus,
			storyBible,
			volumeOutline,
			chapterSummaries,
			currentState,
		},
		RuntimePath: fmt.Sprintf("runtime/chapter-%04d.intent.md", input.ChapterNumber),
	}, nil
}

func (p *PlannerAgent) buildPlannerSystemPrompt(language string) string {
	if strings.EqualFold(language, "en") {
		return `You are a professional novel planning assistant.
Analyze the provided context and produce chapter intent.

Return valid JSON with this structure:
{
  "chapter": 1,
  "goal": "Clear narrative goal",
  "mustKeep": ["key point 1", "key point 2"],
  "mustAvoid": ["avoid 1", "avoid 2"],
  "styleEmphasis": ["style rule"],
  "conflicts": []
}`
	}

	return `你是专业的小说章节规划助手。请结合上下文生成本章意图。
请输出合法 JSON，至少包含以下字段：
{
  "chapter": 1,
  "goal": "本章核心目标",
  "mustKeep": ["必须保留的信息"],
  "mustAvoid": ["必须避免的问题"],
  "styleEmphasis": ["文风要求"],
  "conflicts": []
}`
}

func (p *PlannerAgent) buildPlannerUserPrompt(
	chapterNumber int,
	authorIntent,
	currentFocus,
	storyBible,
	volumeOutline,
	chapterSummaries,
	currentState,
	externalContext string,
) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## 规划章节 %d\n\n", chapterNumber))

	if authorIntent != missingFilePlaceholder {
		sb.WriteString("### 作者意图\n")
		sb.WriteString(authorIntent)
		sb.WriteString("\n\n")
	}

	if currentFocus != missingFilePlaceholder {
		sb.WriteString("### 当前聚焦\n")
		sb.WriteString(currentFocus)
		sb.WriteString("\n\n")
	}

	if storyBible != missingFilePlaceholder {
		sb.WriteString("### 故事设定\n")
		sb.WriteString(truncateRunes(storyBible, 1800))
		sb.WriteString("\n\n")
	}

	if volumeOutline != missingFilePlaceholder {
		sb.WriteString("### 卷纲\n")
		sb.WriteString(volumeOutline)
		sb.WriteString("\n\n")
	}

	if chapterSummaries != missingFilePlaceholder {
		sb.WriteString("### 近期章节摘要\n")
		sb.WriteString(truncateRunes(chapterSummaries, 1800))
		sb.WriteString("\n\n")
	}

	if currentState != missingFilePlaceholder {
		sb.WriteString("### 当前状态\n")
		sb.WriteString(currentState)
		sb.WriteString("\n\n")
	}

	if strings.TrimSpace(externalContext) != "" {
		sb.WriteString("### 外部指令\n")
		sb.WriteString(externalContext)
		sb.WriteString("\n\n")
	}

	sb.WriteString("请输出 JSON。")

	return sb.String()
}

func (p *PlannerAgent) parsePlannerOutput(content string) (models.ChapterIntent, string) {
	intent := models.ChapterIntent{
		Goal:          "Advance the story with clear narrative focus.",
		MustKeep:      []string{},
		MustAvoid:     []string{},
		StyleEmphasis: []string{},
		Conflicts:     []models.ChapterConflict{},
		HookAgenda: models.HookAgenda{
			PressureMap:          []models.HookPressure{},
			MustAdvance:          []string{},
			EligibleResolve:      []string{},
			StaleDebt:            []string{},
			AvoidNewHookFamilies: []string{},
		},
	}

	jsonBlock := extractFirstJSONObject(content)
	if jsonBlock == "" {
		intent.Goal = firstMeaningfulLine(content)
		return intent, content
	}

	type plannerPayload struct {
		Chapter        int                      `json:"chapter"`
		Goal           string                   `json:"goal"`
		OutlineNode    *string                  `json:"outlineNode"`
		SceneDirective *string                  `json:"sceneDirective"`
		ArcDirective   *string                  `json:"arcDirective"`
		MoodDirective  *string                  `json:"moodDirective"`
		TitleDirective *string                  `json:"titleDirective"`
		MustKeep       []string                 `json:"mustKeep"`
		MustAvoid      []string                 `json:"mustAvoid"`
		StyleEmphasis  []string                 `json:"styleEmphasis"`
		Conflicts      []models.ChapterConflict `json:"conflicts"`
		HookAgenda     *models.HookAgenda       `json:"hookAgenda"`
	}

	var payload plannerPayload
	if err := json.Unmarshal([]byte(jsonBlock), &payload); err != nil {
		intent.Goal = firstMeaningfulLine(content)
		return intent, content
	}

	intent.Chapter = payload.Chapter
	if strings.TrimSpace(payload.Goal) != "" {
		intent.Goal = strings.TrimSpace(payload.Goal)
	}
	intent.OutlineNode = payload.OutlineNode
	intent.SceneDirective = payload.SceneDirective
	intent.ArcDirective = payload.ArcDirective
	intent.MoodDirective = payload.MoodDirective
	intent.TitleDirective = payload.TitleDirective
	intent.MustKeep = normalizeStrings(payload.MustKeep)
	intent.MustAvoid = normalizeStrings(payload.MustAvoid)
	intent.StyleEmphasis = normalizeStrings(payload.StyleEmphasis)
	intent.Conflicts = payload.Conflicts
	if intent.Conflicts == nil {
		intent.Conflicts = []models.ChapterConflict{}
	}
	if payload.HookAgenda != nil {
		intent.HookAgenda = *payload.HookAgenda
	}
	if intent.HookAgenda.PressureMap == nil {
		intent.HookAgenda.PressureMap = []models.HookPressure{}
	}
	if intent.HookAgenda.MustAdvance == nil {
		intent.HookAgenda.MustAdvance = []string{}
	}
	if intent.HookAgenda.EligibleResolve == nil {
		intent.HookAgenda.EligibleResolve = []string{}
	}
	if intent.HookAgenda.StaleDebt == nil {
		intent.HookAgenda.StaleDebt = []string{}
	}
	if intent.HookAgenda.AvoidNewHookFamilies == nil {
		intent.HookAgenda.AvoidNewHookFamilies = []string{}
	}

	return intent, content
}

func (p *PlannerAgent) readFileOrDefault(path string) (string, error) {
	return readFileWithFallback(path)
}

func normalizeStrings(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return []string{}
	}
	return out
}

func firstMeaningfulLine(content string) string {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "}") {
			continue
		}
		return trimmed
	}
	return "Advance the story with clear narrative focus."
}
