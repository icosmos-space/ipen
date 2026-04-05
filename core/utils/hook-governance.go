package utils

import (
	"regexp"
	"sort"
	"strings"

	"github.com/icosmos-space/ipen/core/models"
)

// HookDisposition 描述how one hook was treated in a delta。
type HookDisposition string

const (
	HookDispositionNone    HookDisposition = "none"
	HookDispositionMention HookDisposition = "mention"
	HookDispositionAdvance HookDisposition = "advance"
	HookDispositionResolve HookDisposition = "resolve"
	HookDispositionDefer   HookDisposition = "defer"
)

// HookAdmissionCandidate 是a candidate hook for admission checks。
type HookAdmissionCandidate struct {
	Type           string
	ExpectedPayoff string
	PayoffTiming   string
	Notes          string
}

// HookAdmissionDecision 是the admission decision。
type HookAdmissionDecision struct {
	Admit         bool
	Reason        string
	MatchedHookID string
}

// CollectStaleHookDebt 返回stale/overdue unresolved hooks, sorted stalest-first。
func CollectStaleHookDebt(hooks []models.HookRecord, chapterNumber int, targetChapters int, staleAfterChapters ...int) []models.HookRecord {
	result := make([]models.HookRecord, 0, len(hooks))
	manualThreshold := -1
	if len(staleAfterChapters) > 0 {
		manualThreshold = staleAfterChapters[0]
	}

	for _, hook := range hooks {
		if hook.Status == models.HookStatusResolvedRT || hook.Status == models.HookStatusDeferred {
			continue
		}
		if hook.StartChapter > chapterNumber {
			continue
		}

		lifecycle := DescribeHookLifecycle(
			valueOrEmpty(hook.PayoffTiming),
			hook.ExpectedPayoff,
			hook.Notes,
			hook.StartChapter,
			hook.LastAdvancedChapter,
			string(hook.Status),
			chapterNumber,
			targetChapters,
		)

		if manualThreshold >= 0 {
			if hook.LastAdvancedChapter <= chapterNumber-manualThreshold {
				result = append(result, hook)
			}
			continue
		}

		if lifecycle.Stale || lifecycle.Overdue {
			result = append(result, hook)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].LastAdvancedChapter != result[j].LastAdvancedChapter {
			return result[i].LastAdvancedChapter < result[j].LastAdvancedChapter
		}
		if result[i].StartChapter != result[j].StartChapter {
			return result[i].StartChapter < result[j].StartChapter
		}
		return result[i].HookID < result[j].HookID
	})

	return result
}

// EvaluateHookAdmission decides whether a candidate should be admitted.
func EvaluateHookAdmissionForUtils(candidate HookAdmissionCandidate, activeHooks []models.HookRecord) HookAdmissionDecision {
	candidateType := normalizeHookText(candidate.Type)
	if candidateType == "" {
		return HookAdmissionDecision{Admit: false, Reason: "missing_type"}
	}

	payoffSignal := strings.TrimSpace(candidate.ExpectedPayoff + " " + candidate.Notes)
	if payoffSignal == "" {
		return HookAdmissionDecision{Admit: false, Reason: "missing_payoff_signal"}
	}

	candidateNormalized := normalizeHookText(strings.Join([]string{
		candidate.Type,
		candidate.ExpectedPayoff,
		candidate.PayoffTiming,
		candidate.Notes,
	}, " "))
	candidateTerms := extractHookTerms(candidateNormalized)
	candidateBigrams := extractHookChineseBigrams(candidateNormalized)

	for _, hook := range activeHooks {
		activeNormalized := normalizeHookText(strings.Join([]string{
			hook.Type,
			hook.ExpectedPayoff,
			valueOrEmpty(hook.PayoffTiming),
			hook.Notes,
		}, " "))

		if candidateNormalized == activeNormalized {
			return HookAdmissionDecision{Admit: false, Reason: "duplicate_family", MatchedHookID: hook.HookID}
		}

		if candidateType != normalizeHookText(hook.Type) {
			continue
		}

		activeTerms := extractHookTerms(activeNormalized)
		overlap := 0
		for term := range candidateTerms {
			if _, ok := activeTerms[term]; ok {
				overlap++
			}
		}

		activeBigrams := extractHookChineseBigrams(activeNormalized)
		chOverlap := 0
		for gram := range candidateBigrams {
			if _, ok := activeBigrams[gram]; ok {
				chOverlap++
			}
		}

		if overlap >= 2 || chOverlap >= 3 {
			return HookAdmissionDecision{Admit: false, Reason: "duplicate_family", MatchedHookID: hook.HookID}
		}
	}

	return HookAdmissionDecision{Admit: true, Reason: "admit"}
}

// ClassifyHookDisposition 分类per-hook operation in one delta。
func ClassifyHookDisposition(hookID string, delta models.RuntimeStateDelta) HookDisposition {
	for _, id := range delta.HookOps.Defer {
		if id == hookID {
			return HookDispositionDefer
		}
	}
	for _, id := range delta.HookOps.Resolve {
		if id == hookID {
			return HookDispositionResolve
		}
	}
	for _, hook := range delta.HookOps.Upsert {
		if hook.HookID == hookID && hook.LastAdvancedChapter == delta.Chapter {
			return HookDispositionAdvance
		}
	}
	for _, id := range delta.HookOps.Mention {
		if id == hookID {
			return HookDispositionMention
		}
	}
	return HookDispositionNone
}

func normalizeHookText(value string) string {
	normalized := strings.TrimSpace(strings.ToLower(value))
	normalized = regexp.MustCompile(`[^a-z0-9\p{Han}]+`).ReplaceAllString(normalized, " ")
	normalized = strings.Join(strings.Fields(normalized), " ")
	return normalized
}

func extractHookTerms(value string) map[string]struct{} {
	terms := map[string]struct{}{}
	for _, term := range strings.Fields(value) {
		if len(term) >= 4 {
			if _, blocked := hookStopWords[term]; !blocked {
				terms[term] = struct{}{}
			}
		}
	}
	for _, ch := range regexp.MustCompile(`[\p{Han}]{2,6}`).FindAllString(value, -1) {
		terms[ch] = struct{}{}
	}
	return terms
}

func extractHookChineseBigrams(value string) map[string]struct{} {
	result := map[string]struct{}{}
	segments := regexp.MustCompile(`[\p{Han}]+`).FindAllString(value, -1)
	for _, segment := range segments {
		runes := []rune(segment)
		if len(runes) < 2 {
			continue
		}
		for i := 0; i <= len(runes)-2; i++ {
			result[string(runes[i:i+2])] = struct{}{}
		}
	}
	return result
}

var hookStopWords = map[string]struct{}{
	"that": {}, "this": {}, "with": {}, "from": {}, "into": {}, "still": {}, "just": {}, "have": {}, "will": {}, "reveal": {},
}

func valueOrEmpty(value *models.HookPayoffTiming) string {
	if value == nil {
		return ""
	}
	return string(*value)
}
