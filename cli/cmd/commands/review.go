package commands

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/icosmos-space/ipen/core/agents"
	"github.com/icosmos-space/ipen/core/models"
	"github.com/icosmos-space/ipen/core/state"
	coreutils "github.com/icosmos-space/ipen/core/utils"
	"github.com/spf13/cobra"
)

// ReviewCommand 审阅并批准章节。
func ReviewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "review",
		Short: T(TR.CmdReviewShort),
		Long:  T(TR.CmdReviewLong),
	}

	cmd.AddCommand(reviewListCommand())
	cmd.AddCommand(reviewApproveCommand())
	cmd.AddCommand(reviewApproveAllCommand())
	cmd.AddCommand(reviewRejectCommand())
	return cmd
}

func reviewListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [book-id]",
		Short: T(TR.CmdReviewListShort),
		Long:  T(TR.CmdReviewListShort),
		Args:  cobra.MaximumNArgs(1),
		RunE:  runReviewList,
	}
	cmd.Flags().Bool("json", false, "Output JSON")
	return cmd
}

func runReviewList(cmd *cobra.Command, args []string) error {
	root, err := findProjectRoot()
	if err != nil {
		return err
	}
	asJSON, _ := cmd.Flags().GetBool("json")

	sm := state.NewStateManager(root)
	bookIDs, err := sm.ListBooks()
	if err != nil {
		return err
	}
	if len(args) > 0 {
		bookIDs = []string{args[0]}
	}

	type pendingItem struct {
		BookID       string   `json:"bookId"`
		Title        string   `json:"title"`
		Chapter      int      `json:"chapter"`
		ChapterTitle string   `json:"chapterTitle"`
		WordCount    int      `json:"wordCount"`
		Status       string   `json:"status"`
		Issues       []string `json:"issues"`
	}

	pending := []pendingItem{}

	for _, id := range bookIDs {
		index, err := sm.LoadChapterIndex(id)
		if err != nil {
			continue
		}
		book, err := sm.LoadBookConfig(id)
		if err != nil {
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

		shownHeader := false
		for _, chapter := range index {
			if chapter.Status != models.StatusReadyForReview && chapter.Status != models.StatusAuditFailed {
				continue
			}

			item := pendingItem{
				BookID:       id,
				Title:        book.Title,
				Chapter:      chapter.Number,
				ChapterTitle: chapter.Title,
				WordCount:    chapter.WordCount,
				Status:       string(chapter.Status),
				Issues:       chapter.AuditIssues,
			}
			pending = append(pending, item)

			if asJSON {
				continue
			}
			if !shownHeader {
				fmt.Printf("\n%s (%s)\n", book.Title, id)
				shownHeader = true
			}
			fmt.Printf("  Ch.%d %q | %s | %s\n",
				chapter.Number,
				chapter.Title,
				coreutils.FormatLengthCount(chapter.WordCount, countingMode),
				chapter.Status,
			)
			for _, issue := range chapter.AuditIssues {
				fmt.Printf("    - %s\n", issue)
			}
		}
	}

	if asJSON {
		data, _ := json.MarshalIndent(map[string]any{"pending": pending}, "", "  ")
		fmt.Println(string(data))
		return nil
	}
	if len(pending) == 0 {
		fmt.Println("No chapters pending review.")
	}
	return nil
}

func reviewApproveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "approve [book-id] <chapter>",
		Short: T(TR.CmdReviewApproveShort),
		Long:  T(TR.CmdReviewApproveShort),
		Args:  cobra.MinimumNArgs(1),
		RunE:  runReviewApprove,
	}
	cmd.Flags().Bool("json", false, "Output JSON")
	return cmd
}

func runReviewApprove(cmd *cobra.Command, args []string) error {
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
	index, err := sm.LoadChapterIndex(bookID)
	if err != nil {
		return err
	}

	found := false
	for i := range index {
		if index[i].Number == chapter {
			index[i].Status = models.StatusApproved
			index[i].UpdatedAt = time.Now()
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("chapter %d not found in %q", chapter, bookID)
	}
	if err := sm.SaveChapterIndex(bookID, index); err != nil {
		return err
	}

	if asJSON {
		data, _ := json.MarshalIndent(map[string]any{
			"bookId":  bookID,
			"chapter": chapter,
			"status":  "approved",
		}, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Chapter %d approved.\n", chapter)
	return nil
}

func reviewApproveAllCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "approve-all [book-id]",
		Short: "Approve all pending chapters",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runReviewApproveAll,
	}
	cmd.Flags().Bool("json", false, "Output JSON")
	return cmd
}

func runReviewApproveAll(cmd *cobra.Command, args []string) error {
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

	now := time.Now()
	count := 0
	for i := range index {
		if index[i].Status == models.StatusReadyForReview || index[i].Status == models.StatusAuditFailed {
			index[i].Status = models.StatusApproved
			index[i].UpdatedAt = now
			count++
		}
	}

	if err := sm.SaveChapterIndex(bookID, index); err != nil {
		return err
	}

	if asJSON {
		data, _ := json.MarshalIndent(map[string]any{
			"bookId":        bookID,
			"approvedCount": count,
		}, "", "  ")
		fmt.Println(string(data))
		return nil
	}
	fmt.Printf("%d chapter(s) approved.\n", count)
	return nil
}

func reviewRejectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reject [book-id] <chapter>",
		Short: "Reject a chapter and optionally roll back state",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runReviewReject,
	}
	cmd.Flags().String("reason", "", "Rejection reason")
	cmd.Flags().Bool("keep-subsequent", false, "Only reject this chapter and keep later chapters")
	cmd.Flags().Bool("json", false, "Output JSON")
	return cmd
}

func runReviewReject(cmd *cobra.Command, args []string) error {
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

	reason, _ := cmd.Flags().GetString("reason")
	keepSubsequent, _ := cmd.Flags().GetBool("keep-subsequent")
	asJSON, _ := cmd.Flags().GetBool("json")

	sm := state.NewStateManager(root)

	if keepSubsequent {
		index, err := sm.LoadChapterIndex(bookID)
		if err != nil {
			return err
		}
		found := false
		for i := range index {
			if index[i].Number == chapter {
				index[i].Status = models.StatusRejected
				index[i].ReviewNote = reason
				index[i].UpdatedAt = time.Now()
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("chapter %d not found in %q", chapter, bookID)
		}
		if err := sm.SaveChapterIndex(bookID, index); err != nil {
			return err
		}
		if asJSON {
			data, _ := json.MarshalIndent(map[string]any{
				"bookId":    bookID,
				"chapter":   chapter,
				"status":    "rejected",
				"discarded": []int{},
			}, "", "  ")
			fmt.Println(string(data))
			return nil
		}
		fmt.Printf("Chapter %d rejected (kept subsequent chapters).\n", chapter)
		return nil
	}

	rollbackTarget := chapter - 1
	discarded, err := sm.RollbackToChapter(bookID, rollbackTarget)
	if err != nil {
		return err
	}

	if asJSON {
		data, _ := json.MarshalIndent(map[string]any{
			"bookId":       bookID,
			"chapter":      chapter,
			"status":       "rejected",
			"rolledBackTo": rollbackTarget,
			"discarded":    discarded,
		}, "", "  ")
		fmt.Println(string(data))
		return nil
	}
	fmt.Printf("Chapter %d rejected. Rolled back to chapter %d.\n", chapter, rollbackTarget)
	return nil
}
