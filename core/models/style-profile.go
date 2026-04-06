package models

// StyleProfile 表示风格指纹配置文件
type StyleProfile struct {
	// 平均句子长度
	AvgSentenceLength float64 `json:"avgSentenceLength"`
	// 句子长度的标准差
	SentenceLengthStdDev float64 `json:"sentenceLengthStdDev"`
	// 平均段落长度
	AvgParagraphLength float64 `json:"avgParagraphLength"`
	// 段落长度的范围
	ParagraphLengthRange LengthRange `json:"paragraphLengthRange"`
	// 词汇多样性
	VocabularyDiversity float64 `json:"vocabularyDiversity"` // TTR (Type-Token Ratio)
	// 顶部模式
	TopPatterns []string `json:"topPatterns"`
	// 口辞特征
	RhetoricalFeatures []string `json:"rhetoricalFeatures"`
	// 来源名称
	SourceName string `json:"sourceName,omitempty"`
	// 分析时间
	AnalyzedAt string `json:"analyzedAt,omitempty"`
}

// LengthRange 表示长度范围
type LengthRange struct {
	// 最小值
	Min float64 `json:"min"`
	// 最大值
	Max float64 `json:"max"`
}
