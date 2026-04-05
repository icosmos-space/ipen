package utils

import (
	"regexp"
	"strings"

	"github.com/icosmos-space/ipen/core/models"
	"github.com/icosmos-space/ipen/core/state"
)

// BuildGovernedHookWorkingSet 限制hooks markdown to selected/agenda/recent hooks。
func BuildGovernedHookWorkingSet(
	hooksMarkdown string,
	contextPackage models.ContextPackage,
	chapterIntent string,
	chapterNumber int,
	language string,
	keepRecent int,
) string {
	_ = language
	if keepRecent <= 0 {
		keepRecent = 5
	}
	if strings.TrimSpace(hooksMarkdown) == "" || hooksMarkdown == "(文件不存在)" || hooksMarkdown == "(文件尚未创建)" {
		return hooksMarkdown
	}

	hooks := ParsePendingHooksMarkdown(hooksMarkdown)
	if len(hooks) == 0 {
		return hooksMarkdown
	}

	selectedIDs := map[string]struct{}{}
	for _, entry := range contextPackage.SelectedContext {
		prefix := "story/pending_hooks.md#"
		if strings.HasPrefix(entry.Source, prefix) {
			value := strings.TrimSpace(strings.TrimPrefix(entry.Source, prefix))
			if value != "" {
				selectedIDs[value] = struct{}{}
			}
		}
	}
	agendaIDs := collectHookAgendaIDs(chapterIntent)

	workingSet := []state.StoredHook{}
	for _, hook := range hooks {
		_, inSelected := selectedIDs[hook.HookID]
		_, inAgenda := agendaIDs[hook.HookID]
		if inSelected || inAgenda || IsHookWithinChapterWindow(hook, chapterNumber, keepRecent) {
			workingSet = append(workingSet, hook)
		}
	}

	if len(workingSet) == 0 || len(workingSet) >= len(hooks) {
		return hooksMarkdown
	}
	return RenderHookSnapshot(workingSet, language)
}

func collectHookAgendaIDs(chapterIntent string) map[string]struct{} {
	ids := map[string]struct{}{}
	if strings.TrimSpace(chapterIntent) == "" {
		return ids
	}

	lines := strings.Split(chapterIntent, "\n")
	inHookAgenda := false
	captureIDs := false
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "## Hook Agenda" {
			inHookAgenda = true
			captureIDs = false
			continue
		}
		if !inHookAgenda {
			continue
		}
		if strings.HasPrefix(line, "## ") && line != "## Hook Agenda" {
			break
		}
		if line == "### Must Advance" || line == "### Eligible Resolve" || line == "### Stale Debt" {
			captureIDs = true
			continue
		}
		if strings.HasPrefix(line, "### ") {
			captureIDs = false
			continue
		}
		if !captureIDs || !strings.HasPrefix(line, "- ") {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(line, "- "))
		if value != "" && !strings.EqualFold(value, "none") {
			ids[value] = struct{}{}
		}
	}
	return ids
}

// MergeTableMarkdownByKey 合并updated markdown table rows into original by key columns。
func MergeTableMarkdownByKey(original string, updated string, keyColumns []int) string {
	originalTable := parseSingleTable(original)
	updatedTable := parseSingleTable(updated)
	if originalTable == nil || updatedTable == nil || len(updatedTable.DataRows) == 0 {
		return updated
	}

	mergedRows := append([][]string{}, originalTable.DataRows...)
	indexByKey := map[string]int{}
	for i, row := range mergedRows {
		indexByKey[buildTableKey(row, keyColumns)] = i
	}

	for _, row := range updatedTable.DataRows {
		key := buildTableKey(row, keyColumns)
		if idx, ok := indexByKey[key]; ok {
			mergedRows[idx] = row
		} else {
			indexByKey[key] = len(mergedRows)
			mergedRows = append(mergedRows, row)
		}
	}

	lines := []string{}
	lines = append(lines, pickScaffold(originalTable.LeadingLines, updatedTable.LeadingLines)...)
	for _, row := range mergedRows {
		lines = append(lines, renderTableRow(row))
	}
	lines = append(lines, pickScaffold(originalTable.TrailingLines, updatedTable.TrailingLines)...)
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

// MergeCharacterMatrixMarkdown 合并character-matrix sections while keeping original scaffold。
func MergeCharacterMatrixMarkdown(original string, updated string) string {
	originalSections := parseSections(original)
	updatedSections := parseSections(updated)
	if len(originalSections.Sections) == 0 || len(updatedSections.Sections) == 0 {
		return updated
	}

	sectionKeyColumns := [][]int{{0}, {0, 1}, {0, 3}}
	mergedSections := []matrixSection{}
	for i, section := range originalSections.Sections {
		next := matrixSection{}
		if i < len(updatedSections.Sections) {
			next = updatedSections.Sections[i]
		} else {
			mergedSections = append(mergedSections, section)
			continue
		}
		keyCols := []int{0}
		if i < len(sectionKeyColumns) {
			keyCols = sectionKeyColumns[i]
		}
		mergedSections = append(mergedSections, matrixSection{
			Heading: section.Heading,
			Body:    MergeTableMarkdownByKey(section.Body, next.Body, keyCols),
		})
	}
	for i := len(originalSections.Sections); i < len(updatedSections.Sections); i++ {
		mergedSections = append(mergedSections, updatedSections.Sections[i])
	}

	lines := append([]string{}, pickScaffold(originalSections.TopLines, updatedSections.TopLines)...)
	for _, section := range mergedSections {
		lines = append(lines, section.Heading, section.Body)
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

// BuildGovernedCharacterMatrixWorkingSet 过滤matrix sections to active governed characters。
func BuildGovernedCharacterMatrixWorkingSet(
	matrixMarkdown string,
	chapterIntent string,
	contextPackage models.ContextPackage,
	protagonistName string,
) string {
	if strings.TrimSpace(matrixMarkdown) == "" || matrixMarkdown == "(文件不存在)" || matrixMarkdown == "(文件尚未创建)" {
		return matrixMarkdown
	}
	parsed := parseSections(matrixMarkdown)
	if len(parsed.Sections) == 0 {
		return matrixMarkdown
	}

	active := collectGovernedCharacterNames(matrixMarkdown, chapterIntent, contextPackage, protagonistName)
	filtered := []matrixSection{}
	for index, section := range parsed.Sections {
		filtered = append(filtered, matrixSection{
			Heading: section.Heading,
			Body:    filterMatrixSection(section.Body, index, active),
		})
	}

	lines := append([]string{}, parsed.TopLines...)
	for _, section := range filtered {
		lines = append(lines, section.Heading, section.Body)
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

type parsedTable struct {
	LeadingLines  []string
	DataRows      [][]string
	TrailingLines []string
}

type matrixSection struct {
	Heading string
	Body    string
}

type parsedSections struct {
	TopLines []string
	Sections []matrixSection
}

func parseSingleTable(content string) *parsedTable {
	lines := strings.Split(content, "\n")
	tableIndexes := []int{}
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "|") {
			tableIndexes = append(tableIndexes, i)
		}
	}
	if len(tableIndexes) == 0 {
		return nil
	}

	headerStart := tableIndexes[0]
	headerEnd := headerStart
	if len(tableIndexes) > 1 && strings.Contains(lines[tableIndexes[1]], "---") {
		headerEnd = tableIndexes[1]
	}
	dataIndexes := []int{}
	for _, index := range tableIndexes {
		if index > headerEnd {
			dataIndexes = append(dataIndexes, index)
		}
	}
	lastData := headerEnd
	if len(dataIndexes) > 0 {
		lastData = dataIndexes[len(dataIndexes)-1]
	}

	dataRows := [][]string{}
	for _, index := range dataIndexes {
		dataRows = append(dataRows, parseTableRow(lines[index]))
	}
	return &parsedTable{
		LeadingLines:  lines[:headerEnd+1],
		DataRows:      dataRows,
		TrailingLines: lines[lastData+1:],
	}
}

func parseSections(content string) parsedSections {
	lines := strings.Split(content, "\n")
	result := parsedSections{TopLines: []string{}, Sections: []matrixSection{}}
	currentHeading := ""
	currentBody := []string{}
	flush := func() {
		if currentHeading == "" {
			return
		}
		result.Sections = append(result.Sections, matrixSection{Heading: currentHeading, Body: strings.TrimRight(strings.Join(currentBody, "\n"), "\n")})
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "### ") {
			flush()
			currentHeading = line
			currentBody = []string{}
			continue
		}
		if currentHeading == "" {
			result.TopLines = append(result.TopLines, line)
		} else {
			currentBody = append(currentBody, line)
		}
	}
	flush()
	return result
}

func parseTableRow(line string) []string {
	parts := strings.Split(line, "|")
	cells := []string{}
	for _, cell := range parts[1 : len(parts)-1] {
		cells = append(cells, strings.TrimSpace(cell))
	}
	return cells
}

func renderTableRow(row []string) string {
	return "| " + strings.Join(row, " | ") + " |"
}

func buildTableKey(row []string, keyColumns []int) string {
	parts := []string{}
	for _, idx := range keyColumns {
		if idx >= 0 && idx < len(row) {
			parts = append(parts, row[idx])
		} else {
			parts = append(parts, "")
		}
	}
	return strings.Join(parts, "::")
}

func pickScaffold(primary []string, fallback []string) []string {
	if len(primary) > 0 {
		return primary
	}
	return fallback
}

func collectGovernedCharacterNames(matrixMarkdown string, chapterIntent string, contextPackage models.ContextPackage, protagonistName string) map[string]struct{} {
	candidates := extractCharacterCandidatesFromMatrix(matrixMarkdown)
	chunks := []string{chapterIntent}
	for _, entry := range contextPackage.SelectedContext {
		chunks = append(chunks, entry.Reason)
		if entry.Excerpt != nil {
			chunks = append(chunks, *entry.Excerpt)
		}
	}
	corpus := strings.Join(chunks, "\n")
	active := map[string]struct{}{}
	for _, candidate := range candidates {
		if protagonistName != "" && matchesName(candidate, protagonistName) {
			active[candidate] = struct{}{}
			continue
		}
		if isNameMentioned(candidate, corpus) {
			active[candidate] = struct{}{}
		}
	}
	if protagonistName != "" {
		for _, candidate := range candidates {
			if matchesName(candidate, protagonistName) {
				active[candidate] = struct{}{}
			}
		}
	}
	return active
}

func extractCharacterCandidatesFromMatrix(matrixMarkdown string) []string {
	parsed := parseSections(matrixMarkdown)
	names := map[string]struct{}{}
	for index, section := range parsed.Sections {
		table := parseSingleTable(section.Body)
		if table == nil {
			continue
		}
		for _, row := range table.DataRows {
			candidates := []string{}
			if index == 1 {
				candidates = append(candidates, getCell(row, 0), getCell(row, 1))
			} else {
				candidates = append(candidates, getCell(row, 0))
			}
			for _, candidate := range candidates {
				if trimmed := strings.TrimSpace(candidate); trimmed != "" {
					names[trimmed] = struct{}{}
				}
			}
		}
	}
	result := []string{}
	for name := range names {
		result = append(result, name)
	}
	return result
}

func filterMatrixSection(sectionBody string, sectionIndex int, activeNames map[string]struct{}) string {
	table := parseSingleTable(sectionBody)
	if table == nil {
		return sectionBody
	}
	filtered := [][]string{}
	for _, row := range table.DataRows {
		if len(row) == 0 {
			continue
		}
		if sectionIndex == 1 {
			left := getCell(row, 0)
			right := getCell(row, 1)
			_, leftActive := activeNames[left]
			_, rightActive := activeNames[right]
			if leftActive && (right == "" || rightActive) {
				filtered = append(filtered, row)
			}
			continue
		}
		if _, ok := activeNames[getCell(row, 0)]; ok {
			filtered = append(filtered, row)
		}
	}

	lines := append([]string{}, table.LeadingLines...)
	for _, row := range filtered {
		lines = append(lines, renderTableRow(row))
	}
	lines = append(lines, table.TrailingLines...)
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

func isNameMentioned(candidate string, corpus string) bool {
	if candidate == "" || corpus == "" {
		return false
	}
	if containsCJK(candidate) {
		return strings.Contains(corpus, candidate)
	}
	pattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(candidate) + `\b`)
	return pattern.MatchString(corpus)
}

func matchesName(left string, right string) bool {
	if left == "" || right == "" {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(left), strings.TrimSpace(right))
}

func containsCJK(value string) bool {
	return regexp.MustCompile(`[\p{Han}]`).MatchString(value)
}
