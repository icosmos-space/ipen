package commands

import (
	"encoding/json"
	"fmt"

	"github.com/icosmos-space/ipen/core/state"
	coreutils "github.com/icosmos-space/ipen/core/utils"
	"github.com/spf13/cobra"
)

// AnalyticsCommand 显示书籍的数据分析与 Token 使用统计。
func AnalyticsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "analytics [book-id]",
		Aliases: []string{"stats"},
		Short:   T(TR.CmdAnalyticsShort),
		Long:    T(TR.CmdAnalyticsShort),
		Args:    cobra.MaximumNArgs(1),
		RunE:    runAnalytics,
	}
	cmd.Flags().Bool("json", false, "Output JSON")
	return cmd
}

func runAnalytics(cmd *cobra.Command, args []string) error {
	root, err := findProjectRoot()
	if err != nil {
		return err
	}
	bookArg := ""
	if len(args) > 0 {
		bookArg = args[0]
	}
	bookID, err := resolveBookID(root, bookArg)
	if err != nil {
		return err
	}

	asJSON, _ := cmd.Flags().GetBool("json")
	sm := state.NewStateManager(root)
	index, err := sm.LoadChapterIndex(bookID)
	if err != nil {
		return err
	}

	chapters := make([]coreutils.ChapterAnalytics, 0, len(index))
	for _, chapter := range index {
		item := coreutils.ChapterAnalytics{
			Number:      chapter.Number,
			Status:      string(chapter.Status),
			WordCount:   chapter.WordCount,
			AuditIssues: chapter.AuditIssues,
		}
		if chapter.TokenUsage != nil {
			item.TokenUsage = &coreutils.TokenUsageAnalytics{
				PromptTokens:     chapter.TokenUsage.PromptTokens,
				CompletionTokens: chapter.TokenUsage.CompletionTokens,
				TotalTokens:      chapter.TokenUsage.TotalTokens,
			}
		}
		chapters = append(chapters, item)
	}

	analytics := coreutils.ComputeAnalytics(bookID, chapters)
	if asJSON {
		data, _ := json.MarshalIndent(analytics, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Analytics for %q\n\n", bookID)
	fmt.Printf("  Total chapters: %d\n", analytics.TotalChapters)
	fmt.Printf("  Total words: %d\n", analytics.TotalWords)
	fmt.Printf("  Avg words/chapter: %d\n", analytics.AvgWordsPerChapter)
	fmt.Printf("  Audit pass rate: %d%%\n\n", analytics.AuditPassRate)

	if len(analytics.StatusDistribution) > 0 {
		fmt.Println("  Status distribution:")
		for status, count := range analytics.StatusDistribution {
			fmt.Printf("    %s: %d\n", status, count)
		}
		fmt.Println()
	}

	if analytics.TokenStats != nil {
		fmt.Println("  Token usage:")
		fmt.Printf("    Total tokens: %d\n", analytics.TokenStats.TotalTokens)
		fmt.Printf("    Prompt tokens: %d\n", analytics.TokenStats.TotalPromptTokens)
		fmt.Printf("    Completion tokens: %d\n", analytics.TokenStats.TotalCompletionTokens)
		fmt.Printf("    Avg tokens/chapter: %d\n", analytics.TokenStats.AvgTokensPerChapter)
		if len(analytics.TokenStats.RecentTrend) > 0 {
			fmt.Println("    Recent trend:")
			for _, trend := range analytics.TokenStats.RecentTrend {
				fmt.Printf("      Ch.%d: %d tokens\n", trend.Chapter, trend.TotalTokens)
			}
		}
		fmt.Println()
	}

	if len(analytics.TopIssueCategories) > 0 {
		fmt.Println("  Top issue categories:")
		for _, item := range analytics.TopIssueCategories {
			fmt.Printf("    %s: %d\n", item.Category, item.Count)
		}
		fmt.Println()
	}

	if len(analytics.ChaptersWithMostIssues) > 0 {
		fmt.Println("  Chapters with most issues:")
		for _, item := range analytics.ChaptersWithMostIssues {
			fmt.Printf("    Ch.%d: %d issues\n", item.Chapter, item.IssueCount)
		}
	}

	return nil
}
