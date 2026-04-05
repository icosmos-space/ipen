package utils

import (
	"fmt"
	"regexp"
	"strings"
)

// SplitChapter holds one parsed chapter title/body pair.
type SplitChapter struct {
	Title   string
	Content string
}

var defaultChapterPattern = `(?i)^#{0,2}\s*(?:第[零〇一二三四五六七八九十百千万\d]+(?:章|回)(?:[:：\-]|\s+)?\s*(.*)|chapter\s+(?:\d+|[IVXLCDM]+)(?:\.|:|\s+)?\s*(.*))$`

// SplitChapters splits a long text into chapters by title headings.
func SplitChapters(text string, pattern ...string) []SplitChapter {
	rawPattern := defaultChapterPattern
	if len(pattern) > 0 && strings.TrimSpace(pattern[0]) != "" {
		rawPattern = pattern[0]
	}

	re, err := regexp.Compile(rawPattern)
	if err != nil {
		return []SplitChapter{}
	}

	lines := strings.Split(text, "\n")
	type heading struct {
		Title string
		Index int
	}
	headings := []heading{}
	for i, line := range lines {
		m := re.FindStringSubmatch(line)
		if len(m) == 0 {
			continue
		}

		title := ""
		if len(m) > 1 {
			title = strings.TrimSpace(m[1])
		}
		if title == "" && len(m) > 2 {
			title = strings.TrimSpace(m[2])
		}

		headings = append(headings, heading{Title: title, Index: i})
	}

	if len(headings) == 0 {
		return []SplitChapter{}
	}

	result := make([]SplitChapter, 0, len(headings))
	for i, h := range headings {
		nextStart := len(lines)
		if i+1 < len(headings) {
			nextStart = headings[i+1].Index
		}

		content := strings.Join(lines[h.Index+1:nextStart], "\n")
		content = strings.TrimSpace(stripTrailingLicense(content))

		title := h.Title
		if title == "" {
			title = inferFallbackTitle(lines[h.Index], i+1)
		}

		result = append(result, SplitChapter{Title: title, Content: content})
	}

	return result
}

func stripTrailingLicense(content string) string {
	re := regexp.MustCompile(`(?im)^\s*Project Gutenberg(?:™|\(TM\))?.*$`)
	loc := re.FindStringIndex(content)
	if loc == nil {
		return content
	}
	return strings.TrimSpace(content[:loc[0]])
}

func inferFallbackTitle(headingLine string, chapterNumber int) string {
	if regexp.MustCompile(`(?i)chapter\s+(?:\d+|[ivxlcdm]+)`).MatchString(headingLine) {
		return fmt.Sprintf("Chapter %d", chapterNumber)
	}

	if regexp.MustCompile(`第[零〇一二三四五六七八九十百千万\d]+回`).MatchString(headingLine) {
		return fmt.Sprintf("第%d回", chapterNumber)
	}

	return fmt.Sprintf("第%d章", chapterNumber)
}
