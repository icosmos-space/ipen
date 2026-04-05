package agents

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/icosmos-space/ipen/core/llm"
	"github.com/icosmos-space/ipen/core/models"
	"github.com/icosmos-space/ipen/core/utils"
)

// ReviseMode controls chapter repair strategy.
type ReviseMode string

const (
	ReviseModePolish     ReviseMode = "polish"
	ReviseModeRewrite    ReviseMode = "rewrite"
	ReviseModeRework     ReviseMode = "rework"
	ReviseModeLocalFix   ReviseMode = "local-fix"
	ReviseModeAntiDetect ReviseMode = "anti-detect"
)

// DEFAULT_REVISE_MODE 是the default revision mode。
const DEFAULT_REVISE_MODE ReviseMode = ReviseModeLocalFix

// RepairChapterInput 是writer repair input。
type RepairChapterInput struct {
	BookDir        string       `json:"bookDir"`
	ChapterContent string       `json:"chapterContent"`
	ChapterNumber  int          `json:"chapterNumber"`
	Issues         []AuditIssue `json:"issues"`
	Mode           ReviseMode   `json:"mode"`
	Genre          string       `json:"genre,omitempty"`
}

// ReviseOutput 是repair result。
type ReviseOutput struct {
	Mode            ReviseMode         `json:"mode"`
	OriginalContent string             `json:"originalContent"`
	RevisedContent  string             `json:"revisedContent"`
	Issues          []AuditIssue       `json:"issues"`
	TokenUsage      *models.TokenUsage `json:"tokenUsage,omitempty"`
}

// RepairChapter repairs chapter content based on issues and mode.
func (w *WriterAgent) RepairChapter(ctx context.Context, input RepairChapterInput) (*ReviseOutput, error) {
	mode := normalizeReviseMode(input.Mode)
	if strings.TrimSpace(input.ChapterContent) == "" {
		return &ReviseOutput{
			Mode:            mode,
			OriginalContent: input.ChapterContent,
			RevisedContent:  input.ChapterContent,
			Issues:          append([]AuditIssue{}, input.Issues...),
		}, nil
	}

	if mode == ReviseModeLocalFix {
		revised := applyDeterministicLocalFix(input.ChapterContent, input.Issues)
		if revised == "" {
			revised = input.ChapterContent
		}
		return &ReviseOutput{
			Mode:            mode,
			OriginalContent: input.ChapterContent,
			RevisedContent:  revised,
			Issues:          append([]AuditIssue{}, input.Issues...),
		}, nil
	}

	systemPrompt, userPrompt := buildRepairPrompts(mode, input)
	response, err := w.Chat(ctx, []llm.LLMMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}, &llm.ChatOptions{Temperature: 0.35, MaxTokens: 12000})
	if err != nil {
		return nil, fmt.Errorf("repair chat failed: %w", err)
	}

	revised := parseRevisedContent(response.Content)
	if strings.TrimSpace(revised) == "" {
		revised = input.ChapterContent
	}
	usage := response.Usage
	return &ReviseOutput{
		Mode:            mode,
		OriginalContent: input.ChapterContent,
		RevisedContent:  revised,
		Issues:          append([]AuditIssue{}, input.Issues...),
		TokenUsage:      &usage,
	}, nil
}

func normalizeReviseMode(mode ReviseMode) ReviseMode {
	switch mode {
	case ReviseModePolish, ReviseModeRewrite, ReviseModeRework, ReviseModeLocalFix, ReviseModeAntiDetect:
		return mode
	default:
		return DEFAULT_REVISE_MODE
	}
}

func applyDeterministicLocalFix(content string, issues []AuditIssue) string {
	revised := strings.TrimSpace(content)
	if revised == "" {
		return content
	}

	for _, issue := range issues {
		if strings.Contains(strings.ToLower(issue.Category), "title") {
			continue
		}
		if strings.Contains(strings.ToLower(issue.Category), "sensitive") {
			revised = regexp.MustCompile(`(?i)(违禁|敏感|政治|色情|暴力)`).ReplaceAllString(revised, "")
		}
	}

	// Basic prose cleanup with minimal semantic drift.
	revised = regexp.MustCompile(`[ \t]{2,}`).ReplaceAllString(revised, " ")
	revised = regexp.MustCompile(`\n{3,}`).ReplaceAllString(revised, "\n\n")

	patches := utils.ParseLocalFixPatches(revised)
	if len(patches) > 0 {
		if result := utils.ApplyLocalFixPatches(content, patches); result.Applied {
			return result.RevisedContent
		}
	}

	return revised
}

func buildRepairPrompts(mode ReviseMode, input RepairChapterInput) (string, string) {
	action := "polish"
	switch mode {
	case ReviseModeRewrite:
		action = "rewrite the chapter while preserving core plot facts"
	case ReviseModeRework:
		action = "rework the chapter substantially while preserving canon facts"
	case ReviseModeAntiDetect:
		action = "reduce AI-detection fingerprints while preserving narrative facts"
	case ReviseModePolish:
		action = "polish style and readability"
	}

	issues := []string{}
	for _, issue := range input.Issues {
		issues = append(issues, fmt.Sprintf("- [%s] %s | %s", issue.Severity, issue.Description, issue.Suggestion))
	}
	if len(issues) == 0 {
		issues = append(issues, "- No explicit issues provided. Improve clarity and readability.")
	}

	system := "You are a fiction revision assistant. Perform one-pass revision and output only the revised chapter body. Do not add analysis."
	user := fmt.Sprintf("Task: %s\n\nIssues:\n%s\n\nOriginal chapter:\n%s", action, strings.Join(issues, "\n"), input.ChapterContent)
	return system, user
}

func parseRevisedContent(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	if block := regexp.MustCompile("(?is)```(?:markdown|md|text)?\\s*([\\s\\S]*?)\\s*```").FindStringSubmatch(trimmed); len(block) >= 2 {
		return strings.TrimSpace(block[1])
	}
	if section := extractTaggedSection(trimmed, "REVISED_CONTENT"); section != "" {
		return section
	}
	return trimmed
}
