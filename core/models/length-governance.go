package models

// LengthCountingMode 表示the mode for counting content length。
type LengthCountingMode string

const (
	CountingModeZHChars LengthCountingMode = "zh_chars"
	CountingModeENWords LengthCountingMode = "en_words"
)

// LengthNormalizeMode 表示the mode for normalizing length。
type LengthNormalizeMode string

const (
	NormalizeModeExpand   LengthNormalizeMode = "expand"
	NormalizeModeCompress LengthNormalizeMode = "compress"
	NormalizeModeNone     LengthNormalizeMode = "none"
)

// LengthSpec 表示the length specification。
type LengthSpec struct {
	Target        int                 `json:"target"`
	SoftMin       int                 `json:"softMin"`
	SoftMax       int                 `json:"softMax"`
	HardMin       int                 `json:"hardMin"`
	HardMax       int                 `json:"hardMax"`
	CountingMode  LengthCountingMode  `json:"countingMode"`
	NormalizeMode LengthNormalizeMode `json:"normalizeMode"`
}

// LengthTelemetry 表示length telemetry。
type LengthTelemetry struct {
	Target                   int                `json:"target"`
	SoftMin                  int                `json:"softMin"`
	SoftMax                  int                `json:"softMax"`
	HardMin                  int                `json:"hardMin"`
	HardMax                  int                `json:"hardMax"`
	CountingMode             LengthCountingMode `json:"countingMode"`
	WriterCount              int                `json:"writerCount"`
	PostWriterNormalizeCount int                `json:"postWriterNormalizeCount"`
	PostReviseCount          int                `json:"postReviseCount"`
	FinalCount               int                `json:"finalCount"`
	NormalizeApplied         bool               `json:"normalizeApplied"`
	LengthWarning            bool               `json:"lengthWarning"`
}

// LengthWarning 表示a length warning。
type LengthWarning struct {
	Chapter      int                `json:"chapter"`
	Target       int                `json:"target"`
	Actual       int                `json:"actual"`
	CountingMode LengthCountingMode `json:"countingMode"`
	Reason       string             `json:"reason"`
}
