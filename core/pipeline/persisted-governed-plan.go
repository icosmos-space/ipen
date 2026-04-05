package pipeline

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/icosmos-space/ipen/core/agents"
	"github.com/icosmos-space/ipen/core/models"
)

// LoadPersistedPlan 加载a previously persisted planner intent markdown file。
func LoadPersistedPlan(bookDir string, chapterNumber int) (*agents.PlanChapterOutput, error) {
	runtimePath := filepath.Join(
		bookDir,
		"story",
		"runtime",
		"chapter-"+pad4(chapterNumber)+".intent.md",
	)

	intentMarkdownBytes, err := os.ReadFile(runtimePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	intentMarkdown := string(intentMarkdownBytes)

	sections := parseIntentSections(intentMarkdown)
	goal := readIntentScalar(sections, "Goal")
	if goal == "" || isInvalidPersistedIntentScalar(goal) {
		return nil, nil
	}

	outlineNode := readIntentScalar(sections, "Outline Node")
	if outlineNode != "" && outlineNode != "(not found)" && isInvalidPersistedIntentScalar(outlineNode) {
		return nil, nil
	}

	conflicts := make([]models.ChapterConflict, 0)
	for _, line := range readIntentList(sections, "Conflicts") {
		sep := strings.Index(line, ":")
		if sep < 0 {
			continue
		}
		t := strings.TrimSpace(line[:sep])
		resolution := strings.TrimSpace(line[sep+1:])
		if t == "" || resolution == "" {
			continue
		}
		conflicts = append(conflicts, models.ChapterConflict{
			Type:       t,
			Resolution: resolution,
		})
	}

	intent := models.ChapterIntent{
		Chapter:       chapterNumber,
		Goal:          goal,
		MustKeep:      readIntentList(sections, "Must Keep"),
		MustAvoid:     readIntentList(sections, "Must Avoid"),
		StyleEmphasis: readIntentList(sections, "Style Emphasis"),
		Conflicts:     conflicts,
		HookAgenda: models.HookAgenda{
			PressureMap:          []models.HookPressure{},
			MustAdvance:          []string{},
			EligibleResolve:      []string{},
			StaleDebt:            []string{},
			AvoidNewHookFamilies: []string{},
		},
	}
	if outlineNode != "" && outlineNode != "(not found)" {
		intent.OutlineNode = &outlineNode
	}

	return &agents.PlanChapterOutput{
		Intent:         intent,
		IntentMarkdown: intentMarkdown,
		PlannerInputs:  []string{runtimePath},
		RuntimePath:    runtimePath,
	}, nil
}

// RelativeToBookDir converts an absolute path to a book-relative path.
func RelativeToBookDir(bookDir string, absolutePath string) string {
	rel, err := filepath.Rel(bookDir, absolutePath)
	if err != nil {
		return filepath.ToSlash(absolutePath)
	}
	return filepath.ToSlash(rel)
}

func parseIntentSections(markdown string) map[string][]string {
	sections := map[string][]string{}
	current := ""
	for _, line := range strings.Split(markdown, "\n") {
		if strings.HasPrefix(line, "## ") {
			current = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			if _, exists := sections[current]; !exists {
				sections[current] = []string{}
			}
			continue
		}
		if current == "" {
			continue
		}
		sections[current] = append(sections[current], line)
	}
	return sections
}

func readIntentScalar(sections map[string][]string, name string) string {
	lines := sections[name]
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if trimmed == "- none" {
			return ""
		}
		return trimmed
	}
	return ""
}

func readIntentList(sections map[string][]string, name string) []string {
	lines := sections[name]
	result := make([]string, 0)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "-") || trimmed == "- none" {
			continue
		}
		item := strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func isInvalidPersistedIntentScalar(value string) bool {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return true
	}
	if regexp.MustCompile(`^[*_` + "`" + `~:：\-.]+$`).MatchString(normalized) {
		return true
	}
	if regexp.MustCompile(`(?i)^\((describe|briefly describe|write)\b[\s\S]*\)$`).MatchString(normalized) {
		return true
	}
	if regexp.MustCompile(`^\((描述|填写|写下)[\s\S]*\)$`).MatchString(normalized) {
		return true
	}
	return false
}

func pad4(v int) string {
	s := strconv.Itoa(v)
	if len(s) >= 4 {
		return s
	}
	return strings.Repeat("0", 4-len(s)) + s
}
