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
	"github.com/spf13/cobra"
)

// FanficCommand 提供同人创作相关命令。
func FanficCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fanfic",
		Short: T(TR.CmdFanficShort),
		Long:  T(TR.CmdFanficLong),
	}
	cmd.AddCommand(fanficInitCommand())
	cmd.AddCommand(fanficShowCommand())
	cmd.AddCommand(fanficRefreshCommand())
	return cmd
}

func fanficInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a fanfic book from source material",
		RunE:  runFanficInit,
	}
	cmd.Flags().String("title", "", "Book title")
	cmd.Flags().String("from", "", "Source file or directory")
	cmd.Flags().String("mode", "canon", "Fanfic mode: canon, au, ooc, cp")
	cmd.Flags().String("genre", "other", "Genre")
	cmd.Flags().String("platform", "other", "Platform")
	cmd.Flags().Int("target-chapters", 100, "Target chapter count")
	cmd.Flags().Int("chapter-words", 3000, "Words per chapter")
	cmd.Flags().String("lang", "", "Writing language: zh or en")
	cmd.Flags().Bool("json", false, "Output JSON")
	_ = cmd.MarkFlagRequired("title")
	_ = cmd.MarkFlagRequired("from")
	return cmd
}

func runFanficInit(cmd *cobra.Command, args []string) error {
	root, err := findProjectRoot()
	if err != nil {
		return err
	}
	config, err := loadConfig(root)
	if err != nil {
		return err
	}

	title, _ := cmd.Flags().GetString("title")
	fromPath, _ := cmd.Flags().GetString("from")
	modeRaw, _ := cmd.Flags().GetString("mode")
	genre, _ := cmd.Flags().GetString("genre")
	platform, _ := cmd.Flags().GetString("platform")
	targetChapters, _ := cmd.Flags().GetInt("target-chapters")
	chapterWords, _ := cmd.Flags().GetInt("chapter-words")
	lang, _ := cmd.Flags().GetString("lang")
	asJSON, _ := cmd.Flags().GetBool("json")

	mode, err := parseFanficMode(modeRaw)
	if err != nil {
		return err
	}

	sourcePath, err := filepath.Abs(fromPath)
	if err != nil {
		return err
	}
	sourceText, sourceName, err := readFanficSourceMaterial(sourcePath)
	if err != nil {
		return err
	}
	if len([]rune(strings.TrimSpace(sourceText))) < 100 {
		return fmt.Errorf("source material is too short (<100 chars)")
	}

	bookID := sanitizeBookID(title)
	sm := state.NewStateManager(root)
	if _, statErr := os.Stat(sm.BookDir(bookID)); statErr == nil {
		return fmt.Errorf("book %q already exists", bookID)
	}

	if strings.TrimSpace(lang) == "" {
		lang = config.Language
	}
	if !strings.EqualFold(lang, "en") {
		lang = "zh"
	}

	now := time.Now()
	book := &models.BookConfig{
		ID:               bookID,
		Title:            title,
		Platform:         models.Platform(platform),
		Genre:            models.Genre(genre),
		Status:           models.StatusOutlining,
		TargetChapters:   targetChapters,
		ChapterWordCount: chapterWords,
		Language:         lang,
		CreatedAt:        now,
		UpdatedAt:        now,
		FanficMode:       mode,
	}
	if err := book.Validate(); err != nil {
		return err
	}
	if err := sm.SaveBookConfig(bookID, book); err != nil {
		return err
	}

	agentCtx := agents.AgentContext{
		Client:      buildPipelineConfig(config, root, true).Client,
		Model:       config.LLM.Model,
		ProjectRoot: root,
		BookID:      bookID,
		Logger:      buildLogger(true),
	}

	canonDoc := fallbackFanficCanon(sourceName, sourceText, mode)
	importer := agents.NewFanficCanonImporter(agentCtx)
	if imported, importErr := importer.ImportFromText(context.Background(), sourceText, sourceName, mode); importErr == nil {
		canonDoc = imported.FullDocument
	}

	architect := agents.NewArchitectAgent(agentCtx)
	foundation, err := architect.GenerateFanficFoundation(context.Background(), book, canonDoc, mode, "")
	if err != nil {
		return err
	}
	bookDir := sm.BookDir(bookID)
	if err := architect.WriteFoundationFiles(bookDir, *foundation, false, book.Language); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(bookDir, "story", "fanfic_canon.md"), []byte(strings.TrimSpace(canonDoc)+"\n"), 0644); err != nil {
		return err
	}
	if err := sm.EnsureControlDocuments(bookID, ""); err != nil {
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

	payload := map[string]any{
		"bookId":      bookID,
		"title":       book.Title,
		"genre":       book.Genre,
		"fanficMode":  mode,
		"source":      sourceName,
		"location":    "books/" + bookID + "/",
		"nextStep":    "ipen write next " + bookID,
		"canonOutput": "story/fanfic_canon.md",
	}
	if asJSON {
		data, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Fanfic created: %s\n", bookID)
	fmt.Printf("Mode: %s\n", mode)
	fmt.Printf("Location: books/%s/\n", bookID)
	fmt.Println("Canon: story/fanfic_canon.md")
	fmt.Printf("Next: ipen write next %s\n", bookID)
	return nil
}

func fanficShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show [book-id]",
		Short: "Show fanfic canon file",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runFanficShow,
	}
	cmd.Flags().Bool("json", false, "Output JSON")
	return cmd
}

func runFanficShow(cmd *cobra.Command, args []string) error {
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
	content, err := os.ReadFile(filepath.Join(sm.BookDir(bookID), "story", "fanfic_canon.md"))
	if err != nil {
		return fmt.Errorf("fanfic canon not found. run `ipen fanfic init` first")
	}
	if asJSON {
		data, _ := json.MarshalIndent(map[string]any{
			"bookId":      bookID,
			"fanficCanon": string(content),
		}, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Fanfic canon for %q:\n\n", bookID)
	fmt.Println(string(content))
	return nil
}

func fanficRefreshCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refresh [book-id]",
		Short: "Refresh fanfic canon from source material",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runFanficRefresh,
	}
	cmd.Flags().String("from", "", "Source file or directory")
	cmd.Flags().Bool("json", false, "Output JSON")
	_ = cmd.MarkFlagRequired("from")
	return cmd
}

func runFanficRefresh(cmd *cobra.Command, args []string) error {
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

	fromPath, _ := cmd.Flags().GetString("from")
	asJSON, _ := cmd.Flags().GetBool("json")

	sourcePath, err := filepath.Abs(fromPath)
	if err != nil {
		return err
	}
	sourceText, sourceName, err := readFanficSourceMaterial(sourcePath)
	if err != nil {
		return err
	}

	sm := state.NewStateManager(root)
	book, err := sm.LoadBookConfig(bookID)
	if err != nil {
		return err
	}
	mode := book.FanficMode
	if mode == "" {
		mode = models.FanficModeCanon
	}

	agentCtx := agents.AgentContext{
		Client:      buildPipelineConfig(config, root, true).Client,
		Model:       config.LLM.Model,
		ProjectRoot: root,
		BookID:      bookID,
		Logger:      buildLogger(true),
	}

	canonDoc := fallbackFanficCanon(sourceName, sourceText, mode)
	importer := agents.NewFanficCanonImporter(agentCtx)
	if imported, importErr := importer.ImportFromText(context.Background(), sourceText, sourceName, mode); importErr == nil {
		canonDoc = imported.FullDocument
	}

	target := filepath.Join(sm.BookDir(bookID), "story", "fanfic_canon.md")
	if err := os.WriteFile(target, []byte(strings.TrimSpace(canonDoc)+"\n"), 0644); err != nil {
		return err
	}
	book.UpdatedAt = time.Now()
	_ = sm.SaveBookConfig(bookID, book)

	payload := map[string]any{
		"bookId":      bookID,
		"source":      sourceName,
		"fanficMode":  mode,
		"canonOutput": "story/fanfic_canon.md",
		"refreshedAt": time.Now().Format(time.RFC3339),
	}
	if asJSON {
		data, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Println(string(data))
		return nil
	}
	fmt.Printf("Fanfic canon refreshed for %q from %s.\n", bookID, sourceName)
	return nil
}

func parseFanficMode(raw string) (models.FanficMode, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "canon":
		return models.FanficModeCanon, nil
	case "au":
		return models.FanficModeAU, nil
	case "ooc":
		return models.FanficModeOOC, nil
	case "cp":
		return models.FanficModeCP, nil
	default:
		return "", fmt.Errorf("invalid fanfic mode %q (allowed: canon, au, ooc, cp)", raw)
	}
}

func readFanficSourceMaterial(path string) (string, string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", "", err
	}
	if !info.IsDir() {
		raw, err := os.ReadFile(path)
		if err != nil {
			return "", "", err
		}
		return string(raw), filepath.Base(path), nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return "", "", err
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := strings.ToLower(entry.Name())
		if strings.HasSuffix(name, ".txt") || strings.HasSuffix(name, ".md") {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)
	if len(files) == 0 {
		return "", "", fmt.Errorf("no .txt/.md files found in %s", path)
	}

	parts := make([]string, 0, len(files))
	for _, name := range files {
		raw, err := os.ReadFile(filepath.Join(path, name))
		if err != nil {
			return "", "", err
		}
		parts = append(parts, string(raw))
	}
	return strings.Join(parts, "\n\n---\n\n"), filepath.Base(path), nil
}

func fallbackFanficCanon(sourceName string, sourceText string, mode models.FanficMode) string {
	preview := sourceText
	if len([]rune(preview)) > 10000 {
		preview = string([]rune(preview)[:10000])
	}
	return strings.Join([]string{
		fmt.Sprintf("# Fanfic Canon (%s)", sourceName),
		"",
		fmt.Sprintf("Mode: %s", mode),
		"",
		"## Notes",
		"Automatic fallback canon (deterministic) because LLM extraction was unavailable.",
		"",
		"## Source Excerpt",
		preview,
	}, "\n")
}
