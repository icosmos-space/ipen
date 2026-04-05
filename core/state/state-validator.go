package state

// RuntimeStateValidationIssue 表示a validation issue。
type RuntimeStateValidationIssue struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Severity string `json:"severity"` // "error", "warning"
}

// ValidateRuntimeState 校验a runtime state snapshot。
func ValidateRuntimeState(snapshot RuntimeStateSnapshot) []RuntimeStateValidationIssue {
	var issues []RuntimeStateValidationIssue

	// Validate manifest
	if snapshot.Manifest.SchemaVersion != 2 {
		issues = append(issues, RuntimeStateValidationIssue{
			Code:     "INVALID_SCHEMA_VERSION",
			Message:  "Schema version must be 2",
			Severity: "error",
		})
	}

	if snapshot.Manifest.Language != "zh" && snapshot.Manifest.Language != "en" {
		issues = append(issues, RuntimeStateValidationIssue{
			Code:     "INVALID_LANGUAGE",
			Message:  "Language must be 'zh' or 'en'",
			Severity: "error",
		})
	}

	if snapshot.Manifest.LastAppliedChapter < 0 {
		issues = append(issues, RuntimeStateValidationIssue{
			Code:     "INVALID_LAST_APPLIED_CHAPTER",
			Message:  "Last applied chapter must be >= 0",
			Severity: "error",
		})
	}

	// Validate current state
	if snapshot.CurrentState.Chapter != snapshot.Manifest.LastAppliedChapter {
		issues = append(issues, RuntimeStateValidationIssue{
			Code:     "CHAPTER_MISMATCH",
			Message:  "Current state chapter must match manifest last applied chapter",
			Severity: "error",
		})
	}

	// Validate hooks
	hookIDs := make(map[string]bool)
	for _, hook := range snapshot.Hooks.Hooks {
		if hookIDs[hook.HookID] {
			issues = append(issues, RuntimeStateValidationIssue{
				Code:     "DUPLICATE_HOOK",
				Message:  "Duplicate hook ID: " + hook.HookID,
				Severity: "error",
			})
		}
		hookIDs[hook.HookID] = true

		if hook.StartChapter < 0 {
			issues = append(issues, RuntimeStateValidationIssue{
				Code:     "INVALID_HOOK_START_CHAPTER",
				Message:  "Hook start chapter must be >= 0",
				Severity: "error",
			})
		}

		if hook.LastAdvancedChapter < hook.StartChapter {
			issues = append(issues, RuntimeStateValidationIssue{
				Code:     "INVALID_HOOK_LAST_ADVANCED",
				Message:  "Hook last advanced chapter must be >= start chapter",
				Severity: "error",
			})
		}
	}

	// Validate chapter summaries
	summaryChapters := make(map[int]bool)
	for _, row := range snapshot.ChapterSummaries.Rows {
		if summaryChapters[row.Chapter] {
			issues = append(issues, RuntimeStateValidationIssue{
				Code:     "DUPLICATE_SUMMARY",
				Message:  "Duplicate chapter summary for chapter " + string(rune(row.Chapter)),
				Severity: "error",
			})
		}
		summaryChapters[row.Chapter] = true

		if row.Chapter < 1 {
			issues = append(issues, RuntimeStateValidationIssue{
				Code:     "INVALID_SUMMARY_CHAPTER",
				Message:  "Chapter summary chapter must be >= 1",
				Severity: "error",
			})
		}
	}

	return issues
}
