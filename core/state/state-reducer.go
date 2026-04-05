package state

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/icosmos-space/ipen/core/models"
)

// RuntimeStateSnapshot 表示a runtime state snapshot。
type RuntimeStateSnapshot struct {
	Manifest         models.StateManifest         `json:"manifest"`
	CurrentState     models.CurrentStateState     `json:"currentState"`
	Hooks            models.HooksState            `json:"hooks"`
	ChapterSummaries models.ChapterSummariesState `json:"chapterSummaries"`
}

// ApplyRuntimeStateDelta 应用a delta to the snapshot。
func ApplyRuntimeStateDelta(snapshot RuntimeStateSnapshot, delta models.RuntimeStateDelta) (RuntimeStateSnapshot, error) {
	if delta.Chapter <= snapshot.Manifest.LastAppliedChapter {
		return RuntimeStateSnapshot{}, fmt.Errorf("delta chapter %d goes backwards", delta.Chapter)
	}

	if delta.ChapterSummary != nil && delta.ChapterSummary.Chapter != delta.Chapter {
		return RuntimeStateSnapshot{}, fmt.Errorf(
			"chapter summary %d does not match delta chapter %d",
			delta.ChapterSummary.Chapter,
			delta.Chapter,
		)
	}

	if delta.ChapterSummary != nil {
		for _, row := range snapshot.ChapterSummaries.Rows {
			if row.Chapter == delta.ChapterSummary.Chapter {
				return RuntimeStateSnapshot{}, fmt.Errorf("duplicate summary row for chapter %d", delta.ChapterSummary.Chapter)
			}
		}
	}

	hooks := applyHookOps(snapshot.Hooks, delta)
	currentState := applyCurrentStatePatch(snapshot.CurrentState, snapshot.Manifest.Language, delta)
	chapterSummaries := applySummaryDelta(snapshot.ChapterSummaries, delta)

	next := RuntimeStateSnapshot{
		Manifest: models.StateManifest{
			SchemaVersion:      snapshot.Manifest.SchemaVersion,
			Language:           snapshot.Manifest.Language,
			LastAppliedChapter: delta.Chapter,
			ProjectionVersion:  snapshot.Manifest.ProjectionVersion,
			MigrationWarnings:  snapshot.Manifest.MigrationWarnings,
		},
		CurrentState:     currentState,
		Hooks:            hooks,
		ChapterSummaries: chapterSummaries,
	}

	issues := ValidateRuntimeState(next)
	if len(issues) > 0 {
		messages := make([]string, len(issues))
		for i, issue := range issues {
			messages[i] = fmt.Sprintf("%s: %s", issue.Code, issue.Message)
		}
		return RuntimeStateSnapshot{}, errors.New(strings.Join(messages, "; "))
	}

	return next, nil
}

func applyHookOps(hooksState models.HooksState, delta models.RuntimeStateDelta) models.HooksState {
	hooksByID := make(map[string]models.HookRecord, len(hooksState.Hooks))
	for _, hook := range hooksState.Hooks {
		hooksByID[hook.HookID] = hook
	}

	for _, hook := range delta.HookOps.Upsert {
		if _, exists := hooksByID[hook.HookID]; !exists {
			activeHooks := make([]models.HookRecord, 0, len(hooksByID))
			for _, candidate := range hooksByID {
				if candidate.Status != "resolved" {
					activeHooks = append(activeHooks, candidate)
				}
			}

			admission := EvaluateHookAdmission(
				HookAdmissionCandidate{
					Type:           hook.Type,
					ExpectedPayoff: hook.ExpectedPayoff,
					Notes:          hook.Notes,
				},
				activeHooks,
			)
			if !admission.Admit && admission.Reason == "duplicate_family" {
				if existing, ok := hooksByID[admission.MatchedHookID]; ok {
					hooksByID[existing.HookID] = mergeDuplicateHookFamily(existing, hook)
					continue
				}
			}
		}

		hooksByID[hook.HookID] = hook
	}

	for _, hookID := range delta.HookOps.Resolve {
		existing, exists := hooksByID[hookID]
		if !exists {
			continue
		}

		existing.Status = "resolved"
		existing.LastAdvancedChapter = maxInt(existing.LastAdvancedChapter, delta.Chapter)
		hooksByID[hookID] = existing
	}

	for _, hookID := range delta.HookOps.Defer {
		existing, exists := hooksByID[hookID]
		if !exists {
			continue
		}

		existing.Status = "deferred"
		existing.LastAdvancedChapter = maxInt(existing.LastAdvancedChapter, delta.Chapter)
		hooksByID[hookID] = existing
	}

	sorted := make([]models.HookRecord, 0, len(hooksByID))
	for _, record := range hooksByID {
		sorted = append(sorted, record)
	}

	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].StartChapter != sorted[j].StartChapter {
			return sorted[i].StartChapter < sorted[j].StartChapter
		}
		if sorted[i].LastAdvancedChapter != sorted[j].LastAdvancedChapter {
			return sorted[i].LastAdvancedChapter < sorted[j].LastAdvancedChapter
		}
		return sorted[i].HookID < sorted[j].HookID
	})

	return models.HooksState{Hooks: sorted}
}

func mergeDuplicateHookFamily(existing models.HookRecord, incoming models.HookRecord) models.HookRecord {
	expectedPayoff := preferRicherText(existing.ExpectedPayoff, incoming.ExpectedPayoff)
	notes := preferRicherText(existing.Notes, incoming.Notes)
	advanced := maxInt(existing.LastAdvancedChapter, incoming.LastAdvancedChapter)
	progressed := advanced > existing.LastAdvancedChapter

	status := existing.Status
	if progressed {
		status = "progressing"
	} else if existing.Status == "progressing" || incoming.Status == "progressing" {
		status = "progressing"
	}

	startChapter := minInt(existing.StartChapter, incoming.StartChapter)
	hookType := preferRicherText(existing.Type, incoming.Type)
	payoffTiming := ResolveHookPayoffTiming(incoming.PayoffTiming, expectedPayoff, notes)

	return models.HookRecord{
		HookID:              existing.HookID,
		StartChapter:        startChapter,
		Type:                hookType,
		Status:              status,
		LastAdvancedChapter: advanced,
		ExpectedPayoff:      expectedPayoff,
		PayoffTiming:        &payoffTiming,
		Notes:               notes,
	}
}

func preferRicherText(primary string, fallback string) string {
	left := strings.TrimSpace(primary)
	right := strings.TrimSpace(fallback)

	if left == "" {
		return right
	}
	if right == "" {
		return left
	}
	if left == right {
		return left
	}
	if len(right) > len(left) {
		return right
	}
	return left
}

func applyCurrentStatePatch(
	currentState models.CurrentStateState,
	language models.RuntimeStateLanguage,
	delta models.RuntimeStateDelta,
) models.CurrentStateState {
	if delta.CurrentStatePatch == nil {
		copied := make([]models.CurrentStateFact, len(currentState.Facts))
		copy(copied, currentState.Facts)
		return models.CurrentStateState{
			Chapter: delta.Chapter,
			Facts:   copied,
		}
	}

	var labels map[string][]string
	if language == "en" {
		labels = map[string][]string{
			"currentLocation":   {"Current Location", "当前位置"},
			"protagonistState":  {"Protagonist State", "主角状态"},
			"currentGoal":       {"Current Goal", "当前目标"},
			"currentConstraint": {"Current Constraint", "当前限制"},
			"currentAlliances":  {"Current Alliances", "Current Relationships", "当前敌我", "当前关系"},
			"currentConflict":   {"Current Conflict", "当前冲突"},
		}
	} else {
		labels = map[string][]string{
			"currentLocation":   {"当前位置", "Current Location"},
			"protagonistState":  {"主角状态", "Protagonist State"},
			"currentGoal":       {"当前目标", "Current Goal"},
			"currentConstraint": {"当前限制", "Current Constraint"},
			"currentAlliances":  {"当前敌我", "当前关系", "Current Alliances", "Current Relationships"},
			"currentConflict":   {"当前冲突", "Current Conflict"},
		}
	}

	nextFacts := make([]models.CurrentStateFact, len(currentState.Facts))
	copy(nextFacts, currentState.Facts)

	patch := *delta.CurrentStatePatch
	if patch.CurrentLocation != nil {
		nextFacts = applyFactPatch(nextFacts, labels["currentLocation"], *patch.CurrentLocation, delta.Chapter)
	}
	if patch.ProtagonistState != nil {
		nextFacts = applyFactPatch(nextFacts, labels["protagonistState"], *patch.ProtagonistState, delta.Chapter)
	}
	if patch.CurrentGoal != nil {
		nextFacts = applyFactPatch(nextFacts, labels["currentGoal"], *patch.CurrentGoal, delta.Chapter)
	}
	if patch.CurrentConstraint != nil {
		nextFacts = applyFactPatch(nextFacts, labels["currentConstraint"], *patch.CurrentConstraint, delta.Chapter)
	}
	if patch.CurrentAlliances != nil {
		nextFacts = applyFactPatch(nextFacts, labels["currentAlliances"], *patch.CurrentAlliances, delta.Chapter)
	}
	if patch.CurrentConflict != nil {
		nextFacts = applyFactPatch(nextFacts, labels["currentConflict"], *patch.CurrentConflict, delta.Chapter)
	}

	sort.Slice(nextFacts, func(i, j int) bool {
		if nextFacts[i].Predicate != nextFacts[j].Predicate {
			return nextFacts[i].Predicate < nextFacts[j].Predicate
		}
		return nextFacts[i].Object < nextFacts[j].Object
	})

	return models.CurrentStateState{
		Chapter: delta.Chapter,
		Facts:   nextFacts,
	}
}

func applyFactPatch(
	facts []models.CurrentStateFact,
	aliases []string,
	value string,
	chapter int,
) []models.CurrentStateFact {
	filtered := make([]models.CurrentStateFact, 0, len(facts)+1)
	for _, fact := range facts {
		matched := false
		for _, alias := range aliases {
			if strings.EqualFold(fact.Predicate, alias) {
				matched = true
				break
			}
		}
		if !matched {
			filtered = append(filtered, fact)
		}
	}

	filtered = append(filtered, models.CurrentStateFact{
		Subject:           "protagonist",
		Predicate:         aliases[0],
		Object:            value,
		ValidFromChapter:  chapter,
		ValidUntilChapter: nil,
		SourceChapter:     chapter,
	})

	return filtered
}

func applySummaryDelta(
	state models.ChapterSummariesState,
	delta models.RuntimeStateDelta,
) models.ChapterSummariesState {
	rows := make([]models.ChapterSummaryRow, len(state.Rows))
	copy(rows, state.Rows)

	if delta.ChapterSummary != nil {
		rows = append(rows, *delta.ChapterSummary)
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Chapter < rows[j].Chapter
	})

	return models.ChapterSummariesState{Rows: rows}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
