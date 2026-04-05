package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/icosmos-space/ipen/core/models"
)

// StateManager 管理book state and control documents。
type StateManager struct {
	ProjectRoot string
}

// NewStateManager 创建新的state manager。
func NewStateManager(projectRoot string) *StateManager {
	return &StateManager{
		ProjectRoot: projectRoot,
	}
}

func defaultAuthorIntent(language string) string {
	if language == "zh" {
		return "# 作者意图\n\n（在这里描述这本书的长期创作方向。）\n"
	}
	return "# Author Intent\n\n(Describe the long-horizon vision for this book here.)\n"
}

func defaultCurrentFocus(language string) string {
	if language == "zh" {
		return "# 当前聚焦\n\n## 当前重点\n\n（描述接下来 1-3 章最需要优先推进的内容。）\n"
	}
	return "# Current Focus\n\n## Active Focus\n\n(Describe what the next 1-3 chapters should prioritize.)\n"
}

// EnsureControlDocuments ensures control documents exist for a book
func (sm *StateManager) EnsureControlDocuments(bookID string, authorIntent string) error {
	language, err := sm.resolveControlDocumentLanguage(bookID)
	if err != nil {
		language = "en"
	}
	return sm.EnsureControlDocumentsAt(sm.BookDir(bookID), language, authorIntent)
}

// EnsureControlDocumentsAt ensures control documents at a specific directory
func (sm *StateManager) EnsureControlDocumentsAt(bookDir string, language string, authorIntent string) error {
	storyDir := filepath.Join(bookDir, "story")
	runtimeDir := filepath.Join(storyDir, "runtime")

	if err := os.MkdirAll(storyDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(runtimeDir, 0755); err != nil {
		return err
	}

	authorIntentContent := authorIntent
	if strings.TrimSpace(authorIntent) == "" {
		authorIntentContent = defaultAuthorIntent(language)
	} else {
		authorIntentContent = strings.TrimRight(authorIntent, "\n") + "\n"
	}

	if err := sm.writeIfMissing(filepath.Join(storyDir, "author_intent.md"), authorIntentContent); err != nil {
		return err
	}

	return sm.writeIfMissing(filepath.Join(storyDir, "current_focus.md"), defaultCurrentFocus(language))
}

// LoadControlDocuments 加载control documents for a book。
func (sm *StateManager) LoadControlDocuments(bookID string) (authorIntent string, currentFocus string, runtimeDir string, err error) {
	if err = sm.EnsureControlDocuments(bookID, ""); err != nil {
		return
	}

	storyDir := filepath.Join(sm.BookDir(bookID), "story")
	runtimeDir = filepath.Join(storyDir, "runtime")

	authorIntentBytes, err := os.ReadFile(filepath.Join(storyDir, "author_intent.md"))
	if err != nil {
		return
	}
	authorIntent = string(authorIntentBytes)

	currentFocusBytes, err := os.ReadFile(filepath.Join(storyDir, "current_focus.md"))
	if err != nil {
		return
	}
	currentFocus = string(currentFocusBytes)

	return
}

func (sm *StateManager) resolveControlDocumentLanguage(bookID string) (string, error) {
	bookJSONPath := filepath.Join(sm.BookDir(bookID), "book.json")
	data, err := os.ReadFile(bookJSONPath)
	if err != nil {
		return "en", err
	}

	var config map[string]any
	if err = json.Unmarshal(data, &config); err != nil {
		return "en", err
	}

	if lang, ok := config["language"].(string); ok && lang == "zh" {
		return "zh", nil
	}
	return "en", nil
}

// BooksDir 返回the books directory。
func (sm *StateManager) BooksDir() string {
	return filepath.Join(sm.ProjectRoot, "books")
}

// BookDir 返回the book directory。
func (sm *StateManager) BookDir(bookID string) string {
	return filepath.Join(sm.BooksDir(), bookID)
}

// StateDir 返回the state directory。
func (sm *StateManager) StateDir(bookID string) string {
	return filepath.Join(sm.BookDir(bookID), "story", "state")
}

// LoadProjectConfig 加载ipen.json。
func (sm *StateManager) LoadProjectConfig() (map[string]any, error) {
	configPath := filepath.Join(sm.ProjectRoot, "ipen.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config map[string]any
	if err = json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return config, nil
}

// SaveProjectConfig saves ipen.json.
func (sm *StateManager) SaveProjectConfig(config map[string]any) error {
	configPath := filepath.Join(sm.ProjectRoot, "ipen.json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

// EnsureRuntimeState bootstraps structured runtime state json files.
func (sm *StateManager) EnsureRuntimeState(bookID string, fallbackChapter int) error {
	_, err := BootstrapStructuredStateFromMarkdown(sm.BookDir(bookID), fallbackChapter)
	return err
}

// AcquireBookLock acquires a simple write lock file and returns unlock callback.
func (sm *StateManager) AcquireBookLock(bookID string) (func() error, error) {
	bookDir := sm.BookDir(bookID)
	if err := os.MkdirAll(bookDir, 0755); err != nil {
		return nil, err
	}

	lockPath := filepath.Join(bookDir, ".write.lock")
	fd, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("book %q is locked by another process: %w", bookID, err)
	}
	defer fd.Close()

	_, _ = fd.WriteString(fmt.Sprintf("pid:%d ts:%d\n", os.Getpid(), time.Now().UnixMilli()))
	return func() error {
		if removeErr := os.Remove(lockPath); removeErr != nil && !os.IsNotExist(removeErr) {
			return removeErr
		}
		return nil
	}, nil
}

// LoadBookConfig 加载book configuration。
func (sm *StateManager) LoadBookConfig(bookID string) (*models.BookConfig, error) {
	configPath := filepath.Join(sm.BookDir(bookID), "book.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, fmt.Errorf("book.json is empty for book %q", bookID)
	}

	var config models.BookConfig
	if err = json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// SaveBookConfig saves book configuration
func (sm *StateManager) SaveBookConfig(bookID string, config *models.BookConfig) error {
	return sm.SaveBookConfigAt(sm.BookDir(bookID), config)
}

// SaveBookConfigAt saves book config at a specific directory
func (sm *StateManager) SaveBookConfigAt(bookDir string, config *models.BookConfig) error {
	if err := os.MkdirAll(bookDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(bookDir, "book.json"), data, 0644)
}

// ListBooks lists all books
func (sm *StateManager) ListBooks() ([]string, error) {
	entries, err := os.ReadDir(sm.BooksDir())
	if err != nil {
		return []string{}, nil
	}

	var bookIDs []string
	for _, entry := range entries {
		bookJSONPath := filepath.Join(sm.BooksDir(), entry.Name(), "book.json")
		if _, err := os.Stat(bookJSONPath); err == nil {
			bookIDs = append(bookIDs, entry.Name())
		}
	}
	return bookIDs, nil
}

// GetNextChapterNumber gets the next chapter number
func (sm *StateManager) GetNextChapterNumber(bookID string) (int, error) {
	durableChapter, err := resolveDurableStoryProgress(sm.BookDir(bookID))
	if err != nil {
		durableChapter = 0
	}
	return durableChapter + 1, nil
}

// GetPersistedChapterCount gets the persisted chapter count
func (sm *StateManager) GetPersistedChapterCount(bookID string) int {
	chaptersDir := filepath.Join(sm.BookDir(bookID), "chapters")
	chapterNumbers := make(map[int]bool)

	files, err := os.ReadDir(chaptersDir)
	if err != nil {
		return 0
	}

	for _, file := range files {
		if matched, _ := filepath.Match("*_*.md", file.Name()); matched {
			parts := strings.SplitN(file.Name(), "_", 2)
			if num, err := strconv.Atoi(parts[0]); err == nil {
				chapterNumbers[num] = true
			}
		}
	}

	return len(chapterNumbers)
}

// LoadChapterIndex 加载the chapter index。
func (sm *StateManager) LoadChapterIndex(bookID string) ([]models.ChapterMeta, error) {
	indexPath := filepath.Join(sm.BookDir(bookID), "chapters", "index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return []models.ChapterMeta{}, nil
	}

	var index []models.ChapterMeta
	if err = json.Unmarshal(data, &index); err != nil {
		return []models.ChapterMeta{}, err
	}
	return index, nil
}

// SaveChapterIndex saves the chapter index
func (sm *StateManager) SaveChapterIndex(bookID string, index []models.ChapterMeta) error {
	return sm.SaveChapterIndexAt(sm.BookDir(bookID), index)
}

// SaveChapterIndexAt saves the chapter index at a specific directory
func (sm *StateManager) SaveChapterIndexAt(bookDir string, index []models.ChapterMeta) error {
	chaptersDir := filepath.Join(bookDir, "chapters")
	if err := os.MkdirAll(chaptersDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(chaptersDir, "index.json"), data, 0644)
}

// SnapshotState 创建a state snapshot。
func (sm *StateManager) SnapshotState(bookID string, chapterNumber int) error {
	return sm.SnapshotStateAt(sm.BookDir(bookID), chapterNumber)
}

// SnapshotStateAt 创建a state snapshot at a specific directory。
func (sm *StateManager) SnapshotStateAt(bookDir string, chapterNumber int) error {
	storyDir := filepath.Join(bookDir, "story")
	snapshotDir := filepath.Join(storyDir, "snapshots", strconv.Itoa(chapterNumber))

	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return err
	}

	files := []string{
		"current_state.md", "particle_ledger.md", "pending_hooks.md",
		"chapter_summaries.md", "subplot_board.md", "emotional_arcs.md", "character_matrix.md",
	}

	for _, f := range files {
		content, err := os.ReadFile(filepath.Join(storyDir, f))
		if err != nil {
			continue // file doesn't exist yet
		}
		if err = os.WriteFile(filepath.Join(snapshotDir, f), content, 0644); err != nil {
			return err
		}
	}

	// Copy state directory files
	stateDir := filepath.Join(bookDir, "story", "state")
	snapshotStateDir := filepath.Join(snapshotDir, "state")

	stateFiles, err := os.ReadDir(stateDir)
	if err != nil {
		return nil // state directory missing - skip
	}

	if len(stateFiles) > 0 {
		if err = os.MkdirAll(snapshotStateDir, 0755); err != nil {
			return err
		}

		for _, sf := range stateFiles {
			content, err := os.ReadFile(filepath.Join(stateDir, sf.Name()))
			if err != nil {
				continue
			}
			if err = os.WriteFile(filepath.Join(snapshotStateDir, sf.Name()), content, 0644); err != nil {
				return err
			}
		}
	}

	return nil
}

// IsCompleteBookDirectory 检查if a directory is a complete book。
func (sm *StateManager) IsCompleteBookDirectory(bookDir string) bool {
	requiredPaths := []string{
		filepath.Join(bookDir, "book.json"),
		filepath.Join(bookDir, "story", "story_bible.md"),
		filepath.Join(bookDir, "story", "volume_outline.md"),
		filepath.Join(bookDir, "story", "book_rules.md"),
		filepath.Join(bookDir, "story", "current_state.md"),
		filepath.Join(bookDir, "story", "pending_hooks.md"),
		filepath.Join(bookDir, "chapters", "index.json"),
	}

	for _, path := range requiredPaths {
		if _, err := os.Stat(path); err != nil {
			return false
		}
	}
	return true
}

// RestoreState restores state from a snapshot
func (sm *StateManager) RestoreState(bookID string, chapterNumber int) (bool, error) {
	storyDir := filepath.Join(sm.BookDir(bookID), "story")
	snapshotDir := filepath.Join(storyDir, "snapshots", strconv.Itoa(chapterNumber))

	files := []string{
		"current_state.md", "particle_ledger.md", "pending_hooks.md",
		"chapter_summaries.md", "subplot_board.md", "emotional_arcs.md", "character_matrix.md",
	}

	// Required files
	requiredFiles := []string{"current_state.md", "pending_hooks.md"}
	requiredSet := make(map[string]bool)
	for _, f := range requiredFiles {
		requiredSet[f] = true
	}

	// Restore required files
	for _, f := range requiredFiles {
		content, err := os.ReadFile(filepath.Join(snapshotDir, f))
		if err != nil {
			return false, err
		}
		if err = os.WriteFile(filepath.Join(storyDir, f), content, 0644); err != nil {
			return false, err
		}
	}

	// Restore optional files
	for _, f := range files {
		if requiredSet[f] {
			continue
		}
		content, err := os.ReadFile(filepath.Join(snapshotDir, f))
		if err != nil {
			continue // optional file missing
		}
		if err = os.WriteFile(filepath.Join(storyDir, f), content, 0644); err != nil {
			return false, err
		}
	}

	// Restore structured state when available, otherwise drop stale structured state.
	stateDir := sm.StateDir(bookID)
	snapshotStateDir := filepath.Join(snapshotDir, "state")
	restoredStructuredState := false
	if stateFiles, readErr := os.ReadDir(snapshotStateDir); readErr == nil && len(stateFiles) > 0 {
		restoredStructuredState = true
		if err := os.MkdirAll(stateDir, 0755); err != nil {
			return false, err
		}
		for _, file := range stateFiles {
			content, err := os.ReadFile(filepath.Join(snapshotStateDir, file.Name()))
			if err != nil {
				return false, err
			}
			if err = os.WriteFile(filepath.Join(stateDir, file.Name()), content, 0644); err != nil {
				return false, err
			}
		}
	}
	if !restoredStructuredState {
		_ = os.RemoveAll(stateDir)
	}

	return true, nil
}

// RollbackToChapter rolls back state to a specific chapter
func (sm *StateManager) RollbackToChapter(bookID string, targetChapter int) ([]int, error) {
	restored, err := sm.RestoreState(bookID, targetChapter)
	if err != nil || !restored {
		return nil, fmt.Errorf("cannot restore snapshot for chapter %d in %q", targetChapter, bookID)
	}

	bookDir := sm.BookDir(bookID)
	chaptersDir := filepath.Join(bookDir, "chapters")
	index, err := sm.LoadChapterIndex(bookID)
	if err != nil {
		return nil, err
	}

	var kept []models.ChapterMeta
	var discarded []int

	for _, entry := range index {
		if entry.Number <= targetChapter {
			kept = append(kept, entry)
		} else {
			discarded = append(discarded, entry.Number)
		}
	}

	// Delete chapter markdown files for discarded chapters
	files, err := os.ReadDir(chaptersDir)
	if err == nil {
		for _, file := range files {
			if matched, _ := filepath.Match("*_*.md", file.Name()); matched {
				parts := strings.SplitN(file.Name(), "_", 2)
				if num, err := strconv.Atoi(parts[0]); err == nil && num > targetChapter {
					os.Remove(filepath.Join(chaptersDir, file.Name()))
				}
			}
		}
	}

	// Delete snapshots for discarded chapters
	snapshotsDir := filepath.Join(bookDir, "story", "snapshots")
	if snapshots, err := os.ReadDir(snapshotsDir); err == nil {
		for _, snap := range snapshots {
			if num, err := strconv.Atoi(snap.Name()); err == nil && num > targetChapter {
				os.RemoveAll(filepath.Join(snapshotsDir, snap.Name()))
			}
		}
	}

	// Delete runtime artifacts for discarded chapters.
	runtimeDir := filepath.Join(bookDir, "story", "runtime")
	if runtimeFiles, err := os.ReadDir(runtimeDir); err == nil {
		for _, file := range runtimeFiles {
			matches := regexp.MustCompile(`^chapter-(\d+)\.`).FindStringSubmatch(file.Name())
			if len(matches) == 2 {
				if num, convErr := strconv.Atoi(matches[1]); convErr == nil && num > targetChapter {
					_ = os.Remove(filepath.Join(runtimeDir, file.Name()))
				}
			}
		}
	}

	// Delete drafts for discarded chapters.
	draftsDir := filepath.Join(bookDir, "story", "drafts")
	if draftFiles, err := os.ReadDir(draftsDir); err == nil {
		for _, file := range draftFiles {
			if matched, _ := filepath.Match("*_*.md", file.Name()); matched {
				parts := strings.SplitN(file.Name(), "_", 2)
				if num, convErr := strconv.Atoi(parts[0]); convErr == nil && num > targetChapter {
					_ = os.Remove(filepath.Join(draftsDir, file.Name()))
				}
			}
		}
	}

	// Drop sqlite sidecar cache to prevent stale retrieval artifacts.
	_ = os.Remove(filepath.Join(bookDir, "story", "memory.db"))
	_ = os.Remove(filepath.Join(bookDir, "story", "memory.db-shm"))
	_ = os.Remove(filepath.Join(bookDir, "story", "memory.db-wal"))

	if err = sm.SaveChapterIndex(bookID, kept); err != nil {
		return nil, err
	}

	return discarded, nil
}

func (sm *StateManager) writeIfMissing(path string, content string) error {
	if _, err := os.Stat(path); err != nil {
		return os.WriteFile(path, []byte(content), 0644)
	}
	return nil
}

// resolveDurableStoryProgress 解析the durable story progress。
func resolveDurableStoryProgress(bookDir string) (int, error) {
	// Resolve durable progress from existing chapter files.
	chaptersDir := filepath.Join(bookDir, "chapters")
	files, err := os.ReadDir(chaptersDir)
	if err != nil {
		return 0, err
	}

	maxChapter := 0
	for _, file := range files {
		if matched, _ := filepath.Match("*_*.md", file.Name()); matched {
			parts := strings.SplitN(file.Name(), "_", 2)
			if num, err := strconv.Atoi(parts[0]); err == nil && num > maxChapter {
				maxChapter = num
			}
		}
	}

	return maxChapter, nil
}
