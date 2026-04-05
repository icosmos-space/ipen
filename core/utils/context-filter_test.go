package utils

import (
	"strings"
	"testing"
)

func TestFilterSummaries_KeepRecentRows(t *testing.T) {
	summaries := strings.Join([]string{
		"# Chapter Summaries",
		"",
		"| 1 | Chapter 1 | Lin Yue | Old event | state-1 | side-quest-1 | tense | drama |",
		"| 97 | Chapter 97 | Lin Yue | Recent event | state-97 | side-quest-97 | tense | drama |",
		"| 98 | Chapter 98 | Lin Yue | New event | state-98 | side-quest-98 | tense | drama |",
		"| 100 | Chapter 100 | Lin Yue | Latest event | state-100 | mentor-oath advanced | tense | drama |",
	}, "\n")

	filtered := FilterSummaries(summaries, 101)
	if strings.Contains(filtered, "| 1 | Chapter 1 |") || strings.Contains(filtered, "| 97 | Chapter 97 |") {
		t.Fatalf("expected old rows removed, got:\n%s", filtered)
	}
	if !strings.Contains(filtered, "| 98 | Chapter 98 |") || !strings.Contains(filtered, "| 100 | Chapter 100 |") {
		t.Fatalf("expected recent rows kept, got:\n%s", filtered)
	}
}

func TestFilterEmotionalArcs_DefaultCadenceWindow(t *testing.T) {
	arcs := strings.Join([]string{
		"# Emotional Arcs",
		"",
		"| Character | Chapter | Emotional State | Trigger Event | Intensity (1-10) | Arc Direction |",
		"| --- | --- | --- | --- | --- | --- |",
		"| Lin Yue | 97 | guarded | old wound | 4 | holding |",
		"| Lin Yue | 98 | tense | harbor clue | 6 | rising |",
		"| Lin Yue | 99 | strained | mentor echo | 7 | tightening |",
		"| Lin Yue | 100 | brittle | oath pressure | 8 | compressing |",
	}, "\n")

	filtered := FilterEmotionalArcs(arcs, 101)
	if strings.Contains(filtered, "| Lin Yue | 97 |") {
		t.Fatalf("expected chapter 97 arc removed, got:\n%s", filtered)
	}
	if !strings.Contains(filtered, "| Lin Yue | 98 |") || !strings.Contains(filtered, "| Lin Yue | 100 |") {
		t.Fatalf("expected recent arc rows kept, got:\n%s", filtered)
	}
}
