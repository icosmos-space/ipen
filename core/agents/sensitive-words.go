package agents

import (
	"fmt"
	"regexp"
	"strings"
)

// SensitiveWordMatch records one matched sensitive term.
type SensitiveWordMatch struct {
	Word     string `json:"word"`
	Count    int    `json:"count"`
	Severity string `json:"severity"` // block | warn
}

// SensitiveWordResult bundles issues + matched words.
type SensitiveWordResult struct {
	Issues []AuditIssue         `json:"issues"`
	Found  []SensitiveWordMatch `json:"found"`
}

type wordListEntry struct {
	Words        []string
	Severity     string
	Label        string
	EnglishLabel string
}

var politicalWords = []string{
	"习近平", "习主席", "习总书记", "共产党", "中国共产党", "共青团",
	"六四", "天安门事件", "天安门广场事件", "法轮功", "法轮大法",
	"台独", "藏独", "疆独", "港独",
	"新疆集中营", "再教育营",
	"维吾尔", "达赖喇嘛", "达赖", "刘晓波", "艾未未", "赵紫阳",
	"文化大革命", "文革", "大跃进",
	"反右运动", "镇压", "六四屠杀",
	"中南海", "政治局常委",
	"翻墙", "防火长城",
}

var sexualWords = []string{
	"性交", "做爱", "口交", "肛交", "自慰", "手淫",
	"阴茎", "阴道", "阴蒂", "乳房", "乳头",
	"射精", "高潮", "潮吹",
	"淫秽", "淫荡", "淫乱", "荡妇", "婊子",
	"强奸", "轮奸",
}

var violenceExtremeWords = []string{
	"肢解", "碎尸", "挖眼", "剥皮", "开膛破肚", "血腥",
	"虐杀", "凌迟", "活剐", "活埋", "烹煮活人",
}

var wordLists = []wordListEntry{
	{Words: politicalWords, Severity: "block", Label: "政治敏感词", EnglishLabel: "political sensitive terms"},
	{Words: sexualWords, Severity: "warn", Label: "色情敏感词", EnglishLabel: "sexual sensitive terms"},
	{Words: violenceExtremeWords, Severity: "warn", Label: "极端暴力词", EnglishLabel: "extreme violence terms"},
}

// AnalyzeSensitiveWords 扫描chapter content for sensitive terms。
func AnalyzeSensitiveWords(content string, customWords []string, language string) SensitiveWordResult {
	isEnglish := strings.EqualFold(language, "en")
	found := []SensitiveWordMatch{}
	issues := []AuditIssue{}

	for _, list := range wordLists {
		matches := scanWords(content, list.Words, list.Severity)
		if len(matches) == 0 {
			continue
		}

		found = append(found, matches...)
		wordSummary := summarizeMatches(matches, isEnglish)

		issue := AuditIssue{Severity: "warning", Category: "Sensitive terms"}
		if list.Severity == "block" {
			issue.Severity = "critical"
		}

		if isEnglish {
			issue.Description = "Detected " + list.EnglishLabel + ": " + wordSummary
			if list.Severity == "block" {
				issue.Suggestion = "You must remove or replace these blocked terms before publication"
			} else {
				issue.Suggestion = "Replace or soften these terms to reduce moderation risk"
			}
		} else {
			issue.Category = "敏感词"
			issue.Description = "检测到" + list.Label + "：" + wordSummary
			if list.Severity == "block" {
				issue.Suggestion = "必须删除或替换这些词，否则无法发布"
			} else {
				issue.Suggestion = "建议替换或弱化这些词，降低审核风险"
			}
		}
		issues = append(issues, issue)
	}

	if len(customWords) > 0 {
		matches := scanWords(content, customWords, "warn")
		if len(matches) > 0 {
			found = append(found, matches...)
			wordSummary := summarizeMatches(matches, isEnglish)
			issue := AuditIssue{Severity: "warning", Category: "Sensitive terms"}
			if isEnglish {
				issue.Description = "Detected custom sensitive term(s): " + wordSummary
				issue.Suggestion = "Replace or remove these terms according to project rules"
			} else {
				issue.Category = "敏感词"
				issue.Description = "检测到自定义敏感词：" + wordSummary
				issue.Suggestion = "根据项目规则替换或删除这些词"
			}
			issues = append(issues, issue)
		}
	}

	return SensitiveWordResult{Issues: issues, Found: found}
}

func scanWords(content string, words []string, severity string) []SensitiveWordMatch {
	result := []SensitiveWordMatch{}
	for _, word := range words {
		trimmed := strings.TrimSpace(word)
		if trimmed == "" {
			continue
		}
		re := regexp.MustCompile(regexp.QuoteMeta(trimmed))
		hits := re.FindAllStringIndex(content, -1)
		if len(hits) == 0 {
			continue
		}
		result = append(result, SensitiveWordMatch{Word: trimmed, Count: len(hits), Severity: severity})
	}
	return result
}

func summarizeMatches(matches []SensitiveWordMatch, english bool) string {
	parts := make([]string, 0, len(matches))
	for _, match := range matches {
		parts = append(parts, fmt.Sprintf("\"%s\"x%d", match.Word, match.Count))
	}
	if english {
		return strings.Join(parts, ", ")
	}
	return strings.Join(parts, "、")
}
