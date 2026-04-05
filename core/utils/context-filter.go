package utils

import (
	"regexp"
	"strconv"
	"strings"
)

var chapterCellPattern = regexp.MustCompile(`\|\s*(\d+)\s*\|`)
var cnNamePattern = regexp.MustCompile(`[\p{Han}]{2,4}`)
var enNamePattern = regexp.MustCompile(`\b[A-Z][a-z]{2,}\b`)

const contextMissingPlaceholder = "(文件尚未创建)"

// FilterTableRows 过滤table rows by a predicate。
// It 保留header/separator rows and falls back to original content。
// when all data rows are filtered out.
func FilterTableRows(board string, predicate func(row string) bool) string {
	if strings.TrimSpace(board) == "" {
		return board
	}

	lines := strings.Split(board, "\n")
	nonTable := make([]string, 0, len(lines))
	headers := make([]string, 0, len(lines))
	data := make([]string, 0, len(lines))

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") {
			nonTable = append(nonTable, line)
			continue
		}
		if strings.Contains(trimmed, "---") || isLikelyTableHeader(trimmed) {
			headers = append(headers, line)
			continue
		}
		data = append(data, line)
	}

	if len(data) == 0 {
		return board
	}

	filteredData := make([]string, 0, len(data))
	for _, line := range data {
		if predicate(line) {
			filteredData = append(filteredData, line)
		}
	}

	if len(filteredData) == 0 {
		return board
	}

	combined := append(append(nonTable, headers...), filteredData...)
	return strings.Join(combined, "\n")
}

// FilterHooks 过滤pending hooks, removing resolved/closed rows。
func FilterHooks(hooks string) string {
	if hooks == "" || isContextMissingPlaceholder(hooks) {
		return hooks
	}

	return FilterTableRows(hooks, func(row string) bool {
		lower := strings.ToLower(row)
		return !strings.Contains(row, "已回收") &&
			!strings.Contains(lower, "resolved") &&
			!strings.Contains(lower, "closed")
	})
}

// FilterSummaries 过滤chapter summaries, keeping only the most recent N chapters。
func FilterSummaries(summaries string, currentChapter int, keepRecent ...int) string {
	if summaries == "" || isContextMissingPlaceholder(summaries) {
		return summaries
	}

	window := DEFAULT_CHAPTER_CADENCE_WINDOW
	if len(keepRecent) > 0 && keepRecent[0] > 0 {
		window = keepRecent[0]
	}

	return FilterTableRows(summaries, func(row string) bool {
		match := chapterCellPattern.FindStringSubmatch(row)
		if len(match) < 2 {
			return true
		}
		chapter, err := strconv.Atoi(match[1])
		if err != nil {
			return true
		}
		return chapter > currentChapter-window
	})
}

// FilterSubplots 过滤subplot board, removing closed/resolved subplots。
func FilterSubplots(board string) string {
	if board == "" || isContextMissingPlaceholder(board) {
		return board
	}

	return FilterTableRows(board, func(row string) bool {
		lower := strings.ToLower(row)
		return !strings.Contains(row, "已回收") &&
			!strings.Contains(row, "已完结") &&
			!strings.Contains(lower, "closed") &&
			!strings.Contains(lower, "resolved")
	})
}

// FilterEmotionalArcs 过滤emotional arcs, keeping only recent N chapters。
func FilterEmotionalArcs(arcs string, currentChapter int, keepRecent ...int) string {
	if arcs == "" || isContextMissingPlaceholder(arcs) {
		return arcs
	}

	window := DEFAULT_CHAPTER_CADENCE_WINDOW
	if len(keepRecent) > 0 && keepRecent[0] > 0 {
		window = keepRecent[0]
	}

	return FilterTableRows(arcs, func(row string) bool {
		match := chapterCellPattern.FindStringSubmatch(row)
		if len(match) < 2 {
			return true
		}
		chapter, err := strconv.Atoi(match[1])
		if err != nil {
			return true
		}
		return chapter > currentChapter-window
	})
}

// FilterCharacterMatrix 保留only names mentioned in the current outline/protagonist context。
func FilterCharacterMatrix(matrix string, volumeOutline string, protagonistName string) string {
	if matrix == "" || isContextMissingPlaceholder(matrix) {
		return matrix
	}

	names := extractNames(volumeOutline)
	if strings.TrimSpace(protagonistName) != "" {
		names[strings.TrimSpace(protagonistName)] = struct{}{}
	}
	if len(names) == 0 {
		return matrix
	}

	sections := splitByHeading(matrix)
	filteredSections := make([]string, 0, len(sections))
	for _, section := range sections {
		filteredSections = append(filteredSections, FilterTableRows(section, func(row string) bool {
			for name := range names {
				if strings.Contains(row, name) {
					return true
				}
			}
			return false
		}))
	}

	result := strings.Join(filteredSections, "\n")
	dataRowCount := 0
	for _, line := range strings.Split(result, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") || strings.Contains(trimmed, "---") || isLikelyTableHeader(trimmed) {
			continue
		}
		dataRowCount++
	}

	if dataRowCount == 0 {
		return matrix
	}
	return result
}

func isLikelyTableHeader(line string) bool {
	trimmed := strings.TrimSpace(line)
	lower := strings.ToLower(trimmed)
	return strings.HasPrefix(lower, "| chapter") ||
		strings.HasPrefix(lower, "| character") ||
		strings.HasPrefix(lower, "| subplot") ||
		strings.HasPrefix(lower, "| hook_id") ||
		strings.HasPrefix(trimmed, "| 章节") ||
		strings.HasPrefix(trimmed, "| 角色") ||
		strings.HasPrefix(trimmed, "| 支线")
}

func isContextMissingPlaceholder(content string) bool {
	trimmed := strings.TrimSpace(content)
	return trimmed == contextMissingPlaceholder || strings.Contains(trimmed, "尚未创建")
}

func extractNames(text string) map[string]struct{} {
	names := map[string]struct{}{}
	for _, name := range cnNamePattern.FindAllString(text, -1) {
		names[name] = struct{}{}
	}
	for _, name := range enNamePattern.FindAllString(text, -1) {
		names[name] = struct{}{}
	}
	return names
}

func splitByHeading(content string) []string {
	lines := strings.Split(content, "\n")
	sections := []string{}
	current := []string{}

	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "###") && len(current) > 0 {
			sections = append(sections, strings.Join(current, "\n"))
			current = []string{}
		}
		current = append(current, line)
	}

	if len(current) > 0 {
		sections = append(sections, strings.Join(current, "\n"))
	}
	if len(sections) == 0 {
		return []string{content}
	}
	return sections
}
