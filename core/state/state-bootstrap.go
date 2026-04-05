package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/icosmos-space/ipen/core/models"
)

// BootstrapStructuredStateResult captures bootstrap artifacts.
type BootstrapStructuredStateResult struct {
	CreatedFiles []string             `json:"createdFiles"`
	Warnings     []string             `json:"warnings"`
	Manifest     models.StateManifest `json:"manifest"`
}

type markdownBootstrapState struct {
	SummariesState       models.ChapterSummariesState
	HooksState           models.HooksState
	CurrentState         models.CurrentStateState
	DurableStoryProgress int
}

// BootstrapStructuredStateFromMarkdown 构建missing runtime JSON state from markdown truth files。
func BootstrapStructuredStateFromMarkdown(bookDir string, fallbackChapter int) (*BootstrapStructuredStateResult, error) {
	storyDir := filepath.Join(bookDir, "story")
	stateDir := filepath.Join(storyDir, "state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, err
	}

	manifestPath := filepath.Join(stateDir, "manifest.json")
	currentStatePath := filepath.Join(stateDir, "current_state.json")
	hooksPath := filepath.Join(stateDir, "hooks.json")
	summariesPath := filepath.Join(stateDir, "chapter_summaries.json")

	createdFiles := []string{}
	warnings := []string{}

	existingManifest, _ := loadJSONIfValid[models.StateManifest](manifestPath, "manifest.json", &warnings)
	language := resolveRuntimeLanguage(bookDir)
	if existingManifest != nil && (existingManifest.Language == models.LanguageEN || existingManifest.Language == models.LanguageZH) {
		language = existingManifest.Language
	}

	markdownState, err := loadMarkdownBootstrapState(bookDir, storyDir, fallbackChapter, &warnings)
	if err != nil {
		return nil, err
	}

	summariesState, err := loadOrBootstrapSummaries(storyDir, summariesPath, markdownState.SummariesState, &createdFiles, &warnings, false)
	if err != nil {
		return nil, err
	}

	hooksState, err := loadOrBootstrapHooks(storyDir, hooksPath, markdownState.HooksState, &createdFiles, &warnings, false)
	if err != nil {
		return nil, err
	}

	currentState, err := loadOrBootstrapCurrentState(storyDir, currentStatePath, markdownState.CurrentState, markdownState.DurableStoryProgress, &createdFiles, &warnings, false)
	if err != nil {
		return nil, err
	}

	derivedProgress := markdownState.DurableStoryProgress
	if existingManifest != nil && existingManifest.LastAppliedChapter > derivedProgress {
		appendWarning(&warnings, fmt.Sprintf("manifest lastAppliedChapter normalized from %d to %d", existingManifest.LastAppliedChapter, derivedProgress))
	}

	manifest := models.StateManifest{
		SchemaVersion:      2,
		Language:           language,
		LastAppliedChapter: derivedProgress,
		ProjectionVersion:  1,
		MigrationWarnings:  uniqueNonEmptyStrings(append(append([]string{}, warnings...), migrationWarnings(existingManifest)...)),
	}
	if existingManifest != nil && existingManifest.ProjectionVersion > 0 {
		manifest.ProjectionVersion = existingManifest.ProjectionVersion
	}

	if err := writeJSON(manifestPath, manifest); err != nil {
		return nil, err
	}
	if existingManifest == nil {
		createdFiles = append(createdFiles, "manifest.json")
	}

	// Ensure chapter alignment to durable progress to avoid stale optimistic chapters in markdown tables.
	if currentState.Chapter != manifest.LastAppliedChapter {
		currentState.Chapter = manifest.LastAppliedChapter
		if err := writeJSON(currentStatePath, currentState); err != nil {
			return nil, err
		}
	}

	// Normalize snapshots into deterministic sort order.
	summariesState.Rows = deduplicateSummaryRows(summariesState.Rows)
	if err := writeJSON(summariesPath, summariesState); err != nil {
		return nil, err
	}
	hooksState.Hooks = sortHooks(hooksState.Hooks)
	if err := writeJSON(hooksPath, hooksState); err != nil {
		return nil, err
	}

	result := &BootstrapStructuredStateResult{
		CreatedFiles: uniqueNonEmptyStrings(createdFiles),
		Warnings:     manifest.MigrationWarnings,
		Manifest:     manifest,
	}
	return result, nil
}

// RewriteStructuredStateFromMarkdown forces a complete rebuild from markdown.
func RewriteStructuredStateFromMarkdown(bookDir string, fallbackChapter int) (*BootstrapStructuredStateResult, error) {
	storyDir := filepath.Join(bookDir, "story")
	stateDir := filepath.Join(storyDir, "state")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, err
	}

	manifestPath := filepath.Join(stateDir, "manifest.json")
	currentStatePath := filepath.Join(stateDir, "current_state.json")
	hooksPath := filepath.Join(stateDir, "hooks.json")
	summariesPath := filepath.Join(stateDir, "chapter_summaries.json")

	warnings := []string{}
	existingManifest, _ := loadJSONIfValid[models.StateManifest](manifestPath, "manifest.json", &warnings)
	language := resolveRuntimeLanguage(bookDir)
	if existingManifest != nil && (existingManifest.Language == models.LanguageEN || existingManifest.Language == models.LanguageZH) {
		language = existingManifest.Language
	}

	markdownState, err := loadMarkdownBootstrapState(bookDir, storyDir, fallbackChapter, &warnings)
	if err != nil {
		return nil, err
	}

	manifest := models.StateManifest{
		SchemaVersion:      2,
		Language:           language,
		LastAppliedChapter: markdownState.DurableStoryProgress,
		ProjectionVersion:  1,
		MigrationWarnings:  uniqueNonEmptyStrings(append(append([]string{}, warnings...), migrationWarnings(existingManifest)...)),
	}
	if existingManifest != nil && existingManifest.ProjectionVersion > 0 {
		manifest.ProjectionVersion = existingManifest.ProjectionVersion
	}

	if err := writeJSON(manifestPath, manifest); err != nil {
		return nil, err
	}
	if err := writeJSON(currentStatePath, markdownState.CurrentState); err != nil {
		return nil, err
	}
	if err := writeJSON(hooksPath, models.HooksState{Hooks: sortHooks(markdownState.HooksState.Hooks)}); err != nil {
		return nil, err
	}
	if err := writeJSON(summariesPath, models.ChapterSummariesState{Rows: deduplicateSummaryRows(markdownState.SummariesState.Rows)}); err != nil {
		return nil, err
	}

	return &BootstrapStructuredStateResult{
		CreatedFiles: []string{},
		Warnings:     manifest.MigrationWarnings,
		Manifest:     manifest,
	}, nil
}

// ResolveDurableStoryProgress 解析contiguous chapter progress from durable artifacts。
func ResolveDurableStoryProgress(bookDir string, fallbackChapter int) (int, error) {
	explicitFallback := normalizeExplicitChapter(fallbackChapter)
	durable, err := resolveContiguousArtifactChapterProgress(bookDir)
	if err != nil {
		return explicitFallback, err
	}
	if durable > explicitFallback {
		return durable, nil
	}
	return explicitFallback, nil
}

func loadMarkdownBootstrapState(bookDir, storyDir string, fallbackChapter int, warnings *[]string) (*markdownBootstrapState, error) {
	summariesState, err := loadMarkdownSummariesState(storyDir)
	if err != nil {
		return nil, err
	}
	hooksState, err := loadMarkdownHooksState(storyDir, warnings)
	if err != nil {
		return nil, err
	}

	explicitFallback := normalizeExplicitChapter(fallbackChapter)
	durableArtifactProgress, err := resolveContiguousArtifactChapterProgress(bookDir)
	if err != nil {
		durableArtifactProgress = 0
	}
	authoritativeProgress := maxInt(explicitFallback, durableArtifactProgress)
	currentState, err := loadMarkdownCurrentState(storyDir, authoritativeProgress, warnings)
	if err != nil {
		return nil, err
	}

	return &markdownBootstrapState{
		SummariesState:       summariesState,
		HooksState:           hooksState,
		CurrentState:         currentState,
		DurableStoryProgress: authoritativeProgress,
	}, nil
}

func loadOrBootstrapCurrentState(
	storyDir string,
	statePath string,
	bootstrap models.CurrentStateState,
	fallbackChapter int,
	createdFiles *[]string,
	warnings *[]string,
	forceBootstrap bool,
) (models.CurrentStateState, error) {
	if !forceBootstrap {
		existing, err := loadJSONIfValid[models.CurrentStateState](statePath, "current_state.json", warnings)
		if err == nil && existing != nil {
			return *existing, nil
		}
	}

	state := bootstrap
	if state.Chapter <= 0 {
		state.Chapter = maxInt(0, fallbackChapter)
	}

	existed := fileExists(statePath)
	if err := writeJSON(statePath, state); err != nil {
		return models.CurrentStateState{}, err
	}
	if !existed {
		*createdFiles = append(*createdFiles, "current_state.json")
	}
	return state, nil
}

func loadOrBootstrapHooks(
	storyDir string,
	statePath string,
	bootstrap models.HooksState,
	createdFiles *[]string,
	warnings *[]string,
	forceBootstrap bool,
) (models.HooksState, error) {
	if !forceBootstrap {
		existing, err := loadJSONIfValid[models.HooksState](statePath, "hooks.json", warnings)
		if err == nil && existing != nil {
			existing.Hooks = sortHooks(existing.Hooks)
			return *existing, nil
		}
	}

	hooksState := models.HooksState{Hooks: sortHooks(bootstrap.Hooks)}
	existed := fileExists(statePath)
	if err := writeJSON(statePath, hooksState); err != nil {
		return models.HooksState{}, err
	}
	if !existed {
		*createdFiles = append(*createdFiles, "hooks.json")
	}
	return hooksState, nil
}

func loadOrBootstrapSummaries(
	storyDir string,
	statePath string,
	bootstrap models.ChapterSummariesState,
	createdFiles *[]string,
	warnings *[]string,
	forceBootstrap bool,
) (models.ChapterSummariesState, error) {
	if !forceBootstrap {
		existing, err := loadJSONIfValid[models.ChapterSummariesState](statePath, "chapter_summaries.json", warnings)
		if err == nil && existing != nil {
			repaired := models.ChapterSummariesState{Rows: deduplicateSummaryRows(existing.Rows)}
			if len(repaired.Rows) < len(existing.Rows) {
				if writeErr := writeJSON(statePath, repaired); writeErr != nil {
					return models.ChapterSummariesState{}, writeErr
				}
			}
			return repaired, nil
		}
	}

	summariesState := models.ChapterSummariesState{Rows: deduplicateSummaryRows(bootstrap.Rows)}
	existed := fileExists(statePath)
	if err := writeJSON(statePath, summariesState); err != nil {
		return models.ChapterSummariesState{}, err
	}
	if !existed {
		*createdFiles = append(*createdFiles, "chapter_summaries.json")
	}
	return summariesState, nil
}

func loadMarkdownSummariesState(storyDir string) (models.ChapterSummariesState, error) {
	markdown, _ := os.ReadFile(filepath.Join(storyDir, "chapter_summaries.md"))
	rows := ParseChapterSummariesMarkdown(string(markdown))
	result := make([]models.ChapterSummaryRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, models.ChapterSummaryRow{
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
	return models.ChapterSummariesState{Rows: deduplicateSummaryRows(result)}, nil
}

func loadMarkdownHooksState(storyDir string, warnings *[]string) (models.HooksState, error) {
	markdown, _ := os.ReadFile(filepath.Join(storyDir, "pending_hooks.md"))
	hooks := ParsePendingHooksMarkdown(string(markdown), warnings)
	result := make([]models.HookRecord, 0, len(hooks))
	for _, hook := range hooks {
		var timing *models.HookPayoffTiming
		if strings.TrimSpace(hook.PayoffTiming) != "" {
			t := models.HookPayoffTiming(strings.TrimSpace(hook.PayoffTiming))
			timing = &t
		}
		result = append(result, models.HookRecord{
			HookID:              hook.HookID,
			StartChapter:        hook.StartChapter,
			Type:                hook.Type,
			Status:              models.HookStatus(strings.TrimSpace(hook.Status)),
			LastAdvancedChapter: hook.LastAdvancedChapter,
			ExpectedPayoff:      hook.ExpectedPayoff,
			PayoffTiming:        timing,
			Notes:               hook.Notes,
		})
	}
	for i := range result {
		if result[i].Status == "" {
			result[i].Status = models.HookStatusOpenRT
		}
	}
	return models.HooksState{Hooks: sortHooks(result)}, nil
}

func loadMarkdownCurrentState(storyDir string, fallbackChapter int, warnings *[]string) (models.CurrentStateState, error) {
	markdown, _ := os.ReadFile(filepath.Join(storyDir, "current_state.md"))
	facts := ParseCurrentStateFacts(string(markdown), fallbackChapter)

	stateChapter := maxInt(0, fallbackChapter)
	for _, fact := range facts {
		if fact.ValidFromChapter > stateChapter {
			stateChapter = fact.ValidFromChapter
		}
	}

	resultFacts := make([]models.CurrentStateFact, 0, len(facts))
	for _, fact := range facts {
		resultFacts = append(resultFacts, models.CurrentStateFact{
			Subject:           fact.Subject,
			Predicate:         fact.Predicate,
			Object:            fact.Object,
			ValidFromChapter:  fact.ValidFromChapter,
			ValidUntilChapter: fact.ValidUntilChapter,
			SourceChapter:     fact.SourceChapter,
		})
	}

	if len(resultFacts) == 0 {
		appendWarning(warnings, "current_state markdown is empty, bootstrapped with fallback chapter")
	}

	return models.CurrentStateState{Chapter: stateChapter, Facts: resultFacts}, nil
}

func resolveRuntimeLanguage(bookDir string) models.RuntimeStateLanguage {
	raw, err := os.ReadFile(filepath.Join(bookDir, "book.json"))
	if err != nil {
		return models.LanguageEN
	}
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return models.LanguageEN
	}
	if lang, ok := parsed["language"].(string); ok && strings.EqualFold(lang, "zh") {
		return models.LanguageZH
	}
	return models.LanguageEN
}

func resolveContiguousArtifactChapterProgress(bookDir string) (int, error) {
	numbers := loadDurableArtifactChapterNumbers(bookDir)
	return ResolveContiguousChapterPrefix(numbers), nil
}

func loadDurableArtifactChapterNumbers(bookDir string) []int {
	chaptersDir := filepath.Join(bookDir, "chapters")
	indexPath := filepath.Join(chaptersDir, "index.json")

	collected := []int{}

	if raw, err := os.ReadFile(indexPath); err == nil {
		var entries []map[string]any
		if json.Unmarshal(raw, &entries) == nil {
			for _, entry := range entries {
				if n, ok := entry["number"].(float64); ok {
					chapter := int(n)
					if chapter > 0 {
						collected = append(collected, chapter)
					}
				}
			}
		}
	}

	if files, err := os.ReadDir(chaptersDir); err == nil {
		re := regexp.MustCompile(`^(\d+)_`)
		for _, file := range files {
			match := re.FindStringSubmatch(file.Name())
			if len(match) == 2 {
				n, err := strconv.Atoi(match[1])
				if err == nil && n > 0 {
					collected = append(collected, n)
				}
			}
		}
	}

	return collected
}

// ResolveContiguousChapterPrefix 返回the highest contiguous chapter prefix starting at 1。
func ResolveContiguousChapterPrefix(chapterNumbers []int) int {
	chapters := map[int]struct{}{}
	for _, chapter := range chapterNumbers {
		if chapter > 0 {
			chapters[chapter] = struct{}{}
		}
	}

	contiguous := 0
	for {
		next := contiguous + 1
		if _, ok := chapters[next]; !ok {
			break
		}
		contiguous = next
	}
	return contiguous
}

func normalizeHookStatus(value string, warnings *[]string, hookID string) models.HookStatus {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch {
	case normalized == "":
		return models.HookStatusOpenRT
	case strings.Contains(normalized, "resolved") || strings.Contains(normalized, "closed") || strings.Contains(normalized, "done"):
		return models.HookStatusResolvedRT
	case strings.Contains(normalized, "deferred") || strings.Contains(normalized, "paused") || strings.Contains(normalized, "hold"):
		return models.HookStatusDeferred
	case strings.Contains(normalized, "progress") || strings.Contains(normalized, "active") || strings.Contains(normalized, "advance"):
		return models.HookStatusProgressingRT
	case strings.Contains(normalized, "open") || strings.Contains(normalized, "pending"):
		return models.HookStatusOpenRT
	default:
		appendWarning(warnings, fmt.Sprintf("%s:status normalized from %q to open", hookID, value))
		return models.HookStatusOpenRT
	}
}

func parseStrictIntegerWithWarning(value string, warnings *[]string, fieldLabel string) int {
	if strings.TrimSpace(value) == "" {
		return 0
	}
	if n, ok := parseStrictIntegerCell(value); ok {
		return n
	}
	appendWarning(warnings, fmt.Sprintf("%s normalized from %q to 0", fieldLabel, value))
	return 0
}

func parseStrictIntegerCell(value string) (int, bool) {
	cleaned := NormalizeHookID(value)
	if !regexp.MustCompile(`^\d+$`).MatchString(cleaned) {
		return 0, false
	}
	n, err := strconv.Atoi(cleaned)
	if err != nil {
		return 0, false
	}
	return n, true
}

func parseIntegerWithFallback(value string, fallback int, warnings *[]string, fieldLabel string) int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return maxInt(0, fallback)
	}
	match := regexp.MustCompile(`\d+`).FindString(trimmed)
	if match == "" {
		appendWarning(warnings, fmt.Sprintf("%s normalized from %q to %d", fieldLabel, value, maxInt(0, fallback)))
		return maxInt(0, fallback)
	}
	n, err := strconv.Atoi(match)
	if err != nil {
		appendWarning(warnings, fmt.Sprintf("%s normalized from %q to %d", fieldLabel, value, maxInt(0, fallback)))
		return maxInt(0, fallback)
	}
	return n
}

func normalizeExplicitChapter(value int) int {
	if value <= 0 {
		return 0
	}
	return value
}

func deduplicateSummaryRows(rows []models.ChapterSummaryRow) []models.ChapterSummaryRow {
	byChapter := map[int]models.ChapterSummaryRow{}
	for _, row := range rows {
		if row.Chapter <= 0 {
			continue
		}
		byChapter[row.Chapter] = row
	}
	result := make([]models.ChapterSummaryRow, 0, len(byChapter))
	for _, row := range byChapter {
		result = append(result, row)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Chapter < result[j].Chapter })
	return result
}

func sortHooks(hooks []models.HookRecord) []models.HookRecord {
	result := make([]models.HookRecord, len(hooks))
	copy(result, hooks)
	sort.Slice(result, func(i, j int) bool {
		if result[i].StartChapter != result[j].StartChapter {
			return result[i].StartChapter < result[j].StartChapter
		}
		if result[i].LastAdvancedChapter != result[j].LastAdvancedChapter {
			return result[i].LastAdvancedChapter < result[j].LastAdvancedChapter
		}
		return result[i].HookID < result[j].HookID
	})
	return result
}

func appendWarning(warnings *[]string, warning string) {
	if strings.TrimSpace(warning) == "" {
		return
	}
	for _, existing := range *warnings {
		if existing == warning {
			return
		}
	}
	*warnings = append(*warnings, warning)
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := []string{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func migrationWarnings(manifest *models.StateManifest) []string {
	if manifest == nil {
		return []string{}
	}
	return manifest.MigrationWarnings
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func writeJSON(path string, value any) error {
	bytes, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, bytes, 0644)
}

func loadJSONIfValid[T any](path string, fileLabel string, warnings *[]string) (*T, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var parsed T
	if err := json.Unmarshal(raw, &parsed); err != nil {
		appendWarning(warnings, fmt.Sprintf("%s invalid, rebuilt from markdown", fileLabel))
		return nil, err
	}

	return &parsed, nil
}

// NormalizeHookID removes markdown wrappers from hook ids.
func NormalizeHookID(value string) string {
	normalized := strings.TrimSpace(value)
	previous := ""
	for normalized != "" && normalized != previous {
		previous = normalized
		normalized = regexp.MustCompile(`^\[(.+?)\]\([^)]+\)$`).ReplaceAllString(normalized, "$1")
		normalized = regexp.MustCompile(`^\*\*(.+)\*\*$`).ReplaceAllString(normalized, "$1")
		normalized = regexp.MustCompile(`^__(.+)__$`).ReplaceAllString(normalized, "$1")
		normalized = regexp.MustCompile(`^\*(.+)\*$`).ReplaceAllString(normalized, "$1")
		normalized = regexp.MustCompile(`^_(.+)_$`).ReplaceAllString(normalized, "$1")
		normalized = regexp.MustCompile("^`(.+)`$").ReplaceAllString(normalized, "$1")
		normalized = regexp.MustCompile(`^~~(.+)~~$`).ReplaceAllString(normalized, "$1")
		normalized = strings.TrimSpace(normalized)
	}
	return normalized
}

// ParseChapterSummariesMarkdown 解析chapter summary markdown rows into stored summaries。
func ParseChapterSummariesMarkdown(markdown string) []StoredSummary {
	rows := parseMarkdownTableRows(markdown)
	result := []StoredSummary{}
	for _, row := range rows {
		if len(row) == 0 {
			continue
		}
		chapter, err := strconv.Atoi(strings.TrimSpace(row[0]))
		if err != nil || chapter <= 0 {
			continue
		}
		item := StoredSummary{Chapter: chapter}
		if len(row) > 1 {
			item.Title = row[1]
		}
		if len(row) > 2 {
			item.Characters = row[2]
		}
		if len(row) > 3 {
			item.Events = row[3]
		}
		if len(row) > 4 {
			item.StateChanges = row[4]
		}
		if len(row) > 5 {
			item.HookActivity = row[5]
		}
		if len(row) > 6 {
			item.Mood = row[6]
		}
		if len(row) > 7 {
			item.ChapterType = row[7]
		}
		result = append(result, item)
	}
	return result
}

// ParsePendingHooksMarkdown 解析pending hooks markdown table or bullet fallback。
func ParsePendingHooksMarkdown(markdown string, warnings *[]string) []StoredHook {
	tableRows := parseMarkdownTableRows(markdown)
	filtered := [][]string{}
	for _, row := range tableRows {
		if strings.EqualFold(strings.TrimSpace(getCell(row, 0)), "hook_id") {
			continue
		}
		filtered = append(filtered, row)
	}

	if len(filtered) > 0 {
		result := []StoredHook{}
		for _, row := range filtered {
			hookID := NormalizeHookID(getCell(row, 0))
			if hookID == "" {
				continue
			}
			legacyShape := len(row) < 8
			status := normalizeHookStatus(getCell(row, 3), warnings, hookID)
			payoffTiming := ""
			notes := ""
			if legacyShape {
				notes = getCell(row, 6)
			} else {
				payoffTiming = getCell(row, 6)
				notes = getCell(row, 7)
			}
			result = append(result, StoredHook{
				HookID:              hookID,
				StartChapter:        parseStrictIntegerWithWarning(getCell(row, 1), warnings, hookID+":startChapter"),
				Type:                defaultString(getCell(row, 2), "unspecified"),
				Status:              string(status),
				LastAdvancedChapter: parseStrictIntegerWithWarning(getCell(row, 4), warnings, hookID+":lastAdvancedChapter"),
				ExpectedPayoff:      getCell(row, 5),
				PayoffTiming:        payoffTiming,
				Notes:               notes,
			})
		}
		return result
	}

	lines := strings.Split(markdown, "\n")
	result := []StoredHook{}
	index := 1
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "-") {
			continue
		}
		note := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
		if note == "" {
			continue
		}
		result = append(result, StoredHook{
			HookID:              fmt.Sprintf("hook-%d", index),
			StartChapter:        0,
			Type:                "unspecified",
			Status:              string(models.HookStatusOpenRT),
			LastAdvancedChapter: 0,
			ExpectedPayoff:      "",
			PayoffTiming:        "",
			Notes:               note,
		})
		index++
	}
	return result
}

// ParseCurrentStateFacts 解析current_state markdown into temporal facts。
func ParseCurrentStateFacts(markdown string, fallbackChapter int) []Fact {
	tableRows := parseMarkdownTableRows(markdown)
	fieldRows := [][]string{}
	for _, row := range tableRows {
		if len(row) < 2 {
			continue
		}
		if isStateTableHeaderRow(row) {
			continue
		}
		fieldRows = append(fieldRows, row)
	}

	if len(fieldRows) > 0 {
		stateChapter := maxInt(0, fallbackChapter)
		for _, row := range fieldRows {
			if isCurrentChapterLabel(getCell(row, 0)) {
				stateChapter = parseIntegerWithFallback(getCell(row, 1), stateChapter, &[]string{}, "current_state:chapter")
			}
		}

		facts := []Fact{}
		for _, row := range fieldRows {
			label := strings.TrimSpace(getCell(row, 0))
			value := strings.TrimSpace(getCell(row, 1))
			if label == "" || value == "" || isCurrentChapterLabel(label) {
				continue
			}
			facts = append(facts, Fact{
				Subject:           inferFactSubject(label),
				Predicate:         label,
				Object:            value,
				ValidFromChapter:  stateChapter,
				ValidUntilChapter: nil,
				SourceChapter:     stateChapter,
			})
		}
		return facts
	}

	bulletFacts := []string{}
	for _, line := range strings.Split(markdown, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "-") {
			continue
		}
		item := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
		if item != "" {
			bulletFacts = append(bulletFacts, item)
		}
	}

	facts := []Fact{}
	for i, item := range bulletFacts {
		facts = append(facts, Fact{
			Subject:           "current_state",
			Predicate:         fmt.Sprintf("note_%d", i+1),
			Object:            item,
			ValidFromChapter:  maxInt(0, fallbackChapter),
			ValidUntilChapter: nil,
			SourceChapter:     maxInt(0, fallbackChapter),
		})
	}
	return facts
}

func parseMarkdownTableRows(markdown string) [][]string {
	rows := [][]string{}
	for _, line := range strings.Split(markdown, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") {
			continue
		}
		if strings.Contains(trimmed, "---") {
			continue
		}
		parts := strings.Split(trimmed, "|")
		if len(parts) < 3 {
			continue
		}
		cells := []string{}
		for _, cell := range parts[1 : len(parts)-1] {
			cells = append(cells, strings.TrimSpace(cell))
		}
		hasValue := false
		for _, cell := range cells {
			if cell != "" {
				hasValue = true
				break
			}
		}
		if hasValue {
			rows = append(rows, cells)
		}
	}
	return rows
}

func isStateTableHeaderRow(row []string) bool {
	first := strings.ToLower(strings.TrimSpace(getCell(row, 0)))
	second := strings.ToLower(strings.TrimSpace(getCell(row, 1)))
	return (first == "字段" && second == "值") || (first == "field" && second == "value")
}

func isCurrentChapterLabel(label string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(label))
	return trimmed == "当前章节" || trimmed == "current chapter"
}

func inferFactSubject(label string) string {
	trimmed := strings.ToLower(strings.TrimSpace(label))
	switch trimmed {
	case "当前位置", "current location", "主角状态", "protagonist state", "当前目标", "current goal", "当前限制", "current constraint", "当前敌我", "当前关系", "current alliances", "current relationships", "当前冲突", "current conflict":
		return "protagonist"
	default:
		return "current_state"
	}
}

func getCell(row []string, index int) string {
	if index >= 0 && index < len(row) {
		return strings.TrimSpace(row[index])
	}
	return ""
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
