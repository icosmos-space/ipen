package utils

import (
	"regexp"
	"strconv"
	"strings"
)

// ExtractPOVFromOutline heuristically extracts chapter POV from volume outline.
func ExtractPOVFromOutline(volumeOutline string, chapterNumber int) string {
	lines := strings.Split(volumeOutline, "\n")
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`第\s*` + strconv.Itoa(chapterNumber) + `\s*章`),
		regexp.MustCompile(`(?i)chapter\s+` + strconv.Itoa(chapterNumber) + `\b`),
		regexp.MustCompile(`\b` + strconv.Itoa(chapterNumber) + `\b.*章`),
	}

	inSection := false
	for _, line := range lines {
		matched := false
		for _, pattern := range patterns {
			if pattern.MatchString(line) {
				matched = true
				break
			}
		}
		if matched {
			inSection = true
		} else if inSection && regexp.MustCompile(`^[#-]`).MatchString(strings.TrimSpace(line)) && !strings.Contains(line, strconv.Itoa(chapterNumber)) {
			break
		}

		if inSection {
			m := regexp.MustCompile(`(?i)(?:POV|视角)[:：\s]+([^\s,，。；;]+)`).FindStringSubmatch(line)
			if len(m) > 1 {
				return strings.TrimSpace(m[1])
			}
		}
	}

	return ""
}

// FilterMatrixByPOV 保留only POV-visible info-boundary rows。
func FilterMatrixByPOV(characterMatrix string, povCharacter string) string {
	if strings.TrimSpace(characterMatrix) == "" || characterMatrix == "(文件尚未创建)" || strings.TrimSpace(povCharacter) == "" {
		return characterMatrix
	}

	sections := regexp.MustCompile(`(?m)(?=^###)`).Split(characterMatrix, -1)
	for i, section := range sections {
		if !regexp.MustCompile(`(?i)信息边界|Information\s+Boundar`).MatchString(section) {
			continue
		}
		lines := strings.Split(section, "\n")
		headerLines := []string{}
		dataLines := []string{}
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, "|") {
				continue
			}
			if strings.Contains(trimmed, "---") || strings.Contains(trimmed, "角色") || strings.Contains(strings.ToLower(trimmed), "character") || strings.Contains(trimmed, "已知") || strings.Contains(strings.ToLower(trimmed), "known") {
				headerLines = append(headerLines, line)
			} else {
				dataLines = append(dataLines, line)
			}
		}

		povRows := []string{}
		for _, row := range dataLines {
			if strings.Contains(row, povCharacter) {
				povRows = append(povRows, row)
			}
		}
		otherCount := len(dataLines) - len(povRows)
		header := "### 信息边界"
		for _, line := range lines {
			if strings.HasPrefix(line, "###") {
				header = line
				break
			}
		}

		sections[i] = strings.Join(append([]string{
			header,
			"（当前视角：" + povCharacter + "，其余 " + strconv.Itoa(otherCount) + " 个角色信息边界已隐藏）",
		}, append(headerLines, povRows...)...), "\n")
	}

	return strings.Join(sections, "")
}

// FilterHooksByPOV 保留hooks likely visible to current POV。
func FilterHooksByPOV(hooks string, povCharacter string, chapterSummaries string) string {
	if strings.TrimSpace(hooks) == "" || hooks == "(文件尚未创建)" || strings.TrimSpace(povCharacter) == "" {
		return hooks
	}

	lines := strings.Split(hooks, "\n")
	headers := []string{}
	data := []string{}
	nonTable := []string{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") {
			nonTable = append(nonTable, line)
			continue
		}
		if strings.Contains(trimmed, "hook_id") || strings.Contains(trimmed, "---") {
			headers = append(headers, line)
		} else {
			data = append(data, line)
		}
	}

	povChapters := map[int]struct{}{}
	for _, line := range strings.Split(chapterSummaries, "\n") {
		if !strings.Contains(line, povCharacter) {
			continue
		}
		m := regexp.MustCompile(`\|\s*(\d+)\s*\|`).FindStringSubmatch(line)
		if len(m) > 1 {
			if chapter, err := strconv.Atoi(m[1]); err == nil {
				povChapters[chapter] = struct{}{}
			}
		}
	}

	filtered := []string{}
	for _, row := range data {
		if strings.Contains(row, povCharacter) {
			filtered = append(filtered, row)
			continue
		}
		m := regexp.MustCompile(`\|\s*(\d+)\s*\|`).FindStringSubmatch(row)
		if len(m) <= 1 {
			filtered = append(filtered, row)
			continue
		}
		chapter, err := strconv.Atoi(m[1])
		if err != nil {
			filtered = append(filtered, row)
			continue
		}
		if _, ok := povChapters[chapter]; ok {
			filtered = append(filtered, row)
		}
	}

	if len(filtered) == 0 && len(data) > 0 {
		return hooks
	}
	return strings.Join(append(append(nonTable, headers...), filtered...), "\n")
}
