package state

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestDB(t *testing.T) *MemoryDB {
	t.Helper()
	dir := t.TempDir()
	bookDir := filepath.Join(dir, "book")
	if err := os.MkdirAll(filepath.Join(bookDir, "story"), 0755); err != nil {
		t.Fatalf("failed to create book dir: %v", err)
	}
	mdb, err := NewMemoryDB(bookDir)
	if err != nil {
		t.Fatalf("failed to create memory db: %v", err)
	}
	t.Cleanup(func() { mdb.Close() })
	return mdb
}

// ========== NewMemoryDB / migrate ==========

func TestNewMemoryDB_CreatesDatabase(t *testing.T) {
	dir := t.TempDir()
	bookDir := filepath.Join(dir, "book")
	if err := os.MkdirAll(filepath.Join(bookDir, "story"), 0755); err != nil {
		t.Fatalf("failed to create book dir: %v", err)
	}

	mdb, err := NewMemoryDB(bookDir)
	if err != nil {
		t.Fatalf("NewMemoryDB failed: %v", err)
	}
	defer mdb.Close()

	if mdb.db == nil {
		t.Fatal("expected non-nil db")
	}
}

// ========== Facts ==========

func TestAddFact_And_GetCurrentFacts(t *testing.T) {
	mdb := setupTestDB(t)

	fact := Fact{
		Subject:          "Alice",
		Predicate:        "has_trait",
		Object:           "brave",
		ValidFromChapter: 1,
		SourceChapter:    1,
	}

	id, err := mdb.AddFact(fact)
	if err != nil {
		t.Fatalf("AddFact failed: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero fact id")
	}

	facts, err := mdb.GetCurrentFacts()
	if err != nil {
		t.Fatalf("GetCurrentFacts failed: %v", err)
	}
	if len(facts) != 1 {
		t.Fatalf("expected 1 fact, got %d", len(facts))
	}
	if facts[0].Subject != "Alice" || facts[0].Predicate != "has_trait" || facts[0].Object != "brave" {
		t.Fatalf("unexpected fact: %+v", facts[0])
	}
}

func TestInvalidateFact(t *testing.T) {
	mdb := setupTestDB(t)

	id, err := mdb.AddFact(Fact{
		Subject:          "Bob",
		Predicate:        "has_item",
		Object:           "sword",
		ValidFromChapter: 1,
		SourceChapter:    1,
	})
	if err != nil {
		t.Fatalf("AddFact failed: %v", err)
	}

	err = mdb.InvalidateFact(id, 5)
	if err != nil {
		t.Fatalf("InvalidateFact failed: %v", err)
	}

	facts, err := mdb.GetCurrentFacts()
	if err != nil {
		t.Fatalf("GetCurrentFacts failed: %v", err)
	}
	if len(facts) != 0 {
		t.Fatalf("expected 0 current facts after invalidation, got %d", len(facts))
	}
}

func TestGetFactsAt(t *testing.T) {
	mdb := setupTestDB(t)

	// Add a fact valid from chapter 1 to 10
	_, err := mdb.AddFact(Fact{
		Subject:           "Charlie",
		Predicate:         "location",
		Object:            "village",
		ValidFromChapter:  1,
		ValidUntilChapter: ptrInt(10),
		SourceChapter:     1,
	})
	if err != nil {
		t.Fatalf("AddFact failed: %v", err)
	}

	// Add a fact valid from chapter 5 onward (no end)
	_, err = mdb.AddFact(Fact{
		Subject:          "Charlie",
		Predicate:        "status",
		Object:           "awake",
		ValidFromChapter: 5,
		SourceChapter:    5,
	})
	if err != nil {
		t.Fatalf("AddFact failed: %v", err)
	}

	// At chapter 3, only first fact should be valid
	facts3, err := mdb.GetFactsAt("Charlie", 3)
	if err != nil {
		t.Fatalf("GetFactsAt failed: %v", err)
	}
	if len(facts3) != 1 || facts3[0].Predicate != "location" {
		t.Fatalf("expected 1 fact (location) at chapter 3, got %d", len(facts3))
	}

	// At chapter 7, both facts should be valid
	facts7, err := mdb.GetFactsAt("Charlie", 7)
	if err != nil {
		t.Fatalf("GetFactsAt failed: %v", err)
	}
	if len(facts7) != 2 {
		t.Fatalf("expected 2 facts at chapter 7, got %d", len(facts7))
	}

	// At chapter 12, only second fact should be valid
	facts12, err := mdb.GetFactsAt("Charlie", 12)
	if err != nil {
		t.Fatalf("GetFactsAt failed: %v", err)
	}
	if len(facts12) != 1 || facts12[0].Predicate != "status" {
		t.Fatalf("expected 1 fact (status) at chapter 12, got %d", len(facts12))
	}
}

func TestGetFactHistory(t *testing.T) {
	mdb := setupTestDB(t)

	_, _ = mdb.AddFact(Fact{
		Subject:          "Dave",
		Predicate:        "role",
		Object:           "student",
		ValidFromChapter: 1,
		SourceChapter:    1,
	})
	_, _ = mdb.AddFact(Fact{
		Subject:          "Dave",
		Predicate:        "role",
		Object:           "teacher",
		ValidFromChapter: 10,
		SourceChapter:    10,
	})

	history, err := mdb.GetFactHistory("Dave")
	if err != nil {
		t.Fatalf("GetFactHistory failed: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected 2 facts in history, got %d", len(history))
	}
	if history[0].ValidFromChapter != 1 || history[1].ValidFromChapter != 10 {
		t.Fatalf("facts not sorted by valid_from_chapter: %+v", history)
	}
}

func TestGetFactsByPredicate(t *testing.T) {
	mdb := setupTestDB(t)

	_, _ = mdb.AddFact(Fact{
		Subject:          "Eve",
		Predicate:        "has_trait",
		Object:           "smart",
		ValidFromChapter: 1,
		SourceChapter:    1,
	})
	_, _ = mdb.AddFact(Fact{
		Subject:          "Frank",
		Predicate:        "has_trait",
		Object:           "strong",
		ValidFromChapter: 1,
		SourceChapter:    1,
	})
	// Add a fact with different predicate
	_, _ = mdb.AddFact(Fact{
		Subject:          "Eve",
		Predicate:        "location",
		Object:           "school",
		ValidFromChapter: 1,
		SourceChapter:    1,
	})

	facts, err := mdb.GetFactsByPredicate("has_trait")
	if err != nil {
		t.Fatalf("GetFactsByPredicate failed: %v", err)
	}
	if len(facts) != 2 {
		t.Fatalf("expected 2 facts with has_trait, got %d", len(facts))
	}
}

func TestGetFactsForCharacters(t *testing.T) {
	mdb := setupTestDB(t)

	_, _ = mdb.AddFact(Fact{
		Subject:          "Grace",
		Predicate:        "has_trait",
		Object:           "brave",
		ValidFromChapter: 1,
		SourceChapter:    1,
	})
	_, _ = mdb.AddFact(Fact{
		Subject:          "Henry",
		Predicate:        "has_trait",
		Object:           "clever",
		ValidFromChapter: 1,
		SourceChapter:    1,
	})
	// Add fact for different character
	_, _ = mdb.AddFact(Fact{
		Subject:          "Ivy",
		Predicate:        "has_trait",
		Object:           "shy",
		ValidFromChapter: 1,
		SourceChapter:    1,
	})

	facts, err := mdb.GetFactsForCharacters([]string{"Grace", "Henry"})
	if err != nil {
		t.Fatalf("GetFactsForCharacters failed: %v", err)
	}
	if len(facts) != 2 {
		t.Fatalf("expected 2 facts for Grace and Henry, got %d", len(facts))
	}

	// Test empty names
	emptyFacts, err := mdb.GetFactsForCharacters([]string{})
	if err != nil {
		t.Fatalf("GetFactsForCharacters with empty names failed: %v", err)
	}
	if len(emptyFacts) != 0 {
		t.Fatalf("expected 0 facts for empty names, got %d", len(emptyFacts))
	}
}

func TestReplaceCurrentFacts(t *testing.T) {
	mdb := setupTestDB(t)

	// Add initial facts
	_, _ = mdb.AddFact(Fact{
		Subject:          "Jack",
		Predicate:        "old",
		Object:           "value1",
		ValidFromChapter: 1,
		SourceChapter:    1,
	})
	_, _ = mdb.AddFact(Fact{
		Subject:          "Jack",
		Predicate:        "new",
		Object:           "value2",
		ValidFromChapter: 1,
		SourceChapter:    1,
	})

	// Replace with new facts
	newFacts := []Fact{
		{
			Subject:          "Jack",
			Predicate:        "updated",
			Object:           "value3",
			ValidFromChapter: 5,
			SourceChapter:    5,
		},
	}

	err := mdb.ReplaceCurrentFacts(newFacts)
	if err != nil {
		t.Fatalf("ReplaceCurrentFacts failed: %v", err)
	}

	facts, err := mdb.GetCurrentFacts()
	if err != nil {
		t.Fatalf("GetCurrentFacts failed: %v", err)
	}
	if len(facts) != 1 {
		t.Fatalf("expected 1 fact after replace, got %d", len(facts))
	}
	if facts[0].Predicate != "updated" {
		t.Fatalf("expected updated predicate, got %s", facts[0].Predicate)
	}
}

func TestResetFacts(t *testing.T) {
	mdb := setupTestDB(t)

	_, _ = mdb.AddFact(Fact{
		Subject:          "Kate",
		Predicate:        "exists",
		Object:           "yes",
		ValidFromChapter: 1,
		SourceChapter:    1,
	})

	err := mdb.ResetFacts()
	if err != nil {
		t.Fatalf("ResetFacts failed: %v", err)
	}

	facts, err := mdb.GetCurrentFacts()
	if err != nil {
		t.Fatalf("GetCurrentFacts failed: %v", err)
	}
	if len(facts) != 0 {
		t.Fatalf("expected 0 facts after reset, got %d", len(facts))
	}
}

// ========== Chapter Summaries ==========

func TestUpsertSummary_And_GetSummaries(t *testing.T) {
	mdb := setupTestDB(t)

	summary := StoredSummary{
		Chapter:      1,
		Title:        "First Chapter",
		Characters:   "Alice, Bob",
		Events:       "Introduction",
		StateChanges: "World revealed",
		HookActivity: "hook1 opened",
		Mood:         "mysterious",
		ChapterType:  "prologue",
	}

	err := mdb.UpsertSummary(summary)
	if err != nil {
		t.Fatalf("UpsertSummary failed: %v", err)
	}

	summaries, err := mdb.GetSummaries(1, 1)
	if err != nil {
		t.Fatalf("GetSummaries failed: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}
	if summaries[0].Title != "First Chapter" {
		t.Fatalf("expected title 'First Chapter', got %q", summaries[0].Title)
	}
	if summaries[0].Characters != "Alice, Bob" {
		t.Fatalf("expected characters 'Alice, Bob', got %q", summaries[0].Characters)
	}
}

func TestReplaceSummaries(t *testing.T) {
	mdb := setupTestDB(t)

	// Add initial summary
	_ = mdb.UpsertSummary(StoredSummary{
		Chapter:    1,
		Title:      "Old Chapter",
		Characters: "Old Char",
		Events:     "Old events",
	})

	// Replace with new summaries
	newSummaries := []StoredSummary{
		{
			Chapter:    1,
			Title:      "New Chapter",
			Characters: "New Char",
			Events:     "New events",
		},
	}

	err := mdb.ReplaceSummaries(newSummaries)
	if err != nil {
		t.Fatalf("ReplaceSummaries failed: %v", err)
	}

	summaries, err := mdb.GetSummaries(1, 1)
	if err != nil {
		t.Fatalf("GetSummaries failed: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}
	if summaries[0].Title != "New Chapter" {
		t.Fatalf("expected title 'New Chapter', got %q", summaries[0].Title)
	}
}

func TestGetSummariesByCharacters(t *testing.T) {
	mdb := setupTestDB(t)

	_ = mdb.UpsertSummary(StoredSummary{
		Chapter:    1,
		Title:      "Ch1",
		Characters: "Alice, Bob",
		Events:     "Event 1",
	})
	_ = mdb.UpsertSummary(StoredSummary{
		Chapter:    2,
		Title:      "Ch2",
		Characters: "Charlie",
		Events:     "Event 2",
	})
	_ = mdb.UpsertSummary(StoredSummary{
		Chapter:    3,
		Title:      "Ch3",
		Characters: "Alice, Charlie",
		Events:     "Event 3",
	})

	summaries, err := mdb.GetSummariesByCharacters([]string{"Alice"})
	if err != nil {
		t.Fatalf("GetSummariesByCharacters failed: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 summaries with Alice, got %d", len(summaries))
	}

	// Test empty names
	emptySummaries, err := mdb.GetSummariesByCharacters([]string{})
	if err != nil {
		t.Fatalf("GetSummariesByCharacters with empty names failed: %v", err)
	}
	if len(emptySummaries) != 0 {
		t.Fatalf("expected 0 summaries for empty names, got %d", len(emptySummaries))
	}
}

func TestGetChapterCount(t *testing.T) {
	mdb := setupTestDB(t)

	count, err := mdb.GetChapterCount()
	if err != nil {
		t.Fatalf("GetChapterCount failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 chapters initially, got %d", count)
	}

	_ = mdb.UpsertSummary(StoredSummary{Chapter: 1, Title: "Ch1", Characters: "A", Events: "E1"})
	_ = mdb.UpsertSummary(StoredSummary{Chapter: 2, Title: "Ch2", Characters: "B", Events: "E2"})

	count, err = mdb.GetChapterCount()
	if err != nil {
		t.Fatalf("GetChapterCount failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 chapters, got %d", count)
	}
}

func TestGetRecentSummaries(t *testing.T) {
	mdb := setupTestDB(t)

	for i := 1; i <= 5; i++ {
		_ = mdb.UpsertSummary(StoredSummary{
			Chapter:    i,
			Title:      "Ch",
			Characters: "Char",
			Events:     "Event",
		})
	}

	recent, err := mdb.GetRecentSummaries(3)
	if err != nil {
		t.Fatalf("GetRecentSummaries failed: %v", err)
	}
	if len(recent) != 3 {
		t.Fatalf("expected 3 recent summaries, got %d", len(recent))
	}
	// Should be in descending order
	if recent[0].Chapter != 5 || recent[1].Chapter != 4 || recent[2].Chapter != 3 {
		t.Fatalf("expected chapters 5,4,3, got %d,%d,%d", recent[0].Chapter, recent[1].Chapter, recent[2].Chapter)
	}
}

// ========== Hooks ==========

func TestUpsertHook_And_GetActiveHooks(t *testing.T) {
	mdb := setupTestDB(t)

	hook := StoredHook{
		HookID:              "hook1",
		StartChapter:        1,
		Type:                "mystery",
		Status:              "open",
		LastAdvancedChapter: 3,
		ExpectedPayoff:      "Reveal truth",
		PayoffTiming:        "soon",
		Notes:               "Important mystery",
	}

	err := mdb.UpsertHook(hook)
	if err != nil {
		t.Fatalf("UpsertHook failed: %v", err)
	}

	hooks, err := mdb.GetActiveHooks()
	if err != nil {
		t.Fatalf("GetActiveHooks failed: %v", err)
	}
	if len(hooks) != 1 {
		t.Fatalf("expected 1 active hook, got %d", len(hooks))
	}
	if hooks[0].HookID != "hook1" {
		t.Fatalf("expected hook1, got %s", hooks[0].HookID)
	}

	// Test resolved hook is excluded
	resolvedHook := StoredHook{
		HookID:              "hook2",
		StartChapter:        2,
		Type:                "romance",
		Status:              "resolved",
		LastAdvancedChapter: 5,
		ExpectedPayoff:      "Kiss",
		PayoffTiming:        "now",
		Notes:               "Resolved",
	}
	_ = mdb.UpsertHook(resolvedHook)

	hooks, err = mdb.GetActiveHooks()
	if err != nil {
		t.Fatalf("GetActiveHooks failed: %v", err)
	}
	if len(hooks) != 1 {
		t.Fatalf("expected 1 active hook after resolving one, got %d", len(hooks))
	}
	if hooks[0].HookID != "hook1" {
		t.Fatalf("expected hook1, got %s", hooks[0].HookID)
	}
}

func TestReplaceHooks(t *testing.T) {
	mdb := setupTestDB(t)

	// Add initial hooks
	_ = mdb.UpsertHook(StoredHook{HookID: "h1", StartChapter: 1, Status: "open"})
	_ = mdb.UpsertHook(StoredHook{HookID: "h2", StartChapter: 2, Status: "open"})

	// Replace with new hooks
	newHooks := []StoredHook{
		{HookID: "h3", StartChapter: 3, Status: "open"},
	}

	err := mdb.ReplaceHooks(newHooks)
	if err != nil {
		t.Fatalf("ReplaceHooks failed: %v", err)
	}

	hooks, err := mdb.GetActiveHooks()
	if err != nil {
		t.Fatalf("GetActiveHooks failed: %v", err)
	}
	if len(hooks) != 1 {
		t.Fatalf("expected 1 hook after replace, got %d", len(hooks))
	}
	if hooks[0].HookID != "h3" {
		t.Fatalf("expected h3, got %s", hooks[0].HookID)
	}
}

func TestGetActiveHooks_ExcludesChineseResolved(t *testing.T) {
	mdb := setupTestDB(t)

	// Add hooks with various resolved statuses
	hooks := []StoredHook{
		{HookID: "open1", Status: "open"},
		{HookID: "resolved1", Status: "resolved"},
		{HookID: "closed1", Status: "closed"},
		{HookID: "chinese_resolved", Status: "已回收"},
		{HookID: "chinese_solved", Status: "已解决"},
	}

	for _, h := range hooks {
		_ = mdb.UpsertHook(h)
	}

	activeHooks, err := mdb.GetActiveHooks()
	if err != nil {
		t.Fatalf("GetActiveHooks failed: %v", err)
	}

	// Only open1 should remain
	if len(activeHooks) != 1 {
		t.Fatalf("expected 1 active hook, got %d", len(activeHooks))
	}
	if activeHooks[0].HookID != "open1" {
		t.Fatalf("expected open1, got %s", activeHooks[0].HookID)
	}
}

// ========== Close ==========

func TestMemoryDB_Close(t *testing.T) {
	dir := t.TempDir()
	bookDir := filepath.Join(dir, "book")
	if err := os.MkdirAll(filepath.Join(bookDir, "story"), 0755); err != nil {
		t.Fatalf("failed to create book dir: %v", err)
	}

	mdb, err := NewMemoryDB(bookDir)
	if err != nil {
		t.Fatalf("NewMemoryDB failed: %v", err)
	}

	err = mdb.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

// ========== Helper ==========

func ptrInt(i int) *int {
	return &i
}
