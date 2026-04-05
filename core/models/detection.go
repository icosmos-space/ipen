package models

// DetectionHistoryEntry 表示a detection/rewrite event。
type DetectionHistoryEntry struct {
	ChapterNumber int     `json:"chapterNumber"`
	Timestamp     string  `json:"timestamp"`
	Provider      string  `json:"provider"`
	Score         float64 `json:"score"`
	Action        string  `json:"action"` // "detect" or "rewrite"
	Attempt       int     `json:"attempt"`
}

// DetectionStats 表示detection statistics。
type DetectionStats struct {
	TotalDetections   int                         `json:"totalDetections"`
	TotalRewrites     int                         `json:"totalRewrites"`
	AvgOriginalScore  float64                     `json:"avgOriginalScore"`
	AvgFinalScore     float64                     `json:"avgFinalScore"`
	AvgScoreReduction float64                     `json:"avgScoreReduction"`
	PassRate          float64                     `json:"passRate"`
	ChapterBreakdown  []ChapterDetectionBreakdown `json:"chapterBreakdown"`
}

// ChapterDetectionBreakdown 表示per-chapter detection breakdown。
type ChapterDetectionBreakdown struct {
	ChapterNumber   int     `json:"chapterNumber"`
	OriginalScore   float64 `json:"originalScore"`
	FinalScore      float64 `json:"finalScore"`
	RewriteAttempts int     `json:"rewriteAttempts"`
}
