package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/icosmos-space/ipen/core/agents"
	"github.com/icosmos-space/ipen/core/pipeline"
	"github.com/icosmos-space/ipen/core/state"
	"github.com/spf13/cobra"
)

// DetectCommand 对章节执行 AIGC 检测。
func DetectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "detect [book-id] [chapter]",
		Short: T(TR.CmdDetectShort),
		Long:  T(TR.CmdDetectLong),
		Args:  cobra.MaximumNArgs(2),
		RunE:  runDetect,
	}

	cmd.Flags().Int("chapter", 0, "Chapter number")
	cmd.Flags().Bool("all", false, "Detect all chapters")
	cmd.Flags().Bool("stats", false, "Show detection statistics")
	cmd.Flags().Bool("json", false, "Output JSON")
	return cmd
}

func runDetect(cmd *cobra.Command, args []string) error {
	root, err := findProjectRoot()
	if err != nil {
		return err
	}

	config, err := loadConfig(root)
	if err != nil {
		return err
	}
	if config.Detection == nil || !config.Detection.Enabled {
		return fmt.Errorf("AIGC detection is not enabled. configure `detection.enabled=true` in ipen.json")
	}

	bookArg := ""
	var chapterFromArgs *int
	if len(args) > 0 {
		if value, err := strconv.Atoi(args[0]); err == nil {
			chapterFromArgs = &value
		} else {
			bookArg = args[0]
		}
	}
	if len(args) > 1 {
		value, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("invalid chapter number %q", args[1])
		}
		chapterFromArgs = &value
	}

	bookID, err := resolveBookID(root, bookArg)
	if err != nil {
		return err
	}

	asJSON, _ := cmd.Flags().GetBool("json")
	showAll, _ := cmd.Flags().GetBool("all")
	showStats, _ := cmd.Flags().GetBool("stats")
	chapterFlag, _ := cmd.Flags().GetInt("chapter")

	sm := state.NewStateManager(root)
	bookDir := sm.BookDir(bookID)

	if showStats {
		history, err := pipeline.LoadDetectionHistory(bookDir)
		if err != nil {
			return err
		}
		converted := make([]agents.DetectionHistoryEntry, 0, len(history))
		for _, item := range history {
			converted = append(converted, agents.DetectionHistoryEntry{
				ChapterNumber: item.ChapterNumber,
				Action:        item.Action,
				Attempt:       item.Attempt,
				Score:         item.Score,
				Timestamp:     item.Timestamp,
			})
		}
		stats := agents.AnalyzeDetectionInsights(converted)

		if asJSON {
			data, _ := json.MarshalIndent(stats, "", "  ")
			fmt.Println(string(data))
			return nil
		}

		fmt.Println("Detection statistics:")
		fmt.Printf("  Total detections: %d\n", stats.TotalDetections)
		fmt.Printf("  Total rewrites: %d\n", stats.TotalRewrites)
		fmt.Printf("  Avg original score: %.3f\n", stats.AvgOriginalScore)
		fmt.Printf("  Avg final score: %.3f\n", stats.AvgFinalScore)
		fmt.Printf("  Avg score reduction: %.3f\n", stats.AvgScoreReduction)
		fmt.Printf("  Pass rate: %.0f%%\n", stats.PassRate*100)
		for _, chapter := range stats.ChapterBreakdown {
			fmt.Printf("  Ch.%d: %.3f -> %.3f (%d rewrites)\n",
				chapter.ChapterNumber,
				chapter.OriginalScore,
				chapter.FinalScore,
				chapter.RewriteAttempts,
			)
		}
		return nil
	}

	if showAll {
		index, err := sm.LoadChapterIndex(bookID)
		if err != nil {
			return err
		}
		results := make([]*pipeline.DetectChapterResult, 0, len(index))
		for _, chapter := range index {
			content, err := readChapterContent(bookDir, chapter.Number)
			if err != nil {
				return err
			}
			result, err := pipeline.DetectChapter(context.Background(), *config.Detection, content, chapter.Number)
			if err != nil {
				return err
			}
			results = append(results, result)
		}
		if asJSON {
			data, _ := json.MarshalIndent(results, "", "  ")
			fmt.Println(string(data))
			return nil
		}
		for _, result := range results {
			printDetectionResult(result, false)
		}
		return nil
	}

	targetChapter := 0
	if chapterFlag > 0 {
		targetChapter = chapterFlag
	} else if chapterFromArgs != nil {
		targetChapter = *chapterFromArgs
	} else {
		next, err := sm.GetNextChapterNumber(bookID)
		if err != nil {
			return err
		}
		targetChapter = next - 1
	}
	if targetChapter < 1 {
		return fmt.Errorf("no chapters to detect")
	}

	content, err := readChapterContent(bookDir, targetChapter)
	if err != nil {
		return err
	}
	result, err := pipeline.DetectChapter(context.Background(), *config.Detection, content, targetChapter)
	if err != nil {
		return err
	}
	if asJSON {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}
	printDetectionResult(result, false)
	return nil
}

func printDetectionResult(result *pipeline.DetectChapterResult, asJSON bool) {
	if asJSON {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return
	}

	icon := "FAIL"
	if result.Passed {
		icon = "PASS"
	}
	fmt.Printf("Chapter %d: score=%.3f (%s) %s\n",
		result.ChapterNumber,
		result.Detection.Score,
		result.Detection.Provider,
		icon,
	)
}

func readChapterContent(bookDir string, chapterNumber int) (string, error) {
	chaptersDir := filepath.Join(bookDir, "chapters")
	entries, err := os.ReadDir(chaptersDir)
	if err != nil {
		return "", err
	}

	prefix := fmt.Sprintf("%04d_", chapterNumber)
	targetFile := ""
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, ".md") {
			targetFile = filepath.Join(chaptersDir, name)
			break
		}
	}
	if targetFile == "" {
		return "", fmt.Errorf("chapter %d file not found", chapterNumber)
	}

	content, err := os.ReadFile(targetFile)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(content), "\n")
	start := 0
	for i, line := range lines {
		if i > 0 && strings.TrimSpace(line) != "" {
			start = i
			break
		}
	}
	return strings.Join(lines[start:], "\n"), nil
}
