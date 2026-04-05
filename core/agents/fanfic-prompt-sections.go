package agents

import (
	"regexp"
	"strings"

	"github.com/icosmos-space/ipen/core/models"
)

var fanficModePreambles = map[models.FanficMode]string{
	models.FanficModeCanon: "You are writing canon-faithful fan fiction. Keep character voice, world rules, and timeline strict.",
	models.FanficModeAU:    "You are writing AU fan fiction. World rules may diverge, but character recognizability must remain high.",
	models.FanficModeOOC:   "You are writing OOC fan fiction. Personality deviation is allowed only when strongly motivated.",
	models.FanficModeCP:    "You are writing CP fan fiction. Relationship progression and chemistry are the primary axis.",
}

var fanficModeChecks = map[models.FanficMode]string{
	models.FanficModeCanon: "- Canon compliance check\n- Information boundary check",
	models.FanficModeAU:    "- AU divergence checklist\n- Character recognizability check",
	models.FanficModeOOC:   "- OOC deviation log\n- Voice preservation check",
	models.FanficModeCP:    "- CP interaction progression check\n- Interaction quality check",
}

// BuildFanficCanonSection 构建canon reference prompt section。
func BuildFanficCanonSection(fanficCanon string, mode models.FanficMode) string {
	preamble := fanficModePreambles[mode]
	if strings.TrimSpace(preamble) == "" {
		preamble = fanficModePreambles[models.FanficModeCanon]
	}
	return strings.TrimSpace(strings.Join([]string{
		"## Fanfic Canon Reference",
		"",
		preamble,
		"",
		"Use the following canon sheet as source of truth:",
		fanficCanon,
	}, "\n"))
}

// BuildCharacterVoiceProfiles 提取rough character voice hints from markdown table blocks。
func BuildCharacterVoiceProfiles(fanficCanon string) string {
	table := regexp.MustCompile(`(?is)##\s*Character\s*Profiles[\s\S]*?(\|[^\n]+\|\n\|[-|\s]+\|\n(?:\|[^\n]+\|\n?)*)`).FindStringSubmatch(fanficCanon)
	if len(table) < 2 {
		return ""
	}

	rows := strings.Split(strings.TrimSpace(table[1]), "\n")
	profiles := []string{}
	for _, row := range rows {
		trimmed := strings.TrimSpace(row)
		if !strings.HasPrefix(trimmed, "|") || strings.Contains(trimmed, "---") {
			continue
		}
		cells := splitMarkdownRow(trimmed)
		if len(cells) < 4 {
			continue
		}
		if strings.EqualFold(cells[0], "character") || strings.EqualFold(cells[0], "角色") {
			continue
		}
		name := strings.TrimSpace(cells[0])
		catchphrase := getCell(cells, 3)
		speakingStyle := getCell(cells, 4)
		behavior := getCell(cells, 5)
		if name == "" {
			continue
		}
		parts := []string{"### " + name}
		if catchphrase != "" {
			parts = append(parts, "- Catchphrase: "+catchphrase)
		}
		if speakingStyle != "" {
			parts = append(parts, "- Speaking style: "+speakingStyle)
		}
		if behavior != "" {
			parts = append(parts, "- Behavior signature: "+behavior)
		}
		profiles = append(profiles, strings.Join(parts, "\n"))
	}

	if len(profiles) == 0 {
		return ""
	}

	return strings.Join(append([]string{"## Character Voice References", ""}, profiles...), "\n\n")
}

// BuildFanficModeInstructions 构建mode-specific pre-write checks。
func BuildFanficModeInstructions(mode models.FanficMode, allowedDeviations []string) string {
	checks := fanficModeChecks[mode]
	if strings.TrimSpace(checks) == "" {
		checks = fanficModeChecks[models.FanficModeCanon]
	}
	deviationBlock := ""
	if len(allowedDeviations) > 0 {
		lines := make([]string, 0, len(allowedDeviations))
		for _, item := range allowedDeviations {
			if strings.TrimSpace(item) == "" {
				continue
			}
			lines = append(lines, "- "+strings.TrimSpace(item))
		}
		if len(lines) > 0 {
			deviationBlock = "\nAllowed Deviations:\n" + strings.Join(lines, "\n")
		}
	}
	return strings.TrimSpace("## Fanfic Self-Check\n\n" + checks + deviationBlock)
}

func splitMarkdownRow(row string) []string {
	parts := strings.Split(strings.Trim(row, "|"), "|")
	cells := make([]string, 0, len(parts))
	for _, part := range parts {
		cells = append(cells, strings.TrimSpace(part))
	}
	return cells
}

func getCell(cells []string, idx int) string {
	if idx >= 0 && idx < len(cells) {
		return strings.TrimSpace(cells[idx])
	}
	return ""
}
