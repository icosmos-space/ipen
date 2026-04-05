package agents

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/icosmos-space/ipen/core/llm"
)

// FoundationReviewDimension 是one scored review dimension。
type FoundationReviewDimension struct {
	Name     string `json:"name"`
	Score    int    `json:"score"`
	Feedback string `json:"feedback"`
}

// FoundationReviewResult 是reviewer result for generated foundation artifacts。
type FoundationReviewResult struct {
	Passed          bool                        `json:"passed"`
	TotalScore      int                         `json:"totalScore"`
	Dimensions      []FoundationReviewDimension `json:"dimensions"`
	OverallFeedback string                      `json:"overallFeedback"`
}

const (
	foundationPassThreshold  = 80
	foundationDimensionFloor = 60
)

// FoundationReviewParams 是review input。
type FoundationReviewParams struct {
	Foundation  ArchitectOutput
	Mode        string // "original" | "fanfic" | "series"
	SourceCanon string
	StyleGuide  string
	Language    string // "zh" | "en"
}

// FoundationReviewerAgent 执行strict editorial scoring for foundation files。
type FoundationReviewerAgent struct {
	*BaseAgent
}

// NewFoundationReviewerAgent 创建foundation reviewer。
func NewFoundationReviewerAgent(ctx AgentContext) *FoundationReviewerAgent {
	return &FoundationReviewerAgent{BaseAgent: NewBaseAgent(ctx)}
}

// Name 返回agent name。
func (a *FoundationReviewerAgent) Name() string { return "foundation-reviewer" }

// Review runs scoring review with strict output contract.
func (a *FoundationReviewerAgent) Review(ctx context.Context, params FoundationReviewParams) (*FoundationReviewResult, error) {
	canonBlock := ""
	if strings.TrimSpace(params.SourceCanon) != "" {
		canonBlock = "\n## Original Canon Reference\n" + truncateRunes(params.SourceCanon, 8000) + "\n"
	}
	styleBlock := ""
	if strings.TrimSpace(params.StyleGuide) != "" {
		styleBlock = "\n## Original Style Reference\n" + truncateRunes(params.StyleGuide, 2000) + "\n"
	}

	dimensions := []string{}
	if strings.EqualFold(params.Mode, "original") {
		dimensions = a.originalDimensions(params.Language)
	} else {
		dimensions = a.derivativeDimensions(params.Language, params.Mode)
	}

	systemPrompt := a.buildChineseReviewPrompt(dimensions, canonBlock, styleBlock)
	if strings.EqualFold(params.Language, "en") {
		systemPrompt = a.buildEnglishReviewPrompt(dimensions, canonBlock, styleBlock)
	}
	userPrompt := a.buildFoundationExcerpt(params.Foundation, params.Language)

	response, err := a.Chat(ctx, []llm.LLMMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}, &llm.ChatOptions{MaxTokens: 4096, Temperature: 0.3})
	if err != nil {
		return nil, err
	}
	return a.parseReviewResult(response.Content, dimensions), nil
}

func (a *FoundationReviewerAgent) originalDimensions(language string) []string {
	if strings.EqualFold(language, "en") {
		return []string{
			"Core Conflict (Is there a clear, compelling central conflict that can sustain 40 chapters?)",
			"Opening Momentum (Can the first 5 chapters create a page-turning hook?)",
			"World Coherence (Is the worldbuilding internally consistent and specific?)",
			"Character Differentiation (Are the main characters distinct in voice and motivation?)",
			"Pacing Feasibility (Does the volume outline have enough variety and avoid repetitive beats?)",
		}
	}
	return []string{
		"核心冲突（是否有清晰且足够有张力的核心冲突支撑 40 章？）",
		"开篇节奏（前 5 章能否形成翻页驱动力？）",
		"世界一致性（世界观是否内在且具体？）",
		"角色区分度（主要角色的声音和动机是否各不相同？）",
		"节奏可行性（卷纲是否有足够变化，避免重复节拍？）",
	}
}

func (a *FoundationReviewerAgent) derivativeDimensions(language, mode string) []string {
	modeLabel := "系列"
	if strings.EqualFold(mode, "fanfic") {
		modeLabel = "同人"
	}
	if strings.EqualFold(language, "en") {
		modeLabel = "Series"
		if strings.EqualFold(mode, "fanfic") {
			modeLabel = "Fan Fiction"
		}
		return []string{
			fmt.Sprintf("Source DNA Preservation (Does the %s respect original world rules, character personality, and established facts?)", modeLabel),
			"New Narrative Space (Is there a clear divergence point with original room, not just retelling?)",
			"Core Conflict (Is the new story conflict compelling and distinct?)",
			"Opening Momentum (Can first 5 chapters hook quickly without heavy setup?)",
			"Pacing Feasibility (Does the outline avoid replaying original beats?)",
		}
	}
	return []string{
		fmt.Sprintf("原作 DNA 保留（%s 是否尊重原作世界规则、角色性格与既有事实？）", modeLabel),
		"新叙事空间（是否有明确分歧点与原创空间，而非复述原作？）",
		"核心冲突（新故事冲突是否有张力且不同于原作？）",
		"开篇节奏（前 5 章能否快速起势，不靠过长铺垫？）",
		"节奏可行性（卷纲是否避免重复原作节拍？）",
	}
}

func (a *FoundationReviewerAgent) buildChineseReviewPrompt(dimensions []string, canonBlock, styleBlock string) string {
	lines := []string{
		"你是一位资深小说编辑，正在审核一本新书的基础设定（世界观+大纲+规则）。",
		"你需要从以下维度逐项打分（0-100），并给出具体意见：",
		"",
	}
	for i, dim := range dimensions {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, dim))
	}
	lines = append(lines,
		"",
		"## 评分标准",
		"- 80+ 通过，可开始写作",
		"- 60-79 需要修订",
		"- <60 方向性问题",
		"",
		"## 输出格式（严格）",
		"=== DIMENSION: 1 ===",
		"分数: {0-100}",
		"意见: {具体反馈}",
		"",
		"...",
		"",
		"=== OVERALL ===",
		"总分: {加权平均}",
		"通过: {是/否}",
		"总评: {1-2段总结}",
	)
	if canonBlock != "" {
		lines = append(lines, canonBlock)
	}
	if styleBlock != "" {
		lines = append(lines, styleBlock)
	}
	lines = append(lines, "请严格审稿，不要宽松打分。")
	return strings.Join(lines, "\n")
}

func (a *FoundationReviewerAgent) buildEnglishReviewPrompt(dimensions []string, canonBlock, styleBlock string) string {
	lines := []string{
		"You are a senior fiction editor reviewing a new book foundation (worldbuilding + outline + rules).",
		"",
		"Score each dimension (0-100) with specific feedback:",
		"",
	}
	for i, dim := range dimensions {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, dim))
	}
	lines = append(lines,
		"",
		"## Scoring",
		"- 80+ Pass, ready to write",
		"- 60-79 Needs revision",
		"- <60 Fundamental direction problem",
		"",
		"## Output format (strict)",
		"=== DIMENSION: 1 ===",
		"Score: {0-100}",
		"Feedback: {specific feedback}",
		"",
		"...",
		"",
		"=== OVERALL ===",
		"Total: {weighted average}",
		"Passed: {yes/no}",
		"Summary: {1-2 paragraphs, biggest problem and best quality}",
	)
	if canonBlock != "" {
		lines = append(lines, canonBlock)
	}
	if styleBlock != "" {
		lines = append(lines, styleBlock)
	}
	lines = append(lines, "Be strict. 80 means ready to write without changes.")
	return strings.Join(lines, "\n")
}

func (a *FoundationReviewerAgent) buildFoundationExcerpt(f ArchitectOutput, language string) string {
	if strings.EqualFold(language, "en") {
		return fmt.Sprintf("## Story Bible\n%s\n\n## Volume Outline\n%s\n\n## Book Rules\n%s\n\n## Initial State\n%s\n\n## Initial Hooks\n%s",
			truncateRunes(f.StoryBible, 3000),
			truncateRunes(f.VolumeOutline, 3000),
			truncateRunes(f.BookRules, 1500),
			truncateRunes(f.CurrentState, 1000),
			truncateRunes(f.PendingHooks, 1000),
		)
	}
	return fmt.Sprintf("## 世界设定\n%s\n\n## 卷纲\n%s\n\n## 规则\n%s\n\n## 初始状态\n%s\n\n## 初始伏笔\n%s",
		truncateRunes(f.StoryBible, 3000),
		truncateRunes(f.VolumeOutline, 3000),
		truncateRunes(f.BookRules, 1500),
		truncateRunes(f.CurrentState, 1000),
		truncateRunes(f.PendingHooks, 1000),
	)
}

func (a *FoundationReviewerAgent) parseReviewResult(content string, dimensions []string) *FoundationReviewResult {
	parsed := make([]FoundationReviewDimension, 0, len(dimensions))
	for i, name := range dimensions {
		re := regexp.MustCompile(`(?s)===\s*DIMENSION:\s*` + strconv.Itoa(i+1) + `\s*===.*?(?:分数|Score)\s*[:：]\s*(\d+).*?(?:意见|Feedback)\s*[:：]\s*(.*?)(?:(?:\n===\s*(?:DIMENSION|OVERALL))|$)`)
		match := re.FindStringSubmatch(content)
		score := 50
		feedback := "(parse failed)"
		if len(match) >= 3 {
			if parsedScore, err := strconv.Atoi(match[1]); err == nil {
				score = parsedScore
			}
			feedback = strings.TrimSpace(match[2])
		}
		parsed = append(parsed, FoundationReviewDimension{
			Name:     name,
			Score:    score,
			Feedback: feedback,
		})
	}

	total := 0
	for _, d := range parsed {
		total += d.Score
	}
	totalScore := 0
	if len(parsed) > 0 {
		totalScore = int(float64(total)/float64(len(parsed)) + 0.5)
	}
	anyBelowFloor := false
	for _, d := range parsed {
		if d.Score < foundationDimensionFloor {
			anyBelowFloor = true
			break
		}
	}
	passed := totalScore >= foundationPassThreshold && !anyBelowFloor

	overallFeedback := "(parse failed)"
	if m := regexp.MustCompile(`(?s)===\s*OVERALL\s*===.*?(?:总评|Summary)\s*[:：]\s*(.*)$`).FindStringSubmatch(content); len(m) >= 2 {
		overallFeedback = strings.TrimSpace(m[1])
	}
	return &FoundationReviewResult{
		Passed:          passed,
		TotalScore:      totalScore,
		Dimensions:      parsed,
		OverallFeedback: overallFeedback,
	}
}
