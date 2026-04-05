package agents

import (
	"regexp"
	"strings"

	"github.com/icosmos-space/ipen/core/models"
)

// SettlementOutput 表示parsed settler sections。
type SettlementOutput struct {
	PostSettlement         string `json:"postSettlement"`
	UpdatedState           string `json:"updatedState"`
	UpdatedLedger          string `json:"updatedLedger"`
	UpdatedHooks           string `json:"updatedHooks"`
	ChapterSummary         string `json:"chapterSummary"`
	UpdatedSubplots        string `json:"updatedSubplots"`
	UpdatedEmotionalArcs   string `json:"updatedEmotionalArcs"`
	UpdatedCharacterMatrix string `json:"updatedCharacterMatrix"`
}

// ParseSettlementOutput 解析settle output sections。
func ParseSettlementOutput(content string, genreProfile *models.GenreProfile) SettlementOutput {
	extract := func(tag string) string {
		re := regexp.MustCompile(`(?s)===\s*` + regexp.QuoteMeta(tag) + `\s*===\s*(.*?)(?:(?:\n===\s*[A-Z_]+\s*===)|$)`)
		match := re.FindStringSubmatch(content)
		if len(match) < 2 {
			return ""
		}
		return strings.TrimSpace(match[1])
	}

	ledger := ""
	if genreProfile != nil && genreProfile.NumericalSystem {
		ledger = extract("UPDATED_LEDGER")
		if ledger == "" {
			ledger = "(ledger not updated)"
		}
	}

	updatedState := extract("UPDATED_STATE")
	if updatedState == "" {
		updatedState = "(state not updated)"
	}
	updatedHooks := extract("UPDATED_HOOKS")
	if updatedHooks == "" {
		updatedHooks = "(hooks not updated)"
	}

	return SettlementOutput{
		PostSettlement:         extract("POST_SETTLEMENT"),
		UpdatedState:           updatedState,
		UpdatedLedger:          ledger,
		UpdatedHooks:           updatedHooks,
		ChapterSummary:         extract("CHAPTER_SUMMARY"),
		UpdatedSubplots:        extract("UPDATED_SUBPLOTS"),
		UpdatedEmotionalArcs:   extract("UPDATED_EMOTIONAL_ARCS"),
		UpdatedCharacterMatrix: extract("UPDATED_CHARACTER_MATRIX"),
	}
}
