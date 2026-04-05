package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/icosmos-space/ipen/core/llm"
	"github.com/icosmos-space/ipen/core/models"
)

// AuditResult 表示audit result。
type AuditResult struct {
	Passed     bool               `json:"passed"`
	Issues     []AuditIssue       `json:"issues"`
	Summary    string             `json:"summary"`
	TokenUsage *models.TokenUsage `json:"tokenUsage,omitempty"`
}

// AuditIssue 表示an audit issue。
type AuditIssue struct {
	Severity    string `json:"severity"` // "critical", "warning", "info"
	Category    string `json:"category"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
}

// ContinuityAuditor 表示the continuity auditor。
type ContinuityAuditor struct {
	*BaseAgent
}

// NewContinuityAuditor 创建新的continuity auditor。
func NewContinuityAuditor(ctx AgentContext) *ContinuityAuditor {
	return &ContinuityAuditor{
		BaseAgent: NewBaseAgent(ctx),
	}
}

// Name 返回the agent name。
func (c *ContinuityAuditor) Name() string {
	return "continuity-auditor"
}

// AuditChapter audits a chapter for continuity
func (c *ContinuityAuditor) AuditChapter(
	ctx context.Context,
	chapterNumber int,
	content string,
	currentState, storyBible, chapterSummaries string,
) (*AuditResult, error) {
	systemPrompt := c.buildAuditSystemPrompt()
	userPrompt := c.buildAuditUserPrompt(chapterNumber, content, currentState, storyBible, chapterSummaries)

	messages := []llm.LLMMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	response, err := c.Chat(ctx, messages, &llm.ChatOptions{
		Temperature: 0.3,
		MaxTokens:   4096,
	})
	if err != nil {
		return nil, fmt.Errorf("audit chat failed: %w", err)
	}

	result := c.parseAuditResult(response.Content)
	usage := response.Usage
	result.TokenUsage = &usage
	return result, nil
}

func (c *ContinuityAuditor) buildAuditSystemPrompt() string {
	return `你是严格的小说连续性审核员。
请检查章节是否存在以下问题：
1. 人物设定一致性（OOC）
2. 时间线合理性
3. 设定冲突
4. 伏笔推进与回收
5. 信息边界越界
6. 情节连贯性

输出必须是JSON格式：
{
  "passed": true,
  "issues": [
    {
      "severity": "critical|warning|info",
      "category": "问题分类",
      "description": "问题描述",
      "suggestion": "修改建议"
    }
  ],
  "summary": "简短总结"
}`
}

func (c *ContinuityAuditor) buildAuditUserPrompt(
	chapterNumber int,
	content, currentState, storyBible, chapterSummaries string,
) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## 审核第%d章\n", chapterNumber))

	if strings.TrimSpace(currentState) != "" {
		sb.WriteString("### 当前状态\n")
		sb.WriteString(currentState)
		sb.WriteString("\n\n")
	}

	if strings.TrimSpace(storyBible) != "" {
		sb.WriteString("### 故事圣经\n")
		sb.WriteString(truncateRunes(storyBible, 2000))
		sb.WriteString("\n\n")
	}

	if strings.TrimSpace(chapterSummaries) != "" {
		sb.WriteString("### 章节摘要（节选）\n")
		sb.WriteString(truncateRunes(chapterSummaries, 2000))
		sb.WriteString("\n\n")
	}

	sb.WriteString("### 章节正文\n")
	sb.WriteString(content)

	return sb.String()
}

func (c *ContinuityAuditor) parseAuditResult(content string) *AuditResult {
	result := &AuditResult{
		Passed:  true,
		Issues:  []AuditIssue{},
		Summary: "审核完成。",
	}

	jsonBlock := extractFirstJSONObject(content)
	if jsonBlock != "" {
		var parsed AuditResult
		if err := json.Unmarshal([]byte(jsonBlock), &parsed); err == nil {
			if parsed.Issues == nil {
				parsed.Issues = []AuditIssue{}
			}
			if strings.TrimSpace(parsed.Summary) == "" {
				parsed.Summary = "审核完成。"
			}
			for i := range parsed.Issues {
				severity := strings.ToLower(strings.TrimSpace(parsed.Issues[i].Severity))
				switch severity {
				case "critical", "warning", "info":
					parsed.Issues[i].Severity = severity
				default:
					parsed.Issues[i].Severity = "warning"
				}
			}
			return &parsed
		}
	}

	trimmed := strings.TrimSpace(content)
	if trimmed != "" {
		result.Summary = trimmed
	}

	lower := strings.ToLower(content)
	if strings.Contains(lower, "critical") || strings.Contains(lower, "\"passed\": false") {
		result.Passed = false
	}

	return result
}
