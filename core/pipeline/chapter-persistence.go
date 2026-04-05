package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/icosmos-space/ipen/core/models"
	"github.com/icosmos-space/ipen/core/utils"
)

// ChapterPersistence 处理chapter file persistence。
type ChapterPersistence struct {
	BookDir string
	Logger  utils.Logger
}

// NewChapterPersistence 创建新的chapter persistence handler。
func NewChapterPersistence(bookDir string, logger utils.Logger) *ChapterPersistence {
	return &ChapterPersistence{
		BookDir: bookDir,
		Logger:  logger,
	}
}

// SaveChapter saves chapter content to file
func (cp *ChapterPersistence) SaveChapter(chapterNumber int, title, content string) error {
	chaptersDir := filepath.Join(cp.BookDir, "chapters")
	if err := os.MkdirAll(chaptersDir, 0755); err != nil {
		return fmt.Errorf("failed to create chapters directory: %w", err)
	}

	filename := fmt.Sprintf("%04d_%s.md", chapterNumber, sanitizeFilename(title))
	filepath := filepath.Join(chaptersDir, filename)

	if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write chapter file: %w", err)
	}

	cp.Logger.Info("Chapter saved", map[string]any{
		"chapter":  chapterNumber,
		"filepath": filepath,
	})

	return nil
}

// LoadChapter 加载chapter content from file。
func (cp *ChapterPersistence) LoadChapter(chapterNumber int) (string, string, error) {
	chaptersDir := filepath.Join(cp.BookDir, "chapters")
	files, err := os.ReadDir(chaptersDir)
	if err != nil {
		return "", "", fmt.Errorf("failed to read chapters directory: %w", err)
	}

	prefix := fmt.Sprintf("%04d_", chapterNumber)
	for _, file := range files {
		if file.Name()[:4] == prefix {
			data, err := os.ReadFile(filepath.Join(chaptersDir, file.Name()))
			if err != nil {
				return "", "", err
			}

			// Extract title from filename
			title := file.Name()[5:]
			if idx := len(title) - 3; idx > 0 && title[idx:] == ".md" {
				title = title[:idx]
			}

			return title, string(data), nil
		}
	}

	return "", "", fmt.Errorf("chapter %d not found", chapterNumber)
}

// ListChapters lists all chapters
func (cp *ChapterPersistence) ListChapters() ([]models.ChapterMeta, error) {
	chaptersDir := filepath.Join(cp.BookDir, "chapters")
	files, err := os.ReadDir(chaptersDir)
	if err != nil {
		return nil, err
	}

	var chapters []models.ChapterMeta
	for _, file := range files {
		if len(file.Name()) < 4 || file.Name()[len(file.Name())-3:] != ".md" {
			continue
		}

		// Parse chapter number from filename
		if num, err := parseChapterFromFilename(file.Name()); err == nil {
			info, err := file.Info()
			if err != nil {
				continue
			}

			chapters = append(chapters, models.ChapterMeta{
				Number:    num,
				Title:     extractTitleFromFilename(file.Name()),
				Status:    models.StatusImported,
				WordCount: 0, // Would need to read file to count
				CreatedAt: info.ModTime(),
				UpdatedAt: info.ModTime(),
			})
		}
	}

	return chapters, nil
}

// DeleteChapter deletes a chapter file
func (cp *ChapterPersistence) DeleteChapter(chapterNumber int) error {
	chaptersDir := filepath.Join(cp.BookDir, "chapters")
	files, err := os.ReadDir(chaptersDir)
	if err != nil {
		return err
	}

	prefix := fmt.Sprintf("%04d_", chapterNumber)
	for _, file := range files {
		if file.Name()[:4] == prefix {
			return os.Remove(filepath.Join(chaptersDir, file.Name()))
		}
	}

	return nil
}

func sanitizeFilename(title string) string {
	// Remove or replace invalid characters
	result := title
	for _, ch := range []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"} {
		result = strings.ReplaceAll(result, ch, "_")
	}
	return result
}

func parseChapterFromFilename(filename string) (int, error) {
	if len(filename) < 4 {
		return 0, fmt.Errorf("invalid filename")
	}

	// Extract first 4 digits
	numStr := filename[:4]
	var num int
	for _, ch := range numStr {
		num = num*10 + int(ch-'0')
	}

	return num, nil
}

func extractTitleFromFilename(filename string) string {
	if len(filename) < 5 {
		return filename
	}

	title := filename[5:]
	if idx := len(title) - 3; idx > 0 && title[idx:] == ".md" {
		title = title[:idx]
	}

	return title
}

// PersistChapterArtifacts persists all chapter artifacts
func PersistChapterArtifacts(ctx context.Context, bookDir string, chapterNumber int,
	content, currentState, pendingHooks, chapterSummary string) error {

	runtimeDir := filepath.Join(bookDir, "story", "runtime")
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		return err
	}

	// Save chapter content
	chapterFile := filepath.Join(runtimeDir, fmt.Sprintf("chapter-%04d.md", chapterNumber))
	if err := os.WriteFile(chapterFile, []byte(content), 0644); err != nil {
		return err
	}

	// Save runtime state
	stateFile := filepath.Join(runtimeDir, fmt.Sprintf("chapter-%04d.state.json", chapterNumber))
	stateData := map[string]any{
		"chapter":        chapterNumber,
		"timestamp":      time.Now().Format(time.RFC3339),
		"currentState":   currentState,
		"pendingHooks":   pendingHooks,
		"chapterSummary": chapterSummary,
	}

	stateJSON, err := toJSON(stateData)
	if err != nil {
		return err
	}

	return os.WriteFile(stateFile, stateJSON, 0644)
}

func toJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}
