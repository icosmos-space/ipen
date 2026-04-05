package commands

import (
	"encoding/json"
	"fmt"

	"github.com/icosmos-space/ipen/core/agents"
	"github.com/icosmos-space/ipen/core/state"
	coreutils "github.com/icosmos-space/ipen/core/utils"
	"github.com/spf13/cobra"
)

// StatusCommand 显示书籍与章节状态。
func StatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status [book-id]",
		Short: T(TR.CmdStatusShort),
		Long:  T(TR.CmdStatusLong),
		Args:  cobra.MaximumNArgs(1),
		RunE:  runStatus,
	}

	cmd.Flags().Bool("chapters", false, "Show per-chapter status")
	cmd.Flags().Bool("json", false, "Output JSON")
	return cmd
}

func runStatus(cmd *cobra.Command, args []string) error {
	root, err := findProjectRoot()
	if err != nil {
		return err
	}

	showChapters, _ := cmd.Flags().GetBool("chapters")
	asJSON, _ := cmd.Flags().GetBool("json")

	sm := state.NewStateManager(root)
	allBookIDs, err := sm.ListBooks()
	if err != nil {
		return err
	}

	bookIDs := allBookIDs
	if len(args) > 0 {
		bookIDs = []string{args[0]}
		if !containsString(allBookIDs, args[0]) {
			return fmt.Errorf("book %q not found. available: %v", args[0], allBookIDs)
		}
	}

	type chapterView struct {
		Number    int      `json:"number"`
		Title     string   `json:"title"`
		Status    string   `json:"status"`
		WordCount int      `json:"wordCount"`
		Issues    []string `json:"issues,omitempty"`
	}
	type bookView struct {
		ID                 string        `json:"id"`
		Title              string        `json:"title"`
		Status             string        `json:"status"`
		Genre              string        `json:"genre"`
		Platform           string        `json:"platform"`
		Chapters           int           `json:"chapters"`
		TargetChapters     int           `json:"targetChapters"`
		TotalWords         int           `json:"totalWords"`
		AvgWordsPerChapter int           `json:"avgWordsPerChapter"`
		Approved           int           `json:"approved"`
		Pending            int           `json:"pending"`
		Failed             int           `json:"failed"`
		Degraded           int           `json:"degraded"`
		MigrationHint      string        `json:"migrationHint,omitempty"`
		ChapterList        []chapterView `json:"chapterList,omitempty"`
	}

	booksData := make([]bookView, 0, len(bookIDs))

	if !asJSON {
		fmt.Printf("iPen Project: %s\n", root)
		fmt.Printf("Books: %d\n\n", len(allBookIDs))
	}

	for _, id := range bookIDs {
		book, err := sm.LoadBookConfig(id)
		if err != nil {
			return err
		}
		index, err := sm.LoadChapterIndex(id)
		if err != nil {
			return err
		}

		totalWords := 0
		approved := 0
		pending := 0
		failed := 0
		degraded := 0
		chapters := make([]chapterView, 0, len(index))

		for _, chapter := range index {
			totalWords += chapter.WordCount
			switch chapter.Status {
			case "approved":
				approved++
			case "ready-for-review":
				pending++
			case "audit-failed":
				failed++
			case "state-degraded":
				degraded++
			}

			if showChapters {
				item := chapterView{
					Number:    chapter.Number,
					Title:     chapter.Title,
					Status:    string(chapter.Status),
					WordCount: chapter.WordCount,
				}
				if chapter.Status == "audit-failed" || chapter.Status == "state-degraded" {
					item.Issues = chapter.AuditIssues
				}
				chapters = append(chapters, item)
			}
		}

		avgWords := 0
		if len(index) > 0 {
			avgWords = totalWords / len(index)
		}

		persistedChapterCount := sm.GetPersistedChapterCount(id)
		migrationHint := getLegacyMigrationHint(root, id)

		bookItem := bookView{
			ID:                 id,
			Title:              book.Title,
			Status:             string(book.Status),
			Genre:              string(book.Genre),
			Platform:           string(book.Platform),
			Chapters:           persistedChapterCount,
			TargetChapters:     book.TargetChapters,
			TotalWords:         totalWords,
			AvgWordsPerChapter: avgWords,
			Approved:           approved,
			Pending:            pending,
			Failed:             failed,
			Degraded:           degraded,
			MigrationHint:      migrationHint,
		}
		if showChapters {
			bookItem.ChapterList = chapters
		}
		booksData = append(booksData, bookItem)

		if asJSON {
			continue
		}

		lang := book.Language
		if lang == "" {
			if parsed, err := agents.ReadGenreProfile(root, string(book.Genre)); err == nil && parsed != nil {
				lang = parsed.Profile.Language
			}
		}
		if lang == "" {
			lang = "zh"
		}
		countingMode := coreutils.ResolveLengthCountingMode(coreutils.LengthLanguage(lang))

		fmt.Printf("%s (%s)\n", book.Title, id)
		fmt.Printf("  Status: %s\n", book.Status)
		fmt.Printf("  Platform/Genre: %s / %s\n", book.Platform, book.Genre)
		fmt.Printf("  Chapters: %d / %d\n", persistedChapterCount, book.TargetChapters)
		fmt.Printf("  Words: %d (avg %d/ch)\n", totalWords, avgWords)
		fmt.Printf("  Approved: %d | Pending: %d | Failed: %d | Degraded: %d\n", approved, pending, failed, degraded)
		if migrationHint != "" {
			fmt.Printf("  Migration: %s\n", migrationHint)
		}

		if showChapters && len(chapters) > 0 {
			fmt.Println("  Chapters:")
			for _, ch := range chapters {
				icon := "~"
				switch ch.Status {
				case "approved":
					icon = "+"
				case "audit-failed":
					icon = "!"
				case "state-degraded":
					icon = "x"
				}
				fmt.Printf("    [%s] Ch.%d %q | %s | %s\n",
					icon,
					ch.Number,
					ch.Title,
					coreutils.FormatLengthCount(ch.WordCount, countingMode),
					ch.Status,
				)
				for _, issue := range ch.Issues {
					fmt.Printf("      - %s\n", issue)
				}
			}
		}
		fmt.Println()
	}

	if asJSON {
		payload := map[string]any{
			"project": root,
			"books":   booksData,
		}
		data, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Println(string(data))
	}
	return nil
}
