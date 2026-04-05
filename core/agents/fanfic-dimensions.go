package agents

import (
	"fmt"

	"github.com/icosmos-space/ipen/core/models"
)

// FanficDimensionConfig controls fanfic-specific audit dimension behavior.
type FanficDimensionConfig struct {
	ActiveIDs         []int          `json:"activeIds"`
	SeverityOverrides map[int]string `json:"severityOverrides"`
	DeactivatedIDs    []int          `json:"deactivatedIds"`
	Notes             map[int]string `json:"notes"`
}

// FanficDimension 表示a fanfic audit dimension。
type FanficDimension struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	BaseNote string `json:"baseNote"`
}

// FANFIC_DIMENSIONS are fanfic-specific dimensions mapped from TS implementation.
var FANFIC_DIMENSIONS = []FanficDimension{
	{ID: 34, Name: "character-faithfulness", BaseNote: "Check voice, behavior, and motivations against fanfic canon."},
	{ID: 35, Name: "world-rule-consistency", BaseNote: "Check world-rule consistency (geography, power system, factions)."},
	{ID: 36, Name: "relationship-dynamics", BaseNote: "Check relationship evolution and interaction plausibility."},
	{ID: 37, Name: "canon-event-consistency", BaseNote: "Check conflicts with canon timeline and key events."},
}

var fanficSeverityMap = map[models.FanficMode]map[int]string{
	models.FanficModeCanon: {34: "critical", 35: "critical", 36: "warning", 37: "critical"},
	models.FanficModeAU:    {34: "critical", 35: "info", 36: "warning", 37: "info"},
	models.FanficModeOOC:   {34: "info", 35: "warning", 36: "warning", 37: "info"},
	models.FanficModeCP:    {34: "warning", 35: "warning", 36: "critical", 37: "info"},
}

var fanficSpinoffDims = []int{28, 29, 30, 31}

const fanficOOCDimension = 1

// GetFanficDimensionConfig 构建dimension configuration for fanfic mode。
func GetFanficDimensionConfig(mode models.FanficMode, _allowedDeviations []string) FanficDimensionConfig {
	severityMap, ok := fanficSeverityMap[mode]
	if !ok {
		severityMap = fanficSeverityMap[models.FanficModeCanon]
	}

	overrides := make(map[int]string, len(FANFIC_DIMENSIONS)+1)
	notes := make(map[int]string, len(FANFIC_DIMENSIONS)+1)
	active := make([]int, 0, len(FANFIC_DIMENSIONS))

	for _, dim := range FANFIC_DIMENSIONS {
		severity := severityMap[dim.ID]
		if severity == "" {
			severity = "warning"
		}
		overrides[dim.ID] = severity
		notes[dim.ID] = fmt.Sprintf("%s (%s)", dim.BaseNote, severity)
		active = append(active, dim.ID)
	}

	if mode == models.FanficModeOOC {
		overrides[fanficOOCDimension] = "info"
		notes[fanficOOCDimension] = "OOC mode: personality deviation is allowed with explicit context and motivation. See fanfic_canon.md for baseline profiles."
	}

	if mode == models.FanficModeCanon {
		notes[fanficOOCDimension] = "Canon mode: keep personalities highly faithful to source characterization in fanfic_canon.md."
	}

	return FanficDimensionConfig{
		ActiveIDs:         active,
		SeverityOverrides: overrides,
		DeactivatedIDs:    append([]int{}, fanficSpinoffDims...),
		Notes:             notes,
	}
}
