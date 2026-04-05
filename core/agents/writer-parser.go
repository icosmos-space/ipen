package agents

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/icosmos-space/ipen/core/models"
	"github.com/icosmos-space/ipen/core/utils"
)

// CreativeOutput 是a lightweight parsed output for creative-only generation。
type CreativeOutput struct {
	Title         string `json:"title"`
	Content       string `json:"content"`
	WordCount     int    `json:"wordCount"`
	PreWriteCheck string `json:"preWriteCheck"`
}

// ParsedWriterOutput 是the structured parse of writer/settler tagged output。
type ParsedWriterOutput struct {
	ChapterNumber          int    `json:"chapterNumber"`
	Title                  string `json:"title"`
	Content                string `json:"content"`
	WordCount              int    `json:"wordCount"`
	PreWriteCheck          string `json:"preWriteCheck"`
	PostSettlement         string `json:"postSettlement"`
	UpdatedState           string `json:"updatedState"`
	UpdatedLedger          string `json:"updatedLedger"`
	UpdatedHooks           string `json:"updatedHooks"`
	ChapterSummary         string `json:"chapterSummary"`
	UpdatedSubplots        string `json:"updatedSubplots"`
	UpdatedEmotionalArcs   string `json:"updatedEmotionalArcs"`
	UpdatedCharacterMatrix string `json:"updatedCharacterMatrix"`
}

// ParseCreativeOutput 解析creative output blocks with fallback extraction。
func ParseCreativeOutput(chapterNumber int, content string, countingMode models.LengthCountingMode) CreativeOutput {
	extract := func(tag string) string { return extractTaggedSection(content, tag) }

	chapterContent := extract("CHAPTER_CONTENT")
	if strings.TrimSpace(chapterContent) == "" {
		chapterContent = fallbackExtractContent(content, countingMode)
	}

	title := extract("CHAPTER_TITLE")
	if strings.TrimSpace(title) == "" {
		title = fallbackExtractTitle(content, chapterNumber, countingMode)
	}

	return CreativeOutput{
		Title:         title,
		Content:       chapterContent,
		WordCount:     utils.CountChapterLength(chapterContent, countingMode),
		PreWriteCheck: extract("PRE_WRITE_CHECK"),
	}
}

// ParseWriterOutput 解析full writer output with state/hook settlement sections。
func ParseWriterOutput(chapterNumber int, content string, genreProfile *models.GenreProfile, countingMode models.LengthCountingMode) ParsedWriterOutput {
	extract := func(tag string) string { return extractTaggedSection(content, tag) }
	chapterContent := extract("CHAPTER_CONTENT")

	title := extract("CHAPTER_TITLE")
	if strings.TrimSpace(title) == "" {
		title = defaultChapterTitle(chapterNumber, countingMode)
	}

	updatedState := extract("UPDATED_STATE")
	if strings.TrimSpace(updatedState) == "" {
		updatedState = defaultStatePlaceholder(countingMode)
	}

	updatedHooks := extract("UPDATED_HOOKS")
	if strings.TrimSpace(updatedHooks) == "" {
		updatedHooks = defaultHooksPlaceholder(countingMode)
	}

	updatedLedger := ""
	if genreProfile != nil && genreProfile.NumericalSystem {
		updatedLedger = extract("UPDATED_LEDGER")
		if strings.TrimSpace(updatedLedger) == "" {
			updatedLedger = defaultLedgerPlaceholder(countingMode)
		}
	}

	return ParsedWriterOutput{
		ChapterNumber:          chapterNumber,
		Title:                  title,
		Content:                chapterContent,
		WordCount:              utils.CountChapterLength(chapterContent, countingMode),
		PreWriteCheck:          extract("PRE_WRITE_CHECK"),
		PostSettlement:         extract("POST_SETTLEMENT"),
		UpdatedState:           updatedState,
		UpdatedLedger:          updatedLedger,
		UpdatedHooks:           updatedHooks,
		ChapterSummary:         extract("CHAPTER_SUMMARY"),
		UpdatedSubplots:        extract("UPDATED_SUBPLOTS"),
		UpdatedEmotionalArcs:   extract("UPDATED_EMOTIONAL_ARCS"),
		UpdatedCharacterMatrix: extract("UPDATED_CHARACTER_MATRIX"),
	}
}

func extractTaggedSection(content, tag string) string {
	re := regexp.MustCompile(`(?s)===\s*` + regexp.QuoteMeta(tag) + `\s*===\s*(.*?)(?:(?:\n===\s*[A-Z_]+\s*===)|$)`)
	match := re.FindStringSubmatch(content)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func fallbackExtractContent(raw string, countingMode models.LengthCountingMode) string {
	if match := regexp.MustCompile(`(?m)^#\s*第\d+章[^\n]*\n+([\s\S]+)$`).FindStringSubmatch(raw); len(match) >= 2 {
		return strings.TrimSpace(match[1])
	}
	if countingMode == models.CountingModeENWords {
		if match := regexp.MustCompile(`(?im)^#\s*Chapter\s+\d+(?::|\s+)[^\n]*\n+([\s\S]+)$`).FindStringSubmatch(raw); len(match) >= 2 {
			return strings.TrimSpace(match[1])
		}
	}

	if match := regexp.MustCompile(`(?:正文|内容|章节内容)[:：]\s*([\s\S]+)`).FindStringSubmatch(raw); len(match) >= 2 {
		return strings.TrimSpace(match[1])
	}
	if countingMode == models.CountingModeENWords {
		if match := regexp.MustCompile(`(?i)(?:content|chapter content)[:：]\s*([\s\S]+)`).FindStringSubmatch(raw); len(match) >= 2 {
			return strings.TrimSpace(match[1])
		}
	}

	lines := strings.Split(raw, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if regexp.MustCompile(`^===\s*[A-Z_]+\s*===`).MatchString(trimmed) {
			continue
		}
		if regexp.MustCompile(`(?i)^(PRE_WRITE_CHECK|CHAPTER_TITLE|章节标题|写作自检)\s*[:：]`).MatchString(trimmed) {
			continue
		}
		kept = append(kept, line)
	}
	candidate := strings.TrimSpace(strings.Join(kept, "\n"))
	if len([]rune(candidate)) > 100 {
		return candidate
	}
	return ""
}

func fallbackExtractTitle(raw string, chapterNumber int, countingMode models.LengthCountingMode) string {
	if match := regexp.MustCompile(`(?m)^#\s*第\d+章\s*(.+)$`).FindStringSubmatch(raw); len(match) >= 2 {
		return strings.TrimSpace(match[1])
	}
	if countingMode == models.CountingModeENWords {
		if match := regexp.MustCompile(`(?im)^#\s*Chapter\s+\d+(?::|\s+)\s*(.+)$`).FindStringSubmatch(raw); len(match) >= 2 {
			return strings.TrimSpace(match[1])
		}
	}
	if match := regexp.MustCompile(`(?im)(?:CHAPTER_TITLE|章节标题)\s*[:：]\s*(.+)$`).FindStringSubmatch(raw); len(match) >= 2 {
		return strings.TrimSpace(match[1])
	}
	return defaultChapterTitle(chapterNumber, countingMode)
}

func defaultChapterTitle(chapterNumber int, countingMode models.LengthCountingMode) string {
	if countingMode == models.CountingModeENWords {
		return fmt.Sprintf("Chapter %d", chapterNumber)
	}
	return fmt.Sprintf("第%d章", chapterNumber)
}

func defaultStatePlaceholder(countingMode models.LengthCountingMode) string {
	if countingMode == models.CountingModeENWords {
		return "(state card not updated)"
	}
	return "(状态卡未更新)"
}

func defaultLedgerPlaceholder(countingMode models.LengthCountingMode) string {
	if countingMode == models.CountingModeENWords {
		return "(ledger not updated)"
	}
	return "(账本未更新)"
}

func defaultHooksPlaceholder(countingMode models.LengthCountingMode) string {
	if countingMode == models.CountingModeENWords {
		return "(hooks pool not updated)"
	}
	return "(伏笔池未更新)"
}
