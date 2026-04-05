package models

// GenreProfile 表示a genre profile。
type GenreProfile struct {
	Name              string   `json:"name"`
	ID                string   `json:"id"`
	Language          string   `json:"language"` // "zh" or "en"
	ChapterTypes      []string `json:"chapterTypes"`
	FatigueWords      []string `json:"fatigueWords"`
	NumericalSystem   bool     `json:"numericalSystem"`
	PowerScaling      bool     `json:"powerScaling"`
	EraResearch       bool     `json:"eraResearch"`
	PacingRule        string   `json:"pacingRule"`
	SatisfactionTypes []string `json:"satisfactionTypes"`
	AuditDimensions   []int    `json:"auditDimensions"`
}

// ParsedGenreProfile 表示a parsed genre profile with body。
type ParsedGenreProfile struct {
	Profile GenreProfile `json:"profile"`
	Body    string       `json:"body"`
}
