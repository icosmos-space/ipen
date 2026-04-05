package utils

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/icosmos-space/ipen/core/models"
)

// HookArbiterDecision records one arbitration result.
type HookArbiterDecision struct {
	Action    string
	Reason    string
	HookID    string
	Candidate models.NewHookCandidate
}

type pendingHookCandidate struct {
	models.NewHookCandidate
	PreferredHookID string
}

// ArbitrateRuntimeStateDeltaHooks 去重hook upserts/new candidates and returns a canonical delta。
func ArbitrateRuntimeStateDeltaHooks(hooks []models.HookRecord, delta models.RuntimeStateDelta) (models.RuntimeStateDelta, []HookArbiterDecision) {
	workingHooks := make([]models.HookRecord, len(hooks))
	copy(workingHooks, hooks)
	knownHookIDs := map[string]struct{}{}
	for _, hook := range workingHooks {
		knownHookIDs[hook.HookID] = struct{}{}
	}

	upsertsByID := map[string]models.HookRecord{}
	mentions := map[string]struct{}{}
	for _, mention := range delta.HookOps.Mention {
		normalized := strings.TrimSpace(mention)
		if normalized != "" {
			mentions[normalized] = struct{}{}
		}
	}
	resolves := uniqueTrimmedStrings(delta.HookOps.Resolve)
	defers := uniqueTrimmedStrings(delta.HookOps.Defer)
	fallbackCandidates := []pendingHookCandidate{}
	decisions := []HookArbiterDecision{}

	for _, hook := range delta.HookOps.Upsert {
		if _, exists := knownHookIDs[hook.HookID]; exists {
			upsertsByID[hook.HookID] = hook
			replaceWorkingHook(&workingHooks, hook)
			continue
		}

		fallbackCandidates = append(fallbackCandidates, pendingHookCandidate{
			NewHookCandidate: models.NewHookCandidate{
				Type:           hook.Type,
				ExpectedPayoff: hook.ExpectedPayoff,
				PayoffTiming:   hook.PayoffTiming,
				Notes:          hook.Notes,
			},
			PreferredHookID: hook.HookID,
		})
	}

	pending := append([]pendingHookCandidate{}, fallbackCandidates...)
	for _, candidate := range delta.NewHookCandidates {
		pending = append(pending, pendingHookCandidate{NewHookCandidate: candidate})
	}

	for _, candidate := range pending {
		activeHooks := []models.HookRecord{}
		for _, hook := range workingHooks {
			if hook.Status != models.HookStatusResolvedRT {
				activeHooks = append(activeHooks, hook)
			}
		}

		admission := EvaluateHookAdmissionForUtils(HookAdmissionCandidate{
			Type:           candidate.Type,
			ExpectedPayoff: candidate.ExpectedPayoff,
			PayoffTiming:   timingPtrToString(candidate.PayoffTiming),
			Notes:          candidate.Notes,
		}, activeHooks)

		if !admission.Admit {
			if admission.Reason == "duplicate_family" && admission.MatchedHookID != "" {
				matched := findHookByID(workingHooks, admission.MatchedHookID)
				if matched == nil {
					decisions = append(decisions, HookArbiterDecision{Action: "rejected", Reason: "duplicate_family_without_match", Candidate: candidate.NewHookCandidate})
					continue
				}

				if isPureRestatement(candidate.NewHookCandidate, *matched) {
					if _, exists := upsertsByID[matched.HookID]; !exists && !containsString(resolves, matched.HookID) && !containsString(defers, matched.HookID) {
						mentions[matched.HookID] = struct{}{}
					}
					decisions = append(decisions, HookArbiterDecision{Action: "mentioned", Reason: "restated_existing_family", HookID: matched.HookID, Candidate: candidate.NewHookCandidate})
					continue
				}

				base := *matched
				if existing, ok := upsertsByID[matched.HookID]; ok {
					base = existing
				}
				mapped := mergeCandidate(base, candidate.NewHookCandidate, delta.Chapter)
				upsertsByID[mapped.HookID] = mapped
				delete(mentions, mapped.HookID)
				replaceWorkingHook(&workingHooks, mapped)
				decisions = append(decisions, HookArbiterDecision{Action: "mapped", Reason: "duplicate_family_with_novelty", HookID: mapped.HookID, Candidate: candidate.NewHookCandidate})
				continue
			}

			decisions = append(decisions, HookArbiterDecision{Action: "rejected", Reason: admission.Reason, Candidate: candidate.NewHookCandidate})
			continue
		}

		existingIDs := map[string]struct{}{}
		for _, hook := range workingHooks {
			existingIDs[hook.HookID] = struct{}{}
		}
		for hookID := range upsertsByID {
			existingIDs[hookID] = struct{}{}
		}

		created := createCanonicalHook(candidate, delta.Chapter, existingIDs)
		upsertsByID[created.HookID] = created
		workingHooks = append(workingHooks, created)
		decisions = append(decisions, HookArbiterDecision{Action: "created", Reason: "admit", HookID: created.HookID, Candidate: candidate.NewHookCandidate})
	}

	upsertList := make([]models.HookRecord, 0, len(upsertsByID))
	for _, hook := range upsertsByID {
		upsertList = append(upsertList, hook)
	}
	sort.Slice(upsertList, func(i, j int) bool {
		if upsertList[i].StartChapter != upsertList[j].StartChapter {
			return upsertList[i].StartChapter < upsertList[j].StartChapter
		}
		if upsertList[i].LastAdvancedChapter != upsertList[j].LastAdvancedChapter {
			return upsertList[i].LastAdvancedChapter < upsertList[j].LastAdvancedChapter
		}
		return upsertList[i].HookID < upsertList[j].HookID
	})

	mentionList := []string{}
	for hookID := range mentions {
		if _, ok := upsertsByID[hookID]; ok {
			continue
		}
		if containsString(resolves, hookID) || containsString(defers, hookID) {
			continue
		}
		mentionList = append(mentionList, hookID)
	}
	sort.Strings(mentionList)

	resolved := delta
	resolved.HookOps.Upsert = upsertList
	resolved.HookOps.Mention = mentionList
	resolved.HookOps.Resolve = resolves
	resolved.HookOps.Defer = defers
	resolved.NewHookCandidates = []models.NewHookCandidate{}

	return resolved, decisions
}

func mergeCandidate(existing models.HookRecord, candidate models.NewHookCandidate, chapter int) models.HookRecord {
	expectedPayoff := richerText(existing.ExpectedPayoff, candidate.ExpectedPayoff)
	notes := richerText(existing.Notes, candidate.Notes)
	payoffTiming := ResolveHookPayoffTiming(timingOrString(candidate.PayoffTiming, existing.PayoffTiming), expectedPayoff, notes)

	status := existing.Status
	if status != models.HookStatusResolvedRT {
		status = models.HookStatusProgressingRT
	}

	startChapter := existing.StartChapter
	if startChapter <= 0 {
		startChapter = chapter
	}

	return models.HookRecord{
		HookID:              existing.HookID,
		StartChapter:        startChapter,
		Type:                richerText(existing.Type, candidate.Type),
		Status:              status,
		LastAdvancedChapter: maxInt(existing.LastAdvancedChapter, chapter),
		ExpectedPayoff:      expectedPayoff,
		PayoffTiming:        &payoffTiming,
		Notes:               notes,
	}
}

func createCanonicalHook(candidate pendingHookCandidate, chapter int, existingIDs map[string]struct{}) models.HookRecord {
	hookID := strings.TrimSpace(candidate.PreferredHookID)
	if hookID == "" {
		hookID = buildCanonicalHookID(candidate.NewHookCandidate, existingIDs)
	} else if _, exists := existingIDs[hookID]; exists {
		hookID = buildCanonicalHookID(candidate.NewHookCandidate, existingIDs)
	}
	payoffTiming := ResolveHookPayoffTiming(timingPtrToString(candidate.PayoffTiming), candidate.ExpectedPayoff, candidate.Notes)

	return models.HookRecord{
		HookID:              hookID,
		StartChapter:        chapter,
		Type:                strings.TrimSpace(candidate.Type),
		Status:              models.HookStatusOpenRT,
		LastAdvancedChapter: chapter,
		ExpectedPayoff:      strings.TrimSpace(candidate.ExpectedPayoff),
		PayoffTiming:        &payoffTiming,
		Notes:               strings.TrimSpace(candidate.Notes),
	}
}

func buildCanonicalHookID(candidate models.NewHookCandidate, existingIDs map[string]struct{}) string {
	base := slugifyHookStem(strings.Join([]string{candidate.Type, candidate.ExpectedPayoff, candidate.Notes}, " "))
	next := base
	suffix := 2
	for {
		if _, exists := existingIDs[next]; !exists {
			return next
		}
		next = base + "-" + strconv.Itoa(suffix)
		suffix++
	}
}

func slugifyHookStem(value string) string {
	normalized := normalizeHookText(value)
	english := regexp.MustCompile(`[a-z0-9]{3,}`).FindAllString(normalized, -1)
	terms := []string{}
	for _, term := range english {
		if _, blocked := hookArbiterStopWords[term]; blocked {
			continue
		}
		terms = append(terms, term)
		if len(terms) >= 5 {
			break
		}
	}
	chinese := regexp.MustCompile(`[\p{Han}]{2,6}`).FindAllString(normalized, -1)
	for i := 0; i < len(chinese) && i < 3; i++ {
		terms = append(terms, chinese[i])
	}
	stem := strings.Trim(strings.Join(terms, "-"), "-")
	if stem == "" {
		stem = "hook"
	}
	if len(stem) > 64 {
		stem = strings.TrimRight(stem[:64], "-")
	}
	return stem
}

func isPureRestatement(candidate models.NewHookCandidate, existing models.HookRecord) bool {
	candidateText := normalizeHookText(strings.Join([]string{candidate.Type, candidate.ExpectedPayoff, candidate.Notes}, " "))
	existingText := normalizeHookText(strings.Join([]string{existing.Type, existing.ExpectedPayoff, existing.Notes}, " "))
	if candidateText == "" || candidateText == existingText {
		return true
	}
	candidateTerms := extractHookTerms(candidateText)
	existingTerms := extractHookTerms(existingText)
	novelTerms := 0
	for term := range candidateTerms {
		if _, ok := existingTerms[term]; !ok {
			novelTerms++
		}
	}
	candidateBigrams := extractHookChineseBigrams(candidateText)
	existingBigrams := extractHookChineseBigrams(existingText)
	novelBigrams := 0
	for bg := range candidateBigrams {
		if _, ok := existingBigrams[bg]; !ok {
			novelBigrams++
		}
	}
	return novelTerms == 0 && novelBigrams < 2
}

func replaceWorkingHook(working *[]models.HookRecord, hook models.HookRecord) {
	for i := range *working {
		if (*working)[i].HookID == hook.HookID {
			(*working)[i] = hook
			return
		}
	}
	*working = append(*working, hook)
}

func findHookByID(hooks []models.HookRecord, hookID string) *models.HookRecord {
	for i := range hooks {
		if hooks[i].HookID == hookID {
			return &hooks[i]
		}
	}
	return nil
}

func timingOrString(primary *models.HookPayoffTiming, fallback *models.HookPayoffTiming) string {
	if primary != nil {
		return string(*primary)
	}
	if fallback != nil {
		return string(*fallback)
	}
	return ""
}

var hookArbiterStopWords = map[string]struct{}{
	"that": {}, "this": {}, "with": {}, "from": {}, "into": {}, "still": {}, "just": {}, "have": {}, "will": {}, "reveal": {}, "about": {}, "already": {}, "question": {}, "chapter": {},
}

func uniqueTrimmedStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := []string{}
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func richerText(primary string, fallback string) string {
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

func timingPtrToString(value *models.HookPayoffTiming) string {
	if value == nil {
		return ""
	}
	return string(*value)
}
