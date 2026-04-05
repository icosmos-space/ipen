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
	"github.com/spf13/cobra"
)

// AuditCommand 审计单章连贯性并更新章节状态。
func AuditCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "audit [book-id] [chapter]",
		Short:   T(TR.CmdAuditShort),
		Long:    T(TR.CmdAuditLong),
		Args:    cobra.MaximumNArgs(2),
		Example: "  ipen audit my-book 12\n  ipen audit 12\n  ipen audit",
		RunE:    runAudit,
	}
	cmd.Flags().Bool("json", false, "Output JSON")
	return cmd
}

func runAudit(cmd *cobra.Command, args []string) error {
	root, err := findProjectRoot()
	if err != nil {
		return err
	}
	config, err := loadConfig(root)
	if err != nil {
		return err
	}

	bookArg, chapterArg, err := parseBookAndOptionalChapter(args)
	if err != nil {
		return err
	}
	bookID, err := resolveBookID(root, bookArg)
	if err != nil {
		return err
	}

	asJSON, _ := cmd.Flags().GetBool("json")
	sm := state.NewStateManager(root)
	chapterNumber, err := resolveTargetChapter(sm, bookID, chapterArg)
	if err != nil {
		return err
	}

	book, err := sm.LoadBookConfig(bookID)
	if err != nil {
		return err
	}
	bookDir := sm.BookDir(bookID)
	chapterPath, chapterContent, err := readChapterFileForNumber(bookDir, chapterNumber)
	if err != nil {
		return err
	}

	auditor := agents.NewContinuityAuditor(agents.AgentContext{
		Client:      buildPipelineConfig(config, root, true).Client,
		Model:       config.LLM.Model,
		ProjectRoot: root,
		BookID:      bookID,
		Logger:      buildLogger(true),
	})

	result, err := auditor.AuditChapter(
		context.Background(),
		chapterNumber,
		chapterContent,
		readStoryFileOrEmpty(bookDir, "current_state.md"),
		readStoryFileOrEmpty(bookDir, "story_bible.md"),
		readStoryFileOrEmpty(bookDir, "chapter_summaries.md"),
	)
	if err != nil {
		return err
	}

	index, err := sm.LoadChapterIndex(bookID)
	if err != nil {
		return err
	}

	issues := make([]string, 0, len(result.Issues))
	for _, issue := range result.Issues {
		formatted := formatAuditIssue(issue)
		if strings.TrimSpace(formatted) != "" {
			issues = append(issues, formatted)
		}
	}

	title := extractTitleFromMarkdown(chapterContent, strings.TrimSuffix(filepath.Base(chapterPath), ".md"))
	status := models.StatusReadyForReview
	if !result.Passed {
		status = models.StatusAuditFailed
	}

	now := time.Now()
	meta := models.ChapterMeta{
		Number:      chapterNumber,
		Title:       title,
		Status:      status,
		WordCount:   chapterLengthForBook(book, chapterContent),
		CreatedAt:   now,
		UpdatedAt:   now,
		AuditIssues: issues,
		TokenUsage:  result.TokenUsage,
	}
	index = upsertChapterMeta(index, meta)
	if err := sm.SaveChapterIndex(bookID, index); err != nil {
		return err
	}

	payload := map[string]any{
		"bookId":        bookID,
		"chapterNumber": chapterNumber,
		"passed":        result.Passed,
		"summary":       result.Summary,
		"issues":        result.Issues,
		"status":        string(status),
	}

	if asJSON {
		data, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Audit chapter %d for %q: %s\n", chapterNumber, bookID, map[bool]string{true: "PASSED", false: "FAILED"}[result.Passed])
	fmt.Printf("Summary: %s\n", result.Summary)
	if len(result.Issues) > 0 {
		fmt.Println("Issues:")
		for _, issue := range result.Issues {
			fmt.Printf("  - [%s] %s: %s\n", issue.Severity, issue.Category, issue.Description)
		}
	}
	return nil
}
