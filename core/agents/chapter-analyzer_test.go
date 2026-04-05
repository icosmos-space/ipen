package agents

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/icosmos-space/ipen/core/llm"
	"github.com/icosmos-space/ipen/core/models"
	"github.com/icosmos-space/ipen/core/utils"
)

func analyzerTestBook(language string) *models.BookConfig {
	now := time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC)
	return &models.BookConfig{
		ID:               "english-book",
		Title:            "English Book",
		Platform:         models.PlatformOther,
		Genre:            models.Genre("other"),
		Status:           models.StatusActive,
		TargetChapters:   120,
		ChapterWordCount: 2200,
		Language:         language,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func writeAnalyzerGenreProfile(t *testing.T, projectRoot string, language string) {
	t.Helper()
	genresDir := filepath.Join(projectRoot, "genres")
	if err := os.MkdirAll(genresDir, 0755); err != nil {
		t.Fatalf("mkdir genres failed: %v", err)
	}
	content := strings.Join([]string{
		"---",
		"name: Other",
		"language: " + language,
		"description: test profile",
		"---",
		"",
		"# Other Genre",
		"test body",
	}, "\n")
	if err := os.WriteFile(filepath.Join(genresDir, "other.md"), []byte(content), 0644); err != nil {
		t.Fatalf("write profile failed: %v", err)
	}
}

func TestChapterAnalyzer_EnglishWordCountingUsesCanonicalContent(t *testing.T) {
	root := t.TempDir()
	bookDir := filepath.Join(root, "book")
	if err := os.MkdirAll(filepath.Join(bookDir, "story"), 0755); err != nil {
		t.Fatalf("mkdir story failed: %v", err)
	}
	writeAnalyzerGenreProfile(t, root, "en")

	englishContent := "He looked at the sky and waited."
	agent := NewChapterAnalyzerAgent(AgentContext{
		Client:      &llm.LLMClient{Provider: "openai", Stream: false, Defaults: llm.LLMDefaults{Temperature: 0.7, MaxTokens: 4096, Extra: map[string]any{}}},
		Model:       "test-model",
		ProjectRoot: root,
	})

	oldHook := ChapterAnalyzerChatHook
	defer func() { ChapterAnalyzerChatHook = oldHook }()
	ChapterAnalyzerChatHook = func(ctx context.Context, agent *ChapterAnalyzerAgent, messages []llm.LLMMessage, options *llm.ChatOptions) (*llm.LLMResponse, error) {
		return &llm.LLMResponse{
			Content: strings.Join([]string{
				"=== CHAPTER_TITLE ===",
				"A Quiet Sky",
				"",
				"=== CHAPTER_CONTENT ===",
				englishContent,
				"",
				"=== PRE_WRITE_CHECK ===",
				"",
				"=== POST_SETTLEMENT ===",
				"",
				"=== UPDATED_STATE ===",
				"| Field | Value |",
				"| --- | --- |",
				"| Current Chapter | 1 |",
				"",
				"=== UPDATED_LEDGER ===",
				"",
				"=== UPDATED_HOOKS ===",
				"| hook_id | status |",
				"| --- | --- |",
				"| h1 | open |",
				"",
				"=== CHAPTER_SUMMARY ===",
				"| 1 | A Quiet Sky |",
			}, "\n"),
		}, nil
	}

	output, err := agent.AnalyzeChapter(context.Background(), AnalyzeChapterInput{
		Book:           analyzerTestBook("en"),
		BookDir:        bookDir,
		ChapterNumber:  1,
		ChapterContent: englishContent,
	})
	if err != nil {
		t.Fatalf("analyze failed: %v", err)
	}
	if output.WordCount != utils.CountChapterLength(englishContent, models.CountingModeENWords) {
		t.Fatalf("unexpected word count: %d", output.WordCount)
	}
	if output.WordCount != 7 {
		t.Fatalf("expected 7 words, got %d", output.WordCount)
	}
}

func TestChapterAnalyzer_EnglishPromptsForImportedEnglishChapters(t *testing.T) {
	root := t.TempDir()
	bookDir := filepath.Join(root, "book")
	if err := os.MkdirAll(filepath.Join(bookDir, "story"), 0755); err != nil {
		t.Fatalf("mkdir story failed: %v", err)
	}
	writeAnalyzerGenreProfile(t, root, "en")

	agent := NewChapterAnalyzerAgent(AgentContext{
		Client:      &llm.LLMClient{Provider: "openai", Stream: false, Defaults: llm.LLMDefaults{Temperature: 0.7, MaxTokens: 4096, Extra: map[string]any{}}},
		Model:       "test-model",
		ProjectRoot: root,
	})

	capturedMessages := []llm.LLMMessage{}
	oldHook := ChapterAnalyzerChatHook
	defer func() { ChapterAnalyzerChatHook = oldHook }()
	ChapterAnalyzerChatHook = func(ctx context.Context, agent *ChapterAnalyzerAgent, messages []llm.LLMMessage, options *llm.ChatOptions) (*llm.LLMResponse, error) {
		capturedMessages = append([]llm.LLMMessage{}, messages...)
		return &llm.LLMResponse{
			Content: strings.Join([]string{
				"=== CHAPTER_TITLE ===",
				"A Quiet Sky",
				"",
				"=== CHAPTER_CONTENT ===",
				"He looked at the sky and waited.",
				"",
				"=== UPDATED_STATE ===",
				"| Field | Value |",
				"| --- | --- |",
				"| Current Chapter | 1 |",
				"",
				"=== UPDATED_HOOKS ===",
				"| hook_id | status |",
				"| --- | --- |",
				"| h1 | open |",
			}, "\n"),
		}, nil
	}

	_, err := agent.AnalyzeChapter(context.Background(), AnalyzeChapterInput{
		Book:           analyzerTestBook("en"),
		BookDir:        bookDir,
		ChapterNumber:  1,
		ChapterContent: "He looked at the sky and waited.",
		ChapterTitle:   "A Quiet Sky",
	})
	if err != nil {
		t.Fatalf("analyze failed: %v", err)
	}
	if len(capturedMessages) < 2 {
		t.Fatalf("expected captured system+user messages, got %#v", capturedMessages)
	}
	systemPrompt := capturedMessages[0].Content
	userPrompt := capturedMessages[1].Content
	if !strings.Contains(systemPrompt, "ALL output MUST be in English") {
		t.Fatalf("expected english system prompt, got %q", systemPrompt)
	}
	if !strings.Contains(userPrompt, "Analyze chapter 1") ||
		!strings.Contains(userPrompt, "## Chapter Content") ||
		!strings.Contains(userPrompt, "## Current State") {
		t.Fatalf("unexpected user prompt: %q", userPrompt)
	}
	if strings.Contains(userPrompt, "请分析第1章正文") {
		t.Fatalf("expected no chinese prompt in english mode: %q", userPrompt)
	}
}

func TestChapterAnalyzer_GovernedModeUsesControlInputs(t *testing.T) {
	root := t.TempDir()
	bookDir := filepath.Join(root, "book")
	storyDir := filepath.Join(bookDir, "story")
	if err := os.MkdirAll(storyDir, 0755); err != nil {
		t.Fatalf("mkdir story failed: %v", err)
	}
	writeAnalyzerGenreProfile(t, root, "en")

	if err := os.WriteFile(filepath.Join(storyDir, "story_bible.md"), []byte("# Story Bible\n\n- Full bible should stay out of governed analyzer prompts.\n"), 0644); err != nil {
		t.Fatalf("write story_bible failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(storyDir, "volume_outline.md"), []byte("# Volume Outline\n\n## Chapter 100\nReturn to the mentor oath conflict.\n"), 0644); err != nil {
		t.Fatalf("write volume_outline failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(storyDir, "current_state.md"), []byte("# Current State\n\n- Lin Yue still carries the oath token.\n"), 0644); err != nil {
		t.Fatalf("write current_state failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(storyDir, "pending_hooks.md"), []byte(strings.Join([]string{
		"# Pending Hooks",
		"",
		"| hook_id | start_chapter | type | status | last_advanced_chapter | expected_payoff | notes |",
		"| --- | --- | --- | --- | --- | --- | --- |",
		"| guild-route | 1 | mystery | open | 2 | 6 | Merchant guild trail |",
		"| mentor-oath | 8 | relationship | open | 99 | 101 | Mentor oath debt |",
	}, "\n")), 0644); err != nil {
		t.Fatalf("write pending_hooks failed: %v", err)
	}

	agent := NewChapterAnalyzerAgent(AgentContext{
		Client:      &llm.LLMClient{Provider: "openai", Stream: false, Defaults: llm.LLMDefaults{Temperature: 0.7, MaxTokens: 4096, Extra: map[string]any{}}},
		Model:       "test-model",
		ProjectRoot: root,
	})

	capturedMessages := []llm.LLMMessage{}
	oldHook := ChapterAnalyzerChatHook
	defer func() { ChapterAnalyzerChatHook = oldHook }()
	ChapterAnalyzerChatHook = func(ctx context.Context, agent *ChapterAnalyzerAgent, messages []llm.LLMMessage, options *llm.ChatOptions) (*llm.LLMResponse, error) {
		capturedMessages = append([]llm.LLMMessage{}, messages...)
		return &llm.LLMResponse{
			Content: "=== CHAPTER_TITLE ===\nMentor Oath Returns\n\n=== UPDATED_STATE ===\n| Current Chapter | 100 |\n\n=== UPDATED_HOOKS ===\n| hook_id | status |\n| --- | --- |\n| h1 | open |",
		}, nil
	}

	contextPackage := models.ContextPackage{
		Chapter: 100,
		SelectedContext: []models.ContextSource{
			{
				Source:  "story/pending_hooks.md#mentor-oath",
				Reason:  "Primary hook for this chapter",
				Excerpt: ptrString("mentor-oath remains unresolved"),
			},
		},
	}
	ruleStack := models.RuleStack{
		Layers: []models.RuleLayer{
			{ID: "L1", Name: "Global", Precedence: 1, Scope: models.ScopeGlobal},
		},
		Sections: models.RuleStackSections{
			Hard:       []string{"story_bible"},
			Soft:       []string{"author_intent"},
			Diagnostic: []string{"anti_ai_checks"},
		},
		ActiveOverrides: []models.ActiveOverride{
			{From: "brief", To: "current_focus", Reason: "Keep the chapter on the oath debt", Target: "focus"},
		},
	}

	_, err := agent.AnalyzeChapter(context.Background(), AnalyzeChapterInput{
		Book:           analyzerTestBook("en"),
		BookDir:        bookDir,
		ChapterNumber:  100,
		ChapterTitle:   "Mentor Oath Returns",
		ChapterContent: "Lin Yue returned to the mentor oath and the missing explanation.",
		ChapterIntent:  "# Chapter Intent\n\n## Goal\nBring the focus back to the mentor oath conflict.\n",
		ContextPackage: &contextPackage,
		RuleStack:      &ruleStack,
	})
	if err != nil {
		t.Fatalf("analyze failed: %v", err)
	}
	userPrompt := capturedMessages[1].Content
	if !strings.Contains(userPrompt, "## Chapter Control Inputs (compiled by Planner/Composer)") {
		t.Fatalf("expected governed control block, got %q", userPrompt)
	}
	if !strings.Contains(userPrompt, "story/pending_hooks.md#mentor-oath") {
		t.Fatalf("expected selected context source, got %q", userPrompt)
	}
	if !strings.Contains(userPrompt, "Selected Hook Evidence") {
		t.Fatalf("expected governed hook evidence block, got %q", userPrompt)
	}
	if strings.Contains(userPrompt, "## Story Bible") || strings.Contains(userPrompt, "Full bible should stay out of governed analyzer prompts") {
		t.Fatalf("expected no full story bible in governed mode, got %q", userPrompt)
	}
	if strings.Contains(userPrompt, "guild-route") {
		t.Fatalf("expected governed hook working set to remove unrelated guild-route hook, got %q", userPrompt)
	}
}

func ptrString(v string) *string { return &v }
