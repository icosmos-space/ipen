package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/icosmos-space/ipen/core/agents"
	"github.com/icosmos-space/ipen/core/models"
	"github.com/icosmos-space/ipen/core/state"
	coreutils "github.com/icosmos-space/ipen/core/utils"
	"github.com/spf13/cobra"
)

// DraftCommand 写作草稿章节（不执行审计与修订）。
func DraftCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "draft [book-id]",
		Short: T(TR.CmdDraftShort),
		Long:  T(TR.CmdDraftLong),
		Args:  cobra.MaximumNArgs(1),
		RunE:  runDraftCreate,
	}
	addDraftCreateFlags(cmd)
	cmd.AddCommand(draftCreateCommand())
	cmd.AddCommand(draftListCommand())
	return cmd
}

func draftCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [book-id]",
		Short: T(TR.CmdDraftCreateShort),
		Args:  cobra.MaximumNArgs(1),
		RunE:  runDraftCreate,
	}
	addDraftCreateFlags(cmd)
	return cmd
}

func addDraftCreateFlags(cmd *cobra.Command) {
	cmd.Flags().String("words", "", "Words per chapter override")
	cmd.Flags().String("context", "", "Creative guidance text")
	cmd.Flags().String("context-file", "", "Read guidance from file")
	cmd.Flags().Bool("json", false, "Output JSON")
	cmd.Flags().BoolP("quiet", "q", false, "Suppress progress output")
}

func runDraftCreate(cmd *cobra.Command, args []string) error {
	root, err := findProjectRoot()
	if err != nil {
		return err
	}
	config, err := loadConfig(root)
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
	sm := state.NewStateManager(root)
	book, err := sm.LoadBookConfig(bookID)
	if err != nil {
		return err
	}
	chapterNumber, err := sm.GetNextChapterNumber(bookID)
	if err != nil {
		return err
	}

	targetWords := book.ChapterWordCount
	if wordOverride != nil && *wordOverride > 0 {
		targetWords = *wordOverride
	}
	lang := coreutils.LanguageZH
	if strings.EqualFold(book.Language, "en") {
		lang = coreutils.LanguageEN
	}
	lengthSpec := coreutils.BuildLengthSpec(targetWords, lang)

	writer := agents.NewWriterAgent(agents.AgentContext{
		Client:      buildPipelineConfig(config, root, quiet).Client,
		Model:       config.LLM.Model,
		ProjectRoot: root,
		BookID:      bookID,
		Logger:      buildLogger(quiet),
	})
	if !asJSON && !quiet {
		fmt.Printf("Writing draft chapter %d for %q...\n", chapterNumber, bookID)
	}
	result, err := writer.WriteChapter(context.Background(), agents.WriteChapterInput{
		Book:              book,
		BookDir:           sm.BookDir(bookID),
		ChapterNumber:     chapterNumber,
		ExternalContext:   contextText,
		LengthSpec:        &lengthSpec,
		WordCountOverride: targetWords,
	})
	if err != nil {
		return err
	}

	content := strings.TrimSpace(result.Content)
	if !strings.HasPrefix(content, "# ") {
		content = fmt.Sprintf("# %s\n\n%s\n", strings.TrimSpace(result.Title), content)
	}

	path, err := writeChapterFileForNumber(sm.BookDir(bookID), chapterNumber, result.Title, content)
	if err != nil {
		return err
	}

	index, err := sm.LoadChapterIndex(bookID)
	if err != nil {
		return err
	}
	now := time.Now()
	meta := models.ChapterMeta{
		Number:    chapterNumber,
		Title:     strings.TrimSpace(result.Title),
		Status:    models.StatusDrafted,
		WordCount: chapterLengthForBook(book, content),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if result.TokenUsage != nil {
		meta.TokenUsage = result.TokenUsage
	}
	index = upsertChapterMeta(index, meta)
	if err := sm.SaveChapterIndex(bookID, index); err != nil {
		return err
	}

	payload := map[string]any{
		"bookId":        bookID,
		"chapterNumber": chapterNumber,
		"title":         result.Title,
		"wordCount":     meta.WordCount,
		"status":        meta.Status,
		"filePath":      filepath.ToSlash(path),
	}
	if asJSON {
		data, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Draft chapter %d created: %s\n", chapterNumber, result.Title)
	fmt.Printf("Words: %d\n", meta.WordCount)
	fmt.Printf("File: %s\n", filepath.ToSlash(path))
	return nil
}

func draftListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [book-id]",
		Short: T(TR.CmdDraftListShort),
		Args:  cobra.MaximumNArgs(1),
		RunE:  runDraftList,
	}
	cmd.Flags().Bool("json", false, "Output JSON")
	return cmd
}

func runDraftList(cmd *cobra.Command, args []string) error {
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

	drafts := make([]models.ChapterMeta, 0, len(index))
	for _, chapter := range index {
		if chapter.Status == models.StatusDrafted || chapter.Status == models.StatusDrafting {
			drafts = append(drafts, chapter)
		}
	}

	if asJSON {
		data, _ := json.MarshalIndent(map[string]any{
			"bookId": bookID,
			"drafts": drafts,
		}, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if len(drafts) == 0 {
		fmt.Printf("No drafts found for %q.\n", bookID)
		return nil
	}
	fmt.Printf("Drafts for %q:\n", bookID)
	for _, chapter := range drafts {
		fmt.Printf("  Ch.%d %q | %d words | %s\n", chapter.Number, chapter.Title, chapter.WordCount, chapter.Status)
	}
	return nil
}
