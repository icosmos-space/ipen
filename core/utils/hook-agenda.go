package utils

import (
	"regexp"
	"sort"
	"strings"

	"github.com/icosmos-space/ipen/core/models"
	"github.com/icosmos-space/ipen/core/state"
)

const DEFAULT_HOOK_LOOKAHEAD_CHAPTERS = 3

// BuildPlannerHookAgenda 构建a lightweight stalest-first hook agenda。
func BuildPlannerHookAgenda(
	hooks []state.StoredHook,
	chapterNumber int,
	targetChapters int,
	language string,
	maxMustAdvance int,
	maxEligibleResolve int,
	maxStaleDebt int,
) models.HookAgenda {
	_ = targetChapters
	_ = language
	if maxMustAdvance <= 0 {
		maxMustAdvance = 2
	}
	if maxEligibleResolve <= 0 {
		maxEligibleResolve = 1
	}
	if maxStaleDebt <= 0 {
		maxStaleDebt = 2
	}

	agendaHooks := make([]state.StoredHook, 0, len(hooks))
	for _, hook := range hooks {
		if IsFuturePlannedHook(hook, chapterNumber, 0) {
			continue
		}
		status := normalizeStoredHookStatus(hook.Status)
		if status == models.HookStatusResolvedRT || status == models.HookStatusDeferred {
			continue
		}
		agendaHooks = append(agendaHooks, hook)
	}

	mustAdvance := append([]state.StoredHook(nil), agendaHooks...)
	sort.Slice(mustAdvance, func(i, j int) bool {
		if mustAdvance[i].LastAdvancedChapter != mustAdvance[j].LastAdvancedChapter {
			return mustAdvance[i].LastAdvancedChapter < mustAdvance[j].LastAdvancedChapter
		}
		if mustAdvance[i].StartChapter != mustAdvance[j].StartChapter {
			return mustAdvance[i].StartChapter < mustAdvance[j].StartChapter
		}
		return mustAdvance[i].HookID < mustAdvance[j].HookID
	})
	if len(mustAdvance) > maxMustAdvance {
		mustAdvance = mustAdvance[:maxMustAdvance]
	}

	staleThreshold := chapterNumber - 10
	staleDebt := []state.StoredHook{}
	for _, hook := range agendaHooks {
		lastTouch := maxInt(hook.StartChapter, hook.LastAdvancedChapter)
		if lastTouch > 0 && lastTouch <= staleThreshold {
			staleDebt = append(staleDebt, hook)
		}
	}
	sort.Slice(staleDebt, func(i, j int) bool {
		if staleDebt[i].LastAdvancedChapter != staleDebt[j].LastAdvancedChapter {
			return staleDebt[i].LastAdvancedChapter < staleDebt[j].LastAdvancedChapter
		}
		if staleDebt[i].StartChapter != staleDebt[j].StartChapter {
			return staleDebt[i].StartChapter < staleDebt[j].StartChapter
		}
		return staleDebt[i].HookID < staleDebt[j].HookID
	})
	if len(staleDebt) > maxStaleDebt {
		staleDebt = staleDebt[:maxStaleDebt]
	}

	eligibleResolve := []state.StoredHook{}
	for _, hook := range agendaHooks {
		if hook.StartChapter <= chapterNumber-3 && hook.LastAdvancedChapter >= chapterNumber-2 {
			eligibleResolve = append(eligibleResolve, hook)
		}
	}
	sort.Slice(eligibleResolve, func(i, j int) bool {
		if eligibleResolve[i].StartChapter != eligibleResolve[j].StartChapter {
			return eligibleResolve[i].StartChapter < eligibleResolve[j].StartChapter
		}
		if eligibleResolve[i].LastAdvancedChapter != eligibleResolve[j].LastAdvancedChapter {
			return eligibleResolve[i].LastAdvancedChapter > eligibleResolve[j].LastAdvancedChapter
		}
		return eligibleResolve[i].HookID < eligibleResolve[j].HookID
	})
	if len(eligibleResolve) > maxEligibleResolve {
		eligibleResolve = eligibleResolve[:maxEligibleResolve]
	}

	avoid := []string{}
	seenFamily := map[string]struct{}{}
	for _, candidate := range append(append(append([]state.StoredHook{}, staleDebt...), mustAdvance...), eligibleResolve...) {
		family := strings.TrimSpace(candidate.Type)
		if family == "" {
			continue
		}
		if _, ok := seenFamily[family]; ok {
			continue
		}
		seenFamily[family] = struct{}{}
		avoid = append(avoid, family)
		if len(avoid) >= 3 {
			break
		}
	}

	return models.HookAgenda{
		PressureMap:          []models.HookPressure{},
		MustAdvance:          mapHooksToIDs(mustAdvance),
		EligibleResolve:      mapHooksToIDs(eligibleResolve),
		StaleDebt:            mapHooksToIDs(staleDebt),
		AvoidNewHookFamilies: avoid,
	}
}

func FilterActiveHooks(hooks []state.StoredHook) []state.StoredHook {
	result := make([]state.StoredHook, 0, len(hooks))
	for _, hook := range hooks {
		if normalizeStoredHookStatus(hook.Status) != models.HookStatusResolvedRT {
			result = append(result, hook)
		}
	}
	return result
}

func IsFuturePlannedHook(hook state.StoredHook, chapterNumber int, lookahead ...int) bool {
	window := DEFAULT_HOOK_LOOKAHEAD_CHAPTERS
	if len(lookahead) > 0 {
		window = lookahead[0]
	}
	return hook.LastAdvancedChapter <= 0 && hook.StartChapter > chapterNumber+window
}

func IsHookWithinChapterWindow(hook state.StoredHook, chapterNumber int, recentWindow ...int) bool {
	recent := 5
	if len(recentWindow) > 0 {
		recent = recentWindow[0]
	}
	lookahead := DEFAULT_HOOK_LOOKAHEAD_CHAPTERS
	recentCutoff := maxInt(0, chapterNumber-recent)

	if hook.LastAdvancedChapter > 0 && hook.LastAdvancedChapter >= recentCutoff {
		return true
	}
	if hook.LastAdvancedChapter > 0 {
		return false
	}
	if hook.StartChapter <= 0 {
		return true
	}
	if hook.StartChapter >= recentCutoff && hook.StartChapter <= chapterNumber {
		return true
	}
	return hook.StartChapter > chapterNumber && hook.StartChapter <= chapterNumber+lookahead
}

func normalizeStoredHookStatus(status string) models.HookStatus {
	normalized := strings.TrimSpace(status)
	if matched(normalized, `(?i)^(resolved|closed|done|已回收|已解决)$`) {
		return models.HookStatusResolvedRT
	}
	if matched(normalized, `(?i)^(deferred|paused|hold|延后|延期|搁置|暂缓)$`) {
		return models.HookStatusDeferred
	}
	if matched(normalized, `(?i)^(progressing|advanced|重大推进|持续推进)$`) {
		return models.HookStatusProgressingRT
	}
	return models.HookStatusOpenRT
}

func mapHooksToIDs(hooks []state.StoredHook) []string {
	ids := make([]string, 0, len(hooks))
	for _, hook := range hooks {
		ids = append(ids, hook.HookID)
	}
	return ids
}

func matched(value string, pattern string) bool {
	return regexpMust(pattern).MatchString(value)
}

var regexpCache = map[string]*regexp.Regexp{}

func regexpMust(pattern string) *regexp.Regexp {
	if re, ok := regexpCache[pattern]; ok {
		return re
	}
	re := regexp.MustCompile(pattern)
	regexpCache[pattern] = re
	return re
}
