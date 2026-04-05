package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/icosmos-space/ipen/core/agents"
	"github.com/icosmos-space/ipen/core/llm"
	"github.com/icosmos-space/ipen/core/models"
)

func TestDetectAndRewrite_UsesAntiDetectRepairAndRecordsHistory(t *testing.T) {
	root := t.TempDir()
	bookDir := filepath.Join(root, "book")
	if err := os.MkdirAll(filepath.Join(bookDir, "story"), 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	config := models.DetectionConfig{
		Provider:    "custom",
		APIURL:      "https://example.invalid/detect",
		APIKeyEnv:   "TEST_DETECTION_KEY",
		Threshold:   0.5,
		Enabled:     true,
		AutoRewrite: true,
		MaxRetries:  2,
	}

	oldDetectHook := DetectAIContentHook
	oldRepairHook := RepairChapterHook
	defer func() {
		DetectAIContentHook = oldDetectHook
		RepairChapterHook = oldRepairHook
	}()

	detectCalls := 0
	DetectAIContentHook = func(ctx context.Context, cfg models.DetectionConfig, content string) (*agents.DetectionResult, error) {
		detectCalls++
		if detectCalls == 1 {
			return &agents.DetectionResult{
				Score:      0.91,
				Provider:   "custom",
				DetectedAt: "2026-04-03T00:00:00.000Z",
			}, nil
		}
		return &agents.DetectionResult{
			Score:      0.14,
			Provider:   "custom",
			DetectedAt: "2026-04-03T00:00:01.000Z",
		}, nil
	}

	var capturedRepairInput agents.RepairChapterInput
	RepairChapterHook = func(ctx context.Context, writer *agents.WriterAgent, input agents.RepairChapterInput) (*agents.ReviseOutput, error) {
		capturedRepairInput = input
		return &agents.ReviseOutput{
			RevisedContent: "humanized chapter",
			TokenUsage:     &models.TokenUsage{},
		}, nil
	}

	agentCtx := agents.AgentContext{
		Client: &llm.LLMClient{
			Provider: "openai",
			Stream:   false,
			Defaults: llm.LLMDefaults{
				Temperature: 0.7,
				MaxTokens:   4096,
				Extra:       map[string]any{},
			},
		},
		Model:       "test-model",
		ProjectRoot: root,
	}

	result, err := DetectAndRewrite(context.Background(), config, agentCtx, bookDir, "raw chapter", 7, "xuanhuan")
	if err != nil {
		t.Fatalf("detectAndRewrite failed: %v", err)
	}
	if capturedRepairInput.BookDir != bookDir ||
		capturedRepairInput.ChapterContent != "raw chapter" ||
		capturedRepairInput.ChapterNumber != 7 ||
		capturedRepairInput.Mode != agents.ReviseModeAntiDetect ||
		capturedRepairInput.Genre != "xuanhuan" {
		t.Fatalf("unexpected repair input: %#v", capturedRepairInput)
	}
	if result.ChapterNumber != 7 || result.OriginalScore != 0.91 || result.FinalScore != 0.14 || result.Attempts != 1 || !result.Passed || result.FinalContent != "humanized chapter" {
		t.Fatalf("unexpected detectAndRewrite result: %#v", result)
	}

	history, err := LoadDetectionHistory(bookDir)
	if err != nil {
		t.Fatalf("load history failed: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(history))
	}
	entry := history[0]
	if entry.ChapterNumber != 7 || entry.Action != "rewrite" || entry.Attempt != 1 || entry.Score != 0.14 {
		t.Fatalf("unexpected history entry: %#v", entry)
	}
}
