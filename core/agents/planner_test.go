package agents

import "testing"

func TestParsePlannerOutput_ParsesJSONPayload(t *testing.T) {
	agent := &PlannerAgent{}
	raw := `Reasoning first.
{
  "chapter": 9,
  "goal": "Recover the ledger before dawn.",
  "mustKeep": ["mentor debt", "warehouse clue"],
  "mustAvoid": ["retcon"],
  "styleEmphasis": ["tight pacing"]
}
Trailing notes.`

	intent, _ := agent.parsePlannerOutput(raw)
	if intent.Chapter != 9 {
		t.Fatalf("expected chapter 9, got %d", intent.Chapter)
	}
	if intent.Goal != "Recover the ledger before dawn." {
		t.Fatalf("unexpected goal %q", intent.Goal)
	}
	if len(intent.MustKeep) != 2 {
		t.Fatalf("expected 2 mustKeep entries, got %d", len(intent.MustKeep))
	}
	if len(intent.MustAvoid) != 1 || intent.MustAvoid[0] != "retcon" {
		t.Fatalf("unexpected mustAvoid: %#v", intent.MustAvoid)
	}
	if len(intent.StyleEmphasis) != 1 || intent.StyleEmphasis[0] != "tight pacing" {
		t.Fatalf("unexpected styleEmphasis: %#v", intent.StyleEmphasis)
	}
}

func TestParsePlannerOutput_FallbackUsesFirstMeaningfulLine(t *testing.T) {
	agent := &PlannerAgent{}
	raw := "# Plan\n\nAdvance the mentor line this chapter.\n- avoid repetition"

	intent, _ := agent.parsePlannerOutput(raw)
	if intent.Goal != "Advance the mentor line this chapter." {
		t.Fatalf("unexpected fallback goal: %q", intent.Goal)
	}
}
