package agents

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/icosmos-space/ipen/core/models"
)

// SettlerDeltaOutput 表示parsed settler delta output。
type SettlerDeltaOutput struct {
	PostSettlement    string                   `json:"postSettlement"`
	RuntimeStateDelta models.RuntimeStateDelta `json:"runtimeStateDelta"`
}

// ParseSettlerDeltaOutput 解析settler output and validates delta json shape。
func ParseSettlerDeltaOutput(content string) (*SettlerDeltaOutput, error) {
	extract := func(tag string) string {
		re := regexp.MustCompile(`(?s)===\s*` + regexp.QuoteMeta(tag) + `\s*===\s*(.*?)(?:(?:\n===\s*[A-Z_]+\s*===)|$)`)
		match := re.FindStringSubmatch(content)
		if len(match) < 2 {
			return ""
		}
		return strings.TrimSpace(match[1])
	}

	rawDelta := extract("RUNTIME_STATE_DELTA")
	if strings.TrimSpace(rawDelta) == "" {
		return nil, fmt.Errorf("runtime state delta block is missing")
	}

	payload := stripCodeFence(rawDelta)
	var parsed models.RuntimeStateDelta
	if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
		return nil, fmt.Errorf("runtime state delta is not valid json: %w", err)
	}

	if parsed.Chapter <= 0 {
		return nil, fmt.Errorf("runtime state delta failed validation: chapter must be > 0")
	}
	if parsed.HookOps.Upsert == nil {
		parsed.HookOps.Upsert = []models.HookRecord{}
	}
	if parsed.HookOps.Mention == nil {
		parsed.HookOps.Mention = []string{}
	}
	if parsed.HookOps.Resolve == nil {
		parsed.HookOps.Resolve = []string{}
	}
	if parsed.HookOps.Defer == nil {
		parsed.HookOps.Defer = []string{}
	}
	if parsed.NewHookCandidates == nil {
		parsed.NewHookCandidates = []models.NewHookCandidate{}
	}
	if parsed.SubplotOps == nil {
		parsed.SubplotOps = []map[string]any{}
	}
	if parsed.EmotionalArcOps == nil {
		parsed.EmotionalArcOps = []map[string]any{}
	}
	if parsed.CharacterMatrixOps == nil {
		parsed.CharacterMatrixOps = []map[string]any{}
	}
	if parsed.Notes == nil {
		parsed.Notes = []string{}
	}

	return &SettlerDeltaOutput{
		PostSettlement:    extract("POST_SETTLEMENT"),
		RuntimeStateDelta: parsed,
	}, nil
}

func stripCodeFence(value string) string {
	trimmed := strings.TrimSpace(value)
	fenced := regexp.MustCompile("(?is)^```(?:json)?\\s*([\\s\\S]*?)\\s*```$").FindStringSubmatch(trimmed)
	if len(fenced) >= 2 {
		return strings.TrimSpace(fenced[1])
	}
	return trimmed
}
