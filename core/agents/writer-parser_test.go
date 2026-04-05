package agents

import (
	"strings"
	"testing"

	"github.com/icosmos-space/ipen/core/models"
	"github.com/icosmos-space/ipen/core/utils"
)

var defaultGenreProfile = &models.GenreProfile{
	Name:              "测试",
	ID:                "test",
	Language:          "zh",
	ChapterTypes:      []string{},
	FatigueWords:      []string{},
	NumericalSystem:   true,
	PowerScaling:      false,
	EraResearch:       false,
	PacingRule:        "",
	SatisfactionTypes: []string{},
	AuditDimensions:   []int{},
}

func callParseOutput(chapterNumber int, content string, profile *models.GenreProfile, countingMode models.LengthCountingMode) ParsedWriterOutput {
	if profile == nil {
		profile = defaultGenreProfile
	}
	if countingMode == "" {
		countingMode = models.CountingModeZHChars
	}
	return ParseWriterOutput(chapterNumber, content, profile, countingMode)
}

func TestParseWriterOutput_ExtractsTaggedSections(t *testing.T) {
	fullOutput := strings.Join([]string{
		"=== PRE_WRITE_CHECK ===",
		"| 检查项 | 记录 |",
		"",
		"=== CHAPTER_TITLE ===",
		"吞天之始",
		"",
		"=== CHAPTER_CONTENT ===",
		"陈风站在悬崖边，俯视脚下的深渊。",
		"一股强烈吸力从深渊中传来。",
		"",
		"=== POST_SETTLEMENT ===",
		"| 结算项 | 记录 |",
		"",
		"=== UPDATED_STATE ===",
		"# 状态卡",
		"",
		"=== UPDATED_LEDGER ===",
		"# 资源账本",
		"",
		"=== UPDATED_HOOKS ===",
		"| H001 | 深渊之物 | open |",
	}, "\n")

	result := callParseOutput(1, fullOutput, defaultGenreProfile, models.CountingModeZHChars)
	if result.ChapterNumber != 1 || result.Title != "吞天之始" {
		t.Fatalf("unexpected parsed title/chapter: %#v", result)
	}
	if !strings.Contains(result.Content, "陈风站在悬崖边") || !strings.Contains(result.Content, "强烈吸力") {
		t.Fatalf("unexpected content: %q", result.Content)
	}
	if !strings.Contains(result.PreWriteCheck, "检查项") || !strings.Contains(result.PostSettlement, "结算项") {
		t.Fatalf("missing pre/post sections: %#v", result)
	}
	if !strings.Contains(result.UpdatedState, "状态卡") || !strings.Contains(result.UpdatedLedger, "资源账本") || !strings.Contains(result.UpdatedHooks, "H001") {
		t.Fatalf("missing updated sections: %#v", result)
	}
}

func TestParseWriterOutput_WordCountUsesSharedCounter(t *testing.T) {
	output := strings.Join([]string{
		"=== CHAPTER_CONTENT ===",
		"陈风站在悬崖边，俯视脚下的深渊。",
		"一股强烈吸力从深渊中传来。",
	}, "\n")
	result := callParseOutput(1, output, defaultGenreProfile, models.CountingModeZHChars)
	expected := utils.CountChapterLength("陈风站在悬崖边，俯视脚下的深渊。\n一股强烈吸力从深渊中传来。", models.CountingModeZHChars)
	if result.WordCount != expected {
		t.Fatalf("expected wordCount=%d, got %d", expected, result.WordCount)
	}
}

func TestParseWriterOutput_MissingSectionsFallbacks(t *testing.T) {
	output := strings.Join([]string{
		"=== CHAPTER_CONTENT ===",
		"Some content here.",
	}, "\n")

	zh := callParseOutput(42, output, defaultGenreProfile, models.CountingModeZHChars)
	if zh.Title != "第42章" {
		t.Fatalf("expected zh fallback title, got %q", zh.Title)
	}
	if zh.UpdatedState == "" || zh.UpdatedLedger == "" || zh.UpdatedHooks == "" {
		t.Fatalf("expected zh fallback placeholders, got %#v", zh)
	}

	en := callParseOutput(42, output, defaultGenreProfile, models.CountingModeENWords)
	if en.Title != "Chapter 42" {
		t.Fatalf("expected english fallback title, got %q", en.Title)
	}
	if en.UpdatedState != "(state card not updated)" || en.UpdatedLedger != "(ledger not updated)" || en.UpdatedHooks != "(hooks pool not updated)" {
		t.Fatalf("unexpected english placeholders: %#v", en)
	}
}

func TestParseWriterOutput_EmptyOrTaglessContent(t *testing.T) {
	empty := callParseOutput(1, "", defaultGenreProfile, models.CountingModeZHChars)
	if empty.Title != "第1章" || empty.Content != "" || empty.WordCount != 0 {
		t.Fatalf("unexpected empty parse result: %#v", empty)
	}

	tagless := callParseOutput(5, "Just some random text without tags", defaultGenreProfile, models.CountingModeZHChars)
	if tagless.Title != "第5章" || tagless.Content != "" || tagless.WordCount != 0 {
		t.Fatalf("unexpected tagless parse result: %#v", tagless)
	}
}

func TestParseWriterOutput_TrimsSectionValues(t *testing.T) {
	output := strings.Join([]string{
		"=== CHAPTER_TITLE ===",
		"   吞天之始   ",
		"",
		"=== CHAPTER_CONTENT ===",
		"  内容  ",
	}, "\n")
	result := callParseOutput(1, output, defaultGenreProfile, models.CountingModeZHChars)
	if result.Title != "吞天之始" || result.Content != "内容" {
		t.Fatalf("expected trimmed values, got %#v", result)
	}
}

func TestParseWriterOutput_EnglishCountMode(t *testing.T) {
	englishContent := "He looked at the sky."
	output := "=== CHAPTER_CONTENT ===\n" + englishContent
	result := callParseOutput(1, output, defaultGenreProfile, models.CountingModeENWords)
	if result.WordCount != utils.CountChapterLength(englishContent, models.CountingModeENWords) {
		t.Fatalf("unexpected english word count: %d", result.WordCount)
	}
}

func TestParseCreativeOutput_Fallbacks(t *testing.T) {
	rawHeading := "# 第1章 觉醒之日\n\n林风缓缓睁开眼。" + strings.Repeat("这是一段很长的正文内容。", 20)
	byHeading := ParseCreativeOutput(1, rawHeading, models.CountingModeZHChars)
	if byHeading.Title != "觉醒之日" || len(byHeading.Content) <= 100 {
		t.Fatalf("expected heading fallback success, got %#v", byHeading)
	}

	rawEN := "# Chapter 1: Awakening Day\n\nHe woke to distant bells. " + strings.Repeat("Long English prose follows. ", 15)
	byENHeading := ParseCreativeOutput(1, rawEN, models.CountingModeENWords)
	if byENHeading.Title != "Awakening Day" || len(byENHeading.Content) <= 100 {
		t.Fatalf("expected english heading fallback success, got %#v", byENHeading)
	}

	rawLabel := "章节标题：暗夜追踪\n正文：" + strings.Repeat("黑暗中一道身影掠过屋顶，无声无息。", 20)
	byLabel := ParseCreativeOutput(5, rawLabel, models.CountingModeZHChars)
	if byLabel.Title != "暗夜追踪" || len(byLabel.Content) <= 100 {
		t.Fatalf("expected label fallback success, got %#v", byLabel)
	}

	tooShort := ParseCreativeOutput(1, "太短了", models.CountingModeZHChars)
	if tooShort.Content != "" || tooShort.Title != "第1章" {
		t.Fatalf("expected short fallback empty content, got %#v", tooShort)
	}

	tooShortEN := ParseCreativeOutput(1, "too short", models.CountingModeENWords)
	if tooShortEN.Content != "" || tooShortEN.Title != "Chapter 1" {
		t.Fatalf("expected short english fallback, got %#v", tooShortEN)
	}
}
