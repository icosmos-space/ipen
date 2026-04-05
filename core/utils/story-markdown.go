package utils

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/icosmos-space/ipen/core/state"
)

// RenderSummarySnapshot renders summary rows as markdown table.
func RenderSummarySnapshot(summaries []state.StoredSummary, language string) string {
	if len(summaries) == 0 {
		return "- none"
	}

	headers := []string{}
	if strings.EqualFold(language, "en") {
		headers = []string{
			"| chapter | title | characters | events | stateChanges | hookActivity | mood | chapterType |",
			"| --- | --- | --- | --- | --- | --- | --- | --- |",
		}
	} else {
		headers = []string{
			"| 章节 | 标题 | 出场人物 | 关键事件 | 状态变化 | 伏笔动态 | 情绪基调 | 章节类型 |",
			"| --- | --- | --- | --- | --- | --- | --- | --- |",
		}
	}

	rows := make([]string, 0, len(summaries))
	for _, summary := range summaries {
		cells := []string{
			escapeTableCell(strconv.Itoa(summary.Chapter)),
			escapeTableCell(summary.Title),
			escapeTableCell(summary.Characters),
			escapeTableCell(summary.Events),
			escapeTableCell(summary.StateChanges),
			escapeTableCell(summary.HookActivity),
			escapeTableCell(summary.Mood),
			escapeTableCell(summary.ChapterType),
		}
		rows = append(rows, "| "+strings.Join(cells, " | ")+" |")
	}

	return strings.Join(append(headers, rows...), "\n")
}

// RenderHookSnapshot renders hooks as markdown table.
func RenderHookSnapshot(hooks []state.StoredHook, language string) string {
	if len(hooks) == 0 {
		return "- none"
	}

	headers := []string{}
	if strings.EqualFold(language, "en") {
		headers = []string{
			"| hook_id | start_chapter | type | status | last_advanced | expected_payoff | payoff_timing | notes |",
			"| --- | --- | --- | --- | --- | --- | --- | --- |",
		}
	} else {
		headers = []string{
			"| hook_id | 起始章节 | 类型 | 状态 | 最近推进 | 预期回收 | 回收节奏 | 备注 |",
			"| --- | --- | --- | --- | --- | --- | --- | --- |",
		}
	}

	rows := make([]string, 0, len(hooks))
	for _, hook := range hooks {
		timing := ResolveHookPayoffTiming(hook.PayoffTiming, hook.ExpectedPayoff, hook.Notes)
		localized := LocalizeHookPayoffTiming(timing, language)
		cells := []string{
			escapeTableCell(hook.HookID),
			escapeTableCell(strconv.Itoa(hook.StartChapter)),
			escapeTableCell(hook.Type),
			escapeTableCell(hook.Status),
			escapeTableCell(strconv.Itoa(hook.LastAdvancedChapter)),
			escapeTableCell(hook.ExpectedPayoff),
			escapeTableCell(localized),
			escapeTableCell(hook.Notes),
		}
		rows = append(rows, "| "+strings.Join(cells, " | ")+" |")
	}

	return strings.Join(append(headers, rows...), "\n")
}

// ParseChapterSummariesMarkdown 解析summary markdown table rows。
func ParseChapterSummariesMarkdown(markdown string) []state.StoredSummary {
	rows := ParseMarkdownTableRows(markdown)
	result := []state.StoredSummary{}
	for _, row := range rows {
		if len(row) == 0 || !regexp.MustCompile(`^\d+$`).MatchString(row[0]) {
			continue
		}
		chapter, _ := strconv.Atoi(row[0])
		item := state.StoredSummary{Chapter: chapter}
		if len(row) > 1 {
			item.Title = row[1]
		}
		if len(row) > 2 {
			item.Characters = row[2]
		}
		if len(row) > 3 {
			item.Events = row[3]
		}
		if len(row) > 4 {
			item.StateChanges = row[4]
		}
		if len(row) > 5 {
			item.HookActivity = row[5]
		}
		if len(row) > 6 {
			item.Mood = row[6]
		}
		if len(row) > 7 {
			item.ChapterType = row[7]
		}
		result = append(result, item)
	}
	return result
}

// ParsePendingHooksMarkdown 解析hooks markdown table or fallback bullet list。
func ParsePendingHooksMarkdown(markdown string) []state.StoredHook {
	rows := ParseMarkdownTableRows(markdown)
	tableRows := [][]string{}
	for _, row := range rows {
		if len(row) > 0 && strings.EqualFold(strings.TrimSpace(row[0]), "hook_id") {
			continue
		}
		tableRows = append(tableRows, row)
	}
	if len(tableRows) > 0 {
		result := []state.StoredHook{}
		for _, row := range tableRows {
			hookID := NormalizeHookID(getCell(row, 0))
			if hookID == "" {
				continue
			}
			result = append(result, parsePendingHookRow(row))
		}
		return result
	}

	lines := strings.Split(markdown, "\n")
	result := []state.StoredHook{}
	idx := 1
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "-") {
			continue
		}
		note := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
		if note == "" {
			continue
		}
		result = append(result, state.StoredHook{
			HookID:              "hook-" + strconv.Itoa(idx),
			StartChapter:        0,
			Type:                "unspecified",
			Status:              "open",
			LastAdvancedChapter: 0,
			ExpectedPayoff:      "",
			PayoffTiming:        "",
			Notes:               note,
		})
		idx++
	}
	return result
}

// ParseCurrentStateFacts 解析current_state markdown into temporal facts。
func ParseCurrentStateFacts(markdown string, fallbackChapter int) []state.Fact {
	tableRows := ParseMarkdownTableRows(markdown)
	fieldRows := [][]string{}
	for _, row := range tableRows {
		if len(row) < 2 {
			continue
		}
		if IsStateTableHeaderRow(row) {
			continue
		}
		fieldRows = append(fieldRows, row)
	}

	if len(fieldRows) > 0 {
		stateChapter := fallbackChapter
		for _, row := range fieldRows {
			if IsCurrentChapterLabel(getCell(row, 0)) {
				if chapter := ParseInteger(getCell(row, 1)); chapter > 0 {
					stateChapter = chapter
				}
			}
		}

		facts := []state.Fact{}
		for _, row := range fieldRows {
			label := strings.TrimSpace(getCell(row, 0))
			value := strings.TrimSpace(getCell(row, 1))
			if IsCurrentChapterLabel(label) || label == "" || value == "" {
				continue
			}
			facts = append(facts, state.Fact{
				Subject:           InferFactSubject(label),
				Predicate:         label,
				Object:            value,
				ValidFromChapter:  stateChapter,
				ValidUntilChapter: nil,
				SourceChapter:     stateChapter,
			})
		}
		return facts
	}

	bulletFacts := []string{}
	for _, line := range strings.Split(markdown, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "-") {
			item := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
			if item != "" {
				bulletFacts = append(bulletFacts, item)
			}
		}
	}

	facts := []state.Fact{}
	for i, item := range bulletFacts {
		facts = append(facts, state.Fact{
			Subject:           "current_state",
			Predicate:         "note_" + strconv.Itoa(i+1),
			Object:            item,
			ValidFromChapter:  fallbackChapter,
			ValidUntilChapter: nil,
			SourceChapter:     fallbackChapter,
		})
	}
	return facts
}

// ParseMarkdownTableRows 解析markdown table lines into cell rows。
func ParseMarkdownTableRows(markdown string) [][]string {
	rows := [][]string{}
	for _, line := range strings.Split(markdown, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") {
			continue
		}
		if strings.Contains(trimmed, "---") {
			continue
		}
		parts := strings.Split(trimmed, "|")
		if len(parts) < 3 {
			continue
		}
		cells := []string{}
		for _, cell := range parts[1 : len(parts)-1] {
			cells = append(cells, strings.TrimSpace(cell))
		}
		hasValue := false
		for _, cell := range cells {
			if cell != "" {
				hasValue = true
				break
			}
		}
		if hasValue {
			rows = append(rows, cells)
		}
	}
	return rows
}

// IsStateTableHeaderRow 检查whether a row is the state field-value header row。
func IsStateTableHeaderRow(row []string) bool {
	first := strings.ToLower(strings.TrimSpace(getCell(row, 0)))
	second := strings.ToLower(strings.TrimSpace(getCell(row, 1)))
	return (first == "字段" && second == "值") || (first == "field" && second == "value")
}

// IsCurrentChapterLabel 检查if a label means current chapter。
func IsCurrentChapterLabel(label string) bool {
	return regexp.MustCompile(`(?i)^(当前章节|current\s+chapter)$`).MatchString(strings.TrimSpace(label))
}

// InferFactSubject 推断fact subject from table label。
func InferFactSubject(label string) string {
	trimmed := strings.TrimSpace(label)
	switch {
	case regexp.MustCompile(`(?i)^(当前位置|current\s+location)$`).MatchString(trimmed):
		return "protagonist"
	case regexp.MustCompile(`(?i)^(主角状态|protagonist\s+state)$`).MatchString(trimmed):
		return "protagonist"
	case regexp.MustCompile(`(?i)^(当前目标|current\s+goal)$`).MatchString(trimmed):
		return "protagonist"
	case regexp.MustCompile(`(?i)^(当前限制|current\s+constraint)$`).MatchString(trimmed):
		return "protagonist"
	case regexp.MustCompile(`(?i)^(当前敌我|当前关系|current\s+alliances|current\s+relationships)$`).MatchString(trimmed):
		return "protagonist"
	case regexp.MustCompile(`(?i)^(当前冲突|current\s+conflict)$`).MatchString(trimmed):
		return "protagonist"
	default:
		return "current_state"
	}
}

// ParseInteger 提取first integer from string。
func ParseInteger(value string) int {
	match := regexp.MustCompile(`\d+`).FindString(value)
	if match == "" {
		return 0
	}
	parsed, err := strconv.Atoi(match)
	if err != nil {
		return 0
	}
	return parsed
}

// NormalizeHookID strips markdown wrappers from hook id cells.
func NormalizeHookID(value string) string {
	normalized := strings.TrimSpace(value)
	previous := ""
	for normalized != "" && normalized != previous {
		previous = normalized
		normalized = regexp.MustCompile(`^\[(.+?)\]\([^)]+\)$`).ReplaceAllString(normalized, "$1")
		normalized = regexp.MustCompile(`^\*\*(.+)\*\*$`).ReplaceAllString(normalized, "$1")
		normalized = regexp.MustCompile(`^__(.+)__$`).ReplaceAllString(normalized, "$1")
		normalized = regexp.MustCompile(`^\*(.+)\*$`).ReplaceAllString(normalized, "$1")
		normalized = regexp.MustCompile(`^_(.+)_$`).ReplaceAllString(normalized, "$1")
		normalized = regexp.MustCompile("^`(.+)`$").ReplaceAllString(normalized, "$1")
		normalized = regexp.MustCompile(`^~~(.+)~~$`).ReplaceAllString(normalized, "$1")
		normalized = strings.TrimSpace(normalized)
	}
	return normalized
}

func parsePendingHookRow(row []string) state.StoredHook {
	legacyShape := len(row) < 8
	timing := ""
	if !legacyShape {
		if parsed, ok := NormalizeHookPayoffTiming(getCell(row, 6)); ok {
			timing = string(parsed)
		}
	}
	notes := ""
	if legacyShape {
		notes = getCell(row, 6)
	} else {
		notes = getCell(row, 7)
	}
	return state.StoredHook{
		HookID:              NormalizeHookID(getCell(row, 0)),
		StartChapter:        parseStrictChapterInteger(getCell(row, 1)),
		Type:                getCell(row, 2),
		Status:              defaultString(getCell(row, 3), "open"),
		LastAdvancedChapter: parseStrictChapterInteger(getCell(row, 4)),
		ExpectedPayoff:      getCell(row, 5),
		PayoffTiming:        timing,
		Notes:               notes,
	}
}

func parseStrictChapterInteger(value string) int {
	cleaned := NormalizeHookID(value)
	if !regexp.MustCompile(`^\d+$`).MatchString(cleaned) {
		return 0
	}
	parsed, err := strconv.Atoi(cleaned)
	if err != nil {
		return 0
	}
	return parsed
}

func getCell(row []string, index int) string {
	if index >= 0 && index < len(row) {
		return strings.TrimSpace(row[index])
	}
	return ""
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func escapeTableCell(value string) string {
	return strings.TrimSpace(strings.ReplaceAll(value, "|", `\|`))
}
