package models

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
	FanficMode                *string             `json:"fanficMode,omitempty"`
	AllowedDeviations         []string            `json:"allowedDeviations"`
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
