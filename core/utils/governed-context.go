package utils

import (
	"strings"

	"github.com/icosmos-space/ipen/core/models"
)

// GovernedMemoryEvidenceBlocks 分组selected context blocks for governed prompts。
type GovernedMemoryEvidenceBlocks struct {
	HookDebtBlock        string
	HooksBlock           string
	SummariesBlock       string
	VolumeSummariesBlock string
	TitleHistoryBlock    string
	MoodTrailBlock       string
	CanonBlock           string
}

// BuildGovernedMemoryEvidenceBlocks renders selected-context evidence blocks by source type.
func BuildGovernedMemoryEvidenceBlocks(contextPackage models.ContextPackage, language string) GovernedMemoryEvidenceBlocks {
	if language == "" {
		language = "zh"
	}

	filterByPrefix := func(prefix string) []models.ContextSource {
		result := []models.ContextSource{}
		for _, entry := range contextPackage.SelectedContext {
			if strings.HasPrefix(entry.Source, prefix) {
				result = append(result, entry)
			}
		}
		return result
	}
	filterByExact := func(values ...string) []models.ContextSource {
		result := []models.ContextSource{}
		for _, entry := range contextPackage.SelectedContext {
			for _, value := range values {
				if entry.Source == value {
					result = append(result, entry)
					break
				}
			}
		}
		return result
	}

	hookEntries := filterByPrefix("story/pending_hooks.md#")
	hookDebtEntries := filterByPrefix("runtime/hook_debt#")
	summaryEntries := filterByPrefix("story/chapter_summaries.md#")
	volumeSummaryEntries := filterByPrefix("story/volume_summaries.md#")
	titleHistoryEntries := filterByExact("story/chapter_summaries.md#recent_titles")
	moodTrailEntries := filterByExact("story/chapter_summaries.md#recent_mood_type_trail")
	canonEntries := filterByExact("story/parent_canon.md", "story/fanfic_canon.md")

	blocks := GovernedMemoryEvidenceBlocks{}
	if len(hookDebtEntries) > 0 {
		blocks.HookDebtBlock = renderHookDebtBlock("Hook Debt Briefs", hookDebtEntries)
	}
	if len(hookEntries) > 0 {
		heading := "已选伏笔证据"
		if strings.EqualFold(language, "en") {
			heading = "Selected Hook Evidence"
		}
		blocks.HooksBlock = renderEvidenceBlock(heading, hookEntries)
	}
	if len(summaryEntries) > 0 {
		heading := "已选章节摘要证据"
		if strings.EqualFold(language, "en") {
			heading = "Selected Chapter Summary Evidence"
		}
		blocks.SummariesBlock = renderEvidenceBlock(heading, summaryEntries)
	}
	if len(volumeSummaryEntries) > 0 {
		heading := "已选卷级摘要证据"
		if strings.EqualFold(language, "en") {
			heading = "Selected Volume Summary Evidence"
		}
		blocks.VolumeSummariesBlock = renderEvidenceBlock(heading, volumeSummaryEntries)
	}
	if len(titleHistoryEntries) > 0 {
		heading := "近期标题历史"
		if strings.EqualFold(language, "en") {
			heading = "Recent Title History"
		}
		blocks.TitleHistoryBlock = renderEvidenceBlock(heading, titleHistoryEntries)
	}
	if len(moodTrailEntries) > 0 {
		heading := "近期情绪/章节类型轨迹"
		if strings.EqualFold(language, "en") {
			heading = "Recent Mood / Chapter Type Trail"
		}
		blocks.MoodTrailBlock = renderEvidenceBlock(heading, moodTrailEntries)
	}
	if len(canonEntries) > 0 {
		heading := "正典约束证据"
		if strings.EqualFold(language, "en") {
			heading = "Canon Evidence"
		}
		blocks.CanonBlock = renderEvidenceBlock(heading, canonEntries)
	}

	return blocks
}

func renderHookDebtBlock(heading string, entries []models.ContextSource) string {
	lines := []string{}
	for _, entry := range entries {
		line := entry.Reason
		if entry.Excerpt != nil && strings.TrimSpace(*entry.Excerpt) != "" {
			line = strings.TrimSpace(*entry.Excerpt)
		}
		lines = append(lines, "- "+line)
	}
	return "\n## " + heading + "\n" + strings.Join(lines, "\n") + "\n"
}

func renderEvidenceBlock(heading string, entries []models.ContextSource) string {
	lines := []string{}
	for _, entry := range entries {
		body := entry.Reason
		if entry.Excerpt != nil && strings.TrimSpace(*entry.Excerpt) != "" {
			body = strings.TrimSpace(*entry.Excerpt)
		}
		lines = append(lines, "- "+entry.Source+": "+body)
	}
	return "\n## " + heading + "\n" + strings.Join(lines, "\n") + "\n"
}
