package agents

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/icosmos-space/ipen/core/llm"
	"github.com/icosmos-space/ipen/core/models"
	"github.com/icosmos-space/ipen/core/state"
	"github.com/icosmos-space/ipen/core/utils"
)

var writerNextTagPattern = regexp.MustCompile(`(?m)^===\s*[A-Z_]+\s*===\s*$`)

// WriteChapterInput 表示write chapter input。
type WriteChapterInput struct {
	Book                *models.BookConfig
	BookDir             string
	ChapterNumber       int
	ExternalContext     string
	ChapterIntent       string
	ContextPackage      *models.ContextPackage
	RuleStack           *models.RuleStack
	Trace               *models.ChapterTrace
	LengthSpec          *models.LengthSpec
	WordCountOverride   int
	TemperatureOverride float64
}

// WriteChapterOutput 表示write chapter output。
type WriteChapterOutput struct {
	ChapterNumber           int
	Title                   string
	Content                 string
	WordCount               int
	PreWriteCheck           string
	PostSettlement          string
	RuntimeStateDelta       *models.RuntimeStateDelta
	RuntimeStateSnapshot    *state.RuntimeStateSnapshot
	UpdatedState            string
	UpdatedLedger           string
	UpdatedHooks            string
	ChapterSummary          string
	UpdatedChapterSummaries string
	UpdatedSubplots         string
	UpdatedEmotionalArcs    string
	UpdatedCharacterMatrix  string
	TokenUsage              *models.TokenUsage
}

// WriterAgent 表示the writer agent。
type WriterAgent struct {
	*BaseAgent
}

// NewWriterAgent 创建新的writer agent。
func NewWriterAgent(ctx AgentContext) *WriterAgent {
	return &WriterAgent{
		BaseAgent: NewBaseAgent(ctx),
	}
}

// Name 返回the agent name。
func (w *WriterAgent) Name() string {
	return "writer"
}

// WriteChapter 写入a chapter。
func (w *WriterAgent) WriteChapter(ctx context.Context, input WriteChapterInput) (*WriteChapterOutput, error) {
	bookDir := input.BookDir
	chapterNumber := input.ChapterNumber

	storyBible, err := w.readFileOrDefault(filepath.Join(bookDir, "story/story_bible.md"))
	if err != nil {
		return nil, fmt.Errorf("read story bible failed: %w", err)
	}
	volumeOutline, err := w.readFileOrDefault(filepath.Join(bookDir, "story/volume_outline.md"))
	if err != nil {
		return nil, fmt.Errorf("read volume outline failed: %w", err)
	}
	currentState, err := w.readFileOrDefault(filepath.Join(bookDir, "story/current_state.md"))
	if err != nil {
		return nil, fmt.Errorf("read current state failed: %w", err)
	}
	pendingHooks, err := w.readFileOrDefault(filepath.Join(bookDir, "story/pending_hooks.md"))
	if err != nil {
		return nil, fmt.Errorf("read pending hooks failed: %w", err)
	}
	chapterSummaries, err := w.readFileOrDefault(filepath.Join(bookDir, "story/chapter_summaries.md"))
	if err != nil {
		return nil, fmt.Errorf("read chapter summaries failed: %w", err)
	}

	systemPrompt := w.buildWriterSystemPrompt(input.Book, storyBible, currentState, volumeOutline)
	userPrompt := w.buildUserPrompt(chapterNumber, chapterSummaries, pendingHooks, input.ChapterIntent)

	messages := []llm.LLMMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	temp := input.TemperatureOverride
	if temp == 0 {
		temp = 0.8
	}

	maxTokens := 16000
	if input.LengthSpec != nil && input.LengthSpec.Target > 0 {
		maxTokens = input.LengthSpec.Target * 2
		if maxTokens < 4096 {
			maxTokens = 4096
		}
	}

	response, err := w.Chat(ctx, messages, &llm.ChatOptions{
		Temperature: temp,
		MaxTokens:   maxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("chat completion failed: %w", err)
	}

	title, chapterContent := w.parseWriterOutput(chapterNumber, input.Book.Language, response.Content)
	preWriteCheck := extractWriterTaggedSection(response.Content, "PRE_WRITE_CHECK")

	countingMode := models.CountingModeZHChars
	if input.LengthSpec != nil {
		countingMode = input.LengthSpec.CountingMode
	} else if strings.EqualFold(input.Book.Language, "en") {
		countingMode = models.CountingModeENWords
	}

	wordCount := utils.CountChapterLength(chapterContent, countingMode)
	usage := response.Usage

	return &WriteChapterOutput{
		ChapterNumber: chapterNumber,
		Title:         title,
		Content:       chapterContent,
		WordCount:     wordCount,
		PreWriteCheck: preWriteCheck,
		TokenUsage:    &usage,
	}, nil
}

func (w *WriterAgent) buildWriterSystemPrompt(book *models.BookConfig, storyBible, currentState, volumeOutline string) string {
	var sb strings.Builder

	sb.WriteString("你是专业的网络小说写作助手。\n")
	sb.WriteString("写作要求：\n")
	sb.WriteString("1. 保持情节连贯。\n")
	sb.WriteString("2. 人物行为要符合既有设定。\n")
	sb.WriteString("3. 节奏稳定，尽量避免模板化句式。\n")
	sb.WriteString("4. 使用自然、可读的叙事语言。\n\n")

	if strings.EqualFold(book.Language, "en") {
		sb.WriteString("Write in English.\n")
	} else {
		sb.WriteString("请使用中文写作。\n")
	}

	if storyBible != missingFilePlaceholder {
		sb.WriteString("\n## 故事圣经\n")
		sb.WriteString(truncateRunes(storyBible, 1200))
		sb.WriteString("\n")
	}

	if currentState != missingFilePlaceholder {
		sb.WriteString("\n## 当前状态\n")
		sb.WriteString(currentState)
		sb.WriteString("\n")
	}

	if volumeOutline != missingFilePlaceholder {
		sb.WriteString("\n## 卷纲\n")
		sb.WriteString(truncateRunes(volumeOutline, 1200))
		sb.WriteString("\n")
	}

	return sb.String()
}

func (w *WriterAgent) buildUserPrompt(chapterNumber int, chapterSummaries, pendingHooks, chapterIntent string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## 第%d章\n", chapterNumber))

	if strings.TrimSpace(chapterIntent) != "" {
		sb.WriteString("## 章节意图\n")
		sb.WriteString(chapterIntent)
		sb.WriteString("\n\n")
	}

	if chapterSummaries != missingFilePlaceholder {
		sb.WriteString("## 章节摘要\n")
		sb.WriteString(chapterSummaries)
		sb.WriteString("\n\n")
	}

	if pendingHooks != missingFilePlaceholder {
		sb.WriteString("## 待回收伏笔\n")
		sb.WriteString(pendingHooks)
		sb.WriteString("\n\n")
	}

	sb.WriteString("请据此生成本章内容。优先保证连贯性和可读性。\n")

	return sb.String()
}

func (w *WriterAgent) parseWriterOutput(chapterNumber int, language string, content string) (string, string) {
	title := extractWriterTaggedSection(content, "CHAPTER_TITLE")
	chapterContent := extractWriterTaggedSection(content, "CHAPTER_CONTENT")

	if chapterContent == "" {
		lines := strings.Split(content, "\n")
		contentStart := 0
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "# ") {
				if title == "" {
					title = strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
				}
				contentStart = i + 1
				break
			}
		}

		chapterContent = strings.TrimSpace(strings.Join(lines[contentStart:], "\n"))
	}

	if chapterContent == "" {
		chapterContent = strings.TrimSpace(content)
	}

	if title == "" {
		if strings.EqualFold(language, "en") {
			title = fmt.Sprintf("Chapter %d", chapterNumber)
		} else {
			title = fmt.Sprintf("第%d章", chapterNumber)
		}
	}

	return title, chapterContent
}

func (w *WriterAgent) readFileOrDefault(path string) (string, error) {
	return readFileWithFallback(path)
}

func extractWriterTaggedSection(content string, tag string) string {
	marker := "=== " + tag + " ==="
	start := strings.Index(content, marker)
	if start < 0 {
		return ""
	}

	rest := content[start+len(marker):]
	nextTag := writerNextTagPattern.FindStringIndex(rest)
	if nextTag == nil {
		return strings.TrimSpace(rest)
	}

	return strings.TrimSpace(rest[:nextTag[0]])
}
