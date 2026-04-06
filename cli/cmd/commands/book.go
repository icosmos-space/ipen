package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/icosmos-space/ipen/core/agents"
	"github.com/icosmos-space/ipen/core/models"
	"github.com/icosmos-space/ipen/core/state"
	"github.com/spf13/cobra"
)

// BookCommand 管理书籍。
func BookCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "book",
		Short: T(TR.CmdBookShort),
		Long:  T(TR.CmdBookLong),
	}

	cmd.AddCommand(bookCreateCommand())
	cmd.AddCommand(bookUpdateCommand())
	cmd.AddCommand(bookListCommand())
	cmd.AddCommand(bookDeleteCommand())

	return cmd
}

func bookCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [title]",
		Short: T(TR.CmdBookCreateShort),
		Long:  T(TR.CmdBookCreateLong),
		Args:  cobra.MaximumNArgs(1),
		RunE:  runBookCreate,
	}

	cmd.Flags().String("title", "", T(TR.CmdBookCreateFlagTitle))
	cmd.Flags().String("genre", "xuanhuan", T(TR.CmdBookCreateFlagGenre))
	cmd.Flags().String("platform", "tomato", T(TR.CmdBookCreateFlagPlatform))
	cmd.Flags().Int("target-chapters", 200, T(TR.CmdBookCreateFlagTargetChapters))
	cmd.Flags().Int("words", 3000, T(TR.CmdBookCreateFlagWords))
	cmd.Flags().Int("chapter-words", 0, "--words 的别名")
	cmd.Flags().String("brief", "", T(TR.CmdBookCreateFlagBrief))
	cmd.Flags().String("lang", "", T(TR.CmdBookCreateFlagLang))
	cmd.Flags().Bool("json", false, "Output JSON")

	return cmd
}

func runBookCreate(cmd *cobra.Command, args []string) error {
	title, _ := cmd.Flags().GetString("title")
	title = strings.TrimSpace(title)
	if len(args) > 0 {
		argTitle := strings.TrimSpace(args[0])
		if title == "" {
			title = argTitle
		} else if title != argTitle {
			return fmt.Errorf("标题冲突: 位置参数为 %q，但 --title 为 %q", argTitle, title)
		}
	}
	if title == "" {
		return fmt.Errorf("缺少书名，请使用位置参数 <title> 或 --title")
	}
	genre, _ := cmd.Flags().GetString("genre")
	platform, _ := cmd.Flags().GetString("platform")
	targetChapters, _ := cmd.Flags().GetInt("target-chapters")
	wordsPerChapter, _ := cmd.Flags().GetInt("words")
	chapterWordsAlias, _ := cmd.Flags().GetInt("chapter-words")
	if cmd.Flags().Changed("words") && cmd.Flags().Changed("chapter-words") && wordsPerChapter != chapterWordsAlias {
		return fmt.Errorf("参数冲突: --words=%d 与 --chapter-words=%d", wordsPerChapter, chapterWordsAlias)
	}
	if cmd.Flags().Changed("chapter-words") {
		wordsPerChapter = chapterWordsAlias
	}
	briefPath, _ := cmd.Flags().GetString("brief")
	lang, _ := cmd.Flags().GetString("lang")
	asJSON, _ := cmd.Flags().GetBool("json")

	root, err := findProjectRoot()
	if err != nil {
		return err
	}
	config, err := loadConfig(root)
	if err != nil {
		return err
	}

	bookID := sanitizeBookID(title)
	sm := state.NewStateManager(root)
	bookDir := sm.BookDir(bookID)

	if _, statErr := os.Stat(bookDir); statErr == nil {
		if sm.IsCompleteBookDirectory(bookDir) {
			return fmt.Errorf("书籍 %q 已经存在于 books/%s", bookID, bookID)
		}
		if err := os.RemoveAll(bookDir); err != nil {
			return err
		}
	}

	bookLanguage := lang
	if bookLanguage == "" {
		bookLanguage = config.Language
	}
	if bookLanguage == "" {
		bookLanguage = "zh"
	}

	now := time.Now()
	book := &models.BookConfig{
		ID:               bookID,
		Title:            title,
		Platform:         models.Platform(platform),
		Genre:            models.Genre(genre),
		Status:           models.StatusOutlining,
		TargetChapters:   targetChapters,
		ChapterWordCount: wordsPerChapter,
		Language:         bookLanguage,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := book.Validate(); err != nil {
		return err
	}

	brief := ""
	if briefPath != "" {
		data, err := os.ReadFile(filepath.Clean(briefPath))
		if err != nil {
			return err
		}
		brief = string(data)
	}

	if err := sm.SaveBookConfig(bookID, book); err != nil {
		return err
	}

	architect := agents.NewArchitectAgent(agents.AgentContext{
		ProjectRoot: root,
		BookID:      bookID,
	})
	foundation, err := architect.GenerateFoundation(context.Background(), book, brief, "")
	if err != nil {
		return err
	}

	if err := architect.WriteFoundationFiles(bookDir, *foundation, false, book.Language); err != nil {
		return err
	}
	if err := sm.EnsureControlDocuments(bookID, brief); err != nil {
		return err
	}
	if err := sm.SaveChapterIndex(bookID, []models.ChapterMeta{}); err != nil {
		return err
	}
	if err := sm.SnapshotState(bookID, 0); err != nil {
		return err
	}
	if err := sm.EnsureRuntimeState(bookID, 0); err != nil {
		return err
	}

	if asJSON {
		payload := map[string]any{
			"bookId":    bookID,
			"title":     book.Title,
			"genre":     book.Genre,
			"platform":  book.Platform,
			"location":  filepath.ToSlash(filepath.Join("books", bookID)),
			"nextStep":  fmt.Sprintf("ipen write next %s", bookID),
			"language":  book.Language,
			"timestamp": now.Format(time.RFC3339),
		}
		data, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("创建书籍 %q (%s)\n", book.Title, bookID)
	fmt.Printf("  类别/平台: %s/%s\n", book.Genre, book.Platform)
	fmt.Printf("  写作目标: %d 章节, %d 字/章\n", book.TargetChapters, book.ChapterWordCount)
	fmt.Printf("  位置: books/%s\n", bookID)
	fmt.Printf("  下一步: ipen write next %s\n", bookID)
	return nil
}

func bookUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update [book-id]",
		Short: T(TR.CmdBookUpdateShort),
		Long:  T(TR.CmdBookUpdateLong),
		Args:  cobra.MaximumNArgs(1),
		RunE:  runBookUpdate,
	}

	cmd.Flags().String("title", "", "更新书名")
	cmd.Flags().String("genre", "", "更新类别")
	cmd.Flags().String("platform", "", "更新平台")
	cmd.Flags().String("status", "", "更新状态")
	cmd.Flags().Int("target-chapters", 0, "更新目标章节数")
	cmd.Flags().Int("words", 0, "更新每章字数")
	cmd.Flags().Int("chapter-words", 0, "--words 的别名")
	cmd.Flags().String("lang", "", "更新写作语言")
	cmd.Flags().Bool("json", false, "输出 JSON")
	return cmd
}

func runBookUpdate(cmd *cobra.Command, args []string) error {
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
	book, err := sm.LoadBookConfig(bookID)
	if err != nil {
		return err
	}

	updated := *book
	hasChanges := false

	if cmd.Flags().Changed("title") {
		value, _ := cmd.Flags().GetString("title")
		updated.Title = value
		hasChanges = true
	}
	if cmd.Flags().Changed("genre") {
		value, _ := cmd.Flags().GetString("genre")
		updated.Genre = models.Genre(value)
		hasChanges = true
	}
	if cmd.Flags().Changed("platform") {
		value, _ := cmd.Flags().GetString("platform")
		updated.Platform = models.Platform(value)
		hasChanges = true
	}
	if cmd.Flags().Changed("status") {
		value, _ := cmd.Flags().GetString("status")
		updated.Status = models.BookStatus(value)
		hasChanges = true
	}
	if cmd.Flags().Changed("target-chapters") {
		value, _ := cmd.Flags().GetInt("target-chapters")
		updated.TargetChapters = value
		hasChanges = true
	}
	if cmd.Flags().Changed("words") {
		value, _ := cmd.Flags().GetInt("words")
		updated.ChapterWordCount = value
		hasChanges = true
	}
	if cmd.Flags().Changed("chapter-words") {
		value, _ := cmd.Flags().GetInt("chapter-words")
		if cmd.Flags().Changed("words") && value != updated.ChapterWordCount {
			return fmt.Errorf("参数冲突: --words=%d 与 --chapter-words=%d", updated.ChapterWordCount, value)
		}
		updated.ChapterWordCount = value
		hasChanges = true
	}
	if cmd.Flags().Changed("lang") {
		value, _ := cmd.Flags().GetString("lang")
		updated.Language = value
		hasChanges = true
	}

	if !hasChanges {
		if asJSON {
			data, _ := json.MarshalIndent(book, "", "  ")
			fmt.Println(string(data))
		} else {
			fmt.Printf("书籍 %s (%s)\n", book.Title, bookID)
			fmt.Printf("  类别/平台: %s/%s\n", book.Genre, book.Platform)
			fmt.Printf("  状态: %s\n", book.Status)
			fmt.Printf("  写作目标: %d 章节, %d 字/章\n", book.TargetChapters, book.ChapterWordCount)
		}
		return nil
	}

	updated.UpdatedAt = time.Now()
	if err := updated.Validate(); err != nil {
		return err
	}
	if err := sm.SaveBookConfig(bookID, &updated); err != nil {
		return err
	}

	if asJSON {
		data, _ := json.MarshalIndent(updated, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("更新书籍 %s\n", bookID)
	return nil
}

func bookListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: T(TR.CmdBookListShort),
		RunE:  runBookList,
	}

	cmd.Flags().Bool("json", false, "输出 JSON")
	return cmd
}

func runBookList(cmd *cobra.Command, args []string) error {
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

	type bookListEntry struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		Genre    string `json:"genre"`
		Platform string `json:"platform"`
		Status   string `json:"status"`
		Chapters int    `json:"chapters"`
	}

	result := make([]bookListEntry, 0, len(bookIDs))
	for _, id := range bookIDs {
		book, err := sm.LoadBookConfig(id)
		if err != nil {
			continue
		}
		nextChapter, _ := sm.GetNextChapterNumber(id)
		chaptersWritten := nextChapter - 1
		if chaptersWritten < 0 {
			chaptersWritten = 0
		}
		entry := bookListEntry{
			ID:       id,
			Title:    book.Title,
			Genre:    string(book.Genre),
			Platform: string(book.Platform),
			Status:   string(book.Status),
			Chapters: chaptersWritten,
		}
		result = append(result, entry)
	}

	if asJSON {
		data, _ := json.MarshalIndent(map[string]any{"books": result}, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if len(result) == 0 {
		fmt.Println("未找到书籍。创建一个书籍： `ipen book create --title ...`")
		return nil
	}
	for _, item := range result {
		fmt.Printf("%s | %s | %s/%s | %s | 章节: %d 章节\n",
			item.ID, item.Title, item.Genre, item.Platform, item.Status, item.Chapters)
	}
	return nil
}

func bookDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <book-id>",
		Short: T(TR.CmdBookDeleteShort),
		Long:  T(TR.CmdBookDeleteLong),
		Args:  cobra.ExactArgs(1),
		RunE:  runBookDelete,
	}

	cmd.Flags().Bool("force", false, "跳过确认提示")
	cmd.Flags().Bool("json", false, "输出 JSON")
	return cmd
}

func runBookDelete(cmd *cobra.Command, args []string) error {
	root, err := findProjectRoot()
	if err != nil {
		return err
	}

	bookID, err := resolveBookID(root, args[0])
	if err != nil {
		return err
	}

	force, _ := cmd.Flags().GetBool("force")
	asJSON, _ := cmd.Flags().GetBool("json")

	sm := state.NewStateManager(root)
	book, err := sm.LoadBookConfig(bookID)
	if err != nil {
		return err
	}
	index, _ := sm.LoadChapterIndex(bookID)

	if !force {
		ok, err := askForConfirmation(
			fmt.Sprintf("删除书籍 %q (%s)? 这将删除 %d 章节。 (y/N) ", book.Title, bookID, len(index)),
		)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("已取消")
			return nil
		}
	}

	if err := os.RemoveAll(sm.BookDir(bookID)); err != nil {
		return err
	}

	if asJSON {
		data, _ := json.MarshalIndent(map[string]any{
			"deleted":  bookID,
			"chapters": len(index),
		}, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("删除书籍 %q (%s): %d 章节已删除。\n", book.Title, bookID, len(index))
	return nil
}
