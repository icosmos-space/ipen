package pipeline

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/icosmos-space/ipen/core/models"
	"github.com/icosmos-space/ipen/core/state"
)

func TestAgentTools_RegistersGovernanceTools(t *testing.T) {
	toolNames := map[string]struct{}{}
	for _, tool := range AGENT_TOOLS {
		toolNames[tool.Name] = struct{}{}
	}
	required := []string{"plan_chapter", "compose_chapter", "update_author_intent", "update_current_focus"}
	for _, name := range required {
		if _, ok := toolNames[name]; !ok {
			t.Fatalf("expected tool %s to be registered", name)
		}
	}
}

func TestExecuteAgentTool_BlocksTruthFileProgressManipulation(t *testing.T) {
	root := t.TempDir()
	stateManager := state.NewStateManager(root)
	bookID := "agent-book"
	now := time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC)

	if err := stateManager.SaveBookConfig(bookID, &models.BookConfig{
		ID:               bookID,
		Title:            "Agent Book",
		Platform:         models.PlatformTomato,
		Genre:            models.Genre("other"),
		Status:           models.StatusActive,
		TargetChapters:   20,
		ChapterWordCount: 3000,
		CreatedAt:        now,
		UpdatedAt:        now,
	}); err != nil {
		t.Fatalf("save book config failed: %v", err)
	}

	pipeline := NewPipelineRunner(PipelineConfig{
		ProjectRoot: root,
		Model:       "test-model",
	})
	config := PipelineConfig{
		ProjectRoot: root,
		Model:       "test-model",
	}

	raw, err := ExecuteAgentTool(context.Background(), pipeline, stateManager, config, "write_truth_file", map[string]any{
		"bookId":   bookID,
		"fileName": "current_state.md",
		"content":  "# Current State\n\n| Current Chapter | 999 |\n",
	})
	if err != nil {
		t.Fatalf("execute tool failed: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("unmarshal result failed: %v", err)
	}
	errMsg, _ := payload["error"].(string)
	if !strings.Contains(errMsg, "章节进度") {
		t.Fatalf("expected progress manipulation error, got %q", errMsg)
	}
}

func TestExecuteAgentTool_BlocksWriteFullPipelineWhenProgressAheadOfIndex(t *testing.T) {
	root := t.TempDir()
	stateManager := state.NewStateManager(root)
	bookID := "agent-book"
	now := time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC)

	if err := stateManager.SaveBookConfig(bookID, &models.BookConfig{
		ID:               bookID,
		Title:            "Agent Book",
		Platform:         models.PlatformTomato,
		Genre:            models.Genre("other"),
		Status:           models.StatusActive,
		TargetChapters:   20,
		ChapterWordCount: 3000,
		CreatedAt:        now,
		UpdatedAt:        now,
	}); err != nil {
		t.Fatalf("save book config failed: %v", err)
	}

	if err := stateManager.SaveChapterIndex(bookID, []models.ChapterMeta{{
		Number:         1,
		Title:          "Existing Chapter",
		Status:         models.StatusApproved,
		WordCount:      120,
		CreatedAt:      now,
		UpdatedAt:      now,
		AuditIssues:    []string{},
		LengthWarnings: []string{},
	}}); err != nil {
		t.Fatalf("save chapter index failed: %v", err)
	}

	chaptersDir := filepath.Join(stateManager.BookDir(bookID), "chapters")
	if err := os.MkdirAll(chaptersDir, 0755); err != nil {
		t.Fatalf("mkdir chapters failed: %v", err)
	}
	files := map[string]string{
		"0001_Existing.md": "# Chapter 1\n",
		"0002_Second.md":   "# Chapter 2\n",
		"0003_Third.md":    "# Chapter 3\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(chaptersDir, name), []byte(body), 0644); err != nil {
			t.Fatalf("write chapter file failed: %v", err)
		}
	}

	pipeline := NewPipelineRunner(PipelineConfig{
		ProjectRoot: root,
		Model:       "test-model",
	})
	config := PipelineConfig{
		ProjectRoot: root,
		Model:       "test-model",
	}

	raw, err := ExecuteAgentTool(context.Background(), pipeline, stateManager, config, "write_full_pipeline", map[string]any{
		"bookId": bookID,
		"count":  1,
	})
	if err != nil {
		t.Fatalf("execute tool failed: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("unmarshal result failed: %v", err)
	}
	errMsg, _ := payload["error"].(string)
	if !strings.Contains(errMsg, "write_full_pipeline") {
		t.Fatalf("expected write_full_pipeline guard error, got %q", errMsg)
	}
}
