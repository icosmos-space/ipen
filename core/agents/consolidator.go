package agents

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/icosmos-space/ipen/core/llm"
)

// ConsolidationResult 是the output for chapter-summary consolidation。
type ConsolidationResult struct {
	VolumeSummaries  string `json:"volumeSummaries"`
	ArchivedVolumes  int    `json:"archivedVolumes"`
	RetainedChapters int    `json:"retainedChapters"`
}

type consolidationVolumeBoundary struct {
	Name    string
	StartCh int
	EndCh   int
}

type consolidationSummaryRow struct {
	Chapter int
	Raw     string
}

// ConsolidatorAgent consolidates detailed chapter summaries into volume summaries.
type ConsolidatorAgent struct {
	*BaseAgent
}

// NewConsolidatorAgent 创建consolidator agent。
func NewConsolidatorAgent(ctx AgentContext) *ConsolidatorAgent {
	return &ConsolidatorAgent{BaseAgent: NewBaseAgent(ctx)}
}

// Name 返回agent name。
func (a *ConsolidatorAgent) Name() string {
	return "consolidator"
}

// Consolidate consolidates chapter summaries by completed volumes.
func (a *ConsolidatorAgent) Consolidate(ctx context.Context, bookDir string) (*ConsolidationResult, error) {
	storyDir := filepath.Join(bookDir, "story")
	summariesPath := filepath.Join(storyDir, "chapter_summaries.md")
	outlinePath := filepath.Join(storyDir, "volume_outline.md")
	volumeSummariesPath := filepath.Join(storyDir, "volume_summaries.md")

	summariesRaw := readOptionalFile(summariesPath)
	outlineRaw := readOptionalFile(outlinePath)
	if strings.TrimSpace(summariesRaw) == "" || strings.TrimSpace(outlineRaw) == "" {
		return &ConsolidationResult{}, nil
	}

	volumeBoundaries := a.parseVolumeBoundaries(outlineRaw)
	if len(volumeBoundaries) == 0 {
		return &ConsolidationResult{}, nil
	}

	header, rows := a.parseSummaryTable(summariesRaw)
	if len(rows) == 0 {
		return &ConsolidationResult{}, nil
	}

	maxChapter := 0
	for _, row := range rows {
		if row.Chapter > maxChapter {
			maxChapter = row.Chapter
		}
	}

	completedVolumes := []struct {
		consolidationVolumeBoundary
		Rows []consolidationSummaryRow
	}{}
	currentVolumeRows := []consolidationSummaryRow{}
	for _, vol := range volumeBoundaries {
		volRows := []consolidationSummaryRow{}
		for _, row := range rows {
			if row.Chapter >= vol.StartCh && row.Chapter <= vol.EndCh {
				volRows = append(volRows, row)
			}
		}
		if vol.EndCh <= maxChapter && len(volRows) > 0 {
			completedVolumes = append(completedVolumes, struct {
				consolidationVolumeBoundary
				Rows []consolidationSummaryRow
			}{consolidationVolumeBoundary: vol, Rows: volRows})
		} else {
			currentVolumeRows = append(currentVolumeRows, volRows...)
		}
	}

	covered := map[int]struct{}{}
	for _, vol := range volumeBoundaries {
		for ch := vol.StartCh; ch <= vol.EndCh; ch++ {
			covered[ch] = struct{}{}
		}
	}
	for _, row := range rows {
		if _, ok := covered[row.Chapter]; !ok {
			currentVolumeRows = append(currentVolumeRows, row)
		}
	}

	if len(completedVolumes) == 0 {
		return &ConsolidationResult{
			RetainedChapters: len(currentVolumeRows),
		}, nil
	}

	existingVolumeSummaries := strings.TrimSpace(readOptionalFile(volumeSummariesPath))
	newSummaries := []string{}
	if existingVolumeSummaries != "" {
		newSummaries = append(newSummaries, existingVolumeSummaries)
	} else {
		newSummaries = append(newSummaries, "# Volume Summaries")
	}

	for _, vol := range completedVolumes {
		volRows := []string{}
		for _, row := range vol.Rows {
			volRows = append(volRows, row.Raw)
		}
		response, err := a.Chat(ctx, []llm.LLMMessage{
			{
				Role:    "system",
				Content: "You are a narrative summarizer. Compress chapter-by-chapter summaries into one coherent paragraph (max 500 words), preserving names and key plot points. Use the same language as input.",
			},
			{
				Role: "user",
				Content: fmt.Sprintf("Volume: %s (Chapters %d-%d)\n\nChapter summaries:\n%s\n%s",
					vol.Name, vol.StartCh, vol.EndCh, header, strings.Join(volRows, "\n")),
			},
		}, &llm.ChatOptions{Temperature: 0.3, MaxTokens: 1024})
		if err != nil {
			return nil, err
		}

		newSummaries = append(newSummaries, fmt.Sprintf("\n## %s (Ch.%d-%d)\n\n%s",
			vol.Name, vol.StartCh, vol.EndCh, strings.TrimSpace(response.Content)))
	}

	finalVolumeSummaries := strings.Join(newSummaries, "\n")
	if err := os.WriteFile(volumeSummariesPath, []byte(finalVolumeSummaries), 0644); err != nil {
		return nil, err
	}

	archiveDir := filepath.Join(storyDir, "summaries_archive")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return nil, err
	}
	for _, vol := range completedVolumes {
		lines := []string{fmt.Sprintf("# %s", vol.Name), "", header}
		for _, row := range vol.Rows {
			lines = append(lines, row.Raw)
		}
		archivePath := filepath.Join(archiveDir, fmt.Sprintf("vol_%d-%d.md", vol.StartCh, vol.EndCh))
		if err := os.WriteFile(archivePath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
			return nil, err
		}
	}

	sort.Slice(currentVolumeRows, func(i, j int) bool {
		return currentVolumeRows[i].Chapter < currentVolumeRows[j].Chapter
	})
	retainedLines := []string{header}
	for _, row := range currentVolumeRows {
		retainedLines = append(retainedLines, row.Raw)
	}
	retained := strings.Join(retainedLines, "\n") + "\n"
	if err := os.WriteFile(summariesPath, []byte(retained), 0644); err != nil {
		return nil, err
	}

	return &ConsolidationResult{
		VolumeSummaries:  finalVolumeSummaries,
		ArchivedVolumes:  len(completedVolumes),
		RetainedChapters: len(currentVolumeRows),
	}, nil
}

func (a *ConsolidatorAgent) parseVolumeBoundaries(outline string) []consolidationVolumeBoundary {
	volumes := []consolidationVolumeBoundary{}
	lines := strings.Split(outline, "\n")
	volumeHeader := regexp.MustCompile(`(?i)^(?:第[一二三四五六七八九十百千万零\d]+卷|volume\s+\d+)`)
	rangePattern := regexp.MustCompile(`(?:\(|（)\s*(?:第|Chapters?\s+)?(\d+)\s*[-—~～至到]\s*(\d+)\s*(?:章)?\s*(?:\)|）)`)

	for _, rawLine := range lines {
		line := strings.TrimSpace(strings.TrimLeft(rawLine, "# "))
		if !volumeHeader.MatchString(line) {
			continue
		}
		rangeMatch := rangePattern.FindStringSubmatch(line)
		if len(rangeMatch) < 3 {
			continue
		}
		startCh, _ := strconv.Atoi(rangeMatch[1])
		endCh, _ := strconv.Atoi(rangeMatch[2])
		if startCh <= 0 || endCh <= 0 {
			continue
		}
		loc := rangePattern.FindStringIndex(line)
		name := line
		if loc != nil {
			name = strings.TrimSpace(strings.TrimRight(line[:loc[0]], "(（"))
		}
		if name == "" {
			continue
		}
		volumes = append(volumes, consolidationVolumeBoundary{
			Name:    name,
			StartCh: startCh,
			EndCh:   endCh,
		})
	}
	return volumes
}

func (a *ConsolidatorAgent) parseSummaryTable(raw string) (string, []consolidationSummaryRow) {
	lines := strings.Split(raw, "\n")
	headerLines := []string{}
	dataLines := []string{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") {
			continue
		}
		if strings.Contains(trimmed, "Chapter") || strings.Contains(trimmed, "章节") || strings.Contains(trimmed, "---") {
			headerLines = append(headerLines, trimmed)
			continue
		}
		dataLines = append(dataLines, trimmed)
	}

	rows := []consolidationSummaryRow{}
	chapterMatcher := regexp.MustCompile(`^\|\s*(\d+)\s*\|`)
	for _, line := range dataLines {
		match := chapterMatcher.FindStringSubmatch(line)
		if len(match) < 2 {
			continue
		}
		chapter, _ := strconv.Atoi(match[1])
		if chapter <= 0 {
			continue
		}
		rows = append(rows, consolidationSummaryRow{
			Chapter: chapter,
			Raw:     line,
		})
	}

	return strings.Join(headerLines, "\n"), rows
}

func readOptionalFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}
