package utils

import (
	"fmt"
	"strings"

	"github.com/icosmos-space/ipen/core/models"
)

// HookHealthIssue mirrors audit warning shape used by hook-health checks.
type HookHealthIssue struct {
	Severity    string
	Category    string
	Description string
	Suggestion  string
}

// AnalyzeHookHealth 检查hook debt pressure and burst/overgrowth risks。
func AnalyzeHookHealth(
	language string,
	chapterNumber int,
	targetChapters int,
	hooks []models.HookRecord,
	delta *models.RuntimeStateDelta,
	existingHookIDs []string,
	maxActiveHooks int,
	staleAfterChapters int,
	noAdvanceWindow int,
	newHookBurstThreshold int,
) []HookHealthIssue {
	if maxActiveHooks <= 0 {
		maxActiveHooks = HOOK_HEALTH_DEFAULTS.MaxActiveHooks
	}
	if staleAfterChapters <= 0 {
		staleAfterChapters = HOOK_HEALTH_DEFAULTS.StaleAfterChapters
	}
	if noAdvanceWindow <= 0 {
		noAdvanceWindow = HOOK_HEALTH_DEFAULTS.NoAdvanceWindow
	}
	if newHookBurstThreshold <= 0 {
		newHookBurstThreshold = HOOK_HEALTH_DEFAULTS.NewHookBurstThreshold
	}

	issues := []HookHealthIssue{}
	active := []models.HookRecord{}
	for _, hook := range hooks {
		if hook.Status != models.HookStatusResolvedRT {
			active = append(active, hook)
		}
	}

	if len(active) > maxActiveHooks {
		issues = append(issues, warningIssue(language,
			fmt.Sprintf("There are %d active hooks, above recommended cap %d.", len(active), maxActiveHooks),
			"Prefer resolving or deferring old debt before opening new families.",
		))
	}

	stale := CollectStaleHookDebt(active, chapterNumber, targetChapters, staleAfterChapters)
	staleIDs := map[string]struct{}{}
	for _, hook := range stale {
		staleIDs[hook.HookID] = struct{}{}
	}

	pressured := []models.HookRecord{}
	for _, hook := range active {
		lifecycle := DescribeHookLifecycle(valueOrEmpty(hook.PayoffTiming), hook.ExpectedPayoff, hook.Notes, hook.StartChapter, hook.LastAdvancedChapter, string(hook.Status), chapterNumber, targetChapters)
		if _, isStale := staleIDs[hook.HookID]; isStale || lifecycle.ReadyToResolve || lifecycle.Overdue {
			pressured = append(pressured, hook)
		}
	}

	unresolvedPressure := []string{}
	for _, hook := range pressured {
		if delta == nil {
			unresolvedPressure = append(unresolvedPressure, hook.HookID)
			continue
		}
		disposition := ClassifyHookDisposition(hook.HookID, *delta)
		if disposition == HookDispositionNone || disposition == HookDispositionMention {
			unresolvedPressure = append(unresolvedPressure, hook.HookID)
		}
	}
	if len(unresolvedPressure) > 0 {
		preview := unresolvedPressure
		if len(preview) > 3 {
			preview = preview[:3]
		}
		issues = append(issues, warningIssue(language,
			"Hooks under payoff pressure were left untouched: "+strings.Join(preview, ", "),
			"Advance, resolve, or explicitly defer at least one pressured hook.",
		))
	} else {
		latestAdvance := 0
		for _, hook := range active {
			if hook.LastAdvancedChapter > latestAdvance {
				latestAdvance = hook.LastAdvancedChapter
			}
		}
		if len(active) > 0 && chapterNumber-latestAdvance >= noAdvanceWindow {
			issues = append(issues, warningIssue(language,
				fmt.Sprintf("No real hook advancement for %d chapters.", chapterNumber-latestAdvance),
				"Schedule one old hook for real movement.",
			))
		}
	}

	if delta != nil {
		existing := map[string]struct{}{}
		for _, id := range existingHookIDs {
			existing[id] = struct{}{}
		}
		resulting := map[string]struct{}{}
		for _, hook := range hooks {
			resulting[hook.HookID] = struct{}{}
		}
		newHookCount := 0
		for _, hook := range delta.HookOps.Upsert {
			if _, existed := existing[hook.HookID]; existed {
				continue
			}
			if _, present := resulting[hook.HookID]; present {
				newHookCount++
			}
		}
		if newHookCount >= newHookBurstThreshold && len(delta.HookOps.Resolve) == 0 {
			issues = append(issues, warningIssue(language,
				fmt.Sprintf("Opened %d new hooks without resolving older debt.", newHookCount),
				"Pair new openings with old payoffs to prevent hook inflation.",
			))
		}
	}

	return issues
}

func warningIssue(language string, description string, suggestion string) HookHealthIssue {
	category := "伏笔债务"
	if strings.EqualFold(language, "en") {
		category = "Hook Debt"
	}
	return HookHealthIssue{Severity: "warning", Category: category, Description: description, Suggestion: suggestion}
}
