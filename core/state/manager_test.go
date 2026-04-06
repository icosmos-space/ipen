package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/icosmos-space/ipen/core/models"
)

// ========== NewStateManager ==========

func TestNewStateManager_Initializes(t *testing.T) {
	sm := NewStateManager("/tmp/test")
	if sm.ProjectRoot != "/tmp/test" {
		t.Fatalf("expected project root /tmp/test, got %s", sm.ProjectRoot)
	}
}

// ========== EnsureControlDocuments ==========

func TestEnsureControlDocuments_CreatesFiles(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)

	bookID := "test-book"
	bookDir := filepath.Join(dir, "books", bookID)
	if err := os.MkdirAll(bookDir, 0755); err != nil {
		t.Fatalf("failed to create book dir: %v", err)
	}
	bookConfig := map[string]string{"language": "en"}
	writeTestJSON(t, filepath.Join(bookDir, "book.json"), bookConfig)

	err := sm.EnsureControlDocuments(bookID, "")
	if err != nil {
		t.Fatalf("EnsureControlDocuments failed: %v", err)
	}

	storyDir := filepath.Join(bookDir, "story")
	if _, err := os.Stat(filepath.Join(storyDir, "author_intent.md")); err != nil {
		t.Fatal("expected author_intent.md to be created")
	}
	if _, err := os.Stat(filepath.Join(storyDir, "current_focus.md")); err != nil {
		t.Fatal("expected current_focus.md to be created")
	}
}

func TestEnsureControlDocuments_UsesAuthorIntent(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)

	bookID := "test-book"
	bookDir := filepath.Join(dir, "books", bookID)
	if err := os.MkdirAll(bookDir, 0755); err != nil {
		t.Fatalf("failed to create book dir: %v", err)
	}
	bookConfig := map[string]string{"language": "en"}
	writeTestJSON(t, filepath.Join(bookDir, "book.json"), bookConfig)

	customIntent := "# My Intent\n\nThis is my custom intent."
	err := sm.EnsureControlDocuments(bookID, customIntent)
	if err != nil {
		t.Fatalf("EnsureControlDocuments failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(bookDir, "story", "author_intent.md"))
	if err != nil {
		t.Fatalf("failed to read author_intent.md: %v", err)
	}

	if string(content) != customIntent+"\n" {
		t.Fatalf("expected content %q, got %q", customIntent+"\n", string(content))
	}
}

func TestEnsureControlDocuments_ChineseLanguage(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)

	bookID := "test-book"
	bookDir := filepath.Join(dir, "books", bookID)
	if err := os.MkdirAll(bookDir, 0755); err != nil {
		t.Fatalf("failed to create book dir: %v", err)
	}
	bookConfig := map[string]string{"language": "zh"}
	writeTestJSON(t, filepath.Join(bookDir, "book.json"), bookConfig)

	err := sm.EnsureControlDocuments(bookID, "")
	if err != nil {
		t.Fatalf("EnsureControlDocuments failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(bookDir, "story", "author_intent.md"))
	if err != nil {
		t.Fatalf("failed to read author_intent.md: %v", err)
	}

	if !stringsContainsGo(string(content), "作者意图") {
		t.Fatalf("expected Chinese default author intent, got:\n%s", string(content))
	}
}

func TestEnsureControlDocuments_DoesNotOverwriteExisting(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)

	bookID := "test-book"
	bookDir := filepath.Join(dir, "books", bookID)
	storyDir := filepath.Join(bookDir, "story")
	if err := os.MkdirAll(storyDir, 0755); err != nil {
		t.Fatalf("failed to create book dir: %v", err)
	}

	// Create existing file
	existingContent := "# Existing Intent\n"
	if err := os.WriteFile(filepath.Join(storyDir, "author_intent.md"), []byte(existingContent), 0644); err != nil {
		t.Fatalf("failed to write existing file: %v", err)
	}

	bookConfig := map[string]string{"language": "en"}
	writeTestJSON(t, filepath.Join(bookDir, "book.json"), bookConfig)

	err := sm.EnsureControlDocuments(bookID, "New Intent")
	if err != nil {
		t.Fatalf("EnsureControlDocuments failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(storyDir, "author_intent.md"))
	if err != nil {
		t.Fatalf("failed to read author_intent.md: %v", err)
	}

	if string(content) != existingContent {
		t.Fatalf("expected existing content to be preserved, got %q", string(content))
	}
}

// ========== LoadControlDocuments ==========

func TestLoadControlDocuments_ReturnsContent(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)

	bookID := "test-book"
	bookDir := filepath.Join(dir, "books", bookID)
	storyDir := filepath.Join(bookDir, "story")
	if err := os.MkdirAll(storyDir, 0755); err != nil {
		t.Fatalf("failed to create book dir: %v", err)
	}

	authorIntent := "# Intent\nTest intent."
	currentFocus := "# Focus\nTest focus."
	if err := os.WriteFile(filepath.Join(storyDir, "author_intent.md"), []byte(authorIntent), 0644); err != nil {
		t.Fatalf("failed to write author_intent.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(storyDir, "current_focus.md"), []byte(currentFocus), 0644); err != nil {
		t.Fatalf("failed to write current_focus.md: %v", err)
	}

	bookConfig := map[string]string{"language": "en"}
	writeTestJSON(t, filepath.Join(bookDir, "book.json"), bookConfig)

	loadedIntent, loadedFocus, _, err := sm.LoadControlDocuments(bookID)
	if err != nil {
		t.Fatalf("LoadControlDocuments failed: %v", err)
	}

	if loadedIntent != authorIntent {
		t.Fatalf("expected intent %q, got %q", authorIntent, loadedIntent)
	}
	if loadedFocus != currentFocus {
		t.Fatalf("expected focus %q, got %q", currentFocus, loadedFocus)
	}
}

// ========== BooksDir / BookDir / StateDir ==========

func TestBooksDir_ReturnsCorrectPath(t *testing.T) {
	sm := NewStateManager("/project")
	expected := filepath.Join("/project", "books")
	if sm.BooksDir() != expected {
		t.Fatalf("expected %s, got %s", expected, sm.BooksDir())
	}
}

func TestBookDir_ReturnsCorrectPath(t *testing.T) {
	sm := NewStateManager("/project")
	expected := filepath.Join("/project", "books", "my-book")
	if sm.BookDir("my-book") != expected {
		t.Fatalf("expected %s, got %s", expected, sm.BookDir("my-book"))
	}
}

func TestStateDir_ReturnsCorrectPath(t *testing.T) {
	sm := NewStateManager("/project")
	expected := filepath.Join("/project", "books", "my-book", "story", "state")
	if sm.StateDir("my-book") != expected {
		t.Fatalf("expected %s, got %s", expected, sm.StateDir("my-book"))
	}
}

// ========== Project Config ==========

func TestLoadProjectConfig_LoadsConfig(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)

	config := map[string]any{
		"version": "1.0",
		"theme":   "dark",
	}
	data, _ := json.MarshalIndent(config, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "ipen.json"), data, 0644); err != nil {
		t.Fatalf("failed to write ipen.json: %v", err)
	}

	loaded, err := sm.LoadProjectConfig()
	if err != nil {
		t.Fatalf("LoadProjectConfig failed: %v", err)
	}

	if loaded["version"] != "1.0" {
		t.Fatalf("expected version 1.0, got %v", loaded["version"])
	}
}

func TestLoadProjectConfig_FailsOnMissingFile(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)

	_, err := sm.LoadProjectConfig()
	if err == nil {
		t.Fatal("expected error for missing ipen.json")
	}
}

func TestSaveProjectConfig_PersistsConfig(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)

	config := map[string]any{
		"version": "2.0",
		"setting": "value",
	}

	err := sm.SaveProjectConfig(config)
	if err != nil {
		t.Fatalf("SaveProjectConfig failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "ipen.json"))
	if err != nil {
		t.Fatalf("failed to read ipen.json: %v", err)
	}

	var loaded map[string]any
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to unmarshal ipen.json: %v", err)
	}

	if loaded["version"] != "2.0" {
		t.Fatalf("expected version 2.0, got %v", loaded["version"])
	}
}

// ========== Book Config ==========

func TestLoadBookConfig_LoadsConfig(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)

	bookID := "my-book"
	bookDir := filepath.Join(dir, "books", bookID)
	if err := os.MkdirAll(bookDir, 0755); err != nil {
		t.Fatalf("failed to create book dir: %v", err)
	}

	config := models.BookConfig{
		ID:       bookID,
		Title:    "My Book",
		Platform: models.PlatformQidian,
		Genre:    models.Genre("fantasy"),
		Status:   models.StatusIncubating,
	}
	writeTestJSON(t, filepath.Join(bookDir, "book.json"), config)

	loaded, err := sm.LoadBookConfig(bookID)
	if err != nil {
		t.Fatalf("LoadBookConfig failed: %v", err)
	}

	if loaded.Title != "My Book" {
		t.Fatalf("expected title 'My Book', got %s", loaded.Title)
	}
}

func TestLoadBookConfig_FailsOnMissingFile(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)

	_, err := sm.LoadBookConfig("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing book.json")
	}
}

func TestSaveBookConfig_PersistsConfig(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)

	bookID := "my-book"
	bookDir := filepath.Join(dir, "books", bookID)
	if err := os.MkdirAll(bookDir, 0755); err != nil {
		t.Fatalf("failed to create book dir: %v", err)
	}

	config := &models.BookConfig{
		ID:               bookID,
		Title:            "Saved Book",
		Platform:         models.PlatformQidian,
		Genre:            models.Genre("fantasy"),
		Status:           models.StatusIncubating,
		TargetChapters:   100,
		ChapterWordCount: 3000,
		Language:         "zh",
	}

	err := sm.SaveBookConfig(bookID, config)
	if err != nil {
		t.Fatalf("SaveBookConfig failed: %v", err)
	}

	loaded, err := sm.LoadBookConfig(bookID)
	if err != nil {
		t.Fatalf("LoadBookConfig failed: %v", err)
	}

	if loaded.Title != "Saved Book" {
		t.Fatalf("expected title 'Saved Book', got %s", loaded.Title)
	}
}

func TestSaveBookConfigAt_SavesToCustomPath(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)

	customBookDir := filepath.Join(dir, "custom", "path")
	config := &models.BookConfig{
		ID:               "custom-book",
		Title:            "Custom Book",
		Platform:         models.PlatformQidian,
		Genre:            models.Genre("fantasy"),
		Status:           models.StatusIncubating,
		TargetChapters:   100,
		ChapterWordCount: 3000,
		Language:         "en",
	}

	err := sm.SaveBookConfigAt(customBookDir, config)
	if err != nil {
		t.Fatalf("SaveBookConfigAt failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(customBookDir, "book.json"))
	if err != nil {
		t.Fatalf("failed to read book.json: %v", err)
	}

	var loaded models.BookConfig
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to unmarshal book.json: %v", err)
	}

	if loaded.Title != "Custom Book" {
		t.Fatalf("expected title 'Custom Book', got %s", loaded.Title)
	}
}

// ========== ListBooks ==========

func TestListBooks_ReturnsBookIDs(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)

	booksDir := filepath.Join(dir, "books")
	if err := os.MkdirAll(filepath.Join(booksDir, "book1"), 0755); err != nil {
		t.Fatalf("failed to create book1 dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(booksDir, "book2"), 0755); err != nil {
		t.Fatalf("failed to create book2 dir: %v", err)
	}

	// Only book1 has book.json
	writeTestJSON(t, filepath.Join(booksDir, "book1", "book.json"), map[string]string{"title": "Book 1"})

	books, err := sm.ListBooks()
	if err != nil {
		t.Fatalf("ListBooks failed: %v", err)
	}

	if len(books) != 1 {
		t.Fatalf("expected 1 book, got %d", len(books))
	}
	if books[0] != "book1" {
		t.Fatalf("expected book1, got %s", books[0])
	}
}

func TestListBooks_ReturnsEmptyWhenNoBooks(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)

	books, err := sm.ListBooks()
	if err != nil {
		t.Fatalf("ListBooks failed: %v", err)
	}

	if len(books) != 0 {
		t.Fatalf("expected 0 books, got %d", len(books))
	}
}

// ========== Chapter Management ==========

func TestGetNextChapterNumber_FromChapterFiles(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)

	bookID := "my-book"
	bookDir := filepath.Join(dir, "books", bookID)
	chaptersDir := filepath.Join(bookDir, "chapters")
	if err := os.MkdirAll(chaptersDir, 0755); err != nil {
		t.Fatalf("failed to create chapters dir: %v", err)
	}

	// Create chapter files
	writeTestFile(t, filepath.Join(chaptersDir, "1_intro.md"), "# Ch1")
	writeTestFile(t, filepath.Join(chaptersDir, "2_development.md"), "# Ch2")
	writeTestFile(t, filepath.Join(chaptersDir, "3_climax.md"), "# Ch3")

	next, err := sm.GetNextChapterNumber(bookID)
	if err != nil {
		t.Fatalf("GetNextChapterNumber failed: %v", err)
	}

	if next != 4 {
		t.Fatalf("expected next chapter 4, got %d", next)
	}
}

func TestGetPersistedChapterCount_CountsChapterFiles(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)

	bookID := "my-book"
	bookDir := filepath.Join(dir, "books", bookID)
	chaptersDir := filepath.Join(bookDir, "chapters")
	if err := os.MkdirAll(chaptersDir, 0755); err != nil {
		t.Fatalf("failed to create chapters dir: %v", err)
	}

	writeTestFile(t, filepath.Join(chaptersDir, "1_intro.md"), "# Ch1")
	writeTestFile(t, filepath.Join(chaptersDir, "2_dev.md"), "# Ch2")
	writeTestFile(t, filepath.Join(chaptersDir, "3_end.md"), "# Ch3")

	count := sm.GetPersistedChapterCount(bookID)
	if count != 3 {
		t.Fatalf("expected chapter count 3, got %d", count)
	}
}

func TestGetPersistedChapterCount_IgnoresNonMatchingFiles(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)

	bookID := "my-book"
	bookDir := filepath.Join(dir, "books", bookID)
	chaptersDir := filepath.Join(bookDir, "chapters")
	if err := os.MkdirAll(chaptersDir, 0755); err != nil {
		t.Fatalf("failed to create chapters dir: %v", err)
	}

	writeTestFile(t, filepath.Join(chaptersDir, "1_intro.md"), "# Ch1")
	writeTestFile(t, filepath.Join(chaptersDir, "index.json"), "[]")
	writeTestFile(t, filepath.Join(chaptersDir, "readme.txt"), "Readme")

	count := sm.GetPersistedChapterCount(bookID)
	if count != 1 {
		t.Fatalf("expected chapter count 1, got %d", count)
	}
}

// ========== Chapter Index ==========

func TestLoadChapterIndex_LoadsIndex(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)

	bookID := "my-book"
	bookDir := filepath.Join(dir, "books", bookID)
	chaptersDir := filepath.Join(bookDir, "chapters")
	if err := os.MkdirAll(chaptersDir, 0755); err != nil {
		t.Fatalf("failed to create chapters dir: %v", err)
	}

	index := []models.ChapterMeta{
		{Number: 1, Title: "Chapter 1"},
		{Number: 2, Title: "Chapter 2"},
	}
	writeTestJSON(t, filepath.Join(chaptersDir, "index.json"), index)

	loaded, err := sm.LoadChapterIndex(bookID)
	if err != nil {
		t.Fatalf("LoadChapterIndex failed: %v", err)
	}

	if len(loaded) != 2 {
		t.Fatalf("expected 2 chapters in index, got %d", len(loaded))
	}
}

func TestLoadChapterIndex_ReturnsEmptyForMissingIndex(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)

	bookID := "my-book"
	index, err := sm.LoadChapterIndex(bookID)
	if err != nil {
		t.Fatalf("LoadChapterIndex failed: %v", err)
	}

	if len(index) != 0 {
		t.Fatalf("expected empty index, got %d entries", len(index))
	}
}

func TestSaveChapterIndex_PersistsIndex(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)

	bookID := "my-book"
	bookDir := filepath.Join(dir, "books", bookID)

	index := []models.ChapterMeta{
		{Number: 1, Title: "First"},
	}

	err := sm.SaveChapterIndex(bookID, index)
	if err != nil {
		t.Fatalf("SaveChapterIndex failed: %v", err)
	}

	indexPath := filepath.Join(bookDir, "chapters", "index.json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("failed to read index.json: %v", err)
	}

	var loaded []models.ChapterMeta
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to unmarshal index.json: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("expected 1 chapter in index, got %d", len(loaded))
	}
}

func TestSaveChapterIndexAt_SavesToCustomPath(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)

	bookDir := filepath.Join(dir, "custom")
	index := []models.ChapterMeta{{Number: 1}}

	err := sm.SaveChapterIndexAt(bookDir, index)
	if err != nil {
		t.Fatalf("SaveChapterIndexAt failed: %v", err)
	}

	indexPath := filepath.Join(bookDir, "chapters", "index.json")
	if _, err := os.Stat(indexPath); err != nil {
		t.Fatal("expected index.json to be created")
	}
}

// ========== Snapshot and Restore ==========

func TestSnapshotState_CreatesSnapshot(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)

	bookID := "my-book"
	bookDir := filepath.Join(dir, "books", bookID)
	storyDir := filepath.Join(bookDir, "story")
	if err := os.MkdirAll(storyDir, 0755); err != nil {
		t.Fatalf("failed to create story dir: %v", err)
	}

	// Create state files
	writeTestFile(t, filepath.Join(storyDir, "current_state.md"), "# State")
	writeTestFile(t, filepath.Join(storyDir, "pending_hooks.md"), "# Hooks")

	err := sm.SnapshotState(bookID, 1)
	if err != nil {
		t.Fatalf("SnapshotState failed: %v", err)
	}

	snapshotDir := filepath.Join(storyDir, "snapshots", "1")
	if _, err := os.Stat(filepath.Join(snapshotDir, "current_state.md")); err != nil {
		t.Fatal("expected current_state.md in snapshot")
	}
}

func TestRestoreState_RestoresFromSnapshot(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)

	bookID := "my-book"
	bookDir := filepath.Join(dir, "books", bookID)
	storyDir := filepath.Join(bookDir, "story")
	if err := os.MkdirAll(filepath.Join(storyDir, "snapshots", "1"), 0755); err != nil {
		t.Fatalf("failed to create snapshot dir: %v", err)
	}

	// Create snapshot
	writeTestFile(t, filepath.Join(storyDir, "snapshots", "1", "current_state.md"), "# Restored State")
	writeTestFile(t, filepath.Join(storyDir, "snapshots", "1", "pending_hooks.md"), "# Restored Hooks")

	restored, err := sm.RestoreState(bookID, 1)
	if err != nil {
		t.Fatalf("RestoreState failed: %v", err)
	}
	if !restored {
		t.Fatal("expected restored=true")
	}

	// Verify files restored
	content, err := os.ReadFile(filepath.Join(storyDir, "current_state.md"))
	if err != nil {
		t.Fatalf("failed to read restored current_state.md: %v", err)
	}
	if string(content) != "# Restored State" {
		t.Fatalf("expected restored content, got %s", string(content))
	}
}

// ========== IsCompleteBookDirectory ==========

func TestIsCompleteBookDirectory_ReturnsTrueForComplete(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)

	bookDir := filepath.Join(dir, "books", "complete-book")
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
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
	}

	if !sm.IsCompleteBookDirectory(bookDir) {
		t.Fatal("expected complete book directory")
	}
}

func TestIsCompleteBookDirectory_ReturnsFalseForIncomplete(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)

	bookDir := filepath.Join(dir, "books", "incomplete-book")
	if err := os.MkdirAll(bookDir, 0755); err != nil {
		t.Fatalf("failed to create book dir: %v", err)
	}

	if sm.IsCompleteBookDirectory(bookDir) {
		t.Fatal("expected incomplete book directory")
	}
}

// ========== RollbackToChapter ==========

func TestRollbackToChapter_RemovesLaterChapters(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)

	bookID := "my-book"
	bookDir := filepath.Join(dir, "books", bookID)
	storyDir := filepath.Join(bookDir, "story")
	chaptersDir := filepath.Join(bookDir, "chapters")

	if err := os.MkdirAll(filepath.Join(storyDir, "snapshots", "2"), 0755); err != nil {
		t.Fatalf("failed to create snapshot dir: %v", err)
	}
	if err := os.MkdirAll(chaptersDir, 0755); err != nil {
		t.Fatalf("failed to create chapters dir: %v", err)
	}

	// Create snapshot
	writeTestFile(t, filepath.Join(storyDir, "snapshots", "2", "current_state.md"), "# State Ch2")
	writeTestFile(t, filepath.Join(storyDir, "snapshots", "2", "pending_hooks.md"), "# Hooks")

	// Create chapter files
	writeTestFile(t, filepath.Join(chaptersDir, "1_intro.md"), "# Ch1")
	writeTestFile(t, filepath.Join(chaptersDir, "2_dev.md"), "# Ch2")
	writeTestFile(t, filepath.Join(chaptersDir, "3_climax.md"), "# Ch3")

	// Create index
	index := []models.ChapterMeta{
		{Number: 1},
		{Number: 2},
		{Number: 3},
	}
	writeTestJSON(t, filepath.Join(chaptersDir, "index.json"), index)

	discarded, err := sm.RollbackToChapter(bookID, 2)
	if err != nil {
		t.Fatalf("RollbackToChapter failed: %v", err)
	}

	if len(discarded) != 1 || discarded[0] != 3 {
		t.Fatalf("expected discarded chapters [3], got %v", discarded)
	}

	// Verify chapter 3 file deleted
	if _, err := os.Stat(filepath.Join(chaptersDir, "3_climax.md")); !os.IsNotExist(err) {
		t.Fatal("expected chapter 3 file to be deleted")
	}
}

// ========== Lock Management ==========

func TestAcquireBookLock_AcquiresAndReleases(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)

	bookID := "my-book"
	bookDir := filepath.Join(dir, "books", bookID)
	if err := os.MkdirAll(bookDir, 0755); err != nil {
		t.Fatalf("failed to create book dir: %v", err)
	}

	unlock, err := sm.AcquireBookLock(bookID)
	if err != nil {
		t.Fatalf("AcquireBookLock failed: %v", err)
	}

	// Lock file should exist
	lockPath := filepath.Join(bookDir, ".write.lock")
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatal("expected lock file to exist")
	}

	// Release lock
	if err := unlock(); err != nil {
		t.Fatalf("unlock failed: %v", err)
	}

	// Lock file should be removed
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatal("expected lock file to be removed")
	}
}

func TestAcquireBookLock_RejectsDoubleLock(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)

	bookID := "my-book"
	bookDir := filepath.Join(dir, "books", bookID)
	if err := os.MkdirAll(bookDir, 0755); err != nil {
		t.Fatalf("failed to create book dir: %v", err)
	}

	unlock1, err := sm.AcquireBookLock(bookID)
	if err != nil {
		t.Fatalf("first lock failed: %v", err)
	}
	defer unlock1()

	// Second lock should fail
	_, err = sm.AcquireBookLock(bookID)
	if err == nil {
		t.Fatal("expected second lock to fail")
	}
}

// ========== EnsureRuntimeState ==========

func TestEnsureRuntimeState_BootstrapsState(t *testing.T) {
	dir := t.TempDir()
	sm := NewStateManager(dir)

	bookID := "my-book"
	bookDir := filepath.Join(dir, "books", bookID)
	if err := os.MkdirAll(bookDir, 0755); err != nil {
		t.Fatalf("failed to create book dir: %v", err)
	}
	bookConfig := map[string]string{"language": "en"}
	writeTestJSON(t, filepath.Join(bookDir, "book.json"), bookConfig)

	err := sm.EnsureRuntimeState(bookID, 0)
	if err != nil {
		t.Fatalf("EnsureRuntimeState failed: %v", err)
	}

	// Verify state directory created
	stateDir := filepath.Join(bookDir, "story", "state")
	if _, err := os.Stat(stateDir); err != nil {
		t.Fatal("expected state directory to exist")
	}
}

// ========== Helper ==========

func stringsContainsGo(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
