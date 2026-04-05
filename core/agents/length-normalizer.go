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

// NormalizeLengthInput defines one chapter normalization request.
type NormalizeLengthInput struct {
	ChapterContent      string            `json:"chapterContent"`
	LengthSpec          models.LengthSpec `json:"lengthSpec"`
	ChapterIntent       string            `json:"chapterIntent,omitempty"`
	ReducedControlBlock string            `json:"reducedControlBlock,omitempty"`
}

// NormalizeLengthOutput 是length normalization result。
type NormalizeLengthOutput struct {
	NormalizedContent string                     `json:"normalizedContent"`
	FinalCount        int                        `json:"finalCount"`
	Applied           bool                       `json:"applied"`
	Mode              models.LengthNormalizeMode `json:"mode"`
	Warning           string                     `json:"warning,omitempty"`
	TokenUsage        *models.TokenUsage         `json:"tokenUsage,omitempty"`
}

// LengthNormalizerAgent 执行one-pass chapter length normalization。
type LengthNormalizerAgent struct {
	*BaseAgent
}

// NewLengthNormalizerAgent 创建新的length normalizer。
func NewLengthNormalizerAgent(ctx AgentContext) *LengthNormalizerAgent {
	return &LengthNormalizerAgent{BaseAgent: NewBaseAgent(ctx)}
}

// Name 返回agent name。
func (l *LengthNormalizerAgent) Name() string {
	return "length-normalizer"
}

// NormalizeChapter 规范化chapter length with single pass strategy。
func (l *LengthNormalizerAgent) NormalizeChapter(ctx context.Context, input NormalizeLengthInput) (*NormalizeLengthOutput, error) {
	originalCount := utils.CountChapterLength(input.ChapterContent, input.LengthSpec.CountingMode)
	mode := input.LengthSpec.NormalizeMode
	if mode == models.NormalizeModeNone {
		mode = utils.ChooseNormalizeMode(originalCount, input.LengthSpec)
	}
	if mode == models.NormalizeModeNone {
		return &NormalizeLengthOutput{
			NormalizedContent: input.ChapterContent,
			FinalCount:        originalCount,
			Applied:           false,
			Mode:              mode,
		}, nil
	}

	systemPrompt := l.buildSystemPrompt(mode)
	userPrompt := l.buildUserPrompt(input, originalCount, mode)
	response, err := l.Chat(ctx, []llm.LLMMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}, &llm.ChatOptions{Temperature: 0.2, MaxTokens: maxInt(4096, int(float64(originalCount)*1.2))})
	if err != nil {
		return nil, err
	}

	normalizedContent := l.sanitizeNormalizedContent(response.Content, input.ChapterContent)
	finalCount := utils.CountChapterLength(normalizedContent, input.LengthSpec.CountingMode)
	warning := buildLengthWarning(finalCount, input.LengthSpec)
	usage := response.Usage

	return &NormalizeLengthOutput{
		NormalizedContent: normalizedContent,
		FinalCount:        finalCount,
		Applied:           true,
		Mode:              mode,
		Warning:           warning,
		TokenUsage:        &usage,
	}, nil
}

func (l *LengthNormalizerAgent) buildSystemPrompt(mode models.LengthNormalizeMode) string {
	action := "expand"
	if mode == models.NormalizeModeCompress {
		action = "compress"
	}
	return "You are a chapter length normalizer. Perform exactly one-pass " + action + " edit while preserving core facts, hooks, names, and scene order. Output only revised chapter body."
}

func (l *LengthNormalizerAgent) buildUserPrompt(input NormalizeLengthInput, originalCount int, mode models.LengthNormalizeMode) string {
	intentBlock := ""
	if strings.TrimSpace(input.ChapterIntent) != "" {
		intentBlock = "\n## Chapter Intent\n" + input.ChapterIntent + "\n"
	}
	controlBlock := ""
	if strings.TrimSpace(input.ReducedControlBlock) != "" {
		controlBlock = "\n## Reduced Control Block\n" + input.ReducedControlBlock + "\n"
	}

	return fmt.Sprintf(`Please %s this chapter in one pass.
## Length Spec
- Target: %d
- Soft Range: %d-%d
- Hard Range: %d-%d
- Counting Mode: %s

## Current Count
%d

Rules:
- Keep plot facts unchanged
- Keep names, hooks, and key constraints
- Do not add meta explanation
- Output revised chapter only%s%s
## Chapter Body
%s`,
		mode,
		input.LengthSpec.Target,
		input.LengthSpec.SoftMin,
		input.LengthSpec.SoftMax,
		input.LengthSpec.HardMin,
		input.LengthSpec.HardMax,
		input.LengthSpec.CountingMode,
		originalCount,
		intentBlock,
		controlBlock,
		input.ChapterContent,
	)
}

func buildLengthWarning(finalCount int, spec models.LengthSpec) string {
	if !utils.IsOutsideSoftRange(finalCount, spec) {
		return ""
	}
	if utils.IsOutsideHardRange(finalCount, spec) {
		return fmt.Sprintf("final count %d is outside hard range %d-%d after one normalization pass", finalCount, spec.HardMin, spec.HardMax)
	}
	return fmt.Sprintf("final count %d is outside soft range %d-%d after one normalization pass", finalCount, spec.SoftMin, spec.SoftMax)
}

func (l *LengthNormalizerAgent) sanitizeNormalizedContent(rawContent, fallbackContent string) string {
	trimmed := strings.TrimSpace(rawContent)
	if trimmed == "" {
		return fallbackContent
	}

	if block := regexp.MustCompile("(?is)```(?:[a-zA-Z-]+)?\\s*([\\s\\S]*?)\\s*```").FindStringSubmatch(trimmed); len(block) >= 2 {
		if strings.TrimSpace(block[1]) != "" {
			return strings.TrimSpace(block[1])
		}
	}

	stripped, changed := stripCommonWrappers(trimmed)
	if changed {
		if strings.TrimSpace(stripped) == "" {
			return fallbackContent
		}
		if len([]rune(stripped)) < len([]rune(trimmed))/2 {
			return trimmed
		}
		return strings.TrimSpace(stripped)
	}

	return trimmed
}

func stripCommonWrappers(content string) (string, bool) {
	lines := strings.Split(content, "\n")
	kept := make([]string, 0, len(lines))
	removed := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isWrapperLine(trimmed) {
			removed = true
			continue
		}
		kept = append(kept, line)
	}
	if !removed {
		return content, false
	}
	return strings.Join(kept, "\n"), true
}

func isWrapperLine(line string) bool {
	if line == "" {
		return false
	}
	if regexp.MustCompile("^```").MatchString(line) {
		return true
	}
	if regexp.MustCompile(`(?i)^#+\s*(analysis|note|说明|解释|注释)`).MatchString(line) {
		return true
	}
	if regexp.MustCompile(`(?i)^(here(?:'s| is)|below is).*(chapter|content|rewrite|output)`).MatchString(line) {
		return true
	}
	if regexp.MustCompile(`(?i)^i(?:'ll| will)\s+(rewrite|revise|compress|expand|normalize|adjust)`).MatchString(line) {
		return true
	}
	return false
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
