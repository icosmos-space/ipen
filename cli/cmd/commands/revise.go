package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/icosmos-space/ipen/core/agents"
	"github.com/icosmos-space/ipen/core/models"
	"github.com/icosmos-space/ipen/core/state"
	"github.com/spf13/cobra"
)

// ReviseCommand 基于审计问题修订单个章节。
func ReviseCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "revise [book-id] [chapter]",
		Short:   T(TR.CmdReviseShort),
		Long:    T(TR.CmdReviseLong),
		Args:    cobra.MaximumNArgs(2),
		Example: "  ipen revise my-book 10 --mode rewrite\n  ipen revise 10",
		RunE:    runRevise,
	}
	cmd.Flags().String("mode", string(agents.DEFAULT_REVISE_MODE), "Revise mode: local-fix, polish, rewrite, rework, anti-detect")
	cmd.Flags().Bool("json", false, "Output JSON")
	return cmd
}

func runRevise(cmd *cobra.Command, args []string) error {
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

	modeRaw, _ := cmd.Flags().GetString("mode")
	mode := normalizeReviseModeCLI(modeRaw)
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
	_, originalContent, err := readChapterFileForNumber(bookDir, chapterNumber)
	if err != nil {
		return err
	}

	index, err := sm.LoadChapterIndex(bookID)
	if err != nil {
		return err
	}

	issues := issuesFromChapterIndex(index, chapterNumber)
	if len(issues) == 0 {
		auditor := agents.NewContinuityAuditor(agents.AgentContext{
			Client:      buildPipelineConfig(config, root, true).Client,
			Model:       config.LLM.Model,
			ProjectRoot: root,
			BookID:      bookID,
			Logger:      buildLogger(true),
		})
		auditResult, auditErr := auditor.AuditChapter(
			context.Background(),
			chapterNumber,
			originalContent,
			readStoryFileOrEmpty(bookDir, "current_state.md"),
			readStoryFileOrEmpty(bookDir, "story_bible.md"),
			readStoryFileOrEmpty(bookDir, "chapter_summaries.md"),
		)
		if auditErr == nil {
			issues = auditResult.Issues
		}
	}

	writer := agents.NewWriterAgent(agents.AgentContext{
		Client:      buildPipelineConfig(config, root, true).Client,
		Model:       config.LLM.Model,
		ProjectRoot: root,
		BookID:      bookID,
		Logger:      buildLogger(true),
	})

	reviseResult, err := writer.RepairChapter(context.Background(), agents.RepairChapterInput{
		BookDir:        bookDir,
		ChapterContent: originalContent,
		ChapterNumber:  chapterNumber,
		Issues:         issues,
		Mode:           mode,
		Genre:          string(book.Genre),
	})
	if err != nil {
		return err
	}

	revisedContent := strings.TrimSpace(reviseResult.RevisedContent)
	applied := revisedContent != "" && strings.TrimSpace(originalContent) != revisedContent
	if applied {
		title := extractTitleFromMarkdown(revisedContent, fmt.Sprintf("Chapter %d", chapterNumber))
		if !strings.HasPrefix(revisedContent, "# ") {
			revisedContent = "# " + title + "\n\n" + revisedContent + "\n"
		}
		if _, err := writeChapterFileForNumber(bookDir, chapterNumber, title, revisedContent); err != nil {
			return err
		}
	}

	updatedStatus := models.StatusDrafted
	if applied {
		updatedStatus = models.StatusReadyForReview
	}

	meta := chapterMetaFromIndex(index, chapterNumber)
	now := time.Now()
	if meta == nil {
		meta = &models.ChapterMeta{
			Number:    chapterNumber,
			Title:     extractTitleFromMarkdown(revisedContent, fmt.Sprintf("Chapter %d", chapterNumber)),
			CreatedAt: now,
		}
	}
	meta.UpdatedAt = now
	meta.Status = updatedStatus
	if applied {
		meta.WordCount = chapterLengthForBook(book, revisedContent)
		meta.AuditIssues = []string{}
	}
	if reviseResult.TokenUsage != nil {
		meta.TokenUsage = reviseResult.TokenUsage
	}

	index = upsertChapterMeta(index, *meta)
	if err := sm.SaveChapterIndex(bookID, index); err != nil {
		return err
	}

	payload := map[string]any{
		"bookId":        bookID,
		"chapterNumber": chapterNumber,
		"mode":          mode,
		"applied":       applied,
		"status":        updatedStatus,
		"wordCount":     meta.WordCount,
		"fixedIssues":   summarizeIssues(issues),
	}
	if !applied {
		payload["skippedReason"] = "revised content equals original"
	}

	if asJSON {
		data, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if !applied {
		fmt.Printf("Chapter %d kept original draft.\n", chapterNumber)
		return nil
	}
	fmt.Printf("Chapter %d revised (%s).\n", chapterNumber, mode)
	fmt.Printf("Words: %d\n", meta.WordCount)
	fmt.Printf("Status: %s\n", updatedStatus)
	return nil
}

func normalizeReviseModeCLI(raw string) agents.ReviseMode {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "spot-fix", "local-fix":
		return agents.ReviseModeLocalFix
	case "polish":
		return agents.ReviseModePolish
	case "rewrite":
		return agents.ReviseModeRewrite
	case "rework":
		return agents.ReviseModeRework
	case "anti-detect":
		return agents.ReviseModeAntiDetect
	default:
		return agents.DEFAULT_REVISE_MODE
	}
}

func issuesFromChapterIndex(index []models.ChapterMeta, chapterNumber int) []agents.AuditIssue {
	for _, chapter := range index {
		if chapter.Number != chapterNumber {
			continue
		}
		result := make([]agents.AuditIssue, 0, len(chapter.AuditIssues))
		for _, issue := range chapter.AuditIssues {
			result = append(result, agents.AuditIssue{
				Severity:    "warning",
				Category:    "audit",
				Description: issue,
				Suggestion:  "Fix continuity and style issues while preserving canonical facts.",
			})
		}
		return result
	}
	return []agents.AuditIssue{}
}

func chapterMetaFromIndex(index []models.ChapterMeta, chapterNumber int) *models.ChapterMeta {
	for i := range index {
		if index[i].Number == chapterNumber {
			copied := index[i]
			return &copied
		}
	}
	return nil
}

func summarizeIssues(issues []agents.AuditIssue) []string {
	out := make([]string, 0, len(issues))
	for _, issue := range issues {
		line := strings.TrimSpace(issue.Description)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}
