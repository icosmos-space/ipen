package models

// GenreProfile 表示a genre profile。
type GenreProfile struct {
	// 名称
	Name string `json:"name"`
	// ID
	ID string `json:"id"`
	// 语言
	Language string `json:"language"` // "zh" or "en"
	// 章节类型
	ChapterTypes []string `json:"chapterTypes"`
	FatigueWords []string `json:"fatigueWords"`
	// 数字系统
	// 是否使用数字系统
	NumericalSystem   bool     `json:"numericalSystem"`
	PowerScaling      bool     `json:"powerScaling"`
	EraResearch       bool     `json:"eraResearch"`
	PacingRule        string   `json:"pacingRule"`
	SatisfactionTypes []string `json:"satisfactionTypes"`
	//
	AuditDimensions []int `json:"auditDimensions"`
}

// ParsedGenreProfile 表示a parsed genre profile with body。
type ParsedGenreProfile struct {
	Profile GenreProfile `json:"profile"`
	Body    string       `json:"body"`
}
