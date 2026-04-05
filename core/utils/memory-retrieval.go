package utils

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/icosmos-space/ipen/core/state"
)

// MemorySelection 是the selected memory package used by planner/composer。
type MemorySelection struct {
	Summaries       []state.StoredSummary
	Hooks           []state.StoredHook
	ActiveHooks     []state.StoredHook
	Facts           []state.Fact
	VolumeSummaries []VolumeSummarySelection
	DBPath          string
}

// VolumeSummarySelection 描述one selected section from volume_summaries.md。
type VolumeSummarySelection struct {
	Heading string
	Content string
	Anchor  string
}

// RetrieveMemorySelection retrieves and ranks narrative memory for current chapter planning.
func RetrieveMemorySelection(bookDir string, chapterNumber int, goal string, outlineNode string, mustKeep []string) (MemorySelection, error) {
	storyDir := filepath.Join(bookDir, "story")
	fallbackChapter := maxInt(0, chapterNumber-1)

	currentStateMarkdown := readTextFile(filepath.Join(storyDir, "current_state.md"))
	volumeSummariesMarkdown := readTextFile(filepath.Join(storyDir, "volume_summaries.md"))
	summariesMarkdown := readTextFile(filepath.Join(storyDir, "chapter_summaries.md"))
	hooksMarkdown := readTextFile(filepath.Join(storyDir, "pending_hooks.md"))

	facts := ParseCurrentStateFacts(currentStateMarkdown, fallbackChapter)
	narrativeQueryTerms := ExtractQueryTerms(goal, outlineNode, nil)
	factQueryTerms := ExtractQueryTerms(goal, outlineNode, mustKeep)
	volumeSummaries := selectRelevantVolumeSummaries(parseVolumeSummariesMarkdown(volumeSummariesMarkdown), narrativeQueryTerms)

	memoryDB := openMemoryDB(bookDir)
	if memoryDB != nil {
		defer memoryDB.Close()

		chapterCount, err := memoryDB.GetChapterCount()
		if err == nil && chapterCount == 0 {
			summaries := ParseChapterSummariesMarkdown(summariesMarkdown)
			if len(summaries) > 0 {
				_ = memoryDB.ReplaceSummaries(summaries)
			}
		}

		activeHooksFromDB, err := memoryDB.GetActiveHooks()
		if err == nil && len(activeHooksFromDB) == 0 {
			hooks := ParsePendingHooksMarkdown(hooksMarkdown)
			if len(hooks) > 0 {
				_ = memoryDB.ReplaceHooks(hooks)
			}
		}

		currentFactsFromDB, err := memoryDB.GetCurrentFacts()
		if err == nil && len(currentFactsFromDB) == 0 && len(facts) > 0 {
			_ = memoryDB.ReplaceCurrentFacts(facts)
		}

		activeHooks, _ := memoryDB.GetActiveHooks()
		summaries, _ := memoryDB.GetSummaries(1, maxInt(1, chapterNumber-1))
		currentFacts, _ := memoryDB.GetCurrentFacts()

		return MemorySelection{
			Summaries:       selectRelevantSummaries(summaries, chapterNumber, narrativeQueryTerms),
			Hooks:           selectRelevantHooks(activeHooks, narrativeQueryTerms, chapterNumber),
			ActiveHooks:     activeHooks,
			Facts:           selectRelevantFacts(currentFacts, factQueryTerms),
			VolumeSummaries: volumeSummaries,
			DBPath:          filepath.Join(storyDir, "memory.db"),
		}, nil
	}

	summaries := ParseChapterSummariesMarkdown(summariesMarkdown)
	hooks := ParsePendingHooksMarkdown(hooksMarkdown)
	activeHooks := FilterActiveHooks(hooks)

	return MemorySelection{
		Summaries:       selectRelevantSummaries(summaries, chapterNumber, narrativeQueryTerms),
		Hooks:           selectRelevantHooks(activeHooks, narrativeQueryTerms, chapterNumber),
		ActiveHooks:     activeHooks,
		Facts:           selectRelevantFacts(facts, factQueryTerms),
		VolumeSummaries: volumeSummaries,
	}, nil
}

// ExtractQueryTerms 提取ranked focus terms from goal/outline/must-keep constraints。
func ExtractQueryTerms(goal string, outlineNode string, mustKeep []string) []string {
	primary := uniqueTerms(append(extractTermsFromText(stripNegativeGuidance(goal)), flattenTerms(mustKeep)...))
	if len(primary) >= 2 {
		if len(primary) > 12 {
			return primary[:12]
		}
		return primary
	}
	secondary := uniqueTerms(append(primary, extractTermsFromText(stripNegativeGuidance(outlineNode))...))
	if len(secondary) > 12 {
		return secondary[:12]
	}
	return secondary
}

func flattenTerms(values []string) []string {
	result := []string{}
	for _, value := range values {
		result = append(result, extractTermsFromText(value)...)
	}
	return result
}

func openMemoryDB(bookDir string) *state.MemoryDB {
	mdb, err := state.NewMemoryDB(bookDir)
	if err != nil {
		return nil
	}
	return mdb
}

func readTextFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func extractTermsFromText(text string) []string {
	if strings.TrimSpace(text) == "" {
		return []string{}
	}

	stopWords := map[string]struct{}{
		"bring": {}, "focus": {}, "back": {}, "chapter": {}, "clear": {}, "narrative": {}, "before": {}, "opening": {},
		"track": {}, "the": {}, "with": {}, "from": {}, "that": {}, "this": {}, "into": {}, "still": {}, "cannot": {},
		"current": {}, "state": {}, "advance": {}, "conflict": {}, "story": {}, "keep": {}, "must": {}, "local": {},
		"does": {}, "not": {}, "only": {}, "just": {}, "then": {}, "than": {},
	}

	normalized := regexp.MustCompile(`第\d+章`).ReplaceAllString(text, " ")
	english := regexp.MustCompile(`[A-Za-z]{4,}`).FindAllString(normalized, -1)
	terms := []string{}
	for _, term := range english {
		lower := strings.ToLower(strings.TrimSpace(term))
		if _, blocked := stopWords[lower]; blocked {
			continue
		}
		if len(lower) >= 2 {
			terms = append(terms, lower)
		}
	}

	segments := regexp.MustCompile(`[\p{Han}]{2,}`).FindAllString(normalized, -1)
	for _, segment := range segments {
		terms = append(terms, extractChineseFocusTerms(segment)...)
	}
	return terms
}

func extractChineseFocusTerms(segment string) []string {
	stripped := regexp.MustCompile(`^(本章|继续|重新|拉回|回到|推进|优先|围绕|鑱氱劍|坚持|淇濇寔|处理)+`).ReplaceAllString(strings.TrimSpace(segment), "")
	target := stripped
	if len([]rune(target)) < 2 {
		target = segment
	}
	runes := []rune(target)
	terms := map[string]struct{}{}
	if len(runes) <= 4 {
		terms[string(runes)] = struct{}{}
	}
	for size := 2; size <= 4; size++ {
		if len(runes) >= size {
			terms[string(runes[len(runes)-size:])] = struct{}{}
		}
	}
	result := []string{}
	for term := range terms {
		if len([]rune(term)) >= 2 {
			result = append(result, term)
		}
	}
	return result
}

func stripNegativeGuidance(text string) string {
	if text == "" {
		return ""
	}
	value := regexp.MustCompile(`(?i)\b(do not|don't|avoid|without|instead of)\b[\s\S]*$`).ReplaceAllString(text, " ")
	value = regexp.MustCompile(`(?:不要|不让|鍒珅绂佹|避免)[\s\S]*$`).ReplaceAllString(value, " ")
	return strings.TrimSpace(value)
}

func uniqueTerms(terms []string) []string {
	seen := map[string]struct{}{}
	result := []string{}
	for _, term := range terms {
		normalized := strings.ToLower(strings.TrimSpace(term))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, strings.TrimSpace(term))
	}
	return result
}

func parseVolumeSummariesMarkdown(markdown string) []VolumeSummarySelection {
	if strings.TrimSpace(markdown) == "" {
		return []VolumeSummarySelection{}
	}
	sections := regexp.MustCompile(`(?m)^##\s+`).Split(markdown, -1)
	result := []VolumeSummarySelection{}
	for _, section := range sections {
		trimmed := strings.TrimSpace(section)
		if trimmed == "" {
			continue
		}
		lines := strings.Split(trimmed, "\n")
		heading := strings.TrimSpace(lines[0])
		content := strings.TrimSpace(strings.Join(lines[1:], "\n"))
		if heading == "" || content == "" {
			continue
		}
		result = append(result, VolumeSummarySelection{Heading: heading, Content: content, Anchor: slugifyAnchor(heading)})
	}
	return result
}

func isUnresolvedHookStatus(status string) bool {
	normalized := strings.ToLower(strings.TrimSpace(status))
	return normalized == "" || strings.Contains(normalized, "open") || strings.Contains(normalized, "active") || strings.Contains(normalized, "progressing") || strings.Contains(normalized, "待定") || strings.Contains(normalized, "推进")
}

func selectRelevantSummaries(summaries []state.StoredSummary, chapterNumber int, queryTerms []string) []state.StoredSummary {
	type ranked struct {
		Summary state.StoredSummary
		Score   int
		Matched bool
	}
	rankedRows := []ranked{}
	for _, summary := range summaries {
		if summary.Chapter >= chapterNumber {
			continue
		}
		target := strings.Join([]string{summary.Title, summary.Characters, summary.Events, summary.StateChanges, summary.HookActivity, summary.ChapterType}, " ")
		rankedRows = append(rankedRows, ranked{
			Summary: summary,
			Score:   scoreSummary(summary, chapterNumber, queryTerms),
			Matched: matchesAny(target, queryTerms),
		})
	}

	filtered := []ranked{}
	for _, entry := range rankedRows {
		if entry.Matched || entry.Summary.Chapter >= chapterNumber-3 {
			filtered = append(filtered, entry)
		}
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Score != filtered[j].Score {
			return filtered[i].Score > filtered[j].Score
		}
		return filtered[i].Summary.Chapter > filtered[j].Summary.Chapter
	})
	if len(filtered) > 4 {
		filtered = filtered[:4]
	}

	result := []state.StoredSummary{}
	for _, entry := range filtered {
		result = append(result, entry.Summary)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Chapter < result[j].Chapter })
	return result
}

func selectRelevantHooks(hooks []state.StoredHook, queryTerms []string, chapterNumber int) []state.StoredHook {
	type ranked struct {
		Hook    state.StoredHook
		Score   int
		Matched bool
	}
	rankedRows := []ranked{}
	for _, hook := range hooks {
		text := strings.Join([]string{hook.HookID, hook.Type, hook.ExpectedPayoff, hook.PayoffTiming, hook.Notes}, " ")
		matched := matchesAny(text, queryTerms)
		if !(matched || isUnresolvedHookStatus(hook.Status)) {
			continue
		}
		rankedRows = append(rankedRows, ranked{Hook: hook, Score: scoreHook(hook, queryTerms), Matched: matched})
	}

	primary := []ranked{}
	for _, entry := range rankedRows {
		if entry.Matched || IsHookWithinChapterWindow(entry.Hook, chapterNumber, 5) {
			primary = append(primary, entry)
		}
	}
	sort.Slice(primary, func(i, j int) bool {
		if primary[i].Score != primary[j].Score {
			return primary[i].Score > primary[j].Score
		}
		return primary[i].Hook.LastAdvancedChapter > primary[j].Hook.LastAdvancedChapter
	})
	if len(primary) > 6 {
		primary = primary[:6]
	}

	selectedIDs := map[string]struct{}{}
	for _, entry := range primary {
		selectedIDs[entry.Hook.HookID] = struct{}{}
	}

	stale := []ranked{}
	for _, entry := range rankedRows {
		if _, selected := selectedIDs[entry.Hook.HookID]; selected {
			continue
		}
		if IsFuturePlannedHook(entry.Hook, chapterNumber) {
			continue
		}
		if !isUnresolvedHookStatus(entry.Hook.Status) {
			continue
		}
		stale = append(stale, entry)
	}
	sort.Slice(stale, func(i, j int) bool {
		if stale[i].Hook.LastAdvancedChapter != stale[j].Hook.LastAdvancedChapter {
			return stale[i].Hook.LastAdvancedChapter < stale[j].Hook.LastAdvancedChapter
		}
		return stale[i].Score > stale[j].Score
	})
	if len(stale) > 2 {
		stale = stale[:2]
	}

	result := []state.StoredHook{}
	for _, entry := range primary {
		result = append(result, entry.Hook)
	}
	for _, entry := range stale {
		result = append(result, entry.Hook)
	}
	return result
}

func selectRelevantFacts(facts []state.Fact, queryTerms []string) []state.Fact {
	prioritized := []*regexp.Regexp{
		regexp.MustCompile(`(?i)^(当前冲突|current\s+conflict)$`),
		regexp.MustCompile(`(?i)^(当前目标|current\s+goal)$`),
		regexp.MustCompile(`(?i)^(主角状态|protagonist\s+state)$`),
		regexp.MustCompile(`(?i)^(当前限制|current\s+constraint)$`),
		regexp.MustCompile(`(?i)^(当前位置|current\s+location)$`),
		regexp.MustCompile(`(?i)^(当前敌我|当前关系|current\s+alliances|current\s+relationships)$`),
	}

	type ranked struct {
		Fact    state.Fact
		Score   int
		Matched bool
	}
	rankedFacts := []ranked{}
	for _, fact := range facts {
		text := strings.Join([]string{fact.Subject, fact.Predicate, fact.Object}, " ")
		priority := -1
		for i, pattern := range prioritized {
			if pattern.MatchString(fact.Predicate) {
				priority = i
				break
			}
		}
		base := 5
		if priority >= 0 {
			base = 20 - priority*2
		}
		termScore := 0
		for _, term := range queryTerms {
			if includesTerm(text, term) {
				termScore += maxInt(8, len([]rune(term))*2)
			}
		}
		rankedFacts = append(rankedFacts, ranked{Fact: fact, Score: base + termScore, Matched: matchesAny(text, queryTerms)})
	}

	filtered := []ranked{}
	for _, entry := range rankedFacts {
		if entry.Matched || entry.Score >= 14 {
			filtered = append(filtered, entry)
		}
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Score > filtered[j].Score })
	if len(filtered) > 4 {
		filtered = filtered[:4]
	}

	result := []state.Fact{}
	for _, entry := range filtered {
		result = append(result, entry.Fact)
	}
	return result
}

func selectRelevantVolumeSummaries(summaries []VolumeSummarySelection, queryTerms []string) []VolumeSummarySelection {
	if len(summaries) == 0 {
		return []VolumeSummarySelection{}
	}
	type ranked struct {
		Index   int
		Summary VolumeSummarySelection
		Score   int
		Matched bool
	}
	rankedRows := []ranked{}
	for idx, summary := range summaries {
		text := summary.Heading + " " + summary.Content
		termScore := 0
		for _, term := range queryTerms {
			if includesTerm(text, term) {
				termScore += maxInt(8, len([]rune(term))*2)
			}
		}
		rankedRows = append(rankedRows, ranked{Index: idx, Summary: summary, Score: termScore + idx, Matched: matchesAny(text, queryTerms)})
	}

	filtered := []ranked{}
	for idx, entry := range rankedRows {
		if entry.Matched || idx == len(rankedRows)-1 {
			filtered = append(filtered, entry)
		}
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Score > filtered[j].Score })
	if len(filtered) > 2 {
		filtered = filtered[:2]
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Index < filtered[j].Index })

	result := []VolumeSummarySelection{}
	for _, entry := range filtered {
		result = append(result, entry.Summary)
	}
	return result
}

func scoreSummary(summary state.StoredSummary, chapterNumber int, queryTerms []string) int {
	text := strings.Join([]string{summary.Title, summary.Characters, summary.Events, summary.StateChanges, summary.HookActivity, summary.ChapterType}, " ")
	age := maxInt(0, chapterNumber-summary.Chapter)
	recency := maxInt(0, 12-age)
	termScore := 0
	for _, term := range queryTerms {
		if includesTerm(text, term) {
			termScore += maxInt(8, len([]rune(term))*2)
		}
	}
	return recency + termScore
}

func scoreHook(hook state.StoredHook, queryTerms []string) int {
	text := strings.Join([]string{hook.HookID, hook.Type, hook.ExpectedPayoff, hook.PayoffTiming, hook.Notes}, " ")
	freshness := maxInt(0, hook.LastAdvancedChapter)
	termScore := 0
	for _, term := range queryTerms {
		if includesTerm(text, term) {
			termScore += maxInt(8, len([]rune(term))*2)
		}
	}
	return freshness + termScore
}

func matchesAny(text string, queryTerms []string) bool {
	for _, term := range queryTerms {
		if includesTerm(text, term) {
			return true
		}
	}
	return false
}

func includesTerm(text string, term string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(term))
}

func slugifyAnchor(value string) string {
	normalized := strings.TrimSpace(strings.ToLower(value))
	normalized = regexp.MustCompile(`[^a-z0-9\p{Han}]+`).ReplaceAllString(normalized, "-")
	normalized = strings.Trim(normalized, "-")
	if normalized == "" {
		return "volume-summary"
	}
	return normalized
}
