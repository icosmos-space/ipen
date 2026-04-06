package commands

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/icosmos-space/ipen/core/agents"
	"github.com/icosmos-space/ipen/core/models"
	"github.com/icosmos-space/ipen/core/state"
	coreutils "github.com/icosmos-space/ipen/core/utils"
	"github.com/spf13/cobra"
)

type chapterEval struct {
	Number            int     `json:"number"`
	Title             string  `json:"title"`
	WordCount         int     `json:"wordCount"`
	AuditIssueCount   int     `json:"auditIssueCount"`
	AITellCount       int     `json:"aiTellCount"`
	AITellDensity     float64 `json:"aiTellDensity"`
	ParagraphWarnings int     `json:"paragraphWarnings"`
	Status            string  `json:"status"`
}

type qualityTrendPoint struct {
	Chapter int `json:"chapter"`
	Score   int `json:"score"`
}

type bookEval struct {
	BookID               string              `json:"bookId"`
	TotalChapters        int                 `json:"totalChapters"`
	TotalWords           int                 `json:"totalWords"`
	AuditPassRate        int                 `json:"auditPassRate"`
	AvgAITellDensity     float64             `json:"avgAiTellDensity"`
	AvgParagraphWarnings float64             `json:"avgParagraphWarnings"`
	HookResolveRate      int                 `json:"hookResolveRate"`
	DuplicateTitles      int                 `json:"duplicateTitles"`
	QualityScore         int                 `json:"qualityScore"`
	Chapters             []chapterEval       `json:"chapters"`
	QualityTrend         []qualityTrendPoint `json:"qualityTrend"`
}

// EvalCommand 评估书籍写作质量并输出结构化报告。
func EvalCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "eval [book-id]",
		Short: T(TR.CmdEvalShort),
		Long:  T(TR.CmdEvalLong),
		Args:  cobra.MaximumNArgs(1),
		RunE:  runEval,
	}
	cmd.Flags().Bool("json", false, "只输出JSON格式报告")
	cmd.Flags().String("chapters", "", "评估的章节范围，例如：1-10")
	return cmd
}

func runEval(cmd *cobra.Command, args []string) error {
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
	rangeSpec, _ := cmd.Flags().GetString("chapters")

	start, end, err := parseChapterRange(rangeSpec)
	if err != nil {
		return err
	}

	sm := state.NewStateManager(root)
	book, err := sm.LoadBookConfig(bookID)
	if err != nil {
		return err
	}
	index, err := sm.LoadChapterIndex(bookID)
	if err != nil {
		return err
	}
	sort.Slice(index, func(i, j int) bool { return index[i].Number < index[j].Number })

	filtered := make([]models.ChapterMeta, 0, len(index))
	for _, chapter := range index {
		if chapter.Number >= start && chapter.Number <= end {
			filtered = append(filtered, chapter)
		}
	}

	lang := "zh"
	if strings.EqualFold(book.Language, "en") {
		lang = "en"
	}

	evals := make([]chapterEval, 0, len(filtered))
	for _, chapter := range filtered {
		_, content, readErr := readChapterFileForNumber(sm.BookDir(bookID), chapter.Number)
		if readErr != nil {
			content = ""
		}
		aiTells := agents.AnalyzeAITells(content, lang)
		paragraphWarningCount := paragraphWarnings(content)
		density := 0.0
		if len(content) > 0 {
			density = (float64(len(aiTells.Issues)) / float64(len(content))) * 1000
		}
		evals = append(evals, chapterEval{
			Number:            chapter.Number,
			Title:             chapter.Title,
			WordCount:         chapter.WordCount,
			AuditIssueCount:   len(chapter.AuditIssues),
			AITellCount:       len(aiTells.Issues),
			AITellDensity:     math.Round(density*100) / 100,
			ParagraphWarnings: paragraphWarningCount,
			Status:            string(chapter.Status),
		})
	}

	chaptersForAnalytics := make([]coreutils.ChapterAnalytics, 0, len(filtered))
	for _, chapter := range filtered {
		chaptersForAnalytics = append(chaptersForAnalytics, coreutils.ChapterAnalytics{
			Number:      chapter.Number,
			Status:      string(chapter.Status),
			WordCount:   chapter.WordCount,
			AuditIssues: chapter.AuditIssues,
		})
	}
	analytics := coreutils.ComputeAnalytics(bookID, chaptersForAnalytics)

	avgAITellDensity := 0.0
	avgParagraphWarnings := 0.0
	totalWords := 0
	for _, chapter := range evals {
		avgAITellDensity += chapter.AITellDensity
		avgParagraphWarnings += float64(chapter.ParagraphWarnings)
		totalWords += chapter.WordCount
	}
	if len(evals) > 0 {
		avgAITellDensity /= float64(len(evals))
		avgParagraphWarnings /= float64(len(evals))
	}

	hookResolveRate := estimateHookResolveRate(filepath.Join(sm.BookDir(bookID), "story", "pending_hooks.md"))
	duplicateTitles := countDuplicateTitles(index)
	qualityScore := int(math.Round(
		float64(analytics.AuditPassRate)*0.3 +
			math.Max(0, 100-avgAITellDensity*30)*0.25 +
			math.Max(0, 100-avgParagraphWarnings*10)*0.15 +
			float64(hookResolveRate)*0.2 +
			math.Max(0, 100-float64(duplicateTitles*20))*0.1,
	))

	trend := make([]qualityTrendPoint, 0, len(evals))
	for _, chapter := range evals {
		trend = append(trend, qualityTrendPoint{
			Chapter: chapter.Number,
			Score:   computeChapterQualityScore(chapter),
		})
	}

	result := bookEval{
		BookID:               bookID,
		TotalChapters:        len(evals),
		TotalWords:           totalWords,
		AuditPassRate:        analytics.AuditPassRate,
		AvgAITellDensity:     math.Round(avgAITellDensity*100) / 100,
		AvgParagraphWarnings: math.Round(avgParagraphWarnings*100) / 100,
		HookResolveRate:      hookResolveRate,
		DuplicateTitles:      duplicateTitles,
		QualityScore:         qualityScore,
		Chapters:             evals,
		QualityTrend:         trend,
	}

	if asJSON {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Quality report for %q\n\n", bookID)
	fmt.Printf("  Quality score: %d/100\n", qualityScore)
	fmt.Printf("  Chapters: %d\n", result.TotalChapters)
	fmt.Printf("  Words: %d\n\n", result.TotalWords)
	fmt.Println("  Dimensions:")
	fmt.Printf("    Audit pass rate: %d%%\n", result.AuditPassRate)
	fmt.Printf("    AI tell density: %.2f / 1k chars\n", result.AvgAITellDensity)
	fmt.Printf("    Paragraph warnings: %.2f avg/chapter\n", result.AvgParagraphWarnings)
	fmt.Printf("    Hook resolve rate: %d%%\n", result.HookResolveRate)
	fmt.Printf("    Duplicate titles: %d\n", result.DuplicateTitles)
	fmt.Println()
	fmt.Println("  Quality trend:")
	for _, point := range trend {
		barLen := int(math.Round(float64(point.Score) / 5))
		if barLen < 0 {
			barLen = 0
		}
		if barLen > 20 {
			barLen = 20
		}
		bar := strings.Repeat("#", barLen) + strings.Repeat("-", 20-barLen)
		fmt.Printf("    Ch.%03d %s %d\n", point.Chapter, bar, point.Score)
	}
	return nil
}

func paragraphWarnings(content string) int {
	paragraphs := strings.Split(content, "\n\n")
	nonEmpty := make([]string, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		trimmed := strings.TrimSpace(paragraph)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			nonEmpty = append(nonEmpty, trimmed)
		}
	}
	if len(nonEmpty) == 0 {
		return 0
	}
	shortCount := 0
	for _, paragraph := range nonEmpty {
		if len([]rune(paragraph)) < 35 {
			shortCount++
		}
	}
	if float64(shortCount) > float64(len(nonEmpty))*0.4 {
		return 1
	}
	return 0
}

func estimateHookResolveRate(path string) int {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	lines := strings.Split(string(raw), "\n")
	total := 0
	resolved := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") || strings.Contains(trimmed, "---") {
			continue
		}
		lower := strings.ToLower(trimmed)
		if strings.Contains(lower, "hook_id") || strings.Contains(lower, "hook") && strings.Contains(lower, "status") {
			continue
		}
		total++
		if strings.Contains(lower, "resolved") || strings.Contains(lower, "closed") || strings.Contains(lower, "done") {
			resolved++
		}
	}
	if total == 0 {
		return 0
	}
	return int(math.Round(float64(resolved) / float64(total) * 100))
}

func countDuplicateTitles(index []models.ChapterMeta) int {
	seen := map[string]struct{}{}
	dup := 0
	for _, chapter := range index {
		key := strings.ToLower(strings.TrimSpace(chapter.Title))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			dup++
		}
		seen[key] = struct{}{}
	}
	return dup
}

func computeChapterQualityScore(ch chapterEval) int {
	score := 100.0
	score -= float64(ch.AuditIssueCount * 5)
	score -= ch.AITellDensity * 20
	score -= float64(ch.ParagraphWarnings * 3)
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return int(math.Round(score))
}
