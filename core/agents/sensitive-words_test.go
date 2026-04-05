package agents

import (
	"strings"
	"testing"
)

func TestAnalyzeSensitiveWords_CleanText(t *testing.T) {
	result := AnalyzeSensitiveWords("陈风握紧长剑，准备迎战。", nil, "zh")
	if len(result.Issues) != 0 || len(result.Found) != 0 {
		t.Fatalf("expected no issues for clean text, got %#v", result)
	}
}

func TestAnalyzeSensitiveWords_PoliticalBlockSeverity(t *testing.T) {
	result := AnalyzeSensitiveWords("他在广场上看到了法轮功的标语。", nil, "zh")
	if len(result.Found) == 0 {
		t.Fatalf("expected sensitive words found")
	}

	foundBlock := false
	for _, item := range result.Found {
		if item.Word == "法轮功" && item.Severity == "block" {
			foundBlock = true
			break
		}
	}
	if !foundBlock {
		t.Fatalf("expected 法轮功 block match, got %#v", result.Found)
	}

	criticalIssue := false
	for _, issue := range result.Issues {
		if issue.Severity == "critical" && issue.Category == "敏感词" {
			criticalIssue = true
			break
		}
	}
	if !criticalIssue {
		t.Fatalf("expected critical issue, got %#v", result.Issues)
	}
}

func TestAnalyzeSensitiveWords_WarnAndCustom(t *testing.T) {
	warn := AnalyzeSensitiveWords("他看到了淫秽画面，现场血腥而且有人被肢解。", nil, "zh")
	warnCount := 0
	for _, item := range warn.Found {
		if item.Severity == "warn" {
			warnCount++
		}
	}
	if warnCount == 0 {
		t.Fatalf("expected warn severity matches, got %#v", warn.Found)
	}

	custom := AnalyzeSensitiveWords("他使用了禁术灭世天火。", []string{"灭世天火", "灭世之力"}, "zh")
	if len(custom.Found) != 1 || custom.Found[0].Word != "灭世天火" || custom.Found[0].Severity != "warn" {
		t.Fatalf("unexpected custom match: %#v", custom.Found)
	}
}

func TestAnalyzeSensitiveWords_CountAndSubstringBehavior(t *testing.T) {
	multi := AnalyzeSensitiveWords("共产党的历史很长，共产党的影响很大，共产党的组织遍布各地。", nil, "zh")
	count := -1
	for _, item := range multi.Found {
		if item.Word == "共产党" {
			count = item.Count
			break
		}
	}
	if count != 3 {
		t.Fatalf("expected 共产党 count=3, got %d (%#v)", count, multi.Found)
	}

	onlyXinjiang := AnalyzeSensitiveWords("他来自新疆，是一名普通的牧民。", nil, "zh")
	for _, item := range onlyXinjiang.Found {
		if item.Word == "新疆集中营" {
			t.Fatalf("did not expect 新疆集中营 match in 新疆 context")
		}
	}

	partial := AnalyzeSensitiveWords("这是一个全新的疆域，充满未知。", nil, "zh")
	if len(partial.Found) != 0 {
		t.Fatalf("expected no false positive, got %#v", partial.Found)
	}
}

func TestAnalyzeSensitiveWords_EnglishLocalizationForCustomWords(t *testing.T) {
	result := AnalyzeSensitiveWords("He saw Falun Gong slogans on the wall and paused.", []string{"Falun Gong"}, "en")
	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %#v", result.Issues)
	}
	if result.Issues[0].Category != "Sensitive terms" {
		t.Fatalf("expected english category, got %#v", result.Issues[0])
	}
	if !strings.Contains(result.Issues[0].Description, "custom sensitive term") {
		t.Fatalf("expected custom term description, got %q", result.Issues[0].Description)
	}
	if !strings.Contains(result.Issues[0].Suggestion, "Replace or remove") {
		t.Fatalf("expected replace/remove suggestion, got %q", result.Issues[0].Suggestion)
	}
}
