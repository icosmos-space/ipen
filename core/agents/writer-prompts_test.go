package agents

import (
	"strings"
	"testing"
	"time"

	"github.com/icosmos-space/ipen/core/models"
)

func testPromptBook() *models.BookConfig {
	now := time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC)
	return &models.BookConfig{
		ID:               "prompt-book",
		Title:            "Prompt Book",
		Platform:         models.PlatformTomato,
		Genre:            models.Genre("other"),
		Status:           models.StatusActive,
		TargetChapters:   20,
		ChapterWordCount: 3000,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func testPromptGenre() *models.GenreProfile {
	return &models.GenreProfile{
		ID:                "other",
		Name:              "综合",
		Language:          "zh",
		ChapterTypes:      []string{"setup", "conflict"},
		FatigueWords:      []string{},
		NumericalSystem:   false,
		PowerScaling:      false,
		EraResearch:       false,
		PacingRule:        "",
		SatisfactionTypes: []string{},
		AuditDimensions:   []int{},
	}
}

func TestBuildWriterSystemPrompt_GovernedDemotesMethodBlocks(t *testing.T) {
	chapter := 3
	prompt := BuildWriterSystemPrompt(
		testPromptBook(),
		testPromptGenre(),
		nil,
		"# Book Rules",
		"# Genre Body",
		"# Style Guide\n\nKeep the prose restrained.",
		nil,
		&chapter,
		"creative",
		nil,
		"zh",
		"governed",
		nil,
	)

	if !strings.Contains(prompt, "## 输入治理契约") || !strings.Contains(prompt, "卷纲是默认规划") {
		t.Fatalf("expected governed contract in prompt, got:\n%s", prompt)
	}
	if strings.Contains(prompt, "## 六步走人物心理分析") || strings.Contains(prompt, "## 读者心理学框架") || strings.Contains(prompt, "## 黄金三章规则") {
		t.Fatalf("expected governed prompt to demote always-on methods, got:\n%s", prompt)
	}
}

func TestBuildWriterSystemPrompt_UsesLengthSpecTargetRangeWordings(t *testing.T) {
	chapter := 3
	lengthSpec := &models.LengthSpec{
		Target:        2200,
		SoftMin:       1900,
		SoftMax:       2500,
		HardMin:       1600,
		HardMax:       2800,
		CountingMode:  models.CountingModeZHChars,
		NormalizeMode: models.NormalizeModeNone,
	}

	prompt := BuildWriterSystemPrompt(
		testPromptBook(),
		testPromptGenre(),
		nil,
		"# Book Rules",
		"# Genre Body",
		"# Style Guide\n\nKeep the prose restrained.",
		nil,
		&chapter,
		"creative",
		nil,
		"zh",
		"governed",
		lengthSpec,
	)

	if !strings.Contains(prompt, "目标字数：2200") || !strings.Contains(prompt, "允许区间：1900-2500") {
		t.Fatalf("expected target/range wording, got:\n%s", prompt)
	}
	if strings.Contains(prompt, "正文不少于2200字") {
		t.Fatalf("unexpected old hard wording in prompt:\n%s", prompt)
	}
}

func TestBuildWriterSystemPrompt_GovernedKeepsHardGuardrailsAndConstraints(t *testing.T) {
	chapter := 3
	prompt := BuildWriterSystemPrompt(
		testPromptBook(),
		testPromptGenre(),
		nil,
		"# Book Rules\n\n- Do not reveal the mastermind.",
		"# Genre Body",
		"# Style Guide\n\nKeep the prose restrained.",
		nil,
		&chapter,
		"creative",
		nil,
		"zh",
		"governed",
		nil,
	)

	if !strings.Contains(prompt, "## 核心规则") || !strings.Contains(prompt, "## 硬性禁令") {
		t.Fatalf("expected hard guardrails sections, got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "Do not reveal the mastermind") || !strings.Contains(prompt, "Keep the prose restrained") {
		t.Fatalf("expected book/style constraints to survive, got:\n%s", prompt)
	}
}

func TestBuildWriterSystemPrompt_GovernedEnglishHasVarianceAndResistanceExchange(t *testing.T) {
	chapter := 3
	book := testPromptBook()
	book.Language = "en"
	genre := testPromptGenre()
	genre.Language = "en"
	genre.Name = "General"

	prompt := BuildWriterSystemPrompt(
		book,
		genre,
		nil,
		"# Book Rules",
		"# Genre Body",
		"# Style Guide\n\nKeep the prose restrained.",
		nil,
		&chapter,
		"creative",
		nil,
		"en",
		"governed",
		nil,
	)

	if !strings.Contains(prompt, "English Variance Brief") || !strings.Contains(prompt, "resistance-bearing exchange") {
		t.Fatalf("expected governed english variance+exchange guidance, got:\n%s", prompt)
	}
}
