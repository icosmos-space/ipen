package state

import (
	"database/sql"
	"fmt"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Fact 表示a temporal fact。
type Fact struct {
	ID                int64  `json:"id,omitempty"`
	Subject           string `json:"subject"`
	Predicate         string `json:"predicate"`
	Object            string `json:"object"`
	ValidFromChapter  int    `json:"validFromChapter"`
	ValidUntilChapter *int   `json:"validUntilChapter"`
	SourceChapter     int    `json:"sourceChapter"`
}

// StoredSummary 表示a stored chapter summary。
type StoredSummary struct {
	Chapter      int    `json:"chapter"`
	Title        string `json:"title"`
	Characters   string `json:"characters"`
	Events       string `json:"events"`
	StateChanges string `json:"stateChanges"`
	HookActivity string `json:"hookActivity"`
	Mood         string `json:"mood"`
	ChapterType  string `json:"chapterType"`
}

// StoredHook 表示a stored hook。
type StoredHook struct {
	HookID              string `json:"hookId"`
	StartChapter        int    `json:"startChapter"`
	Type                string `json:"type"`
	Status              string `json:"status"`
	LastAdvancedChapter int    `json:"lastAdvancedChapter"`
	ExpectedPayoff      string `json:"expectedPayoff"`
	PayoffTiming        string `json:"payoffTiming"`
	Notes               string `json:"notes"`
}

// MemoryDB 表示the temporal memory database。
type MemoryDB struct {
	db *sql.DB
}

// NewMemoryDB 创建新的memory database。
func NewMemoryDB(bookDir string) (*MemoryDB, error) {
	dbPath := filepath.Join(bookDir, "story", "memory.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	mdb := &MemoryDB{db: db}
	if err = mdb.migrate(); err != nil {
		return nil, err
	}

	return mdb, nil
}

func (mdb *MemoryDB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS facts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		subject TEXT NOT NULL,
		predicate TEXT NOT NULL,
		object TEXT NOT NULL,
		valid_from_chapter INTEGER NOT NULL,
		valid_until_chapter INTEGER,
		source_chapter INTEGER NOT NULL,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS chapter_summaries (
		chapter INTEGER PRIMARY KEY,
		title TEXT NOT NULL,
		characters TEXT NOT NULL DEFAULT '',
		events TEXT NOT NULL DEFAULT '',
		state_changes TEXT NOT NULL DEFAULT '',
		hook_activity TEXT NOT NULL DEFAULT '',
		mood TEXT NOT NULL DEFAULT '',
		chapter_type TEXT NOT NULL DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS hooks (
		hook_id TEXT PRIMARY KEY,
		start_chapter INTEGER NOT NULL DEFAULT 0,
		type TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'open',
		last_advanced_chapter INTEGER NOT NULL DEFAULT 0,
		expected_payoff TEXT NOT NULL DEFAULT '',
		payoff_timing TEXT NOT NULL DEFAULT '',
		notes TEXT NOT NULL DEFAULT ''
	);

	CREATE INDEX IF NOT EXISTS idx_facts_subject ON facts(subject);
	CREATE INDEX IF NOT EXISTS idx_facts_valid ON facts(valid_from_chapter, valid_until_chapter);
	CREATE INDEX IF NOT EXISTS idx_facts_source ON facts(source_chapter);
	CREATE INDEX IF NOT EXISTS idx_hooks_status ON hooks(status);
	CREATE INDEX IF NOT EXISTS idx_hooks_last_advanced ON hooks(last_advanced_chapter);
	`
	_, err := mdb.db.Exec(schema)
	return err
}

// AddFact adds a new fact
func (mdb *MemoryDB) AddFact(fact Fact) (int64, error) {
	result, err := mdb.db.Exec(
		`INSERT INTO facts (subject, predicate, object, valid_from_chapter, valid_until_chapter, source_chapter)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		fact.Subject, fact.Predicate, fact.Object,
		fact.ValidFromChapter, fact.ValidUntilChapter, fact.SourceChapter,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// InvalidateFact invalidates a fact
func (mdb *MemoryDB) InvalidateFact(id int64, untilChapter int) error {
	_, err := mdb.db.Exec(
		"UPDATE facts SET valid_until_chapter = ? WHERE id = ?",
		untilChapter, id,
	)
	return err
}

// GetCurrentFacts gets all currently valid facts
func (mdb *MemoryDB) GetCurrentFacts() ([]Fact, error) {
	rows, err := mdb.db.Query(
		`SELECT id, subject, predicate, object, valid_from_chapter, valid_until_chapter, source_chapter
		 FROM facts
		 WHERE valid_until_chapter IS NULL
		 ORDER BY subject, predicate`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var facts []Fact
	for rows.Next() {
		var f Fact
		var validUntil sql.NullInt64
		if err := rows.Scan(&f.ID, &f.Subject, &f.Predicate, &f.Object, &f.ValidFromChapter, &validUntil, &f.SourceChapter); err != nil {
			return nil, err
		}
		if validUntil.Valid {
			v := int(validUntil.Int64)
			f.ValidUntilChapter = &v
		}
		facts = append(facts, f)
	}
	return facts, rows.Err()
}

// GetFactsAt gets facts valid at a specific chapter
func (mdb *MemoryDB) GetFactsAt(subject string, chapter int) ([]Fact, error) {
	rows, err := mdb.db.Query(
		`SELECT id, subject, predicate, object, valid_from_chapter, valid_until_chapter, source_chapter
		 FROM facts
		 WHERE subject = ? AND valid_from_chapter <= ?
		 AND (valid_until_chapter IS NULL OR valid_until_chapter > ?)
		 ORDER BY predicate`,
		subject, chapter, chapter,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var facts []Fact
	for rows.Next() {
		var f Fact
		var validUntil sql.NullInt64
		if err := rows.Scan(&f.ID, &f.Subject, &f.Predicate, &f.Object, &f.ValidFromChapter, &validUntil, &f.SourceChapter); err != nil {
			return nil, err
		}
		if validUntil.Valid {
			v := int(validUntil.Int64)
			f.ValidUntilChapter = &v
		}
		facts = append(facts, f)
	}
	return facts, rows.Err()
}

// GetFactHistory gets all facts about a subject
func (mdb *MemoryDB) GetFactHistory(subject string) ([]Fact, error) {
	rows, err := mdb.db.Query(
		`SELECT id, subject, predicate, object, valid_from_chapter, valid_until_chapter, source_chapter
		 FROM facts
		 WHERE subject = ?
		 ORDER BY valid_from_chapter`,
		subject,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var facts []Fact
	for rows.Next() {
		var f Fact
		var validUntil sql.NullInt64
		if err := rows.Scan(&f.ID, &f.Subject, &f.Predicate, &f.Object, &f.ValidFromChapter, &validUntil, &f.SourceChapter); err != nil {
			return nil, err
		}
		if validUntil.Valid {
			v := int(validUntil.Int64)
			f.ValidUntilChapter = &v
		}
		facts = append(facts, f)
	}
	return facts, rows.Err()
}

// GetFactsByPredicate gets facts by predicate
func (mdb *MemoryDB) GetFactsByPredicate(predicate string) ([]Fact, error) {
	rows, err := mdb.db.Query(
		`SELECT id, subject, predicate, object, valid_from_chapter, valid_until_chapter, source_chapter
		 FROM facts
		 WHERE predicate = ? AND valid_until_chapter IS NULL
		 ORDER BY subject`,
		predicate,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var facts []Fact
	for rows.Next() {
		var f Fact
		var validUntil sql.NullInt64
		if err := rows.Scan(&f.ID, &f.Subject, &f.Predicate, &f.Object, &f.ValidFromChapter, &validUntil, &f.SourceChapter); err != nil {
			return nil, err
		}
		if validUntil.Valid {
			v := int(validUntil.Int64)
			f.ValidUntilChapter = &v
		}
		facts = append(facts, f)
	}
	return facts, rows.Err()
}

// GetFactsForCharacters gets facts for characters
func (mdb *MemoryDB) GetFactsForCharacters(names []string) ([]Fact, error) {
	if len(names) == 0 {
		return []Fact{}, nil
	}

	placeholders := ""
	args := make([]any, len(names))
	for i, name := range names {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args[i] = name
	}

	query := fmt.Sprintf(
		`SELECT id, subject, predicate, object, valid_from_chapter, valid_until_chapter, source_chapter
		 FROM facts
		 WHERE subject IN (%s) AND valid_until_chapter IS NULL
		 ORDER BY subject, predicate`,
		placeholders,
	)

	rows, err := mdb.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var facts []Fact
	for rows.Next() {
		var f Fact
		var validUntil sql.NullInt64
		if err := rows.Scan(&f.ID, &f.Subject, &f.Predicate, &f.Object, &f.ValidFromChapter, &validUntil, &f.SourceChapter); err != nil {
			return nil, err
		}
		if validUntil.Valid {
			v := int(validUntil.Int64)
			f.ValidUntilChapter = &v
		}
		facts = append(facts, f)
	}
	return facts, rows.Err()
}

// ReplaceCurrentFacts replaces all current facts
func (mdb *MemoryDB) ReplaceCurrentFacts(facts []Fact) error {
	tx, err := mdb.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM facts WHERE valid_until_chapter IS NULL"); err != nil {
		return err
	}

	for _, fact := range facts {
		if _, err := tx.Exec(
			`INSERT INTO facts (subject, predicate, object, valid_from_chapter, valid_until_chapter, source_chapter)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			fact.Subject, fact.Predicate, fact.Object,
			fact.ValidFromChapter, fact.ValidUntilChapter, fact.SourceChapter,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ResetFacts resets all facts
func (mdb *MemoryDB) ResetFacts() error {
	_, err := mdb.db.Exec("DELETE FROM facts")
	return err
}

// UpsertSummary upserts a chapter summary
func (mdb *MemoryDB) UpsertSummary(summary StoredSummary) error {
	_, err := mdb.db.Exec(
		`INSERT OR REPLACE INTO chapter_summaries (chapter, title, characters, events, state_changes, hook_activity, mood, chapter_type)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		summary.Chapter, summary.Title, summary.Characters, summary.Events,
		summary.StateChanges, summary.HookActivity, summary.Mood, summary.ChapterType,
	)
	return err
}

// ReplaceSummaries replaces all summaries
func (mdb *MemoryDB) ReplaceSummaries(summaries []StoredSummary) error {
	tx, err := mdb.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM chapter_summaries"); err != nil {
		return err
	}

	for _, summary := range summaries {
		if err := mdb.UpsertSummary(summary); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetSummaries gets summaries for a chapter range
func (mdb *MemoryDB) GetSummaries(fromChapter int, toChapter int) ([]StoredSummary, error) {
	rows, err := mdb.db.Query(
		`SELECT chapter, title, characters, events, state_changes, hook_activity, mood, chapter_type
		 FROM chapter_summaries
		 WHERE chapter >= ? AND chapter <= ?
		 ORDER BY chapter`,
		fromChapter, toChapter,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []StoredSummary
	for rows.Next() {
		var s StoredSummary
		if err := rows.Scan(&s.Chapter, &s.Title, &s.Characters, &s.Events, &s.StateChanges, &s.HookActivity, &s.Mood, &s.ChapterType); err != nil {
			return nil, err
		}
		summaries = append(summaries, s)
	}
	return summaries, rows.Err()
}

// GetSummariesByCharacters gets summaries matching character names
func (mdb *MemoryDB) GetSummariesByCharacters(names []string) ([]StoredSummary, error) {
	if len(names) == 0 {
		return []StoredSummary{}, nil
	}

	conditions := ""
	args := make([]any, len(names))
	for i, name := range names {
		if i > 0 {
			conditions += " OR "
		}
		conditions += "characters LIKE ?"
		args[i] = "%" + name + "%"
	}

	query := fmt.Sprintf(
		`SELECT chapter, title, characters, events, state_changes, hook_activity, mood, chapter_type
		 FROM chapter_summaries
		 WHERE %s
		 ORDER BY chapter`,
		conditions,
	)

	rows, err := mdb.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []StoredSummary
	for rows.Next() {
		var s StoredSummary
		if err := rows.Scan(&s.Chapter, &s.Title, &s.Characters, &s.Events, &s.StateChanges, &s.HookActivity, &s.Mood, &s.ChapterType); err != nil {
			return nil, err
		}
		summaries = append(summaries, s)
	}
	return summaries, rows.Err()
}

// GetChapterCount gets the chapter count
func (mdb *MemoryDB) GetChapterCount() (int, error) {
	var count int
	err := mdb.db.QueryRow("SELECT COUNT(*) as count FROM chapter_summaries").Scan(&count)
	return count, err
}

// GetRecentSummaries gets recent summaries
func (mdb *MemoryDB) GetRecentSummaries(count int) ([]StoredSummary, error) {
	rows, err := mdb.db.Query(
		`SELECT chapter, title, characters, events, state_changes, hook_activity, mood, chapter_type
		 FROM chapter_summaries
		 ORDER BY chapter DESC
		 LIMIT ?`,
		count,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []StoredSummary
	for rows.Next() {
		var s StoredSummary
		if err := rows.Scan(&s.Chapter, &s.Title, &s.Characters, &s.Events, &s.StateChanges, &s.HookActivity, &s.Mood, &s.ChapterType); err != nil {
			return nil, err
		}
		summaries = append(summaries, s)
	}
	return summaries, rows.Err()
}

// UpsertHook upserts a hook
func (mdb *MemoryDB) UpsertHook(hook StoredHook) error {
	_, err := mdb.db.Exec(
		`INSERT OR REPLACE INTO hooks (hook_id, start_chapter, type, status, last_advanced_chapter, expected_payoff, payoff_timing, notes)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		hook.HookID, hook.StartChapter, hook.Type, hook.Status,
		hook.LastAdvancedChapter, hook.ExpectedPayoff, hook.PayoffTiming, hook.Notes,
	)
	return err
}

// ReplaceHooks replaces all hooks
func (mdb *MemoryDB) ReplaceHooks(hooks []StoredHook) error {
	tx, err := mdb.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM hooks"); err != nil {
		return err
	}

	for _, hook := range hooks {
		if err := mdb.UpsertHook(hook); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetActiveHooks gets active hooks
func (mdb *MemoryDB) GetActiveHooks() ([]StoredHook, error) {
	rows, err := mdb.db.Query(
		`SELECT hook_id, start_chapter, type, status, last_advanced_chapter, expected_payoff, payoff_timing, notes
		 FROM hooks
		 WHERE lower(status) NOT IN ('resolved', 'closed', '已回收', '已解决')
		 ORDER BY last_advanced_chapter DESC, start_chapter DESC, hook_id ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hooks []StoredHook
	for rows.Next() {
		var h StoredHook
		if err := rows.Scan(&h.HookID, &h.StartChapter, &h.Type, &h.Status, &h.LastAdvancedChapter, &h.ExpectedPayoff, &h.PayoffTiming, &h.Notes); err != nil {
			return nil, err
		}
		hooks = append(hooks, h)
	}
	return hooks, rows.Err()
}

// Close closes the database
func (mdb *MemoryDB) Close() error {
	return mdb.db.Close()
}
