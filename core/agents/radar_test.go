package agents

import "testing"

func TestParseRadarResult_ParsesJSONAndClampsConfidence(t *testing.T) {
	agent := &RadarAgent{}
	raw := `Result:
{
  "recommendations": [
    {
      "platform": "qidian",
      "genre": "xuanhuan",
      "concept": "Debt-and-oath investigation arc",
      "confidence": 1.4,
      "reasoning": "Top lists show investigation growth.",
      "benchmarkTitles": ["A", "B"]
    }
  ],
  "marketSummary": "Investigation stories are rising."
}`

	result := agent.parseRadarResult(raw)
	if result.MarketSummary != "Investigation stories are rising." {
		t.Fatalf("unexpected market summary: %q", result.MarketSummary)
	}
	if len(result.Recommendations) != 1 {
		t.Fatalf("expected 1 recommendation, got %d", len(result.Recommendations))
	}
	if result.Recommendations[0].Confidence != 1 {
		t.Fatalf("expected confidence to be clamped to 1, got %f", result.Recommendations[0].Confidence)
	}
	if result.Timestamp == "" {
		t.Fatalf("expected timestamp to be set")
	}
}

func TestParseRadarResult_FallbackWhenNoJSON(t *testing.T) {
	agent := &RadarAgent{}
	raw := "plain text summary only"

	result := agent.parseRadarResult(raw)
	if result.MarketSummary != "plain text summary only" {
		t.Fatalf("unexpected fallback summary: %q", result.MarketSummary)
	}
	if len(result.Recommendations) != 0 {
		t.Fatalf("expected no recommendations, got %d", len(result.Recommendations))
	}
}
