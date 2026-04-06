/** A single detection/rewrite event recorded in detection_history.json. */
package models

// DetectionHistoryEntry 检测&重写事件内容
type DetectionHistoryEntry struct {
	// 章节
	ChapterNumber int `json:"chapterNumber"`
	// 时间戳
	Timestamp string `json:"timestamp"`
	// 提供方
	Provider string `json:"provider"`
	// 分数
	Score float64 `json:"score"`
	// 操作
	Action string `json:"action" validate:"oneof=detect rewrite"` // "detect" or "rewrite"
	// 尝试次数
	Attempt int `json:"attempt"`
}

// DetectionStats 检测&重写统计
type DetectionStats struct {
	// 总检测数
	TotalDetections int `json:"totalDetections"`
	// 总重写数
	TotalRewrites int `json:"totalRewrites"`
	// 平均原始分数
	AvgOriginalScore float64 `json:"avgOriginalScore"`
	// 平均最终分数
	AvgFinalScore float64 `json:"avgFinalScore"`
	// 平均分数减少量
	AvgScoreReduction float64 `json:"avgScoreReduction"`
	// 通过率
	PassRate float64 `json:"passRate"`
	// 每章检测&重写统计
	ChapterBreakdown []ChapterDetectionBreakdown `json:"chapterBreakdown"`
}

// ChapterDetectionBreakdown 每章检测&重写统计
type ChapterDetectionBreakdown struct {
	// 章节
	ChapterNumber int `json:"chapterNumber"`
	// 原始分数
	OriginalScore float64 `json:"originalScore"`
	// 最终分数
	FinalScore float64 `json:"finalScore"`
	// 重写尝试次数
	RewriteAttempts int `json:"rewriteAttempts"`
}
