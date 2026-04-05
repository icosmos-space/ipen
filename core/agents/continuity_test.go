package agents

import "testing"

func TestParseAuditResult_ParsesJSONAndNormalizesSeverity(t *testing.T) {
	agent := &ContinuityAuditor{}
	raw := `analysis...
{
  "passed": false,
  "issues": [
    {
      "severity": "CRITICAL",
      "category": "consistency",
      "description": "Character motivation flips without setup.",
      "suggestion": "Add bridge scene."
    }
  ],
  "summary": "Found one critical issue."
}`

	result := agent.parseAuditResult(raw)
	if result.Passed {
		t.Fatalf("expected passed=false")
	}
	if result.Summary != "Found one critical issue." {
		t.Fatalf("unexpected summary: %q", result.Summary)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	if result.Issues[0].Severity != "critical" {
		t.Fatalf("expected severity normalized to critical, got %q", result.Issues[0].Severity)
	}
}

func TestParseAuditResult_FallbackWhenNoJSON(t *testing.T) {
	agent := &ContinuityAuditor{}
	result := agent.parseAuditResult("critical continuity risk detected")

	if result.Passed {
		t.Fatalf("expected passed=false for critical fallback")
	}
	if result.Summary == "" {
		t.Fatalf("expected summary from raw content")
	}
}
