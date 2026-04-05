package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/icosmos-space/ipen/core/models"
)

// RuntimeStateArtifacts contains updated runtime snapshot and markdown projections.
type RuntimeStateArtifacts struct {
	Snapshot                 RuntimeStateSnapshot     `json:"snapshot"`
	ResolvedDelta            models.RuntimeStateDelta `json:"resolvedDelta"`
	CurrentStateMarkdown     string                   `json:"currentStateMarkdown"`
	HooksMarkdown            string                   `json:"hooksMarkdown"`
	ChapterSummariesMarkdown string                   `json:"chapterSummariesMarkdown"`
}

// NarrativeMemorySeed 是a compact memory seed for retrieval components。
type NarrativeMemorySeed struct {
	Summaries []StoredSummary `json:"summaries"`
	Hooks     []StoredHook    `json:"hooks"`
}

// LoadRuntimeStateSnapshot 加载runtime snapshot from story/state json files。
func LoadRuntimeStateSnapshot(bookDir string) (*RuntimeStateSnapshot, error) {
	if _, err := BootstrapStructuredStateFromMarkdown(bookDir, 0); err != nil {
		return nil, err
	}

	stateDir := filepath.Join(bookDir, "story", "state")
	manifest, err := readJSONFile[models.StateManifest](filepath.Join(stateDir, "manifest.json"))
	if err != nil {
		return nil, err
	}
	currentState, err := readJSONFile[models.CurrentStateState](filepath.Join(stateDir, "current_state.json"))
	if err != nil {
		return nil, err
	}
	hooks, err := readJSONFile[models.HooksState](filepath.Join(stateDir, "hooks.json"))
	if err != nil {
		return nil, err
	}
	chapterSummaries, err := readJSONFile[models.ChapterSummariesState](filepath.Join(stateDir, "chapter_summaries.json"))
	if err != nil {
		return nil, err
	}

	snapshot := &RuntimeStateSnapshot{
		Manifest:         manifest,
		CurrentState:     currentState,
		Hooks:            hooks,
		ChapterSummaries: chapterSummaries,
	}

	issues := ValidateRuntimeState(*snapshot)
	if len(issues) > 0 {
		summary := ""
		for i, issue := range issues {
			if i > 0 {
				summary += ", "
			}
			summary += issue.Code
		}
		return nil, fmt.Errorf("invalid persisted runtime state: %s", summary)
	}

	return snapshot, nil
}

// BuildRuntimeStateArtifacts 应用a delta and renders projections。
func BuildRuntimeStateArtifacts(bookDir string, delta models.RuntimeStateDelta, language string) (*RuntimeStateArtifacts, error) {
	snapshot, err := LoadRuntimeStateSnapshot(bookDir)
	if err != nil {
		return nil, err
	}

	// Keep delta as-is in state package to avoid cross-package cycle from hook arbiter helpers.
	resolvedDelta := delta

	next, err := ApplyRuntimeStateDelta(*snapshot, resolvedDelta)
	if err != nil {
		return nil, err
	}

	return &RuntimeStateArtifacts{
		Snapshot:                 next,
		ResolvedDelta:            resolvedDelta,
		CurrentStateMarkdown:     RenderCurrentStateProjection(next.CurrentState, language),
		HooksMarkdown:            RenderHooksProjection(next.Hooks, language),
		ChapterSummariesMarkdown: RenderChapterSummariesProjection(next.ChapterSummaries, language),
	}, nil
}

// SaveRuntimeStateSnapshot persists snapshot to story/state json files.
func SaveRuntimeStateSnapshot(bookDir string, snapshot RuntimeStateSnapshot) error {
	stateDir := filepath.Join(bookDir, "story", "state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return err
	}

	if err := writeJSONFile(filepath.Join(stateDir, "manifest.json"), snapshot.Manifest); err != nil {
		return err
	}
	if err := writeJSONFile(filepath.Join(stateDir, "current_state.json"), snapshot.CurrentState); err != nil {
		return err
	}
	if err := writeJSONFile(filepath.Join(stateDir, "hooks.json"), snapshot.Hooks); err != nil {
		return err
	}
	if err := writeJSONFile(filepath.Join(stateDir, "chapter_summaries.json"), snapshot.ChapterSummaries); err != nil {
		return err
	}
	return nil
}

// LoadNarrativeMemorySeed 加载summaries/hooks from runtime snapshot。
func LoadNarrativeMemorySeed(bookDir string) (*NarrativeMemorySeed, error) {
	snapshot, err := LoadRuntimeStateSnapshot(bookDir)
	if err != nil {
		return nil, err
	}

	summaries := make([]StoredSummary, 0, len(snapshot.ChapterSummaries.Rows))
	for _, row := range snapshot.ChapterSummaries.Rows {
		summaries = append(summaries, StoredSummary{
			Chapter:      row.Chapter,
			Title:        row.Title,
			Characters:   row.Characters,
			Events:       row.Events,
			StateChanges: row.StateChanges,
			HookActivity: row.HookActivity,
			Mood:         row.Mood,
			ChapterType:  row.ChapterType,
		})
	}

	hooks := make([]StoredHook, 0, len(snapshot.Hooks.Hooks))
	for _, hook := range snapshot.Hooks.Hooks {
		timing := ""
		if hook.PayoffTiming != nil {
			timing = string(*hook.PayoffTiming)
		}
		hooks = append(hooks, StoredHook{
			HookID:              hook.HookID,
			StartChapter:        hook.StartChapter,
			Type:                hook.Type,
			Status:              string(hook.Status),
			LastAdvancedChapter: hook.LastAdvancedChapter,
			ExpectedPayoff:      hook.ExpectedPayoff,
			PayoffTiming:        timing,
			Notes:               hook.Notes,
		})
	}

	return &NarrativeMemorySeed{
		Summaries: summaries,
		Hooks:     hooks,
	}, nil
}

// LoadSnapshotCurrentStateFacts 加载current-state facts from a historical snapshot。
func LoadSnapshotCurrentStateFacts(bookDir string, chapterNumber int) ([]Fact, error) {
	snapshotDir := filepath.Join(bookDir, "story", "snapshots", fmt.Sprintf("%d", chapterNumber))
	structuredPath := filepath.Join(snapshotDir, "state", "current_state.json")
	if fileExists(structuredPath) {
		structured, err := readJSONFile[models.CurrentStateState](structuredPath)
		if err == nil {
			facts := make([]Fact, 0, len(structured.Facts))
			for _, fact := range structured.Facts {
				facts = append(facts, Fact{
					Subject:           fact.Subject,
					Predicate:         fact.Predicate,
					Object:            fact.Object,
					ValidFromChapter:  fact.ValidFromChapter,
					ValidUntilChapter: fact.ValidUntilChapter,
					SourceChapter:     fact.SourceChapter,
				})
			}
			return facts, nil
		}
	}

	markdownPath := filepath.Join(snapshotDir, "current_state.md")
	markdown, _ := os.ReadFile(markdownPath)
	return ParseCurrentStateFacts(string(markdown), chapterNumber), nil
}

func readJSONFile[T any](path string) (T, error) {
	var target T
	raw, err := os.ReadFile(path)
	if err != nil {
		return target, err
	}
	if err := json.Unmarshal(raw, &target); err != nil {
		return target, err
	}
	return target, nil
}

func writeJSONFile(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0644)
}
