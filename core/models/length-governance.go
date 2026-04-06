package models

// LengthCountingMode 长度计数模式
type LengthCountingMode string

const (
	CountingModeZHChars LengthCountingMode = "中文字符"
	CountingModeENWords LengthCountingMode = "英文单词"
)

// LengthNormalizeMode 长度归一化模式
type LengthNormalizeMode string

const (
	NormalizeModeExpand   LengthNormalizeMode = "扩展"
	NormalizeModeCompress LengthNormalizeMode = "压缩"
	NormalizeModeNone     LengthNormalizeMode = "无"
)

// LengthSpec 长度规格
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
