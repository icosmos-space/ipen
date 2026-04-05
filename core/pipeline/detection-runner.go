package pipeline

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/icosmos-space/ipen/core/agents"
	"github.com/icosmos-space/ipen/core/models"
)

// DetectChapterResult 是the single-pass detection result。
type DetectChapterResult struct {
	ChapterNumber int                     `json:"chapterNumber"`
	Detection     *agents.DetectionResult `json:"detection"`
	Passed        bool                    `json:"passed"`
}

// DetectAndRewriteResult 是the detect->rewrite loop result。
type DetectAndRewriteResult struct {
	ChapterNumber int     `json:"chapterNumber"`
	OriginalScore float64 `json:"originalScore"`
	FinalScore    float64 `json:"finalScore"`
	Attempts      int     `json:"attempts"`
	Passed        bool    `json:"passed"`
	FinalContent  string  `json:"finalContent"`
}

// DetectAIContentHook enables tests to stub detection behavior.
var DetectAIContentHook = func(ctx context.Context, cfg models.DetectionConfig, content string) (*agents.DetectionResult, error) {
	return agents.DetectAIContent(ctx, agents.DetectionConfig{
		Provider:  cfg.Provider,
		APIURL:    cfg.APIURL,
		APIKeyEnv: cfg.APIKeyEnv,
	}, content)
}

// RepairChapterHook enables tests to stub writer anti-detect revision behavior.
var RepairChapterHook = func(ctx context.Context, writer *agents.WriterAgent, input agents.RepairChapterInput) (*agents.ReviseOutput, error) {
	return writer.RepairChapter(ctx, input)
}

// DetectChapter runs one detection pass.
func DetectChapter(ctx context.Context, config models.DetectionConfig, content string, chapterNumber int) (*DetectChapterResult, error) {
	detection, err := DetectAIContentHook(ctx, config, content)
	if err != nil {
		return nil, err
	}
	return &DetectChapterResult{
		ChapterNumber: chapterNumber,
		Detection:     detection,
		Passed:        detection.Score <= config.Threshold,
	}, nil
}

// DetectAndRewrite 执行detect->repair->re-detect loop using writer anti-detect mode。
func DetectAndRewrite(
	ctx context.Context,
	config models.DetectionConfig,
	agentCtx agents.AgentContext,
	bookDir string,
	content string,
	chapterNumber int,
	genre string,
) (*DetectAndRewriteResult, error) {
	maxRetries := config.MaxRetries
	currentContent := content

	firstDetection, err := DetectAIContentHook(ctx, config, currentContent)
	if err != nil {
		return nil, err
	}
	originalScore := firstDetection.Score

	if firstDetection.Score <= config.Threshold {
		if err := recordDetectionHistory(bookDir, models.DetectionHistoryEntry{
			ChapterNumber: chapterNumber,
			Timestamp:     firstDetection.DetectedAt,
			Provider:      firstDetection.Provider,
			Score:         firstDetection.Score,
			Action:        "detect",
			Attempt:       0,
		}); err != nil {
			return nil, err
		}
		return &DetectAndRewriteResult{
			ChapterNumber: chapterNumber,
			OriginalScore: originalScore,
			FinalScore:    firstDetection.Score,
			Attempts:      0,
			Passed:        true,
			FinalContent:  currentContent,
		}, nil
	}

	finalScore := firstDetection.Score
	attempts := 0
	for i := 0; i < maxRetries; i++ {
		attempts = i + 1

		writer := agents.NewWriterAgent(agentCtx)
		reviseOutput, err := RepairChapterHook(ctx, writer, agents.RepairChapterInput{
			BookDir:        bookDir,
			ChapterContent: currentContent,
			ChapterNumber:  chapterNumber,
			Issues: []agents.AuditIssue{{
				Severity:    "warning",
				Category:    "AIGC检测",
				Description: "AI 检测分数超过阈值",
				Suggestion:  "降低AI生成痕迹：增加段落长度差异、减少套话、增加口语化表达",
			}},
			Mode:  agents.ReviseModeAntiDetect,
			Genre: genre,
		})
		if err != nil {
			return nil, err
		}
		if reviseOutput.RevisedContent == "" {
			break
		}
		currentContent = reviseOutput.RevisedContent

		reDetection, err := DetectAIContentHook(ctx, config, currentContent)
		if err != nil {
			return nil, err
		}
		finalScore = reDetection.Score

		if err := recordDetectionHistory(bookDir, models.DetectionHistoryEntry{
			ChapterNumber: chapterNumber,
			Timestamp:     reDetection.DetectedAt,
			Provider:      reDetection.Provider,
			Score:         reDetection.Score,
			Action:        "rewrite",
			Attempt:       attempts,
		}); err != nil {
			return nil, err
		}

		if finalScore <= config.Threshold {
			break
		}
	}

	return &DetectAndRewriteResult{
		ChapterNumber: chapterNumber,
		OriginalScore: originalScore,
		FinalScore:    finalScore,
		Attempts:      attempts,
		Passed:        finalScore <= config.Threshold,
		FinalContent:  currentContent,
	}, nil
}

func recordDetectionHistory(bookDir string, entry models.DetectionHistoryEntry) error {
	historyPath := filepath.Join(bookDir, "story", "detection_history.json")
	history := []models.DetectionHistoryEntry{}

	if raw, err := os.ReadFile(historyPath); err == nil {
		_ = json.Unmarshal(raw, &history)
	}

	history = append(history, entry)

	if err := os.MkdirAll(filepath.Dir(historyPath), 0755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(historyPath, payload, 0644)
}

// LoadDetectionHistory 读取detection history from disk。
func LoadDetectionHistory(bookDir string) ([]models.DetectionHistoryEntry, error) {
	historyPath := filepath.Join(bookDir, "story", "detection_history.json")
	raw, err := os.ReadFile(historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []models.DetectionHistoryEntry{}, nil
		}
		return nil, err
	}
	result := []models.DetectionHistoryEntry{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return []models.DetectionHistoryEntry{}, nil
	}
	return result, nil
}
