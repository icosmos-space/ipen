package utils

import (
	"testing"
)

func TestComputeAnalytics_EmptyChapters(t *testing.T) {
	analytics := ComputeAnalytics("test-book", []ChapterAnalytics{})

	if analytics.BookID != "test-book" {
		t.Errorf("expected book ID 'test-book', got '%s'", analytics.BookID)
	}
	if analytics.TotalChapters != 0 {
		t.Errorf("expected 0 total chapters, got %d", analytics.TotalChapters)
	}
	if analytics.TotalWords != 0 {
		t.Errorf("expected 0 total words, got %d", analytics.TotalWords)
	}
	if analytics.AvgWordsPerChapter != 0 {
		t.Errorf("expected 0 avg words per chapter, got %d", analytics.AvgWordsPerChapter)
	}
	if analytics.AuditPassRate != 100 {
		t.Errorf("expected 100 audit pass rate for empty chapters, got %d", analytics.AuditPassRate)
	}
	if analytics.TokenStats != nil {
		t.Error("expected nil token stats for empty chapters")
	}
}

func TestComputeAnalytics_TotalWords(t *testing.T) {
	chapters := []ChapterAnalytics{
		{Number: 1, WordCount: 3000},
		{Number: 2, WordCount: 3500},
		{Number: 3, WordCount: 2800},
	}

	analytics := ComputeAnalytics("test-book", chapters)

	if analytics.TotalChapters != 3 {
		t.Errorf("expected 3 total chapters, got %d", analytics.TotalChapters)
	}
	if analytics.TotalWords != 9300 {
		t.Errorf("expected 9300 total words, got %d", analytics.TotalWords)
	}
	expectedAvg := 9300 / 3
	if analytics.AvgWordsPerChapter != expectedAvg {
		t.Errorf("expected avg %d words per chapter, got %d", expectedAvg, analytics.AvgWordsPerChapter)
	}
}

func TestComputeAnalytics_AuditPassRate_AllPassed(t *testing.T) {
	chapters := []ChapterAnalytics{
		{Number: 1, Status: "approved"},
		{Number: 2, Status: "published"},
		{Number: 3, Status: "ready-for-review"},
	}

	analytics := ComputeAnalytics("test-book", chapters)

	if analytics.AuditPassRate != 100 {
		t.Errorf("expected 100%% audit pass rate, got %d%%", analytics.AuditPassRate)
	}
}

func TestComputeAnalytics_AuditPassRate_NonePassed(t *testing.T) {
	chapters := []ChapterAnalytics{
		{Number: 1, Status: "audit-failed"},
		{Number: 2, Status: "revising"},
		{Number: 3, Status: "state-degraded"},
	}

	analytics := ComputeAnalytics("test-book", chapters)

	if analytics.AuditPassRate != 0 {
		t.Errorf("expected 0%% audit pass rate, got %d%%", analytics.AuditPassRate)
	}
}

func TestComputeAnalytics_AuditPassRate_Mixed(t *testing.T) {
	chapters := []ChapterAnalytics{
		{Number: 1, Status: "approved"},
		{Number: 2, Status: "audit-failed"},
		{Number: 3, Status: "published"},
		{Number: 4, Status: "revising"},
	}

	analytics := ComputeAnalytics("test-book", chapters)

	// 2 passed out of 4 audited = 50%
	expectedRate := 50
	if analytics.AuditPassRate != expectedRate {
		t.Errorf("expected %d%% audit pass rate, got %d%%", expectedRate, analytics.AuditPassRate)
	}
}

func TestComputeAnalytics_AuditPassRate_ExcludesDrafts(t *testing.T) {
	chapters := []ChapterAnalytics{
		{Number: 1, Status: "drafted"},
		{Number: 2, Status: "drafting"},
		{Number: 3, Status: "card-generated"},
		{Number: 4, Status: "approved"},
	}

	analytics := ComputeAnalytics("test-book", chapters)

	// Only chapter 4 is audited and passed = 100%
	if analytics.AuditPassRate != 100 {
		t.Errorf("expected 100%% audit pass rate (excluding drafts), got %d%%", analytics.AuditPassRate)
	}
}

func TestComputeAnalytics_StatusDistribution(t *testing.T) {
	chapters := []ChapterAnalytics{
		{Number: 1, Status: "approved"},
		{Number: 2, Status: "approved"},
		{Number: 3, Status: "drafted"},
		{Number: 4, Status: "audit-failed"},
	}

	analytics := ComputeAnalytics("test-book", chapters)

	if analytics.StatusDistribution["approved"] != 2 {
		t.Errorf("expected 2 approved, got %d", analytics.StatusDistribution["approved"])
	}
	if analytics.StatusDistribution["drafted"] != 1 {
		t.Errorf("expected 1 drafted, got %d", analytics.StatusDistribution["drafted"])
	}
	if analytics.StatusDistribution["audit-failed"] != 1 {
		t.Errorf("expected 1 audit-failed, got %d", analytics.StatusDistribution["audit-failed"])
	}
}

func TestExtractIssueCategory_Critical(t *testing.T) {
	issue := "[critical] 逻辑矛盾: 主角在两个地方同时出现"
	category := extractIssueCategory(issue)

	if category != "逻辑矛盾" {
		t.Errorf("expected category '逻辑矛盾', got '%s'", category)
	}
}

func TestExtractIssueCategory_Warning(t *testing.T) {
	issue := "[warning] 节奏问题: 情节推进过快"
	category := extractIssueCategory(issue)

	if category != "节奏问题" {
		t.Errorf("expected category '节奏问题', got '%s'", category)
	}
}

func TestExtractIssueCategory_Info(t *testing.T) {
	issue := "[info] 细节补充: 可以增加更多环境描写"
	category := extractIssueCategory(issue)

	if category != "细节补充" {
		t.Errorf("expected category '细节补充', got '%s'", category)
	}
}

func TestExtractIssueCategory_NoCategory(t *testing.T) {
	issue := "这是一个没有分类的问题"
	category := extractIssueCategory(issue)

	if category != "未分类" {
		t.Errorf("expected category '未分类', got '%s'", category)
	}
}

func TestExtractIssueCategory_ChineseColon(t *testing.T) {
	issue := "[critical] 人物设定：主角年龄前后不一致"
	category := extractIssueCategory(issue)

	if category != "人物设定" {
		t.Errorf("expected category '人物设定', got '%s'", category)
	}
}

func TestSortCategories(t *testing.T) {
	chapters := []ChapterAnalytics{
		{
			Number: 1,
			AuditIssues: []string{
				"[critical] 逻辑矛盾: 问题1",
				"[critical] 逻辑矛盾: 问题2",
				"[critical] 逻辑矛盾: 问题3",
				"[critical] 逻辑矛盾: 问题4",
				"[critical] 逻辑矛盾: 问题5",
			},
		},
		{
			Number: 2,
			AuditIssues: []string{
				"[warning] 节奏问题: 问题1",
				"[warning] 节奏问题: 问题2",
				"[warning] 节奏问题: 问题3",
			},
		},
		{
			Number: 3,
			AuditIssues: []string{
				"[info] 人物设定: 问题1",
				"[info] 人物设定: 问题2",
				"[info] 人物设定: 问题3",
				"[info] 人物设定: 问题4",
				"[info] 人物设定: 问题5",
				"[info] 人物设定: 问题6",
				"[info] 人物设定: 问题7",
				"[info] 人物设定: 问题8",
			},
		},
		{
			Number: 4,
			AuditIssues: []string{
				"[info] 细节补充: 问题1",
				"[info] 细节补充: 问题2",
			},
		},
	}

	analytics := ComputeAnalytics("test-book", chapters)

	if len(analytics.TopIssueCategories) != 4 {
		t.Errorf("expected 4 categories, got %d", len(analytics.TopIssueCategories))
	}

	// Should be sorted by count descending
	if analytics.TopIssueCategories[0].Category != "人物设定" || analytics.TopIssueCategories[0].Count != 8 {
		t.Errorf("expected first category '人物设定' with count 8, got '%s' with count %d",
			analytics.TopIssueCategories[0].Category, analytics.TopIssueCategories[0].Count)
	}
	if analytics.TopIssueCategories[1].Category != "逻辑矛盾" || analytics.TopIssueCategories[1].Count != 5 {
		t.Errorf("expected second category '逻辑矛盾' with count 5, got '%s' with count %d",
			analytics.TopIssueCategories[1].Category, analytics.TopIssueCategories[1].Count)
	}
}

func TestGetChaptersWithMostIssues(t *testing.T) {
	chapters := []ChapterAnalytics{
		{Number: 1, AuditIssues: []string{"issue1", "issue2", "issue3"}},
		{Number: 2, AuditIssues: []string{"issue1"}},
		{Number: 3, AuditIssues: []string{"issue1", "issue2", "issue3", "issue4", "issue5"}},
		{Number: 4, AuditIssues: []string{}},
		{Number: 5, AuditIssues: []string{"issue1", "issue2"}},
	}

	analytics := ComputeAnalytics("test-book", chapters)

	if len(analytics.ChaptersWithMostIssues) != 4 {
		t.Errorf("expected 4 chapters with issues, got %d", len(analytics.ChaptersWithMostIssues))
	}

	// Should be sorted by issue count descending
	if analytics.ChaptersWithMostIssues[0].Chapter != 3 || analytics.ChaptersWithMostIssues[0].IssueCount != 5 {
		t.Errorf("expected chapter 3 with 5 issues, got chapter %d with %d issues",
			analytics.ChaptersWithMostIssues[0].Chapter, analytics.ChaptersWithMostIssues[0].IssueCount)
	}
	if analytics.ChaptersWithMostIssues[1].Chapter != 1 || analytics.ChaptersWithMostIssues[1].IssueCount != 3 {
		t.Errorf("expected chapter 1 with 3 issues, got chapter %d with %d issues",
			analytics.ChaptersWithMostIssues[1].Chapter, analytics.ChaptersWithMostIssues[1].IssueCount)
	}
}

func TestGetChaptersWithMostIssues_Limit5(t *testing.T) {
	chapters := []ChapterAnalytics{
		{Number: 1, AuditIssues: []string{"i1", "i2", "i3", "i4", "i5", "i6"}},
		{Number: 2, AuditIssues: []string{"i1", "i2", "i3", "i4", "i5"}},
		{Number: 3, AuditIssues: []string{"i1", "i2", "i3", "i4"}},
		{Number: 4, AuditIssues: []string{"i1", "i2", "i3"}},
		{Number: 5, AuditIssues: []string{"i1", "i2"}},
		{Number: 6, AuditIssues: []string{"i1"}},
	}

	analytics := ComputeAnalytics("test-book", chapters)

	// Should be limited to 5
	if len(analytics.ChaptersWithMostIssues) != 5 {
		t.Errorf("expected 5 chapters max, got %d", len(analytics.ChaptersWithMostIssues))
	}
}

func TestComputeTokenStats_NoUsage(t *testing.T) {
	chapters := []ChapterAnalytics{
		{Number: 1, TokenUsage: nil},
		{Number: 2, TokenUsage: nil},
	}

	analytics := ComputeAnalytics("test-book", chapters)

	if analytics.TokenStats != nil {
		t.Error("expected nil token stats when no usage data")
	}
}

func TestComputeTokenStats_WithUsage(t *testing.T) {
	chapters := []ChapterAnalytics{
		{
			Number: 1,
			TokenUsage: &TokenUsageAnalytics{
				PromptTokens:     1000,
				CompletionTokens: 500,
				TotalTokens:      1500,
			},
		},
		{
			Number: 2,
			TokenUsage: &TokenUsageAnalytics{
				PromptTokens:     1200,
				CompletionTokens: 600,
				TotalTokens:      1800,
			},
		},
		{
			Number: 3,
			TokenUsage: &TokenUsageAnalytics{
				PromptTokens:     800,
				CompletionTokens: 400,
				TotalTokens:      1200,
			},
		},
	}

	analytics := ComputeAnalytics("test-book", chapters)

	if analytics.TokenStats == nil {
		t.Fatal("expected non-nil token stats")
	}

	if analytics.TokenStats.TotalPromptTokens != 3000 {
		t.Errorf("expected 3000 prompt tokens, got %d", analytics.TokenStats.TotalPromptTokens)
	}
	if analytics.TokenStats.TotalCompletionTokens != 1500 {
		t.Errorf("expected 1500 completion tokens, got %d", analytics.TokenStats.TotalCompletionTokens)
	}
	if analytics.TokenStats.TotalTokens != 4500 {
		t.Errorf("expected 4500 total tokens, got %d", analytics.TokenStats.TotalTokens)
	}

	expectedAvg := 4500 / 3
	if analytics.TokenStats.AvgTokensPerChapter != expectedAvg {
		t.Errorf("expected avg %d tokens per chapter, got %d", expectedAvg, analytics.TokenStats.AvgTokensPerChapter)
	}

	if len(analytics.TokenStats.RecentTrend) != 3 {
		t.Errorf("expected 3 trend entries, got %d", len(analytics.TokenStats.RecentTrend))
	}
}

func TestComputeTokenStats_RecentTrendLimit(t *testing.T) {
	chapters := []ChapterAnalytics{
		{Number: 1, TokenUsage: &TokenUsageAnalytics{TotalTokens: 1000}},
		{Number: 2, TokenUsage: &TokenUsageAnalytics{TotalTokens: 1100}},
		{Number: 3, TokenUsage: &TokenUsageAnalytics{TotalTokens: 1200}},
		{Number: 4, TokenUsage: &TokenUsageAnalytics{TotalTokens: 1300}},
		{Number: 5, TokenUsage: &TokenUsageAnalytics{TotalTokens: 1400}},
		{Number: 6, TokenUsage: &TokenUsageAnalytics{TotalTokens: 1500}},
		{Number: 7, TokenUsage: &TokenUsageAnalytics{TotalTokens: 1600}},
	}

	analytics := ComputeAnalytics("test-book", chapters)

	if analytics.TokenStats == nil {
		t.Fatal("expected non-nil token stats")
	}

	// Should only have last 5 chapters
	if len(analytics.TokenStats.RecentTrend) != 5 {
		t.Errorf("expected 5 trend entries, got %d", len(analytics.TokenStats.RecentTrend))
	}

	// Should be chapters 3-7
	if analytics.TokenStats.RecentTrend[0].Chapter != 3 {
		t.Errorf("expected first trend entry chapter 3, got chapter %d", analytics.TokenStats.RecentTrend[0].Chapter)
	}
	if analytics.TokenStats.RecentTrend[4].Chapter != 7 {
		t.Errorf("expected last trend entry chapter 7, got chapter %d", analytics.TokenStats.RecentTrend[4].Chapter)
	}
}

func TestComputeTokenStats_MixedUsage(t *testing.T) {
	chapters := []ChapterAnalytics{
		{Number: 1, TokenUsage: &TokenUsageAnalytics{TotalTokens: 1000}},
		{Number: 2, TokenUsage: nil},
		{Number: 3, TokenUsage: &TokenUsageAnalytics{TotalTokens: 1500}},
		{Number: 4, TokenUsage: nil},
		{Number: 5, TokenUsage: &TokenUsageAnalytics{TotalTokens: 2000}},
	}

	analytics := ComputeAnalytics("test-book", chapters)

	if analytics.TokenStats == nil {
		t.Fatal("expected non-nil token stats")
	}

	// Should only count chapters with usage
	if analytics.TokenStats.TotalTokens != 4500 {
		t.Errorf("expected 4500 total tokens, got %d", analytics.TokenStats.TotalTokens)
	}

	expectedAvg := 4500 / 3
	if analytics.TokenStats.AvgTokensPerChapter != expectedAvg {
		t.Errorf("expected avg %d tokens per chapter, got %d", expectedAvg, analytics.TokenStats.AvgTokensPerChapter)
	}
}

func TestComputeAnalytics_FullIntegration(t *testing.T) {
	chapters := []ChapterAnalytics{
		{
			Number:     1,
			Status:     "approved",
			WordCount:  3000,
			TokenUsage: &TokenUsageAnalytics{PromptTokens: 1000, CompletionTokens: 500, TotalTokens: 1500},
		},
		{
			Number:    2,
			Status:    "audit-failed",
			WordCount: 2800,
			AuditIssues: []string{
				"[critical] 逻辑矛盾: 时间线混乱",
				"[warning] 节奏问题: 情节拖沓",
			},
			TokenUsage: &TokenUsageAnalytics{PromptTokens: 1200, CompletionTokens: 600, TotalTokens: 1800},
		},
		{
			Number:     3,
			Status:     "drafted",
			WordCount:  3200,
			TokenUsage: &TokenUsageAnalytics{PromptTokens: 800, CompletionTokens: 400, TotalTokens: 1200},
		},
		{
			Number:      4,
			Status:      "published",
			WordCount:   3500,
			AuditIssues: []string{"[info] 细节补充: 环境描写不足"},
			TokenUsage:  &TokenUsageAnalytics{PromptTokens: 1100, CompletionTokens: 550, TotalTokens: 1650},
		},
	}

	analytics := ComputeAnalytics("my-book", chapters)

	// Verify basic stats
	if analytics.BookID != "my-book" {
		t.Errorf("expected book ID 'my-book', got '%s'", analytics.BookID)
	}
	if analytics.TotalChapters != 4 {
		t.Errorf("expected 4 total chapters, got %d", analytics.TotalChapters)
	}
	if analytics.TotalWords != 12500 {
		t.Errorf("expected 12500 total words, got %d", analytics.TotalWords)
	}

	// Verify audit pass rate (excluding drafted)
	// 3 audited, 2 passed = 67% (rounded)
	expectedRate := 67
	if analytics.AuditPassRate != expectedRate {
		t.Errorf("expected %d%% audit pass rate, got %d%%", expectedRate, analytics.AuditPassRate)
	}

	// Verify status distribution
	if analytics.StatusDistribution["approved"] != 1 {
		t.Errorf("expected 1 approved, got %d", analytics.StatusDistribution["approved"])
	}
	if analytics.StatusDistribution["audit-failed"] != 1 {
		t.Errorf("expected 1 audit-failed, got %d", analytics.StatusDistribution["audit-failed"])
	}
	if analytics.StatusDistribution["drafted"] != 1 {
		t.Errorf("expected 1 drafted, got %d", analytics.StatusDistribution["drafted"])
	}
	if analytics.StatusDistribution["published"] != 1 {
		t.Errorf("expected 1 published, got %d", analytics.StatusDistribution["published"])
	}

	// Verify issue categories
	if len(analytics.TopIssueCategories) != 3 {
		t.Errorf("expected 3 issue categories, got %d", len(analytics.TopIssueCategories))
	}

	// Verify chapters with most issues
	if len(analytics.ChaptersWithMostIssues) != 2 {
		t.Errorf("expected 2 chapters with issues, got %d", len(analytics.ChaptersWithMostIssues))
	}
	if analytics.ChaptersWithMostIssues[0].Chapter != 2 {
		t.Errorf("expected chapter 2 to have most issues, got chapter %d",
			analytics.ChaptersWithMostIssues[0].Chapter)
	}

	// Verify token stats
	if analytics.TokenStats == nil {
		t.Fatal("expected non-nil token stats")
	}
	if analytics.TokenStats.TotalTokens != 6150 {
		t.Errorf("expected 6150 total tokens, got %d", analytics.TokenStats.TotalTokens)
	}
}

func TestAnalyticsData_Structure(t *testing.T) {
	analytics := AnalyticsData{
		BookID:             "test",
		TotalChapters:      10,
		TotalWords:         30000,
		AvgWordsPerChapter: 3000,
		AuditPassRate:      80,
		TopIssueCategories: []IssueCategory{
			{Category: "逻辑矛盾", Count: 5},
		},
		ChaptersWithMostIssues: []ChapterIssueCount{
			{Chapter: 3, IssueCount: 3},
		},
		StatusDistribution: map[string]int{
			"approved": 8,
			"failed":   2,
		},
		TokenStats: &TokenStats{
			TotalPromptTokens:     10000,
			TotalCompletionTokens: 5000,
			TotalTokens:           15000,
			AvgTokensPerChapter:   1500,
			RecentTrend: []ChapterTokenTrend{
				{Chapter: 8, TotalTokens: 1600},
				{Chapter: 9, TotalTokens: 1500},
				{Chapter: 10, TotalTokens: 1400},
			},
		},
	}

	if analytics.BookID != "test" {
		t.Errorf("expected book ID 'test', got '%s'", analytics.BookID)
	}
	if analytics.TotalChapters != 10 {
		t.Errorf("expected 10 total chapters, got %d", analytics.TotalChapters)
	}
	if analytics.TokenStats.TotalTokens != 15000 {
		t.Errorf("expected 15000 total tokens, got %d", analytics.TokenStats.TotalTokens)
	}
	if len(analytics.TokenStats.RecentTrend) != 3 {
		t.Errorf("expected 3 trend entries, got %d", len(analytics.TokenStats.RecentTrend))
	}
}

func TestTokenUsageAnalytics_Structure(t *testing.T) {
	usage := TokenUsageAnalytics{
		PromptTokens:     1000,
		CompletionTokens: 500,
		TotalTokens:      1500,
	}

	if usage.PromptTokens != 1000 {
		t.Errorf("expected 1000 prompt tokens, got %d", usage.PromptTokens)
	}
	if usage.CompletionTokens != 500 {
		t.Errorf("expected 500 completion tokens, got %d", usage.CompletionTokens)
	}
	if usage.TotalTokens != 1500 {
		t.Errorf("expected 1500 total tokens, got %d", usage.TotalTokens)
	}
}
