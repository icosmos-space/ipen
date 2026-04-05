package agents

// DetectionHistoryEntry 表示a detection history entry。
type DetectionHistoryEntry struct {
	ChapterNumber int     `json:"chapterNumber"`
	Action        string  `json:"action"` // "detect" or "rewrite"
	Attempt       int     `json:"attempt"`
	Score         float64 `json:"score"`
	Timestamp     string  `json:"timestamp"`
}

// ChapterDetectionBreakdown 表示per-chapter detection breakdown。
type ChapterDetectionBreakdown struct {
	ChapterNumber   int     `json:"chapterNumber"`
	OriginalScore   float64 `json:"originalScore"`
	FinalScore      float64 `json:"finalScore"`
	RewriteAttempts int     `json:"rewriteAttempts"`
}

// DetectionStats 表示aggregated detection statistics。
type DetectionStats struct {
	TotalDetections   int                         `json:"totalDetections"`
	TotalRewrites     int                         `json:"totalRewrites"`
	AvgOriginalScore  float64                     `json:"avgOriginalScore"`
	AvgFinalScore     float64                     `json:"avgFinalScore"`
	AvgScoreReduction float64                     `json:"avgScoreReduction"`
	PassRate          float64                     `json:"passRate"`
	ChapterBreakdown  []ChapterDetectionBreakdown `json:"chapterBreakdown"`
}

// AnalyzeDetectionInsights 分析detection history and produces aggregated statistics。
func AnalyzeDetectionInsights(history []DetectionHistoryEntry) DetectionStats {
	if len(history) == 0 {
		return DetectionStats{
			TotalDetections:   0,
			TotalRewrites:     0,
			AvgOriginalScore:  0,
			AvgFinalScore:     0,
			AvgScoreReduction: 0,
			PassRate:          0,
			ChapterBreakdown:  []ChapterDetectionBreakdown{},
		}
	}

	detections := 0
	rewrites := 0
	for _, h := range history {
		if h.Action == "detect" {
			detections++
		}
		if h.Action == "rewrite" {
			rewrites++
		}
	}

	// Group by chapter
	chapterMap := make(map[int][]DetectionHistoryEntry)
	for _, entry := range history {
		chapterMap[entry.ChapterNumber] = append(chapterMap[entry.ChapterNumber], entry)
	}

	var chapterBreakdown []ChapterDetectionBreakdown
	totalOriginal := 0.0
	totalFinal := 0.0

	for chapterNumber, entries := range chapterMap {
		// Sort by attempt
		sorted := make([]DetectionHistoryEntry, len(entries))
		copy(sorted, entries)
		sortByAttempt(sorted)

		originalScore := 0.0
		finalScore := 0.0
		if len(sorted) > 0 {
			originalScore = sorted[0].Score
			finalScore = sorted[len(sorted)-1].Score
		}

		rewriteAttempts := 0
		for _, e := range sorted {
			if e.Action == "rewrite" {
				rewriteAttempts++
			}
		}

		chapterBreakdown = append(chapterBreakdown, ChapterDetectionBreakdown{
			ChapterNumber:   chapterNumber,
			OriginalScore:   originalScore,
			FinalScore:      finalScore,
			RewriteAttempts: rewriteAttempts,
		})
		totalOriginal += originalScore
		totalFinal += finalScore
	}

	// Sort chapter breakdown by chapter number
	sortChapterBreakdown(chapterBreakdown)

	chapterCount := len(chapterBreakdown)
	avgOriginalScore := 0.0
	avgFinalScore := 0.0
	if chapterCount > 0 {
		avgOriginalScore = totalOriginal / float64(chapterCount)
		avgFinalScore = totalFinal / float64(chapterCount)
	}

	// Pass rate = chapters where final score decreased (or no rewrite needed)
	passedChapters := 0
	for _, c := range chapterBreakdown {
		if c.FinalScore <= c.OriginalScore {
			passedChapters++
		}
	}

	passRate := 0.0
	if chapterCount > 0 {
		passRate = float64(passedChapters) / float64(chapterCount)
	}

	return DetectionStats{
		TotalDetections:   detections,
		TotalRewrites:     rewrites,
		AvgOriginalScore:  roundTo3(avgOriginalScore),
		AvgFinalScore:     roundTo3(avgFinalScore),
		AvgScoreReduction: roundTo3(avgOriginalScore - avgFinalScore),
		PassRate:          roundTo2(passRate),
		ChapterBreakdown:  chapterBreakdown,
	}
}

func sortByAttempt(entries []DetectionHistoryEntry) {
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].Attempt < entries[i].Attempt {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
}

func sortChapterBreakdown(breakdown []ChapterDetectionBreakdown) {
	for i := 0; i < len(breakdown); i++ {
		for j := i + 1; j < len(breakdown); j++ {
			if breakdown[j].ChapterNumber < breakdown[i].ChapterNumber {
				breakdown[i], breakdown[j] = breakdown[j], breakdown[i]
			}
		}
	}
}

func roundTo3(f float64) float64 {
	return float64(int(f*1000+0.5)) / 1000.0
}

func roundTo2(f float64) float64 {
	return float64(int(f*100+0.5)) / 100.0
}
