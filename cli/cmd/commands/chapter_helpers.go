package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/icosmos-space/ipen/core/agents"
	"github.com/icosmos-space/ipen/core/models"
	coreutils "github.com/icosmos-space/ipen/core/utils"
)

var invalidFilenameChars = regexp.MustCompile(`[\\/:*?"<>|]+`)

func chapterFileForNumber(bookDir string, chapterNumber int) (string, error) {
	chaptersDir := filepath.Join(bookDir, "chapters")
	entries, err := os.ReadDir(chaptersDir)
	if err != nil {
		return "", err
	}

	prefix := fmt.Sprintf("%04d_", chapterNumber)
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(strings.ToLower(name), ".md") {
			return filepath.Join(chaptersDir, name), nil
		}
	}

	return "", fmt.Errorf("chapter %d not found", chapterNumber)
}

func readChapterFileForNumber(bookDir string, chapterNumber int) (string, string, error) {
	path, err := chapterFileForNumber(bookDir, chapterNumber)
	if err != nil {
		return "", "", err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	return path, string(content), nil
}

func writeChapterFileForNumber(bookDir string, chapterNumber int, title string, content string) (string, error) {
	chaptersDir := filepath.Join(bookDir, "chapters")
	if err := os.MkdirAll(chaptersDir, 0755); err != nil {
		return "", err
	}

	prefix := fmt.Sprintf("%04d_", chapterNumber)
	entries, _ := os.ReadDir(chaptersDir)
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(strings.ToLower(name), ".md") {
			_ = os.Remove(filepath.Join(chaptersDir, name))
		}
	}

	safeTitle := normalizeChapterFilename(title)
	path := filepath.Join(chaptersDir, fmt.Sprintf("%04d_%s.md", chapterNumber, safeTitle))
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", err
	}
	return path, nil
}

func normalizeChapterFilename(title string) string {
	cleaned := strings.TrimSpace(title)
	if cleaned == "" {
		cleaned = "chapter"
	}
	cleaned = invalidFilenameChars.ReplaceAllString(cleaned, "_")
	cleaned = strings.ReplaceAll(cleaned, " ", "_")
	cleaned = strings.Trim(cleaned, "._-")
	if cleaned == "" {
		return "chapter"
	}
	return cleaned
}

func extractTitleFromMarkdown(content string, fallback string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			title := strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
			if title != "" {
				return title
			}
		}
		if trimmed != "" {
			break
		}
	}
	return fallback
}

func chapterLengthForBook(book *models.BookConfig, content string) int {
	lang := "zh"
	if book != nil && strings.EqualFold(book.Language, "en") {
		lang = "en"
	}
	countingMode := coreutils.ResolveLengthCountingMode(coreutils.LengthLanguage(lang))
	return coreutils.CountChapterLength(content, countingMode)
}

func normalizeRuntimeArtifactPath(raw string) string {
	normalized := filepath.ToSlash(strings.TrimSpace(raw))
	if normalized == "" {
		return ""
	}
	if strings.HasPrefix(normalized, "story/") {
		return normalized
	}
	if strings.HasPrefix(normalized, "runtime/") {
		return "story/" + normalized
	}
	if strings.Contains(normalized, "/story/") {
		idx := strings.Index(normalized, "/story/")
		return normalized[idx+1:]
	}
	return filepath.ToSlash(filepath.Join("story", "runtime", filepath.Base(normalized)))
}

func writeRuntimeArtifact(bookDir string, runtimePath string, payload []byte) (string, error) {
	relative := normalizeRuntimeArtifactPath(runtimePath)
	if relative == "" {
		return "", fmt.Errorf("runtime path is empty")
	}
	target := filepath.Join(bookDir, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(target, payload, 0644); err != nil {
		return "", err
	}
	return relative, nil
}

func parseBookAndOptionalChapter(args []string) (string, *int, error) {
	if len(args) == 0 {
		return "", nil, nil
	}
	if len(args) == 1 {
		if n, err := strconv.Atoi(args[0]); err == nil {
			return "", &n, nil
		}
		return args[0], nil, nil
	}
	if len(args) == 2 {
		chapter, err := strconv.Atoi(args[1])
		if err != nil {
			return "", nil, fmt.Errorf("invalid chapter number %q", args[1])
		}
		return args[0], &chapter, nil
	}
	return "", nil, fmt.Errorf("expected [book-id] [chapter]")
}

func parseChapterRange(raw string) (int, int, error) {
	spec := strings.TrimSpace(raw)
	if spec == "" {
		return 1, int(^uint(0) >> 1), nil
	}
	parts := strings.Split(spec, "-")
	if len(parts) == 1 {
		n, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil || n <= 0 {
			return 0, 0, fmt.Errorf("invalid chapter range %q", raw)
		}
		return n, n, nil
	}
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid chapter range %q", raw)
	}
	start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || start <= 0 {
		return 0, 0, fmt.Errorf("invalid chapter range %q", raw)
	}
	end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || end < start {
		return 0, 0, fmt.Errorf("invalid chapter range %q", raw)
	}
	return start, end, nil
}

func resolveTargetChapter(sm interface {
	GetNextChapterNumber(bookID string) (int, error)
	LoadChapterIndex(bookID string) ([]models.ChapterMeta, error)
}, bookID string, chapter *int) (int, error) {
	if chapter != nil && *chapter > 0 {
		return *chapter, nil
	}
	index, err := sm.LoadChapterIndex(bookID)
	if err == nil && len(index) > 0 {
		maxNumber := 0
		for _, item := range index {
			if item.Number > maxNumber {
				maxNumber = item.Number
			}
		}
		if maxNumber > 0 {
			return maxNumber, nil
		}
	}
	next, err := sm.GetNextChapterNumber(bookID)
	if err != nil {
		return 0, err
	}
	if next <= 1 {
		return 0, fmt.Errorf("no chapters found")
	}
	return next - 1, nil
}

func upsertChapterMeta(index []models.ChapterMeta, meta models.ChapterMeta) []models.ChapterMeta {
	replaced := false
	for i := range index {
		if index[i].Number == meta.Number {
			meta.CreatedAt = index[i].CreatedAt
			index[i] = meta
			replaced = true
			break
		}
	}
	if !replaced {
		index = append(index, meta)
	}
	sort.Slice(index, func(i, j int) bool {
		return index[i].Number < index[j].Number
	})
	return index
}

func readStoryFileOrEmpty(bookDir string, filename string) string {
	data, err := os.ReadFile(filepath.Join(bookDir, "story", filename))
	if err != nil {
		return ""
	}
	return string(data)
}

func formatAuditIssue(issue agents.AuditIssue) string {
	category := strings.TrimSpace(issue.Category)
	description := strings.TrimSpace(issue.Description)
	if category == "" && description == "" {
		return ""
	}
	severity := strings.ToLower(strings.TrimSpace(issue.Severity))
	if severity == "" {
		severity = "warning"
	}
	if category != "" {
		return fmt.Sprintf("[%s] %s: %s", severity, category, description)
	}
	return fmt.Sprintf("[%s] %s", severity, description)
}

func nowOrFallback(ts time.Time) time.Time {
	if ts.IsZero() {
		return time.Now()
	}
	return ts
}
