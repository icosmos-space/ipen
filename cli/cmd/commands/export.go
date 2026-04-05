package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	epub "github.com/go-shiori/go-epub"
	"github.com/icosmos-space/ipen/core/models"
	"github.com/icosmos-space/ipen/core/state"
	"github.com/spf13/cobra"
	"github.com/yuin/goldmark"
)

// ExportCommand 将书籍章节导出为单一文件。
func ExportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export [book-id]",
		Short: T(TR.CmdExportShort),
		Long:  T(TR.CmdExportLong),
		Args:  cobra.MaximumNArgs(1),
		RunE:  runExport,
	}
	cmd.Flags().StringP("format", "f", "txt", "Output format: txt, md, epub")
	cmd.Flags().StringP("output", "o", "", "Output file path")
	cmd.Flags().Bool("approved-only", false, "Only export approved chapters")
	cmd.Flags().Bool("json", false, "Output JSON metadata")
	return cmd
}

func runExport(cmd *cobra.Command, args []string) error {
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

	format, _ := cmd.Flags().GetString("format")
	outputPath, _ := cmd.Flags().GetString("output")
	approvedOnly, _ := cmd.Flags().GetBool("approved-only")
	asJSON, _ := cmd.Flags().GetBool("json")
	format = strings.ToLower(strings.TrimSpace(format))

	if format != "txt" && format != "md" && format != "epub" {
		return fmt.Errorf("unsupported format %q", format)
	}

	sm := state.NewStateManager(root)
	book, err := sm.LoadBookConfig(bookID)
	if err != nil {
		return err
	}
	index, err := sm.LoadChapterIndex(bookID)
	if err != nil {
		return err
	}

	selected := make([]models.ChapterMeta, 0, len(index))
	for _, chapter := range index {
		if approvedOnly && chapter.Status != models.StatusApproved {
			continue
		}
		selected = append(selected, chapter)
	}
	if len(selected) == 0 {
		return fmt.Errorf("no chapters to export")
	}
	sort.Slice(selected, func(i, j int) bool {
		return selected[i].Number < selected[j].Number
	})

	if strings.TrimSpace(outputPath) == "" {
		if format == "epub" {
			outputPath = filepath.Join(root, fmt.Sprintf("%s.epub", bookID))
		} else {
			outputPath = filepath.Join(root, fmt.Sprintf("%s_export.%s", bookID, format))
		}
	}

	totalWords := 0
	for _, chapter := range selected {
		totalWords += chapter.WordCount
	}

	if format == "epub" {
		if err := exportAsEPUB(book.Title, selected, sm.BookDir(bookID), outputPath); err != nil {
			return err
		}
	} else {
		parts := []string{}
		if format == "md" {
			parts = append(parts, "# "+book.Title, "", "---", "")
		} else {
			parts = append(parts, book.Title, "")
		}

		for _, chapter := range selected {
			_, content, err := readChapterFileForNumber(sm.BookDir(bookID), chapter.Number)
			if err != nil {
				continue
			}
			parts = append(parts, strings.TrimSpace(content), "")
		}

		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return err
		}
		finalContent := strings.TrimSpace(strings.Join(parts, "\n")) + "\n"
		if err := os.WriteFile(outputPath, []byte(finalContent), 0644); err != nil {
			return err
		}
	}

	payload := map[string]any{
		"bookId":           bookID,
		"chaptersExported": len(selected),
		"totalWords":       totalWords,
		"format":           format,
		"outputPath":       filepath.ToSlash(outputPath),
	}
	if asJSON {
		data, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Println(string(data))
		return nil
	}
	fmt.Printf("Exported %d chapter(s) (%d words).\n", len(selected), totalWords)
	fmt.Printf("Output: %s\n", filepath.ToSlash(outputPath))
	return nil
}

func exportAsEPUB(bookTitle string, chapters []models.ChapterMeta, bookDir string, outputPath string) error {
	book, err := epub.NewEpub(bookTitle)
	if err != nil {
		return fmt.Errorf("create epub failed: %w", err)
	}
	book.SetLang("zh-CN")

	for _, chapter := range chapters {
		_, markdown, err := readChapterFileForNumber(bookDir, chapter.Number)
		if err != nil {
			continue
		}
		title := extractTitleFromMarkdown(markdown, fmt.Sprintf("Chapter %d", chapter.Number))
		html, err := markdownToHTML(markdown)
		if err != nil {
			return fmt.Errorf("convert chapter %d to html failed: %w", chapter.Number, err)
		}
		if _, err := book.AddSection(html, title, "", ""); err != nil {
			return fmt.Errorf("add chapter %d to epub failed: %w", chapter.Number, err)
		}
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}
	if err := book.Write(outputPath); err != nil {
		return fmt.Errorf("write epub failed: %w", err)
	}
	return nil
}

func markdownToHTML(markdown string) (string, error) {
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(markdown), &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}
