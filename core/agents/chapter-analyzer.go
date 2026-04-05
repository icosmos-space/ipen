package agents

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/icosmos-space/ipen/core/llm"
	"github.com/icosmos-space/ipen/core/models"
	"github.com/icosmos-space/ipen/core/utils"
)

// AnalyzeChapterInput 是chapter analyzer input。
type AnalyzeChapterInput struct {
	Book           *models.BookConfig
	BookDir        string
	ChapterNumber  int
	ChapterContent string
	ChapterTitle   string
	ChapterIntent  string
	ContextPackage *models.ContextPackage
	RuleStack      *models.RuleStack
}

// AnalyzeChapterOutput 是analyzer output shape。
type AnalyzeChapterOutput = ParsedWriterOutput

// ChapterAnalyzerAgent 读取a completed chapter and updates truth-file deltas。
type ChapterAnalyzerAgent struct {
	*BaseAgent
}

// ChapterAnalyzerChatHook allows tests to intercept analyzer chat calls.
var ChapterAnalyzerChatHook func(ctx context.Context, agent *ChapterAnalyzerAgent, messages []llm.LLMMessage, options *llm.ChatOptions) (*llm.LLMResponse, error)

// NewChapterAnalyzerAgent 创建chapter analyzer agent。
func NewChapterAnalyzerAgent(ctx AgentContext) *ChapterAnalyzerAgent {
	return &ChapterAnalyzerAgent{BaseAgent: NewBaseAgent(ctx)}
}

// Name 返回agent name。
func (a *ChapterAnalyzerAgent) Name() string { return "chapter-analyzer" }

// AnalyzeChapter 分析chapter content and returns parsed writer-style tags。
func (a *ChapterAnalyzerAgent) AnalyzeChapter(ctx context.Context, input AnalyzeChapterInput) (*AnalyzeChapterOutput, error) {
	if input.Book == nil {
		return nil, fmt.Errorf("book is required")
	}
	book := input.Book
	bookDir := input.BookDir
	chapterNumber := input.ChapterNumber
	chapterContent := input.ChapterContent
	chapterTitle := input.ChapterTitle

	genreProfile, err := ReadGenreProfile(a.Ctx.ProjectRoot, string(book.Genre))
	if err != nil {
		return nil, err
	}
	resolvedLanguage := strings.TrimSpace(book.Language)
	if resolvedLanguage == "" {
		resolvedLanguage = strings.TrimSpace(genreProfile.Profile.Language)
	}
	if resolvedLanguage == "" {
		resolvedLanguage = "zh"
	}

	currentState, _ := a.readFileOrDefault(filepath.Join(bookDir, "story", "current_state.md"), resolvedLanguage)
	ledger, _ := a.readFileOrDefault(filepath.Join(bookDir, "story", "particle_ledger.md"), resolvedLanguage)
	hooks, _ := a.readFileOrDefault(filepath.Join(bookDir, "story", "pending_hooks.md"), resolvedLanguage)
	subplotBoard, _ := a.readFileOrDefault(filepath.Join(bookDir, "story", "subplot_board.md"), resolvedLanguage)
	emotionalArcs, _ := a.readFileOrDefault(filepath.Join(bookDir, "story", "emotional_arcs.md"), resolvedLanguage)
	characterMatrix, _ := a.readFileOrDefault(filepath.Join(bookDir, "story", "character_matrix.md"), resolvedLanguage)
	storyBible, _ := a.readFileOrDefault(filepath.Join(bookDir, "story", "story_bible.md"), resolvedLanguage)
	volumeOutline, _ := a.readFileOrDefault(filepath.Join(bookDir, "story", "volume_outline.md"), resolvedLanguage)

	parsedBookRules, _ := ReadBookRules(bookDir)
	bookRulesBody := ""
	protagonistName := ""
	if parsedBookRules != nil {
		bookRulesBody = parsedBookRules.Raw
		if p, ok := parsedBookRules.Rules["protagonist"].(map[string]any); ok {
			if name, ok := p["name"].(string); ok {
				protagonistName = name
			}
		}
	}

	governedMode := strings.TrimSpace(input.ChapterIntent) != "" && input.ContextPackage != nil && input.RuleStack != nil
	memorySelection, _ := utils.RetrieveMemorySelection(
		bookDir,
		chapterNumber,
		a.buildMemoryGoal(chapterTitle, chapterContent),
		a.findOutlineNode(volumeOutline, chapterNumber, resolvedLanguage),
		nil,
	)
	chapterSummaries := utils.RenderSummarySnapshot(memorySelection.Summaries, resolvedLanguage)
	if strings.TrimSpace(chapterSummaries) == "" || chapterSummaries == "- none" {
		chapterSummaries = a.missingFilePlaceholder(resolvedLanguage)
	}

	var governedMemoryBlocks utils.GovernedMemoryEvidenceBlocks
	if input.ContextPackage != nil {
		governedMemoryBlocks = utils.BuildGovernedMemoryEvidenceBlocks(*input.ContextPackage, resolvedLanguage)
	}

	hooksWorkingSet := hooks
	if governedMode && input.ContextPackage != nil {
		hooksWorkingSet = utils.BuildGovernedHookWorkingSet(
			hooks,
			*input.ContextPackage,
			input.ChapterIntent,
			chapterNumber,
			resolvedLanguage,
			5,
		)
	}

	subplotWorkingSet := subplotBoard
	if governedMode {
		subplotWorkingSet = utils.FilterSubplots(subplotBoard)
	}
	emotionalWorkingSet := emotionalArcs
	if governedMode {
		emotionalWorkingSet = utils.FilterEmotionalArcs(emotionalArcs, chapterNumber)
	}
	matrixWorkingSet := characterMatrix
	if governedMode && input.ChapterIntent != "" && input.ContextPackage != nil {
		matrixWorkingSet = utils.BuildGovernedCharacterMatrixWorkingSet(
			characterMatrix,
			input.ChapterIntent,
			*input.ContextPackage,
			protagonistName,
		)
	}

	reducedControlBlock := ""
	if governedMode {
		reducedControlBlock = a.buildReducedControlBlock(input.ChapterIntent, *input.ContextPackage, *input.RuleStack, resolvedLanguage)
	}

	systemPrompt := a.buildSystemPrompt(book, genreProfile.Profile.Name, genreProfile.Body, bookRulesBody, resolvedLanguage, false)

	hooksBlock := governedMemoryBlocks.HooksBlock
	if hooksBlock == "" {
		if hooksWorkingSet != a.missingFilePlaceholder(resolvedLanguage) {
			if strings.EqualFold(resolvedLanguage, "en") {
				hooksBlock = "\n## Current Hooks\n" + hooksWorkingSet + "\n"
			} else {
				hooksBlock = "\n## 当前伏笔池\n" + hooksWorkingSet + "\n"
			}
		}
	}

	summariesBlock := governedMemoryBlocks.SummariesBlock
	if summariesBlock == "" {
		if chapterSummaries != a.missingFilePlaceholder(resolvedLanguage) {
			if strings.EqualFold(resolvedLanguage, "en") {
				summariesBlock = "\n## Existing Chapter Summaries\n" + chapterSummaries + "\n"
			} else {
				summariesBlock = "\n## 已有章节摘要\n" + chapterSummaries + "\n"
			}
		}
	}

	outlineOrControlBlock := reducedControlBlock
	if outlineOrControlBlock == "" && volumeOutline != a.missingFilePlaceholder(resolvedLanguage) {
		if strings.EqualFold(resolvedLanguage, "en") {
			outlineOrControlBlock = "\n## Volume Outline\n" + volumeOutline + "\n"
		} else {
			outlineOrControlBlock = "\n## 卷纲\n" + volumeOutline + "\n"
		}
	}

	bibleBlock := ""
	if !governedMode && storyBible != a.missingFilePlaceholder(resolvedLanguage) {
		if strings.EqualFold(resolvedLanguage, "en") {
			bibleBlock = "\n## Story Bible\n" + storyBible + "\n"
		} else {
			bibleBlock = "\n## 世界观设定\n" + storyBible + "\n"
		}
	}

	userPrompt := a.buildUserPrompt(chapterAnalyzerUserPromptParams{
		Language:           resolvedLanguage,
		ChapterNumber:      chapterNumber,
		ChapterContent:     chapterContent,
		ChapterTitle:       chapterTitle,
		CurrentState:       currentState,
		Ledger:             ledger,
		HooksBlock:         hooksBlock,
		SummariesBlock:     summariesBlock,
		VolumeSummaryBlock: governedMemoryBlocks.VolumeSummariesBlock,
		SubplotBlock: conditionalBlock(
			subplotWorkingSet != a.missingFilePlaceholder(resolvedLanguage),
			resolvedLanguage,
			"Current Subplot Board",
			"当前支线进度板",
			subplotWorkingSet,
		),
		EmotionalBlock: conditionalBlock(
			emotionalWorkingSet != a.missingFilePlaceholder(resolvedLanguage),
			resolvedLanguage,
			"Current Emotional Arcs",
			"当前情感弧线",
			emotionalWorkingSet,
		),
		MatrixBlock: conditionalBlock(
			matrixWorkingSet != a.missingFilePlaceholder(resolvedLanguage),
			resolvedLanguage,
			"Current Character Matrix",
			"当前角色交互矩阵",
			matrixWorkingSet,
		),
		BibleBlock:            bibleBlock,
		OutlineOrControlBlock: outlineOrControlBlock,
	})

	messages := []llm.LLMMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
	chatOptions := &llm.ChatOptions{MaxTokens: 16384, Temperature: 0.3}
	var response *llm.LLMResponse
	if ChapterAnalyzerChatHook != nil {
		response, err = ChapterAnalyzerChatHook(ctx, a, messages, chatOptions)
	} else {
		response, err = a.Chat(ctx, messages, chatOptions)
	}
	if err != nil {
		return nil, err
	}

	countingMode := utils.ResolveLengthCountingMode(utils.LengthLanguage(resolvedLanguage))
	modelGenreProfile := &models.GenreProfile{
		Name:            genreProfile.Profile.Name,
		Language:        resolvedLanguage,
		NumericalSystem: false,
	}
	output := ParseWriterOutput(chapterNumber, response.Content, modelGenreProfile, countingMode)
	canonicalContent := chapterContent
	canonicalWordCount := utils.CountChapterLength(canonicalContent, countingMode)

	if chapterTitle != "" && (output.Title == a.defaultChapterTitle(chapterNumber, resolvedLanguage) || output.Title == fmt.Sprintf("第%d章", chapterNumber)) {
		output.Title = chapterTitle
	}
	output.Content = canonicalContent
	output.WordCount = canonicalWordCount
	return &output, nil
}

func (a *ChapterAnalyzerAgent) buildSystemPrompt(book *models.BookConfig, genreName, genreBody, bookRulesBody, language string, numericalSystem bool) string {
	if strings.EqualFold(language, "en") {
		numericalBlock := "- This genre has no numerical system; leave UPDATED_LEDGER empty."
		if numericalSystem {
			numericalBlock = "- This genre tracks numerical/resources systems; UPDATED_LEDGER must capture every resource change shown in the chapter."
		}
		return strings.Join([]string{
			"LANGUAGE OVERRIDE: ALL output MUST be in English. The === TAG === markers remain unchanged.",
			"",
			"You are a fiction continuity analyst. Analyze a finished chapter and update tracking files incrementally.",
			"",
			"Book:",
			"- Title: " + book.Title,
			"- Genre: " + genreName + " (" + string(book.Genre) + ")",
			"- Platform: " + string(book.Platform),
			numericalBlock,
			"",
			"Genre guidance:",
			genreBody,
			"",
			conditionalText(bookRulesBody != "", "Book rules:\n"+bookRulesBody, ""),
			"",
			"Output sections: CHAPTER_TITLE, CHAPTER_CONTENT, PRE_WRITE_CHECK, POST_SETTLEMENT, UPDATED_STATE, UPDATED_LEDGER, UPDATED_HOOKS, CHAPTER_SUMMARY, UPDATED_SUBPLOTS, UPDATED_EMOTIONAL_ARCS, UPDATED_CHARACTER_MATRIX.",
		}, "\n")
	}

	return "你是小说连续性分析师。请分析已完成章节并更新追踪文件，输出必须使用 === TAG === 分段。"
}

type chapterAnalyzerUserPromptParams struct {
	Language              string
	ChapterNumber         int
	ChapterContent        string
	ChapterTitle          string
	CurrentState          string
	Ledger                string
	HooksBlock            string
	SummariesBlock        string
	VolumeSummaryBlock    string
	SubplotBlock          string
	EmotionalBlock        string
	MatrixBlock           string
	BibleBlock            string
	OutlineOrControlBlock string
}

func (a *ChapterAnalyzerAgent) buildUserPrompt(params chapterAnalyzerUserPromptParams) string {
	if strings.EqualFold(params.Language, "en") {
		titleLine := ""
		if strings.TrimSpace(params.ChapterTitle) != "" {
			titleLine = "Chapter Title: " + params.ChapterTitle + "\n"
		}
		ledgerBlock := ""
		if strings.TrimSpace(params.Ledger) != "" && params.Ledger != a.missingFilePlaceholder(params.Language) {
			ledgerBlock = "\n## Current Resource Ledger\n" + params.Ledger + "\n"
		}
		return strings.Join([]string{
			fmt.Sprintf("Analyze chapter %d and update all tracking files.", params.ChapterNumber),
			titleLine,
			"## Chapter Content",
			"",
			params.ChapterContent,
			"",
			"## Current State",
			params.CurrentState,
			ledgerBlock,
			params.HooksBlock + params.VolumeSummaryBlock + params.SubplotBlock + params.EmotionalBlock + params.MatrixBlock + params.SummariesBlock + params.OutlineOrControlBlock + params.BibleBlock,
			"",
			"Please return the result strictly in the === TAG === format.",
		}, "\n")
	}
	return fmt.Sprintf("请分析第%d章正文，更新所有追踪文件。\n\n## 正文内容\n\n%s\n\n## 当前状态卡\n%s\n%s%s%s%s%s%s%s\n\n请严格按 === TAG === 格式输出。",
		params.ChapterNumber,
		params.ChapterContent,
		params.CurrentState,
		params.HooksBlock,
		params.VolumeSummaryBlock,
		params.SubplotBlock,
		params.EmotionalBlock,
		params.MatrixBlock,
		params.SummariesBlock,
		params.OutlineOrControlBlock+params.BibleBlock,
	)
}

func (a *ChapterAnalyzerAgent) buildReducedControlBlock(
	chapterIntent string,
	contextPackage models.ContextPackage,
	ruleStack models.RuleStack,
	language string,
) string {
	selected := []string{}
	for _, entry := range contextPackage.SelectedContext {
		line := "- " + entry.Source + ": " + entry.Reason
		if entry.Excerpt != nil && strings.TrimSpace(*entry.Excerpt) != "" {
			line += " | " + strings.TrimSpace(*entry.Excerpt)
		}
		selected = append(selected, line)
	}
	if len(selected) == 0 {
		selected = append(selected, "- none")
	}
	overrides := []string{}
	for _, ov := range ruleStack.ActiveOverrides {
		overrides = append(overrides, fmt.Sprintf("- %s -> %s: %s (%s)", ov.From, ov.To, ov.Reason, ov.Target))
	}
	if len(overrides) == 0 {
		overrides = append(overrides, "- none")
	}

	if strings.EqualFold(language, "en") {
		return strings.Join([]string{
			"\n## Chapter Control Inputs (compiled by Planner/Composer)",
			chapterIntent,
			"",
			"### Selected Context",
			strings.Join(selected, "\n"),
			"",
			"### Rule Stack",
			"- Hard guardrails: " + joinOrNone(ruleStack.Sections.Hard),
			"- Soft constraints: " + joinOrNone(ruleStack.Sections.Soft),
			"- Diagnostic rules: " + joinOrNone(ruleStack.Sections.Diagnostic),
			"",
			"### Active Overrides",
			strings.Join(overrides, "\n"),
			"",
		}, "\n")
	}

	return strings.Join([]string{
		"\n## 本章控制输入（由 Planner/Composer 编译）",
		chapterIntent,
		"",
		"### 已选上下文",
		strings.Join(selected, "\n"),
		"",
		"### 规则栈",
		"- 硬护栏: " + joinOrNone(ruleStack.Sections.Hard),
		"- 软约束: " + joinOrNone(ruleStack.Sections.Soft),
		"- 诊断规则: " + joinOrNone(ruleStack.Sections.Diagnostic),
		"",
		"### 当前覆盖",
		strings.Join(overrides, "\n"),
		"",
	}, "\n")
}

func (a *ChapterAnalyzerAgent) buildMemoryGoal(chapterTitle string, chapterContent string) string {
	parts := []string{}
	if strings.TrimSpace(chapterTitle) != "" {
		parts = append(parts, chapterTitle)
	}
	if strings.TrimSpace(chapterContent) != "" {
		parts = append(parts, truncateRunes(chapterContent, 1500))
	}
	return strings.Join(parts, "\n\n")
}

func (a *ChapterAnalyzerAgent) findOutlineNode(volumeOutline string, chapterNumber int, language string) string {
	if volumeOutline == "" || volumeOutline == a.missingFilePlaceholder("zh") || volumeOutline == a.missingFilePlaceholder("en") {
		return ""
	}
	lines := []string{}
	for _, line := range strings.Split(volumeOutline, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			lines = append(lines, trimmed)
		}
	}

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)^#+\s*Chapter\s*` + fmt.Sprintf("%d", chapterNumber) + `\b`),
		regexp.MustCompile(`^#+\s*第\s*` + fmt.Sprintf("%d", chapterNumber) + `\s*章`),
	}
	_ = language
	for i, line := range lines {
		for _, p := range patterns {
			if p.MatchString(line) {
				if i+1 < len(lines) && !strings.HasPrefix(lines[i+1], "#") {
					return lines[i+1]
				}
				return strings.TrimSpace(strings.TrimLeft(line, "# "))
			}
		}
	}
	return ""
}

func (a *ChapterAnalyzerAgent) readFileOrDefault(path string, language string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return a.missingFilePlaceholder(language), nil
	}
	return string(data), nil
}

func (a *ChapterAnalyzerAgent) missingFilePlaceholder(language string) string {
	if strings.EqualFold(language, "en") {
		return "(file not created yet)"
	}
	return "(文件尚未创建)"
}

func (a *ChapterAnalyzerAgent) defaultChapterTitle(chapterNumber int, language string) string {
	if strings.EqualFold(language, "en") {
		return fmt.Sprintf("Chapter %d", chapterNumber)
	}
	return fmt.Sprintf("第%d章", chapterNumber)
}

func conditionalBlock(enabled bool, language string, enTitle string, zhTitle string, body string) string {
	if !enabled {
		return ""
	}
	if strings.EqualFold(language, "en") {
		return "\n## " + enTitle + "\n" + body + "\n"
	}
	return "\n## " + zhTitle + "\n" + body + "\n"
}

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "(none)"
	}
	return strings.Join(values, ", ")
}

func conditionalText(enabled bool, yes string, no string) string {
	if enabled {
		return yes
	}
	return no
}
