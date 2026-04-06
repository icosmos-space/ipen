package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/icosmos-space/ipen/core/models"
)

// ========== BootstrapStructuredStateFromMarkdown ==========

func TestBootstrapStructuredStateFromMarkdown_CreatesStateFiles(t *testing.T) {
	bookDir := setupBootstrapTestBookDir(t)

	// Create markdown source files
	createMarkdownSources(t, bookDir, 5)

	result, err := BootstrapStructuredStateFromMarkdown(bookDir, 5)
	if err != nil {
		t.Fatalf("BootstrapStructuredStateFromMarkdown failed: %v", err)
	}

	if len(result.CreatedFiles) == 0 {
		t.Fatal("expected some created files")
	}

	stateDir := filepath.Join(bookDir, "story", "state")
	if _, err := os.Stat(filepath.Join(stateDir, "manifest.json")); err != nil {
		t.Fatal("expected manifest.json to be created")
	}
}

func TestBootstrapStructuredStateFromMarkdown_Idempotent(t *testing.T) {
	bookDir := setupBootstrapTestBookDir(t)
	createMarkdownSources(t, bookDir, 3)

	// First bootstrap
	result1, err := BootstrapStructuredStateFromMarkdown(bookDir, 3)
	if err != nil {
		t.Fatalf("first bootstrap failed: %v", err)
	}

	// Second bootstrap - should not create new files
	result2, err := BootstrapStructuredStateFromMarkdown(bookDir, 3)
	if err != nil {
		t.Fatalf("second bootstrap failed: %v", err)
	}

	if len(result2.CreatedFiles) != 0 {
		t.Fatalf("expected no new files on second bootstrap, got %v", result2.CreatedFiles)
	}

	if len(result1.CreatedFiles) == 0 {
		t.Fatal("expected first bootstrap to create files")
	}
}

func TestBootstrapStructuredStateFromMarkdown_ParsesSummaries(t *testing.T) {
	bookDir := setupBootstrapTestBookDir(t)

	summariesMD := `| Chapter | Title | Characters | Key Events | State Changes | Hook Activity | Mood | Chapter Type |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | First Chapter | Alice, Bob | Introduction | World revealed | hook1 opened | mysterious | prologue |
| 2 | Second Chapter | Charlie | Development | Power gained | - | tense | mainline |
`
	writeTestFile(t, filepath.Join(bookDir, "story", "chapter_summaries.md"), summariesMD)
	writeTestFile(t, filepath.Join(bookDir, "story", "current_state.md"), "| Field | Value |\n| --- | --- |\n")
	writeTestFile(t, filepath.Join(bookDir, "story", "pending_hooks.md"), "")

	result, err := BootstrapStructuredStateFromMarkdown(bookDir, 2)
	if err != nil {
		t.Fatalf("bootstrap failed: %v", err)
	}

	// Load and verify
	summariesPath := filepath.Join(bookDir, "story", "state", "chapter_summaries.json")
	var summaries models.ChapterSummariesState
	readTestJSON(t, summariesPath, &summaries)

	if len(summaries.Rows) != 2 {
		t.Fatalf("expected 2 summary rows, got %d", len(summaries.Rows))
	}
	if summaries.Rows[0].Title != "First Chapter" {
		t.Fatalf("expected title 'First Chapter', got %s", summaries.Rows[0].Title)
	}

	_ = result
}

func TestBootstrapStructuredStateFromMarkdown_ParsesHooks(t *testing.T) {
	bookDir := setupBootstrapTestBookDir(t)

	hooksMD := `| hook_id | start_chapter | type | status | last_advanced_chapter | expected_payoff | payoff_timing | notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| mystery-1 | 1 | mystery | open | 3 | Reveal truth | near-term | Important mystery |
`
	writeTestFile(t, filepath.Join(bookDir, "story", "pending_hooks.md"), hooksMD)
	writeTestFile(t, filepath.Join(bookDir, "story", "current_state.md"), "| Field | Value |\n| --- | --- |\n")
	writeTestFile(t, filepath.Join(bookDir, "story", "chapter_summaries.md"), "")

	result, err := BootstrapStructuredStateFromMarkdown(bookDir, 3)
	if err != nil {
		t.Fatalf("bootstrap failed: %v", err)
	}

	hooksPath := filepath.Join(bookDir, "story", "state", "hooks.json")
	var hooksState models.HooksState
	readTestJSON(t, hooksPath, &hooksState)

	if len(hooksState.Hooks) != 1 {
		t.Fatalf("expected 1 hook, got %d", len(hooksState.Hooks))
	}
	if hooksState.Hooks[0].HookID != "mystery-1" {
		t.Fatalf("expected hook ID 'mystery-1', got %s", hooksState.Hooks[0].HookID)
	}

	_ = result
}

func TestBootstrapStructuredStateFromMarkdown_ParsesCurrentState(t *testing.T) {
	bookDir := setupBootstrapTestBookDir(t)

	currentStateMD := `| Field | Value |
| --- | --- |
| Current Chapter | 5 |
| Current Location | Village |
| Current Goal | Find the sword |
`
	writeTestFile(t, filepath.Join(bookDir, "story", "current_state.md"), currentStateMD)
	writeTestFile(t, filepath.Join(bookDir, "story", "pending_hooks.md"), "")
	writeTestFile(t, filepath.Join(bookDir, "story", "chapter_summaries.md"), "")

	// Create chapter files to set durable progress
	chaptersDir := filepath.Join(bookDir, "chapters")
	if err := os.MkdirAll(chaptersDir, 0755); err != nil {
		t.Fatalf("failed to create chapters dir: %v", err)
	}
	for i := 1; i <= 5; i++ {
		writeTestFile(t, filepath.Join(chaptersDir, string(rune('0'+i))+"_ch.md"), "Ch")
	}

	result, err := BootstrapStructuredStateFromMarkdown(bookDir, 5)
	if err != nil {
		t.Fatalf("bootstrap failed: %v", err)
	}

	statePath := filepath.Join(bookDir, "story", "state", "current_state.json")
	var currentState models.CurrentStateState
	readTestJSON(t, statePath, &currentState)

	if currentState.Chapter != 5 {
		t.Fatalf("expected chapter 5, got %d", currentState.Chapter)
	}
	if len(currentState.Facts) != 2 {
		t.Fatalf("expected 2 facts, got %d", len(currentState.Facts))
	}

	_ = result
}

func TestBootstrapStructuredStateFromMarkdown_DetectsLanguage(t *testing.T) {
	bookDir := setupBootstrapTestBookDir(t)

	// Set Chinese language in book.json
	bookConfig := map[string]string{"language": "zh"}
	writeTestJSON(t, filepath.Join(bookDir, "book.json"), bookConfig)

	writeTestFile(t, filepath.Join(bookDir, "story", "current_state.md"), "| Field | Value |\n| --- | --- |\n")
	writeTestFile(t, filepath.Join(bookDir, "story", "pending_hooks.md"), "")
	writeTestFile(t, filepath.Join(bookDir, "story", "chapter_summaries.md"), "")

	result, err := BootstrapStructuredStateFromMarkdown(bookDir, 0)
	if err != nil {
		t.Fatalf("bootstrap failed: %v", err)
	}

	if result.Manifest.Language != models.LanguageZH {
		t.Fatalf("expected language zh, got %s", result.Manifest.Language)
	}
}

func TestBootstrapStructuredStateFromMarkdown_ResolvesDurableProgress(t *testing.T) {
	bookDir := setupBootstrapTestBookDir(t)

	// Create chapter files
	chaptersDir := filepath.Join(bookDir, "chapters")
	if err := os.MkdirAll(chaptersDir, 0755); err != nil {
		t.Fatalf("failed to create chapters dir: %v", err)
	}
	writeTestFile(t, filepath.Join(chaptersDir, "1_intro.md"), "# Chapter 1")
	writeTestFile(t, filepath.Join(chaptersDir, "2_development.md"), "# Chapter 2")
	writeTestFile(t, filepath.Join(chaptersDir, "3_climax.md"), "# Chapter 3")

	writeTestFile(t, filepath.Join(bookDir, "story", "current_state.md"), "| Field | Value |\n| --- | --- |\n")
	writeTestFile(t, filepath.Join(bookDir, "story", "pending_hooks.md"), "")
	writeTestFile(t, filepath.Join(bookDir, "story", "chapter_summaries.md"), "")

	result, err := BootstrapStructuredStateFromMarkdown(bookDir, 0)
	if err != nil {
		t.Fatalf("bootstrap failed: %v", err)
	}

	if result.Manifest.LastAppliedChapter != 3 {
		t.Fatalf("expected last applied chapter 3, got %d", result.Manifest.LastAppliedChapter)
	}
}

func TestBootstrapStructuredStateFromMarkdown_UsesFallbackChapter(t *testing.T) {
	bookDir := setupBootstrapTestBookDir(t)

	writeTestFile(t, filepath.Join(bookDir, "story", "current_state.md"), "| Field | Value |\n| --- | --- |\n| Current Chapter | 7 |")
	writeTestFile(t, filepath.Join(bookDir, "story", "pending_hooks.md"), "")
	writeTestFile(t, filepath.Join(bookDir, "story", "chapter_summaries.md"), "")

	result, err := BootstrapStructuredStateFromMarkdown(bookDir, 10)
	if err != nil {
		t.Fatalf("bootstrap failed: %v", err)
	}

	// Should use max of fallback and chapter from markdown
	if result.Manifest.LastAppliedChapter != 10 {
		t.Fatalf("expected last applied chapter 10, got %d", result.Manifest.LastAppliedChapter)
	}
}

func TestBootstrapStructuredStateFromMarkdown_WarnsOnEmptyCurrentState(t *testing.T) {
	bookDir := setupBootstrapTestBookDir(t)

	writeTestFile(t, filepath.Join(bookDir, "story", "current_state.md"), "")
	writeTestFile(t, filepath.Join(bookDir, "story", "pending_hooks.md"), "")
	writeTestFile(t, filepath.Join(bookDir, "story", "chapter_summaries.md"), "")

	result, err := BootstrapStructuredStateFromMarkdown(bookDir, 0)
	if err != nil {
		t.Fatalf("bootstrap failed: %v", err)
	}

	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "current_state") && strings.Contains(w, "empty") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected warning about empty current_state, got: %v", result.Warnings)
	}
}

// ========== RewriteStructuredStateFromMarkdown ==========

func TestRewriteStructuredStateFromMarkdown_ForcesRebuild(t *testing.T) {
	bookDir := setupBootstrapTestBookDir(t)
	createMarkdownSources(t, bookDir, 4)

	// Bootstrap first
	_, err := BootstrapStructuredStateFromMarkdown(bookDir, 4)
	if err != nil {
		t.Fatalf("initial bootstrap failed: %v", err)
	}

	// Modify state files directly
	stateDir := filepath.Join(bookDir, "story", "state")
	modifiedState := models.CurrentStateState{
		Chapter: 99,
		Facts:   []models.CurrentStateFact{},
	}
	writeTestJSON(t, filepath.Join(stateDir, "current_state.json"), modifiedState)

	// Rewrite should force rebuild from markdown
	result, err := RewriteStructuredStateFromMarkdown(bookDir, 4)
	if err != nil {
		t.Fatalf("rewrite failed: %v", err)
	}

	// Should have reset chapter to 4 (from markdown)
	var currentState models.CurrentStateState
	readTestJSON(t, filepath.Join(stateDir, "current_state.json"), &currentState)

	if currentState.Chapter != 4 {
		t.Fatalf("expected chapter 4 after rewrite, got %d", currentState.Chapter)
	}

	if len(result.Warnings) > 0 {
		t.Logf("rewrite warnings: %v", result.Warnings)
	}
}

// ========== ResolveDurableStoryProgress ==========

func TestResolveDurableStoryProgress_UsesChapterFiles(t *testing.T) {
	bookDir := setupBootstrapTestBookDir(t)

	chaptersDir := filepath.Join(bookDir, "chapters")
	if err := os.MkdirAll(chaptersDir, 0755); err != nil {
		t.Fatalf("failed to create chapters dir: %v", err)
	}
	writeTestFile(t, filepath.Join(chaptersDir, "1_ch.md"), "Ch1")
	writeTestFile(t, filepath.Join(chaptersDir, "2_ch.md"), "Ch2")
	writeTestFile(t, filepath.Join(chaptersDir, "3_ch.md"), "Ch3")
	// Skip 4
	writeTestFile(t, filepath.Join(chaptersDir, "5_ch.md"), "Ch5")

	progress, err := ResolveDurableStoryProgress(bookDir, 0)
	if err != nil {
		t.Fatalf("ResolveDurableStoryProgress failed: %v", err)
	}

	// Should be 3 (contiguous from 1)
	if progress != 3 {
		t.Fatalf("expected progress 3, got %d", progress)
	}
}

func TestResolveDurableStoryProgress_UsesFallback(t *testing.T) {
	bookDir := setupBootstrapTestBookDir(t)

	progress, err := ResolveDurableStoryProgress(bookDir, 10)
	if err != nil {
		// Error is OK if no chapters exist
		t.Logf("ResolveDurableStoryProgress returned error (expected): %v", err)
	}

	// Should use fallback
	if progress != 10 {
		t.Fatalf("expected fallback progress 10, got %d", progress)
	}
}

func TestResolveDurableStoryProgress_PrefersMaxOfDurableAndFallback(t *testing.T) {
	bookDir := setupBootstrapTestBookDir(t)

	chaptersDir := filepath.Join(bookDir, "chapters")
	if err := os.MkdirAll(chaptersDir, 0755); err != nil {
		t.Fatalf("failed to create chapters dir: %v", err)
	}
	writeTestFile(t, filepath.Join(chaptersDir, "1_ch.md"), "Ch1")
	writeTestFile(t, filepath.Join(chaptersDir, "2_ch.md"), "Ch2")

	progress, err := ResolveDurableStoryProgress(bookDir, 5)
	if err != nil {
		t.Fatalf("ResolveDurableStoryProgress failed: %v", err)
	}

	// Should use max(2, 5) = 5
	if progress != 5 {
		t.Fatalf("expected progress 5, got %d", progress)
	}
}

// ========== ResolveContiguousChapterPrefix ==========

func TestResolveContiguousChapterPrefix_FromOne(t *testing.T) {
	chapters := []int{1, 2, 3, 5, 7}
	result := ResolveContiguousChapterPrefix(chapters)

	if result != 3 {
		t.Fatalf("expected contiguous prefix 3, got %d", result)
	}
}

func TestResolveContiguousChapterPrefix_EmptyList(t *testing.T) {
	chapters := []int{}
	result := ResolveContiguousChapterPrefix(chapters)

	if result != 0 {
		t.Fatalf("expected 0 for empty list, got %d", result)
	}
}

func TestResolveContiguousChapterPrefix_StartsFromTwo(t *testing.T) {
	chapters := []int{2, 3, 4}
	result := ResolveContiguousChapterPrefix(chapters)

	// Should be 0 since it doesn't start from 1
	if result != 0 {
		t.Fatalf("expected 0 when not starting from 1, got %d", result)
	}
}

func TestResolveContiguousChapterPrefix_FullSequence(t *testing.T) {
	chapters := []int{1, 2, 3, 4, 5}
	result := ResolveContiguousChapterPrefix(chapters)

	if result != 5 {
		t.Fatalf("expected 5, got %d", result)
	}
}

func TestResolveContiguousChapterPrefix_IgnoresZeroAndNegative(t *testing.T) {
	chapters := []int{0, -1, 1, 2, 3}
	result := ResolveContiguousChapterPrefix(chapters)

	if result != 3 {
		t.Fatalf("expected 3, got %d", result)
	}
}

// ========== Helper functions ==========

func setupBootstrapTestBookDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bookDir := filepath.Join(dir, "books", "test-book")
	if err := os.MkdirAll(filepath.Join(bookDir, "story"), 0755); err != nil {
		t.Fatalf("failed to create book dir: %v", err)
	}
	// Create default book.json
	bookConfig := map[string]string{"language": "en"}
	writeTestJSON(t, filepath.Join(bookDir, "book.json"), bookConfig)
	return bookDir
}

func createMarkdownSources(t *testing.T, bookDir string, chapter int) {
	t.Helper()

	currentStateMD := `| Field | Value |
| --- | --- |
| Current Chapter | ` + string(rune('0'+chapter)) + ` |
| Current Location | Village |
`
	writeTestFile(t, filepath.Join(bookDir, "story", "current_state.md"), currentStateMD)

	summariesMD := "| Chapter | Title | Characters | Key Events | State Changes | Hook Activity | Mood | Chapter Type |\n| --- | --- | --- | --- | --- | --- | --- | --- |\n"
	for i := 1; i <= chapter; i++ {
		summariesMD += "| " + string(rune('0'+i)) + " | Chapter " + string(rune('0'+i)) + " | Alice | Events | Changes | Hook | Mood | mainline |\n"
	}
	writeTestFile(t, filepath.Join(bookDir, "story", "chapter_summaries.md"), summariesMD)

	hooksMD := `| hook_id | start_chapter | type | status | last_advanced_chapter | expected_payoff | payoff_timing | notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| mystery-1 | 1 | mystery | open | ` + string(rune('0'+chapter)) + ` | Reveal truth | near-term | Important |
`
	writeTestFile(t, filepath.Join(bookDir, "story", "pending_hooks.md"), hooksMD)
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}

func readTestJSON(t *testing.T, path string, target any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file %s: %v", path, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("failed to unmarshal JSON from %s: %v", path, err)
	}
}
