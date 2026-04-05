package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/icosmos-space/ipen/core/llm"
)

// ValidationWarning 表示a state validation warning。
type ValidationWarning struct {
	Category    string `json:"category"`
	Description string `json:"description"`
}

// ValidationResult 表示state validation result。
type ValidationResult struct {
	Warnings []ValidationWarning `json:"warnings"`
	Passed   bool                `json:"passed"`
}

// StateValidatorAgent 校验truth-file delta consistency against chapter body。
type StateValidatorAgent struct {
	*BaseAgent
}

// NewStateValidatorAgent 创建新的state validator agent。
func NewStateValidatorAgent(ctx AgentContext) *StateValidatorAgent {
	return &StateValidatorAgent{BaseAgent: NewBaseAgent(ctx)}
}

// Name 返回agent name。
func (s *StateValidatorAgent) Name() string {
	return "state-validator"
}

// Validate 校验state/hook transitions against chapter content。
func (s *StateValidatorAgent) Validate(
	ctx context.Context,
	chapterContent string,
	chapterNumber int,
	oldState string,
	newState string,
	oldHooks string,
	newHooks string,
	language string,
) (*ValidationResult, error) {
	stateDiff := s.computeDiff(oldState, newState, "State Card")
	hooksDiff := s.computeDiff(oldHooks, newHooks, "Hooks Pool")

	if strings.TrimSpace(stateDiff) == "" && strings.TrimSpace(hooksDiff) == "" {
		return &ValidationResult{Warnings: []ValidationWarning{}, Passed: true}, nil
	}

	langInstruction := "Respond in Chinese."
	if strings.EqualFold(language, "en") {
		langInstruction = "Respond in English."
	}

	systemPrompt := `You are a continuity validator for a novel writing system. ` + langInstruction + `

Given chapter text and changes to truth files (state card + hooks pool), check:
1. Unsupported state change
2. Missing state change
3. Temporal impossibility
4. Hook anomaly
5. Retroactive edit

Output strict JSON:
{
  "warnings": [
    {"category": "missing_state_change", "description": "..."}
  ],
  "passed": true
}

Set passed=false only for hard contradictions directly conflicting with chapter text.`

	userPrompt := fmt.Sprintf(`Chapter %d validation:

## State Card Changes
%s

## Hooks Pool Changes
%s

## Chapter Text (truncated)
%s`,
		chapterNumber,
		emptyIfBlank(stateDiff, "(no changes)"),
		emptyIfBlank(hooksDiff, "(no changes)"),
		truncate(chapterContent, 6000),
	)

	response, err := s.Chat(ctx, []llm.LLMMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}, &llm.ChatOptions{Temperature: 0.1, MaxTokens: 2048})
	if err != nil {
		s.Log().Warn("state validation failed", map[string]any{"error": err.Error()})
		return nil, err
	}

	result, parseErr := s.parseResult(response.Content)
	if parseErr != nil {
		return nil, parseErr
	}
	return result, nil
}

func (s *StateValidatorAgent) computeDiff(oldText, newText, label string) string {
	if oldText == newText {
		return ""
	}

	oldLines := normalizeLines(oldText)
	newLines := normalizeLines(newText)

	added := []string{}
	removed := []string{}
	oldSet := map[string]struct{}{}
	newSet := map[string]struct{}{}
	for _, line := range oldLines {
		oldSet[line] = struct{}{}
	}
	for _, line := range newLines {
		newSet[line] = struct{}{}
	}
	for _, line := range newLines {
		if _, ok := oldSet[line]; !ok {
			added = append(added, line)
		}
	}
	for _, line := range oldLines {
		if _, ok := newSet[line]; !ok {
			removed = append(removed, line)
		}
	}
	if len(added) == 0 && len(removed) == 0 {
		return ""
	}

	parts := []string{"### " + label}
	if len(removed) > 0 {
		parts = append(parts, "Removed:\n- "+strings.Join(removed, "\n- "))
	}
	if len(added) > 0 {
		parts = append(parts, "Added:\n+ "+strings.Join(added, "\n+ "))
	}
	return strings.Join(parts, "\n")
}

func (s *StateValidatorAgent) parseResult(content string) (*ValidationResult, error) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil, fmt.Errorf("llm returned empty response")
	}

	parsed := extractFirstValidJSONObject(trimmed)
	if parsed == nil {
		return nil, fmt.Errorf("state validator returned invalid json")
	}

	warningsRaw, _ := (*parsed)["warnings"].([]any)
	warnings := make([]ValidationWarning, 0, len(warningsRaw))
	for _, item := range warningsRaw {
		row, _ := item.(map[string]any)
		warnings = append(warnings, ValidationWarning{
			Category:    asString(row["category"], "unknown"),
			Description: asString(row["description"], ""),
		})
	}

	passed, ok := (*parsed)["passed"].(bool)
	if !ok {
		return nil, fmt.Errorf("state validator missing boolean field 'passed'")
	}

	return &ValidationResult{Warnings: warnings, Passed: passed}, nil
}

func extractFirstValidJSONObject(text string) *map[string]any {
	if parsed := tryParseObject(text); parsed != nil {
		return parsed
	}

	for i := 0; i < len(text); i++ {
		if text[i] != '{' {
			continue
		}
		candidate := extractBalancedJSONObject(text, i)
		if candidate == "" {
			continue
		}
		if parsed := tryParseObject(candidate); parsed != nil {
			return parsed
		}
	}
	return nil
}

func tryParseObject(text string) *map[string]any {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return nil
	}
	return &parsed
}

func extractBalancedJSONObject(text string, start int) string {
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(text); i++ {
		ch := text[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}
		if ch == '"' {
			inString = true
			continue
		}
		if ch == '{' {
			depth++
			continue
		}
		if ch == '}' {
			depth--
			if depth == 0 {
				return text[start : i+1]
			}
			if depth < 0 {
				return ""
			}
		}
	}
	return ""
}

func normalizeLines(text string) []string {
	lines := strings.Split(text, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func truncate(text string, maxChars int) string {
	runes := []rune(text)
	if len(runes) <= maxChars {
		return text
	}
	return string(runes[:maxChars])
}

func emptyIfBlank(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func asString(value any, fallback string) string {
	if s, ok := value.(string); ok {
		return s
	}
	return fallback
}
