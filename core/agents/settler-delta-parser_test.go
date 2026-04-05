package agents

import (
	"strings"
	"testing"
)

func TestParseSettlerDeltaOutput_ValidPayload(t *testing.T) {
	content := strings.Join([]string{
		"=== POST_SETTLEMENT ===",
		"| 伏笔变动 | mentor-oath 推进 | 同步更新 |",
		"",
		"=== RUNTIME_STATE_DELTA ===",
		"```json",
		`{`,
		`  "chapter": 12,`,
		`  "currentStatePatch": {`,
		`    "currentGoal": "追到旧账尽头",`,
		`    "currentConflict": "商会噪音仍在干扰主线"`,
		`  },`,
		`  "hookOps": {`,
		`    "upsert": [`,
		`      {`,
		`        "hookId": "mentor-oath",`,
		`        "startChapter": 8,`,
		`        "type": "relationship",`,
		`        "status": "progressing",`,
		`        "lastAdvancedChapter": 12,`,
		`        "expectedPayoff": "揭开师债真相",`,
		`        "notes": "旧账线索向前推进"`,
		`      }`,
		`    ],`,
		`    "resolve": [],`,
		`    "defer": []`,
		`  },`,
		`  "chapterSummary": {`,
		`    "chapter": 12,`,
		`    "title": "旧账对照",`,
		`    "characters": "林月",`,
		`    "events": "林月核对旧账",`,
		`    "stateChanges": "师债线索进一步收束",`,
		`    "hookActivity": "mentor-oath advanced",`,
		`    "mood": "紧绷",`,
		`    "chapterType": "主线推进"`,
		`  },`,
		`  "notes": ["保留噪音支线，但不盖过主线"]`,
		`}`,
		"```",
	}, "\n")

	parsed, err := ParseSettlerDeltaOutput(content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if !strings.Contains(parsed.PostSettlement, "mentor-oath") {
		t.Fatalf("expected post settlement block, got %q", parsed.PostSettlement)
	}
	if parsed.RuntimeStateDelta.Chapter != 12 {
		t.Fatalf("expected chapter=12, got %d", parsed.RuntimeStateDelta.Chapter)
	}
	if len(parsed.RuntimeStateDelta.HookOps.Upsert) == 0 || parsed.RuntimeStateDelta.HookOps.Upsert[0].HookID != "mentor-oath" {
		t.Fatalf("expected upsert hook mentor-oath, got %#v", parsed.RuntimeStateDelta.HookOps.Upsert)
	}
	if parsed.RuntimeStateDelta.ChapterSummary == nil || parsed.RuntimeStateDelta.ChapterSummary.Title != "旧账对照" {
		t.Fatalf("expected chapter summary title, got %#v", parsed.RuntimeStateDelta.ChapterSummary)
	}
}

func TestParseSettlerDeltaOutput_InvalidPayload(t *testing.T) {
	content := strings.Join([]string{
		"=== RUNTIME_STATE_DELTA ===",
		"```json",
		`{`,
		`  "chapter": 12,`,
		`  "hookOps": {`,
		`    "upsert": [`,
		`      {`,
		`        "hookId": "mentor-oath",`,
		`        "startChapter": 8,`,
		`        "type": "relationship",`,
		`        "status": "open",`,
		`        "lastAdvancedChapter": "chapter twelve"`,
		`      }`,
		`    ],`,
		`    "resolve": [],`,
		`    "defer": []`,
		`  }`,
		`}`,
		"```",
	}, "\n")

	if _, err := ParseSettlerDeltaOutput(content); err == nil || !strings.Contains(strings.ToLower(err.Error()), "runtime state delta") {
		t.Fatalf("expected runtime state delta error, got %v", err)
	}
}

func TestParseSettlerDeltaOutput_HookOpsAndNewCandidates(t *testing.T) {
	content := strings.Join([]string{
		"=== RUNTIME_STATE_DELTA ===",
		"```json",
		`{`,
		`  "chapter": 21,`,
		`  "hookOps": {`,
		`    "upsert": [],`,
		`    "mention": ["mentor-oath"],`,
		`    "resolve": ["old-seal"],`,
		`    "defer": ["guild-route"]`,
		`  },`,
		`  "newHookCandidates": [`,
		`    {`,
		`      "type": "source-risk",`,
		`      "expectedPayoff": "Reveal what the source knew",`,
		`      "notes": "fresh unresolved source question"`,
		`    }`,
		`  ],`,
		`  "notes": []`,
		`}`,
		"```",
	}, "\n")

	parsed, err := ParseSettlerDeltaOutput(content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(parsed.RuntimeStateDelta.HookOps.Mention) != 1 || parsed.RuntimeStateDelta.HookOps.Mention[0] != "mentor-oath" {
		t.Fatalf("unexpected mention ops: %#v", parsed.RuntimeStateDelta.HookOps.Mention)
	}
	if len(parsed.RuntimeStateDelta.HookOps.Resolve) != 1 || parsed.RuntimeStateDelta.HookOps.Resolve[0] != "old-seal" {
		t.Fatalf("unexpected resolve ops: %#v", parsed.RuntimeStateDelta.HookOps.Resolve)
	}
	if len(parsed.RuntimeStateDelta.HookOps.Defer) != 1 || parsed.RuntimeStateDelta.HookOps.Defer[0] != "guild-route" {
		t.Fatalf("unexpected defer ops: %#v", parsed.RuntimeStateDelta.HookOps.Defer)
	}
	if len(parsed.RuntimeStateDelta.NewHookCandidates) != 1 || parsed.RuntimeStateDelta.NewHookCandidates[0].Type != "source-risk" {
		t.Fatalf("unexpected newHookCandidates: %#v", parsed.RuntimeStateDelta.NewHookCandidates)
	}
}
