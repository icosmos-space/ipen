package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/icosmos-space/ipen/core/agents"
	"github.com/icosmos-space/ipen/core/models"
	"github.com/icosmos-space/ipen/core/state"
	coreutils "github.com/icosmos-space/ipen/core/utils"
	"github.com/spf13/cobra"
)

// ImportCommand 向书籍导入外部数据。
func ImportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: T(TR.CmdImportShort),
		Long:  T(TR.CmdImportLong),
	}
	cmd.AddCommand(importCanonCommand())
	cmd.AddCommand(importChaptersCommand())
	return cmd
}

func importCanonCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "canon [target-book-id]",
		Short: T(TR.CmdImportCanonShort),
		Long:  T(TR.CmdImportCanonLong),
		Args:  cobra.MaximumNArgs(1),
		RunE:  runImportCanon,
	}
	cmd.Flags().String("from", "", "Parent book ID")
	cmd.Flags().Bool("json", false, "Output JSON")
	_ = cmd.MarkFlagRequired("from")
	return cmd
}

func runImportCanon(cmd *cobra.Command, args []string) error {
	root, err := findProjectRoot()
	if err != nil {
		return err
	}
	targetArg := ""
	if len(args) > 0 {
		targetArg = args[0]
	}
	targetBookID, err := resolveBookID(root, targetArg)
	if err != nil {
		return err
	}

	parentArg, _ := cmd.Flags().GetString("from")
	parentBookID, err := resolveBookID(root, parentArg)
	if err != nil {
		return err
	}
	asJSON, _ := cmd.Flags().GetBool("json")

	sm := state.NewStateManager(root)
	targetBook, err := sm.LoadBookConfig(targetBookID)
	if err != nil {
		return err
	}
	parentBook, err := sm.LoadBookConfig(parentBookID)
	if err != nil {
		return err
	}

	parentDir := sm.BookDir(parentBookID)
	parts := []string{
		"# Parent Canon Reference",
		"",
		fmt.Sprintf("Source Book: %s (%s)", parentBook.Title, parentBookID),
		"",
		"## Story Bible",
		readStoryFileOrEmpty(parentDir, "story_bible.md"),
		"",
		"## Volume Outline",
		readStoryFileOrEmpty(parentDir, "volume_outline.md"),
		"",
		"## Book Rules",
		readStoryFileOrEmpty(parentDir, "book_rules.md"),
	}
	sourceText := strings.Join(parts, "\n")
	canonDoc := sourceText

	if config, cfgErr := loadConfig(root); cfgErr == nil {
		importer := agents.NewFanficCanonImporter(agents.AgentContext{
			Client:      buildPipelineConfig(config, root, true).Client,
			Model:       config.LLM.Model,
			ProjectRoot: root,
			BookID:      targetBookID,
			Logger:      buildLogger(true),
		})
		if imported, impErr := importer.ImportFromText(context.Background(), sourceText, parentBookID, models.FanficModeCanon); impErr == nil {
			canonDoc = imported.FullDocument
		}
	}

	targetStoryDir := filepath.Join(sm.BookDir(targetBookID), "story")
	if err := os.MkdirAll(targetStoryDir, 0755); err != nil {
		return err
	}
	outputPath := filepath.Join(targetStoryDir, "parent_canon.md")
	if err := os.WriteFile(outputPath, []byte(canonDoc), 0644); err != nil {
		return err
	}

	targetBook.ParentBookID = parentBookID
	targetBook.UpdatedAt = time.Now()
	_ = sm.SaveBookConfig(targetBookID, targetBook)

	payload := map[string]any{
		"targetBookId": targetBookID,
		"parentBookId": parentBookID,
		"output":       "story/parent_canon.md",
	}
	if asJSON {
		data, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Println(string(data))
		return nil
	}
	fmt.Printf("Imported parent canon from %q to %q.\n", parentBookID, targetBookID)
	fmt.Println("Output: story/parent_canon.md")
	return nil
}

func importChaptersCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chapters [book-id]",
		Short: T(TR.CmdImportChaptersShort),
		Long:  T(TR.CmdImportChaptersLong),
		Args:  cobra.MaximumNArgs(1),
		RunE:  runImportChapters,
	}
	cmd.Flags().String("from", "", "Path to a text file or directory of .md/.txt files")
	cmd.Flags().String("split", "", "Custom regex for chapter splitting (single-file mode)")
	cmd.Flags().Int("resume-from", 0, "Resume from source chapter number")
	cmd.Flags().Bool("json", false, "Output JSON")
	_ = cmd.MarkFlagRequired("from")
	return cmd
}

func runImportChapters(cmd *cobra.Command, args []string) error {
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

	fromPath, _ := cmd.Flags().GetString("from")
	splitPattern, _ := cmd.Flags().GetString("split")
	resumeFrom, _ := cmd.Flags().GetInt("resume-from")
	asJSON, _ := cmd.Flags().GetBool("json")

	sm := state.NewStateManager(root)
	book, err := sm.LoadBookConfig(bookID)
	if err != nil {
		return err
	}
	index, err := sm.LoadChapterIndex(bookID)
	if err != nil {
		return err
	}
	existingCount := len(index)
	if existingCount > 0 && resumeFrom <= 0 {
		return fmt.Errorf("book %q already has %d chapter(s). use --resume-from to continue", bookID, existingCount)
	}

	absFrom, err := filepath.Abs(fromPath)
	if err != nil {
		return err
	}
	info, err := os.Stat(absFrom)
	if err != nil {
		return err
	}

	type chapterSource struct {
		Title   string
		Content string
	}
	chapters := []chapterSource{}

	if info.IsDir() {
		entries, err := os.ReadDir(absFrom)
		if err != nil {
			return err
		}
		files := make([]string, 0, len(entries))
		for _, entry := range entries {
			name := strings.ToLower(entry.Name())
			if strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".txt") {
				files = append(files, entry.Name())
			}
		}
		sort.Strings(files)
		if len(files) == 0 {
			return fmt.Errorf("no .md/.txt files found in %s", absFrom)
		}
		for _, name := range files {
			raw, err := os.ReadFile(filepath.Join(absFrom, name))
			if err != nil {
				return err
			}
			title := strings.TrimSuffix(strings.TrimSuffix(name, ".md"), ".txt")
			title = strings.TrimLeft(title, "0123456789-_ ")
			if title == "" {
				title = name
			}
			chapters = append(chapters, chapterSource{
				Title:   title,
				Content: string(raw),
			})
		}
	} else {
		raw, err := os.ReadFile(absFrom)
		if err != nil {
			return err
		}
		splits := coreutils.SplitChapters(string(raw), splitPattern)
		if len(splits) == 0 {
			return fmt.Errorf("no chapters found in %s", absFrom)
		}
		for _, item := range splits {
			title := strings.TrimSpace(item.Title)
			if title == "" {
				title = "Imported Chapter"
			}
			chapters = append(chapters, chapterSource{
				Title:   title,
				Content: item.Content,
			})
		}
	}

	startNumber := 1
	if resumeFrom > 0 {
		startNumber = resumeFrom
		if resumeFrom > len(chapters) {
			return fmt.Errorf("--resume-from %d exceeds source chapter count %d", resumeFrom, len(chapters))
		}
		chapters = chapters[resumeFrom-1:]
	}
	if existingCount >= startNumber {
		shift := existingCount - startNumber + 1
		if shift >= len(chapters) {
			return fmt.Errorf("nothing left to import after resume adjustment")
		}
		chapters = chapters[shift:]
		startNumber = existingCount + 1
	}

	totalWords := 0
	imported := 0
	bookDir := sm.BookDir(bookID)
	for i, chapter := range chapters {
		chapterNumber := startNumber + i
		title := strings.TrimSpace(chapter.Title)
		if title == "" {
			title = fmt.Sprintf("Chapter %d", chapterNumber)
		}
		content := strings.TrimSpace(chapter.Content)
		if !strings.HasPrefix(content, "# ") {
			content = "# " + title + "\n\n" + content + "\n"
		}
		if _, err := writeChapterFileForNumber(bookDir, chapterNumber, title, content); err != nil {
			return err
		}

		wordCount := chapterLengthForBook(book, content)
		totalWords += wordCount
		imported++

		meta := models.ChapterMeta{
			Number:    chapterNumber,
			Title:     title,
			Status:    models.StatusImported,
			WordCount: wordCount,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		index = upsertChapterMeta(index, meta)
	}
	if err := sm.SaveChapterIndex(bookID, index); err != nil {
		return err
	}

	if imported > 0 {
		joined := make([]string, 0, imported)
		for _, chapter := range chapters {
			joined = append(joined, chapter.Content)
		}
		profile := agents.AnalyzeStyle(strings.Join(joined, "\n\n"), filepath.Base(absFrom))
		profileData, _ := json.MarshalIndent(profile, "", "  ")
		_ = os.MkdirAll(filepath.Join(bookDir, "story"), 0755)
		_ = os.WriteFile(filepath.Join(bookDir, "story", "style_profile.json"), profileData, 0644)
	}

	nextChapter, _ := sm.GetNextChapterNumber(bookID)
	payload := map[string]any{
		"bookId":        bookID,
		"importedCount": imported,
		"totalWords":    totalWords,
		"nextChapter":   nextChapter,
	}
	if asJSON {
		data, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Imported %d chapter(s) into %q.\n", imported, bookID)
	fmt.Printf("Total words: %d\n", totalWords)
	fmt.Printf("Next chapter: %d\n", nextChapter)
	return nil
}
