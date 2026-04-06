package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/icosmos-space/ipen/core/llm"
	"github.com/icosmos-space/ipen/core/models"
	"github.com/icosmos-space/ipen/core/state"
	"github.com/spf13/cobra"
)

// WriteCommand 负责章节写作与状态修复。
func WriteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "write",
		Short: T(TR.CmdWriteShort),
		Long:  T(TR.CmdWriteLong),
	}

	cmd.AddCommand(writeNextCommand())
	cmd.AddCommand(writeRewriteCommand())
	cmd.AddCommand(writeRepairStateCommand())

	return cmd
}

func writeNextCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "next [book-id]",
		Short: T(TR.CmdWriteNextShort),
		Long:  T(TR.CmdWriteNextLong),
		Args:  cobra.MaximumNArgs(1),
		RunE:  runWriteNext,
	}

	cmd.Flags().Int("count", 1, "Number of chapters to write")
	cmd.Flags().String("words", "", "Words per chapter override")
	cmd.Flags().String("context", "", "Creative guidance text")
	cmd.Flags().String("context-file", "", "Read guidance from file")
	cmd.Flags().Bool("json", false, "Output JSON")
	cmd.Flags().BoolP("quiet", "q", false, "Suppress progress output")
	return cmd
}

func runWriteNext(cmd *cobra.Command, args []string) error {
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

	count, _ := cmd.Flags().GetInt("count")
	if count <= 0 {
		return fmt.Errorf("--count must be > 0")
	}

	wordOverride, err := parseOptionalIntFlag(cmd, "words")
	if err != nil {
		return err
	}
	contextText, err := resolveContextInput(cmd)
	if err != nil {
		return err
	}

	asJSON, _ := cmd.Flags().GetBool("json")
	quiet, _ := cmd.Flags().GetBool("quiet")

	config, err := loadConfig(root)
	if err != nil {
		return err
	}

	streamOutput := newCLIStreamOutput(!asJSON && !quiet, os.Stdout, "writer")
	runner := buildRunner(config, root, quiet, streamOutput.Callback(), nil)
	sm := state.NewStateManager(root)
	results := make([]map[string]any, 0, count)

	if contextText != "" && !asJSON && !quiet {
		fmt.Println("[note] external context is accepted but not yet wired into Go pipeline write input.")
	}
	if wordOverride != nil && !asJSON && !quiet {
		fmt.Println("[note] --words override is accepted but not yet wired into Go pipeline write input.")
	}
	if hint := getLegacyMigrationHint(root, bookID); hint != "" && !asJSON && !quiet {
		fmt.Printf("[migration] %s\n", hint)
	}

	for i := 0; i < count; i++ {
		chapterNumber, err := sm.GetNextChapterNumber(bookID)
		if err != nil {
			return err
		}

		if !asJSON && !quiet {
			fmt.Printf("Writing chapter %d for %s (%d/%d)\n", chapterNumber, bookID, i+1, count)
		}

		releaseLock, err := sm.AcquireBookLock(bookID)
		if err != nil {
			return err
		}

		result, runErr := runner.RunChapterPipeline(context.Background(), bookID, chapterNumber)
		streamOutput.Finish()
		unlockErr := releaseLock()
		if runErr != nil {
			return runErr
		}
		if unlockErr != nil {
			return unlockErr
		}

		item := map[string]any{
			"chapterNumber": result.ChapterNumber,
			"title":         result.Title,
			"wordCount":     result.WordCount,
			"status":        result.Status,
			"revised":       result.Revised,
		}
		if result.TokenUsage != nil {
			item["tokenUsage"] = result.TokenUsage
		}
		results = append(results, item)

		if !asJSON && !quiet {
			fmt.Printf("  Ch.%d %q | %d words | %s\n", result.ChapterNumber, result.Title, result.WordCount, result.Status)
		}
		if strings.EqualFold(result.Status, string(models.StatusStateDegraded)) {
			if !asJSON && !quiet {
				fmt.Println("  State degraded. Stopping batch run.")
			}
			break
		}
	}

	if asJSON {
		data, _ := json.MarshalIndent(results, "", "  ")
		fmt.Println(string(data))
		return nil
	}
	if !quiet {
		fmt.Println("Write complete.")
	}
	return nil
}

func writeRewriteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rewrite [book-id] <chapter>",
		Short: T(TR.CmdWriteRewriteShort),
		Long:  T(TR.CmdWriteRewriteLong),
		Args:  cobra.MinimumNArgs(1),
		RunE:  runWriteRewrite,
	}

	cmd.Flags().Bool("force", false, "Skip confirmation prompt")
	cmd.Flags().String("words", "", "Words per chapter override")
	cmd.Flags().Bool("json", false, "Output JSON")
	return cmd
}

func runWriteRewrite(cmd *cobra.Command, args []string) error {
	root, err := findProjectRoot()
	if err != nil {
		return err
	}

	bookArg, chapter, err := parseBookAndChapterArgs(args)
	if err != nil {
		return err
	}
	bookID, err := resolveBookID(root, bookArg)
	if err != nil {
		return err
	}

	force, _ := cmd.Flags().GetBool("force")
	asJSON, _ := cmd.Flags().GetBool("json")

	if !force {
		confirmed, err := askForConfirmation(
			fmt.Sprintf("Rewrite chapter %d of %q? This removes chapter %d and all later chapters. (y/N) ", chapter, bookID, chapter),
		)
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	config, err := loadConfig(root)
	if err != nil {
		return err
	}

	sm := state.NewStateManager(root)
	releaseLock, err := sm.AcquireBookLock(bookID)
	if err != nil {
		return err
	}
	defer releaseLock()

	chaptersDir := filepath.Join(sm.BookDir(bookID), "chapters")
	entries, _ := os.ReadDir(chaptersDir)
	for _, entry := range entries {
		num := chapterNumberFromFilename(entry.Name())
		if num >= chapter && strings.HasSuffix(entry.Name(), ".md") {
			_ = os.Remove(filepath.Join(chaptersDir, entry.Name()))
			if !asJSON {
				fmt.Printf("Removed: %s\n", entry.Name())
			}
		}
	}

	index, err := sm.LoadChapterIndex(bookID)
	if err != nil {
		return err
	}
	trimmed := make([]models.ChapterMeta, 0, len(index))
	for _, chapterMeta := range index {
		if chapterMeta.Number < chapter {
			trimmed = append(trimmed, chapterMeta)
		}
	}
	if err := sm.SaveChapterIndex(bookID, trimmed); err != nil {
		return err
	}

	restoreFrom := chapter - 1
	restored, restoreErr := sm.RestoreState(bookID, restoreFrom)
	if restoreErr != nil {
		if !asJSON {
			fmt.Printf("Warning: could not restore state from chapter %d snapshot: %v\n", restoreFrom, restoreErr)
		}
	} else if restored && !asJSON {
		fmt.Printf("State restored from chapter %d snapshot.\n", restoreFrom)
	}

	streamOutput := newCLIStreamOutput(!asJSON, os.Stdout, "writer")
	runner := buildRunner(config, root, false, streamOutput.Callback(), nil)
	result, err := runner.RunChapterPipeline(context.Background(), bookID, chapter)
	streamOutput.Finish()
	if err != nil {
		return err
	}

	if asJSON {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}
	fmt.Printf("Rewrote chapter %d: %q (%d words, %s)\n", result.ChapterNumber, result.Title, result.WordCount, result.Status)
	return nil
}

func writeRepairStateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repair-state [book-id] <chapter>",
		Short: T(TR.CmdWriteRepairShort),
		Long:  T(TR.CmdWriteRepairLong),
		Args:  cobra.MinimumNArgs(1),
		RunE:  runWriteRepairState,
	}

	cmd.Flags().Bool("json", false, "Output JSON")
	return cmd
}

func runWriteRepairState(cmd *cobra.Command, args []string) error {
	root, err := findProjectRoot()
	if err != nil {
		return err
	}

	bookArg, chapter, err := parseBookAndChapterArgs(args)
	if err != nil {
		return err
	}
	bookID, err := resolveBookID(root, bookArg)
	if err != nil {
		return err
	}

	asJSON, _ := cmd.Flags().GetBool("json")
	sm := state.NewStateManager(root)

	releaseLock, err := sm.AcquireBookLock(bookID)
	if err != nil {
		return err
	}
	defer releaseLock()

	result, err := state.RewriteStructuredStateFromMarkdown(sm.BookDir(bookID), chapter)
	if err != nil {
		return err
	}

	index, err := sm.LoadChapterIndex(bookID)
	if err == nil {
		changed := false
		for i := range index {
			if index[i].Number == chapter && index[i].Status == models.StatusStateDegraded {
				index[i].Status = models.StatusReadyForReview
				index[i].UpdatedAt = time.Now()
				changed = true
			}
		}
		if changed {
			_ = sm.SaveChapterIndex(bookID, index)
		}
	}

	payload := map[string]any{
		"bookId":       bookID,
		"chapter":      chapter,
		"createdFiles": result.CreatedFiles,
		"warnings":     result.Warnings,
		"manifest":     result.Manifest,
	}

	if asJSON {
		data, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Repaired runtime state for %s chapter %d.\n", bookID, chapter)
	if len(result.Warnings) > 0 {
		fmt.Println("Warnings:")
		for _, warning := range result.Warnings {
			fmt.Printf("  - %s\n", warning)
		}
	}
	return nil
}

func parseBookAndChapterArgs(args []string) (string, int, error) {
	if len(args) == 1 {
		chapter, err := strconv.Atoi(args[0])
		if err != nil {
			return "", 0, fmt.Errorf("expected chapter number, got %q", args[0])
		}
		return "", chapter, nil
	}
	if len(args) == 2 {
		chapter, err := strconv.Atoi(args[1])
		if err != nil {
			return "", 0, fmt.Errorf("expected chapter number, got %q", args[1])
		}
		return args[0], chapter, nil
	}
	return "", 0, fmt.Errorf("usage: [book-id] <chapter>")
}

func chapterNumberFromFilename(filename string) int {
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) != 2 {
		return -1
	}
	number, err := strconv.Atoi(parts[0])
	if err != nil {
		return -1
	}
	return number
}

type cliStreamOutput struct {
	enabled       bool
	out           io.Writer
	allowedLabels map[string]struct{}
	mu            sync.Mutex
	wrote         bool
}

func newCLIStreamOutput(enabled bool, out io.Writer, labels ...string) *cliStreamOutput {
	allowedLabels := make(map[string]struct{}, len(labels))
	for _, label := range labels {
		label = strings.TrimSpace(label)
		if label != "" {
			allowedLabels[label] = struct{}{}
		}
	}
	return &cliStreamOutput{
		enabled:       enabled,
		out:           out,
		allowedLabels: allowedLabels,
	}
}

func (s *cliStreamOutput) Callback() llm.OnStreamChunk {
	if !s.enabled || s.out == nil {
		return nil
	}

	return func(chunk llm.StreamChunk) {
		if chunk.Text == "" {
			return
		}
		if len(s.allowedLabels) > 0 {
			if _, ok := s.allowedLabels[chunk.Label]; !ok {
				return
			}
		}

		text := strings.ToValidUTF8(chunk.Text, "\uFFFD")
		if text == "" {
			return
		}

		s.mu.Lock()
		defer s.mu.Unlock()
		_, _ = io.WriteString(s.out, text)
		s.wrote = true
	}
}

func (s *cliStreamOutput) Finish() {
	if !s.enabled || s.out == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.wrote {
		return
	}

	_, _ = io.WriteString(s.out, "\n")
	s.wrote = false
}
