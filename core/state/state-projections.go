package state

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/icosmos-space/ipen/core/models"
)

// RenderHooksProjection renders hooks state as markdown.
func RenderHooksProjection(state models.HooksState, language string) string {
	isEnglish := strings.EqualFold(language, "en")
	title := "# Pending Hooks"
	headers := []string{
		"| hook_id | start_chapter | type | status | last_advanced_chapter | expected_payoff | payoff_timing | notes |",
		"| --- | --- | --- | --- | --- | --- | --- | --- |",
	}
	if !isEnglish {
		title = "# Pending Hooks"
		headers = []string{
			"| hook_id | 起始章节 | 类型 | 状态 | 最近推进 | 预期回收 | 回收节奏 | 备注 |",
			"| --- | --- | --- | --- | --- | --- | --- | --- |",
		}
	}

	hooks := make([]models.HookRecord, len(state.Hooks))
	copy(hooks, state.Hooks)
	sort.Slice(hooks, func(i, j int) bool {
		if hooks[i].StartChapter != hooks[j].StartChapter {
			return hooks[i].StartChapter < hooks[j].StartChapter
		}
		if hooks[i].LastAdvancedChapter != hooks[j].LastAdvancedChapter {
			return hooks[i].LastAdvancedChapter < hooks[j].LastAdvancedChapter
		}
		return hooks[i].HookID < hooks[j].HookID
	})

	rows := make([]string, 0, len(hooks))
	for _, hook := range hooks {
		timing := localizeHookTiming(hook.PayoffTiming, isEnglish)
		cells := []string{
			escapeTableCell(hook.HookID),
			escapeTableCell(strconv.Itoa(hook.StartChapter)),
			escapeTableCell(hook.Type),
			escapeTableCell(string(hook.Status)),
			escapeTableCell(strconv.Itoa(hook.LastAdvancedChapter)),
			escapeTableCell(hook.ExpectedPayoff),
			escapeTableCell(timing),
			escapeTableCell(hook.Notes),
		}
		rows = append(rows, "| "+strings.Join(cells, " | ")+" |")
	}

	if len(rows) == 0 {
		rows = append(rows, "| - | - | - | - | - | - | - | - |")
	}

	return strings.Join(append([]string{title, ""}, append(headers, append(rows, "")...)...), "\n")
}

// RenderChapterSummariesProjection renders chapter summaries as markdown.
func RenderChapterSummariesProjection(state models.ChapterSummariesState, language string) string {
	isEnglish := strings.EqualFold(language, "en")
	title := "# Chapter Summaries"
	headers := []string{
		"| Chapter | Title | Characters | Key Events | State Changes | Hook Activity | Mood | Chapter Type |",
		"| --- | --- | --- | --- | --- | --- | --- | --- |",
	}
	if !isEnglish {
		title = "# Chapter Summaries"
		headers = []string{
			"| 章节 | 标题 | 出场人物 | 关键事件 | 状态变化 | 伏笔动态 | 情绪基调 | 章节类型 |",
			"| --- | --- | --- | --- | --- | --- | --- | --- |",
		}
	}

	rows := make([]models.ChapterSummaryRow, len(state.Rows))
	copy(rows, state.Rows)
	sort.Slice(rows, func(i, j int) bool { return rows[i].Chapter < rows[j].Chapter })

	result := make([]string, 0, len(rows))
	for _, row := range rows {
		cells := []string{
			escapeTableCell(strconv.Itoa(row.Chapter)),
			escapeTableCell(row.Title),
			escapeTableCell(row.Characters),
			escapeTableCell(row.Events),
			escapeTableCell(row.StateChanges),
			escapeTableCell(row.HookActivity),
			escapeTableCell(row.Mood),
			escapeTableCell(row.ChapterType),
		}
		result = append(result, "| "+strings.Join(cells, " | ")+" |")
	}

	if len(result) == 0 {
		result = append(result, "| - | - | - | - | - | - | - | - |")
	}

	return strings.Join(append([]string{title, ""}, append(headers, append(result, "")...)...), "\n")
}

// RenderCurrentStateProjection renders current state facts as markdown.
func RenderCurrentStateProjection(state models.CurrentStateState, language string) string {
	isEnglish := strings.EqualFold(language, "en")
	layout := struct {
		Title           string
		TableHeader     string
		ChapterLabel    string
		Placeholder     string
		AdditionalTitle string
		Slots           []slotConfig
	}{
		Title:           "# Current State",
		TableHeader:     "| Field | Value |",
		ChapterLabel:    "Current Chapter",
		Placeholder:     "(not set)",
		AdditionalTitle: "## Additional State",
		Slots: []slotConfig{
			{Label: "Current Location", Aliases: []string{"Current Location", "当前位置"}},
			{Label: "Protagonist State", Aliases: []string{"Protagonist State", "主角状态"}},
			{Label: "Current Goal", Aliases: []string{"Current Goal", "当前目标"}},
			{Label: "Current Constraint", Aliases: []string{"Current Constraint", "当前限制"}},
			{Label: "Current Alliances", Aliases: []string{"Current Alliances", "Current Relationships", "当前敌我", "当前关系"}},
			{Label: "Current Conflict", Aliases: []string{"Current Conflict", "当前冲突"}},
		},
	}
	if !isEnglish {
		layout.Title = "# 当前状态"
		layout.TableHeader = "| 字段 | 值 |"
		layout.ChapterLabel = "当前章节"
		layout.Placeholder = "(未设定)"
		layout.AdditionalTitle = "## 其他状态"
		layout.Slots = []slotConfig{
			{Label: "当前位置", Aliases: []string{"当前位置", "Current Location"}},
			{Label: "主角状态", Aliases: []string{"主角状态", "Protagonist State"}},
			{Label: "当前目标", Aliases: []string{"当前目标", "Current Goal"}},
			{Label: "当前限制", Aliases: []string{"当前限制", "Current Constraint"}},
			{Label: "当前敌我", Aliases: []string{"当前敌我", "当前关系", "Current Alliances", "Current Relationships"}},
			{Label: "当前冲突", Aliases: []string{"当前冲突", "Current Conflict"}},
		}
	}

	lines := []string{
		layout.Title,
		"",
		layout.TableHeader,
		"| --- | --- |",
		fmt.Sprintf("| %s | %d |", layout.ChapterLabel, state.Chapter),
	}

	knownPredicates := map[string]struct{}{}
	for _, slot := range layout.Slots {
		for _, alias := range slot.Aliases {
			knownPredicates[normalizePredicate(alias)] = struct{}{}
		}
		value := findFactValue(state, slot.Aliases)
		if strings.TrimSpace(value) == "" {
			value = layout.Placeholder
		}
		lines = append(lines, fmt.Sprintf("| %s | %s |", slot.Label, escapeTableCell(value)))
	}

	additional := make([]models.CurrentStateFact, 0)
	for _, fact := range state.Facts {
		if _, ok := knownPredicates[normalizePredicate(fact.Predicate)]; ok {
			continue
		}
		additional = append(additional, fact)
	}

	sort.Slice(additional, func(i, j int) bool {
		left := strings.ToLower(strings.TrimSpace(additional[i].Predicate))
		right := strings.ToLower(strings.TrimSpace(additional[j].Predicate))
		leftNote, leftIdx := parseNoteIndex(left)
		rightNote, rightIdx := parseNoteIndex(right)
		if leftNote && rightNote {
			return leftIdx < rightIdx
		}
		if leftNote {
			return true
		}
		if rightNote {
			return false
		}
		return left < right
	})

	if len(additional) == 0 {
		return strings.Join(append(lines, ""), "\n")
	}

	lines = append(lines, "", layout.AdditionalTitle)
	for _, fact := range additional {
		predicate := strings.TrimSpace(fact.Predicate)
		if isNotePredicate(predicate) {
			lines = append(lines, "- "+strings.TrimSpace(fact.Object))
			continue
		}
		lines = append(lines, "- "+predicate+": "+strings.TrimSpace(fact.Object))
	}
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

type slotConfig struct {
	Label   string
	Aliases []string
}

func findFactValue(state models.CurrentStateState, aliases []string) string {
	aliasSet := map[string]struct{}{}
	for _, alias := range aliases {
		aliasSet[normalizePredicate(alias)] = struct{}{}
	}
	for _, fact := range state.Facts {
		if _, ok := aliasSet[normalizePredicate(fact.Predicate)]; ok {
			return fact.Object
		}
	}
	return ""
}

func normalizePredicate(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func parseNoteIndex(value string) (bool, int) {
	if !strings.HasPrefix(value, "note_") {
		return false, 0
	}
	index, err := strconv.Atoi(strings.TrimPrefix(value, "note_"))
	if err != nil {
		return true, 0
	}
	return true, index
}

func isNotePredicate(value string) bool {
	ok, _ := parseNoteIndex(strings.ToLower(strings.TrimSpace(value)))
	return ok
}

func localizeHookTiming(timing *models.HookPayoffTiming, isEnglish bool) string {
	if timing == nil {
		if isEnglish {
			return "near-term"
		}
		return "近期"
	}
	if isEnglish {
		return string(*timing)
	}
	switch *timing {
	case models.TimingImmediate:
		return "立即"
	case models.TimingNearTerm:
		return "近期"
	case models.TimingMidArc:
		return "中程"
	case models.TimingSlowBurn:
		return "慢烧"
	case models.TimingEndgame:
		return "终局"
	default:
		return string(*timing)
	}
}

func escapeTableCell(value string) string {
	return strings.TrimSpace(strings.ReplaceAll(value, "|", `\\|`))
}
