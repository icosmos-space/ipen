package state

import (
	"testing"

	"github.com/icosmos-space/ipen/core/models"
)

// ========== EvaluateHookAdmission ==========

func TestEvaluateHookAdmission_AdmitWhenNoDuplicates(t *testing.T) {
	candidate := HookAdmissionCandidate{
		Type:           "mystery",
		ExpectedPayoff: "Reveal the killer",
		Notes:          "Important mystery",
	}
	activeHooks := []models.HookRecord{
		{HookID: "romance-1", Type: "romance", Status: models.HookStatusOpenRT},
	}

	decision := EvaluateHookAdmission(candidate, activeHooks)

	if !decision.Admit {
		t.Fatalf("expected admit=true, got false")
	}
	if decision.Reason != "" {
		t.Fatalf("expected empty reason, got %q", decision.Reason)
	}
}

func TestEvaluateHookAdmission_RejectDuplicateFamily(t *testing.T) {
	candidate := HookAdmissionCandidate{
		Type:           "mystery",
		ExpectedPayoff: "Reveal the killer",
		Notes:          "",
	}
	activeHooks := []models.HookRecord{
		{HookID: "mystery-1", Type: "mystery", Status: models.HookStatusOpenRT},
	}

	decision := EvaluateHookAdmission(candidate, activeHooks)

	if decision.Admit {
		t.Fatalf("expected admit=false, got true")
	}
	if decision.Reason != "duplicate_family" {
		t.Fatalf("expected reason=duplicate_family, got %q", decision.Reason)
	}
	if decision.MatchedHookID != "mystery-1" {
		t.Fatalf("expected matchedHookId=mystery-1, got %q", decision.MatchedHookID)
	}
}

func TestEvaluateHookAdmission_RejectWithMultipleActiveHooks(t *testing.T) {
	candidate := HookAdmissionCandidate{
		Type: "romance",
	}
	activeHooks := []models.HookRecord{
		{HookID: "mystery-1", Type: "mystery", Status: models.HookStatusOpenRT},
		{HookID: "romance-1", Type: "romance", Status: models.HookStatusProgressingRT},
		{HookID: "hook-3", Type: "action", Status: models.HookStatusOpenRT},
	}

	decision := EvaluateHookAdmission(candidate, activeHooks)

	if decision.Admit {
		t.Fatalf("expected admit=false, got true")
	}
	if decision.MatchedHookID != "romance-1" {
		t.Fatalf("expected matchedHookId=romance-1, got %q", decision.MatchedHookID)
	}
}

func TestEvaluateHookAdmission_EmptyActiveHooks(t *testing.T) {
	candidate := HookAdmissionCandidate{
		Type: "any",
	}

	decision := EvaluateHookAdmission(candidate, []models.HookRecord{})

	if !decision.Admit {
		t.Fatalf("expected admit=true for empty active hooks, got false")
	}
}

// ========== isSameHookFamily ==========

func TestIsSameHookFamily_SameType(t *testing.T) {
	if !isSameHookFamily("mystery", "mystery") {
		t.Fatal("expected same family for identical types")
	}
}

func TestIsSameHookFamily_DifferentType(t *testing.T) {
	if isSameHookFamily("mystery", "romance") {
		t.Fatal("expected different family for different types")
	}
}

func TestIsSameHookFamily_CaseSensitive(t *testing.T) {
	if isSameHookFamily("Mystery", "mystery") {
		t.Fatal("expected case sensitive comparison")
	}
}

// ========== ResolveHookPayoffTiming ==========

func TestResolveHookPayoffTiming_UsesProvidedTiming(t *testing.T) {
	timing := models.TimingImmediate
	result := ResolveHookPayoffTiming(&timing, "short", "notes")

	if result != models.TimingImmediate {
		t.Fatalf("expected immediate, got %s", result)
	}
}

func TestResolveHookPayoffTiming_NilTiming_LongPayoff(t *testing.T) {
	// Create a string > 100 characters
	longPayoff := ""
	for i := 0; i < 150; i++ {
		longPayoff += "x"
	}
	result := ResolveHookPayoffTiming(nil, longPayoff, "notes")

	if result != models.TimingSlowBurn {
		t.Fatalf("expected slow-burn for long payoff, got %s", result)
	}
}

func TestResolveHookPayoffTiming_NilTiming_LongNotes(t *testing.T) {
	result := ResolveHookPayoffTiming(nil, "short", "these are very long notes that should trigger the slow burn timing because the notes exceed 100 characters easily")

	if result != models.TimingSlowBurn {
		t.Fatalf("expected slow-burn for long notes, got %s", result)
	}
}

func TestResolveHookPayoffTiming_NilTiming_ShortContent(t *testing.T) {
	result := ResolveHookPayoffTiming(nil, "short", "notes")

	if result != models.TimingNearTerm {
		t.Fatalf("expected near-term for short content, got %s", result)
	}
}

func TestResolveHookPayoffTiming_NilTiming_EmptyContent(t *testing.T) {
	result := ResolveHookPayoffTiming(nil, "", "")

	if result != models.TimingNearTerm {
		t.Fatalf("expected near-term for empty content, got %s", result)
	}
}

func TestResolveHookPayoffTiming_NilTiming_Exactly100Chars(t *testing.T) {
	// Exactly 100 characters
	payload := ""
	for i := 0; i < 100; i++ {
		payload += "a"
	}
	result := ResolveHookPayoffTiming(nil, payload, "")

	// Should be near-term since it's not > 100
	if result != models.TimingNearTerm {
		t.Fatalf("expected near-term for exactly 100 chars, got %s", result)
	}
}

func TestResolveHookPayoffTiming_NilTiming_101Chars(t *testing.T) {
	payload := ""
	for i := 0; i < 101; i++ {
		payload += "a"
	}
	result := ResolveHookPayoffTiming(nil, payload, "")

	if result != models.TimingSlowBurn {
		t.Fatalf("expected slow-burn for 101 chars, got %s", result)
	}
}
