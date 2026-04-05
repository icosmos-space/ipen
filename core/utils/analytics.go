package utils

import "math"

// TokenStats 表示token statistics。
type TokenStats struct {
	TotalPromptTokens     int                 `json:"totalPromptTokens"`
	TotalCompletionTokens int                 `json:"totalCompletionTokens"`
	TotalTokens           int                 `json:"totalTokens"`
	AvgTokensPerChapter   int                 `json:"avgTokensPerChapter"`
	RecentTrend           []ChapterTokenTrend `json:"recentTrend"`
}

// ChapterTokenTrend 表示per-chapter token trend。
type ChapterTokenTrend struct {
	Chapter     int `json:"chapter"`
	TotalTokens int `json:"totalTokens"`
}

// AnalyticsData 表示analytics data。
type AnalyticsData struct {
	BookID                 string              `json:"bookId"`
	TotalChapters          int                 `json:"totalChapters"`
	TotalWords             int                 `json:"totalWords"`
	AvgWordsPerChapter     int                 `json:"avgWordsPerChapter"`
	AuditPassRate          int                 `json:"auditPassRate"`
	TopIssueCategories     []IssueCategory     `json:"topIssueCategories"`
	ChaptersWithMostIssues []ChapterIssueCount `json:"chaptersWithMostIssues"`
	StatusDistribution     map[string]int      `json:"statusDistribution"`
	TokenStats             *TokenStats         `json:"tokenStats,omitempty"`
}

// IssueCategory 表示an issue category with count。
type IssueCategory struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

// ChapterIssueCount 表示chapter issue count。
type ChapterIssueCount struct {
	Chapter    int `json:"chapter"`
	IssueCount int `json:"issueCount"`
}

// ChapterAnalytics 表示a chapter for analytics。
type ChapterAnalytics struct {
	Number      int
	Status      string
	WordCount   int
	AuditIssues []string
	TokenUsage  *TokenUsageAnalytics
}

// TokenUsageAnalytics 表示token usage for analytics。
type TokenUsageAnalytics struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// ComputeAnalytics 计算analytics for a book。
func ComputeAnalytics(bookID string, chapters []ChapterAnalytics) AnalyticsData {
	totalChapters := len(chapters)
	totalWords := 0
	for _, ch := range chapters {
		totalWords += ch.WordCount
	}

	avgWordsPerChapter := 0
	if totalChapters > 0 {
		avgWordsPerChapter = int(math.Round(float64(totalWords) / float64(totalChapters)))
	}

	passedStatuses := map[string]bool{
		"ready-for-review": true,
		"approved":         true,
		"published":        true,
	}

	var auditedChapters []ChapterAnalytics
	var passedChapters []ChapterAnalytics
	for _, ch := range chapters {
		if ch.Status != "drafted" && ch.Status != "drafting" && ch.Status != "card-generated" {
			auditedChapters = append(auditedChapters, ch)
			if passedStatuses[ch.Status] {
				passedChapters = append(passedChapters, ch)
			}
		}
	}

	auditPassRate := 100
	if len(auditedChapters) > 0 {
		auditPassRate = int(math.Round(float64(len(passedChapters)) / float64(len(auditedChapters)) * 100))
	}

	// Count issue categories
	categoryCounts := make(map[string]int)
	for _, ch := range chapters {
		for _, issue := range ch.AuditIssues {
			category := extractIssueCategory(issue)
			categoryCounts[category]++
		}
	}

	var topIssueCategories []IssueCategory
	for cat, count := range categoryCounts {
		topIssueCategories = append(topIssueCategories, IssueCategory{Category: cat, Count: count})
	}
	// Sort by count descending
	for i := 0; i < len(topIssueCategories); i++ {
		for j := i + 1; j < len(topIssueCategories); j++ {
			if topIssueCategories[j].Count > topIssueCategories[i].Count {
				topIssueCategories[i], topIssueCategories[j] = topIssueCategories[j], topIssueCategories[i]
			}
		}
	}
	if len(topIssueCategories) > 10 {
		topIssueCategories = topIssueCategories[:10]
	}

	// Chapters with most issues
	var chaptersWithMostIssues []ChapterIssueCount
	for _, ch := range chapters {
		if len(ch.AuditIssues) > 0 {
			chaptersWithMostIssues = append(chaptersWithMostIssues, ChapterIssueCount{
				Chapter:    ch.Number,
				IssueCount: len(ch.AuditIssues),
			})
		}
	}
	// Sort by issue count descending
	for i := 0; i < len(chaptersWithMostIssues); i++ {
		for j := i + 1; j < len(chaptersWithMostIssues); j++ {
			if chaptersWithMostIssues[j].IssueCount > chaptersWithMostIssues[i].IssueCount {
				chaptersWithMostIssues[i], chaptersWithMostIssues[j] = chaptersWithMostIssues[j], chaptersWithMostIssues[i]
			}
		}
	}
	if len(chaptersWithMostIssues) > 5 {
		chaptersWithMostIssues = chaptersWithMostIssues[:5]
	}

	// Status distribution
	statusDistribution := make(map[string]int)
	for _, ch := range chapters {
		statusDistribution[ch.Status]++
	}

	// Token stats
	var tokenStats *TokenStats
	var chaptersWithUsage []ChapterAnalytics
	for _, ch := range chapters {
		if ch.TokenUsage != nil {
			chaptersWithUsage = append(chaptersWithUsage, ch)
		}
	}

	if len(chaptersWithUsage) > 0 {
		totalPromptTokens := 0
		totalCompletionTokens := 0
		totalTokens := 0
		for _, ch := range chaptersWithUsage {
			if ch.TokenUsage != nil {
				totalPromptTokens += ch.TokenUsage.PromptTokens
				totalCompletionTokens += ch.TokenUsage.CompletionTokens
				totalTokens += ch.TokenUsage.TotalTokens
			}
		}

		avgTokensPerChapter := int(math.Round(float64(totalTokens) / float64(len(chaptersWithUsage))))

		var recentTrend []ChapterTokenTrend
		// Get last 5 chapters
		start := len(chaptersWithUsage) - 5
		if start < 0 {
			start = 0
		}
		for _, ch := range chaptersWithUsage[start:] {
			if ch.TokenUsage != nil {
				recentTrend = append(recentTrend, ChapterTokenTrend{
					Chapter:     ch.Number,
					TotalTokens: ch.TokenUsage.TotalTokens,
				})
			}
		}

		tokenStats = &TokenStats{
			TotalPromptTokens:     totalPromptTokens,
			TotalCompletionTokens: totalCompletionTokens,
			TotalTokens:           totalTokens,
			AvgTokensPerChapter:   avgTokensPerChapter,
			RecentTrend:           recentTrend,
		}
	}

	return AnalyticsData{
		BookID:                 bookID,
		TotalChapters:          totalChapters,
		TotalWords:             totalWords,
		AvgWordsPerChapter:     avgWordsPerChapter,
		AuditPassRate:          auditPassRate,
		TopIssueCategories:     topIssueCategories,
		ChaptersWithMostIssues: chaptersWithMostIssues,
		StatusDistribution:     statusDistribution,
		TokenStats:             tokenStats,
	}
}

// extractIssueCategory 提取category from issue string。
func extractIssueCategory(issue string) string {
	// Try to match pattern: [critical|warning|info] category:
	for _, prefix := range []string{"[critical] ", "[warning] ", "[info] "} {
		start := -1
		for i := 0; i <= len(issue)-len(prefix); i++ {
			if issue[i:i+len(prefix)] == prefix {
				start = i + len(prefix)
				break
			}
		}
		if start != -1 {
			// Find the colon (English or Chinese)
			end := start
			for end < len(issue) {
				if issue[end] == ':' {
					break
				}
				// Check for Chinese colon (3 bytes)
				if end+3 <= len(issue) && issue[end:end+3] == "：" {
					break
				}
				end++
			}
			if end > start {
				return issue[start:end]
			}
			return "未分类"
		}
	}
	return "未分类"
}
