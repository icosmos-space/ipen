package utils

import (
	"regexp"
	"strings"
)

// LocalFixPatch 描述one local find/replace patch。
type LocalFixPatch struct {
	TargetText      string
	ReplacementText string
}

// LocalFixPatchApplyResult 是the apply result and safety metadata。
type LocalFixPatchApplyResult struct {
	Applied           bool
	RevisedContent    string
	RejectedReason    string
	AppliedPatchCount int
	TouchedChars      int
}

const maxLocalFixTouchedRatio = 0.25

// ParseLocalFixPatches 解析"--- PATCH ---" blocks from model output。
func ParseLocalFixPatches(raw string) []LocalFixPatch {
	normalized := raw
	marker := "=== PATCHES ==="
	if idx := strings.Index(raw, marker); idx >= 0 {
		normalized = raw[idx+len(marker):]
	}

	re := regexp.MustCompile(`(?s)--- PATCH(?:\s+\d+)? ---\s*TARGET_TEXT:\s*(.*?)\s*REPLACEMENT_TEXT:\s*(.*?)\s*--- END PATCH ---`)
	matches := re.FindAllStringSubmatch(normalized, -1)
	result := make([]LocalFixPatch, 0, len(matches))
	for _, m := range matches {
		target := trimPatchField(getMatch(m, 1))
		replacement := trimPatchField(getMatch(m, 2))
		if strings.TrimSpace(target) == "" {
			continue
		}
		result = append(result, LocalFixPatch{TargetText: target, ReplacementText: replacement})
	}
	return result
}

// ApplyLocalFixPatches 应用exact-once local patches with touched-ratio guard。
func ApplyLocalFixPatches(original string, patches []LocalFixPatch) LocalFixPatchApplyResult {
	if len(patches) == 0 {
		return LocalFixPatchApplyResult{
			Applied:           false,
			RevisedContent:    original,
			RejectedReason:    "No valid patches returned.",
			AppliedPatchCount: 0,
			TouchedChars:      0,
		}
	}

	touchedChars := 0
	for _, patch := range patches {
		touchedChars += len([]rune(patch.TargetText))
	}

	if len([]rune(original)) > 0 {
		if float64(touchedChars)/float64(len([]rune(original))) > maxLocalFixTouchedRatio {
			return LocalFixPatchApplyResult{
				Applied:           false,
				RevisedContent:    original,
				RejectedReason:    "Patch set would touch too much of the chapter.",
				AppliedPatchCount: 0,
				TouchedChars:      touchedChars,
			}
		}
	}

	current := original
	for _, patch := range patches {
		start := strings.Index(current, patch.TargetText)
		if start < 0 {
			return LocalFixPatchApplyResult{
				Applied:           false,
				RevisedContent:    original,
				RejectedReason:    "Each TARGET_TEXT must match the chapter exactly once.",
				AppliedPatchCount: 0,
				TouchedChars:      touchedChars,
			}
		}

		next := strings.Index(current[start+len(patch.TargetText):], patch.TargetText)
		if next >= 0 {
			return LocalFixPatchApplyResult{
				Applied:           false,
				RevisedContent:    original,
				RejectedReason:    "Each TARGET_TEXT must match the chapter exactly once.",
				AppliedPatchCount: 0,
				TouchedChars:      touchedChars,
			}
		}

		current = current[:start] + patch.ReplacementText + current[start+len(patch.TargetText):]
	}

	return LocalFixPatchApplyResult{
		Applied:           current != original,
		RevisedContent:    current,
		AppliedPatchCount: len(patches),
		TouchedChars:      touchedChars,
	}
}

func getMatch(parts []string, index int) string {
	if index >= 0 && index < len(parts) {
		return parts[index]
	}
	return ""
}

func trimPatchField(value string) string {
	result := strings.TrimPrefix(value, "\n")
	result = strings.TrimSuffix(result, "\n")
	return strings.TrimSpace(result)
}
