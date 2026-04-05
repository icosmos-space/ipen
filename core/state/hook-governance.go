package state

import "github.com/icosmos-space/ipen/core/models"

// HookAdmissionCandidate 表示a hook admission candidate。
type HookAdmissionCandidate struct {
	Type           string `json:"type"`
	ExpectedPayoff string `json:"expectedPayoff"`
	Notes          string `json:"notes"`
}

// HookAdmissionDecision 表示a hook admission decision。
type HookAdmissionDecision struct {
	Admit         bool   `json:"admit"`
	Reason        string `json:"reason"`
	MatchedHookID string `json:"matchedHookId,omitempty"`
}

// EvaluateHookAdmission evaluates if a hook should be admitted
func EvaluateHookAdmission(candidate HookAdmissionCandidate, activeHooks []models.HookRecord) HookAdmissionDecision {
	// Check for duplicate families
	for _, hook := range activeHooks {
		if isSameHookFamily(hook.Type, candidate.Type) {
			return HookAdmissionDecision{
				Admit:         false,
				Reason:        "duplicate_family",
				MatchedHookID: hook.HookID,
			}
		}
	}

	return HookAdmissionDecision{
		Admit:  true,
		Reason: "",
	}
}

// isSameHookFamily 检查if two hooks belong to the same family。
func isSameHookFamily(type1 string, type2 string) bool {
	// Simple implementation - can be enhanced
	return type1 == type2
}

// ResolveHookPayoffTiming 解析hook payoff timing。
func ResolveHookPayoffTiming(timing *models.HookPayoffTiming, expectedPayoff string, notes string) models.HookPayoffTiming {
	if timing != nil {
		return *timing
	}

	// Infer timing from content
	if len(expectedPayoff) > 100 || len(notes) > 100 {
		return models.TimingSlowBurn
	}
	return models.TimingNearTerm
}
