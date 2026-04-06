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
	Target        int                 `json:"target"  validate:"required,min=1"`
	SoftMin       int                 `json:"softMin"  validate:"required,min=1"`
	SoftMax       int                 `json:"softMax"  validate:"required,min=1"`
	HardMin       int                 `json:"hardMin"  validate:"required,min=1"`
	HardMax       int                 `json:"hardMax"  validate:"required,min=1"`
	CountingMode  LengthCountingMode  `json:"countingMode"`
	NormalizeMode LengthNormalizeMode `json:"normalizeMode"`
}

// LengthTelemetry 表示length telemetry。
type LengthTelemetry struct {
	Target                   int                `json:"target"   validate:"required,min=1"`
	SoftMin                  int                `json:"softMin"  validate:"required,min=1"`
	SoftMax                  int                `json:"softMax"  validate:"required,min=1"`
	HardMin                  int                `json:"hardMin"  validate:"required,min=1"`
	HardMax                  int                `json:"hardMax"  validate:"required,min=1"`
	CountingMode             LengthCountingMode `json:"countingMode"`
	WriterCount              int                `json:"writerCount"  validate:"required,min=0"`
	PostWriterNormalizeCount int                `json:"postWriterNormalizeCount"  validate:"required,min=0"`
	PostReviseCount          int                `json:"postReviseCount"  validate:"required,min=0"`
	FinalCount               int                `json:"finalCount"  validate:"required,min=0"`
	NormalizeApplied         bool               `json:"normalizeApplied"`
	LengthWarning            bool               `json:"lengthWarning"`
}

// LengthWarning 长度警告
type LengthWarning struct {
	Chapter      int                `json:"chapter"  validate:"required,min=1"`
	Target       int                `json:"target"  validate:"required,min=1"`
	Actual       int                `json:"actual"  validate:"required,min=0"`
	CountingMode LengthCountingMode `json:"countingMode"`
	Reason       string             `json:"reason"  validate:"required"`
}

func (lw *LengthWarning) Validate() error {
	return validate.Struct(lw)
}
