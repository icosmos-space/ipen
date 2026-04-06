package models

import (
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// ProtagonistRules 表示protagonist rules。
type ProtagonistRules struct {
	Name                  string   `json:"name"`
	PersonalityLock       []string `json:"personalityLock"`
	BehavioralConstraints []string `json:"behavioralConstraints"`
}

// GenreLockRules 表示genre lock rules。
type GenreLockRules struct {
	Primary   string   `json:"primary"`
	Forbidden []string `json:"forbidden"`
}

// NumericalOverrides 表示numerical system overrides。
type NumericalOverrides struct {
	HardCap       *string  `json:"hardCap,omitempty"` // can be number or string
	ResourceTypes []string `json:"resourceTypes"`
}

// EraConstraints 表示era constraints。
type EraConstraints struct {
	Enabled bool    `json:"enabled"`
	Period  *string `json:"period,omitempty"`
	Region  *string `json:"region,omitempty"`
}

// BookRules 表示the rules for a book。
type BookRules struct {
	Version                   string              `json:"version"`
	Protagonist               *ProtagonistRules   `json:"protagonist,omitempty"`
	GenreLock                 *GenreLockRules     `json:"genreLock,omitempty"`
	NumericalSystemOverrides  *NumericalOverrides `json:"numericalSystemOverrides,omitempty"`
	EraConstraints            *EraConstraints     `json:"eraConstraints,omitempty"`
	Prohibitions              []string            `json:"prohibitions"`
	ChapterTypesOverride      []string            `json:"chapterTypesOverride"`
	FatigueWordsOverride      []string            `json:"fatigueWordsOverride"`
	AdditionalAuditDimensions []any               `json:"additionalAuditDimensions"` // number or string
	EnableFullCastTracking    bool                `json:"enableFullCastTracking"`
	FanficMode                *FanficMode         `json:"fanficMode,omitempty" validate:"oneof=正典延续 架空世界 性格重塑 CP向"`
	AllowedDeviations         []string            `json:"allowedDeviations"`
}

func (bk *BookRules) Validate() error {
	return validate.Struct(bk)
}

// ParsedBookRules 表示parsed book rules with body。
type ParsedBookRules struct {
	Rules BookRules `json:"rules"`
	Body  string    `json:"body"`
}

// DefaultBookRules 返回default book rules。
func DefaultBookRules() BookRules {
	return BookRules{
		Version: "1.0",
	}
}

func ParseBookRules(raw string) (ParsedBookRules, error) {
	// Strip markdown code block wrappers if present (LLM often wraps output in ```md ... ```)
	stripped := raw
	re := regexp.MustCompile(`^` + "```" + `(?:md|markdown|yaml)?\s*\n`)
	stripped = re.ReplaceAllString(stripped, "")
	reEnd := regexp.MustCompile(`\n` + "```" + `\s*$`)
	stripped = reEnd.ReplaceAllString(stripped, "")

	// Try to find YAML frontmatter anywhere in the text (not just at the start)
	fmRe := regexp.MustCompile(`---\s*\n([\s\S]*?)\n---\s*\n?([\s\S]*)$`)
	fmMatch := fmRe.FindStringSubmatch(stripped)

	if fmMatch != nil {
		frontmatter := fmMatch[1]
		body := strings.TrimSpace(fmMatch[2])

		var rules BookRules
		if err := yaml.Unmarshal([]byte(frontmatter), &rules); err == nil {
			// YAML parse succeeded
			return ParsedBookRules{
				Rules: rules,
				Body:  body,
			}, nil
		}
		// YAML parse failed — fall through to default
	}

	// No valid frontmatter found — return default rules with the raw content as body
	rules := DefaultBookRules()
	return ParsedBookRules{
		Rules: rules,
		Body:  strings.TrimSpace(stripped),
	}, nil
}
