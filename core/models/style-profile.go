package models

// StyleProfile 表示a style fingerprint profile。
type StyleProfile struct {
	AvgSentenceLength    float64     `json:"avgSentenceLength"`
	SentenceLengthStdDev float64     `json:"sentenceLengthStdDev"`
	AvgParagraphLength   float64     `json:"avgParagraphLength"`
	ParagraphLengthRange LengthRange `json:"paragraphLengthRange"`
	VocabularyDiversity  float64     `json:"vocabularyDiversity"` // TTR (Type-Token Ratio)
	TopPatterns          []string    `json:"topPatterns"`
	RhetoricalFeatures   []string    `json:"rhetoricalFeatures"`
	SourceName           string      `json:"sourceName,omitempty"`
	AnalyzedAt           string      `json:"analyzedAt,omitempty"`
}

// LengthRange 表示a length range。
type LengthRange struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}
