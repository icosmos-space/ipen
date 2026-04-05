package utils

import (
	"regexp"
	"sort"
	"strings"
)

// CadenceSummaryRow 是a compact summary row used for cadence checks。
type CadenceSummaryRow struct {
	Chapter     int
	Title       string
	Mood        string
	ChapterType string
}

// SceneCadencePressure 表示repeated scene-type pressure。
type SceneCadencePressure struct {
	Pressure     string
	RepeatedType string
	Streak       int
}

// MoodCadencePressure 表示sustained high-tension mood pressure。
type MoodCadencePressure struct {
	Pressure          string
	HighTensionStreak int
	RecentMoods       []string
}

// TitleCadencePressure 表示repeated title-token pressure。
type TitleCadencePressure struct {
	Pressure      string
	RepeatedToken string
	Count         int
	RecentTitles  []string
}

// ChapterCadenceAnalysis aggregates scene/mood/title cadence checks.
type ChapterCadenceAnalysis struct {
	ScenePressure *SceneCadencePressure
	MoodPressure  *MoodCadencePressure
	TitlePressure *TitleCadencePressure
}

const DEFAULT_CHAPTER_CADENCE_WINDOW = 4

var highTensionKeywords = []string{
	"紧张", "压抑", "冷硬", "阴沉", "危机", "对峙", "肃杀", "窒息", "焦灼", "凛冽",
	"tense", "cold", "oppressive", "grim", "ominous", "dark", "hostile", "threatening",
}

var englishStopWords = map[string]struct{}{
	"the": {}, "and": {}, "with": {}, "from": {}, "into": {}, "after": {}, "before": {},
	"over": {}, "under": {}, "this": {}, "that": {}, "your": {}, "their": {},
}

// AnalyzeChapterCadence 计算recent scene/mood/title repetition pressure。
func AnalyzeChapterCadence(rows []CadenceSummaryRow, language string) ChapterCadenceAnalysis {
	sorted := append([]CadenceSummaryRow(nil), rows...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Chapter < sorted[j].Chapter })
	if len(sorted) > CADENCE_WINDOW_DEFAULTS.SummaryLookback {
		sorted = sorted[len(sorted)-CADENCE_WINDOW_DEFAULTS.SummaryLookback:]
	}

	analysis := ChapterCadenceAnalysis{}
	analysis.ScenePressure = analyzeScenePressure(sorted)
	analysis.MoodPressure = analyzeMoodPressure(sorted)
	analysis.TitlePressure = analyzeTitlePressure(sorted, language)
	return analysis
}

// IsHighTensionMood 返回true when a mood string reads as high-tension。
func IsHighTensionMood(mood string) bool {
	lower := strings.ToLower(mood)
	for _, keyword := range highTensionKeywords {
		if strings.Contains(lower, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func analyzeScenePressure(rows []CadenceSummaryRow) *SceneCadencePressure {
	types := make([]string, 0, len(rows))
	for _, row := range rows {
		value := strings.TrimSpace(row.ChapterType)
		if isMeaningfulCadenceValue(value) {
			types = append(types, value)
		}
	}
	if len(types) < 2 {
		return nil
	}

	repeatedType := types[len(types)-1]
	streak := 0
	for i := len(types) - 1; i >= 0; i-- {
		if !strings.EqualFold(types[i], repeatedType) {
			break
		}
		streak++
	}

	pressure := ResolveCadencePressure(
		streak,
		len(types),
		CADENCE_PRESSURE_THRESHOLDS.Scene.HighCount,
		CADENCE_PRESSURE_THRESHOLDS.Scene.MediumCount,
		CADENCE_PRESSURE_THRESHOLDS.Scene.MediumWindowFloor,
	)
	if pressure == "" {
		return nil
	}

	return &SceneCadencePressure{Pressure: pressure, RepeatedType: repeatedType, Streak: streak}
}

func analyzeMoodPressure(rows []CadenceSummaryRow) *MoodCadencePressure {
	moods := make([]string, 0, len(rows))
	for _, row := range rows {
		value := strings.TrimSpace(row.Mood)
		if isMeaningfulCadenceValue(value) {
			moods = append(moods, value)
		}
	}
	if len(moods) < 2 {
		return nil
	}

	recent := []string{}
	streak := 0
	for i := len(moods) - 1; i >= 0; i-- {
		if !IsHighTensionMood(moods[i]) {
			break
		}
		recent = append([]string{moods[i]}, recent...)
		streak++
	}

	pressure := ResolveCadencePressure(
		streak,
		len(moods),
		CADENCE_PRESSURE_THRESHOLDS.Mood.HighCount,
		CADENCE_PRESSURE_THRESHOLDS.Mood.MediumCount,
		CADENCE_PRESSURE_THRESHOLDS.Mood.MediumWindowFloor,
	)
	if pressure == "" {
		return nil
	}

	return &MoodCadencePressure{Pressure: pressure, HighTensionStreak: streak, RecentMoods: recent}
}

func analyzeTitlePressure(rows []CadenceSummaryRow, language string) *TitleCadencePressure {
	titles := make([]string, 0, len(rows))
	for _, row := range rows {
		value := strings.TrimSpace(row.Title)
		if isMeaningfulCadenceValue(value) {
			titles = append(titles, value)
		}
	}
	if len(titles) < 2 {
		return nil
	}

	counts := map[string]int{}
	for _, title := range titles {
		for _, token := range extractTitleTokens(title, language) {
			counts[token]++
		}
	}

	type tokenCount struct {
		Token string
		Count int
	}
	all := make([]tokenCount, 0, len(counts))
	for token, count := range counts {
		if count >= 2 {
			all = append(all, tokenCount{Token: token, Count: count})
		}
	}
	if len(all) == 0 {
		return nil
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Count != all[j].Count {
			return all[i].Count > all[j].Count
		}
		if len(all[i].Token) != len(all[j].Token) {
			return len(all[i].Token) > len(all[j].Token)
		}
		return all[i].Token < all[j].Token
	})

	repeated := all[0]
	pressure := ResolveCadencePressure(
		repeated.Count,
		len(titles),
		CADENCE_PRESSURE_THRESHOLDS.Title.HighCount,
		CADENCE_PRESSURE_THRESHOLDS.Title.MediumCount,
		CADENCE_PRESSURE_THRESHOLDS.Title.MediumWindowFloor,
	)
	if pressure == "" {
		return nil
	}

	return &TitleCadencePressure{
		Pressure:      pressure,
		RepeatedToken: repeated.Token,
		Count:         repeated.Count,
		RecentTitles:  titles,
	}
}

func extractTitleTokens(title string, language string) []string {
	if strings.EqualFold(language, "en") {
		matches := regexp.MustCompile(`[A-Za-z]{4,}`).FindAllString(title, -1)
		seen := map[string]struct{}{}
		result := make([]string, 0, len(matches))
		for _, match := range matches {
			normalized := strings.ToLower(match)
			if _, blocked := englishStopWords[normalized]; blocked {
				continue
			}
			if _, ok := seen[normalized]; ok {
				continue
			}
			seen[normalized] = struct{}{}
			result = append(result, normalized)
		}
		return result
	}

	segments := regexp.MustCompile(`[\p{Han}]{2,}`).FindAllString(title, -1)
	seen := map[string]struct{}{}
	result := []string{}
	for _, segment := range segments {
		runes := []rune(segment)
		maxSize := 4
		if len(runes) < maxSize {
			maxSize = len(runes)
		}
		for size := 2; size <= maxSize; size++ {
			for i := 0; i <= len(runes)-size; i++ {
				token := string(runes[i : i+size])
				if _, ok := seen[token]; ok {
					continue
				}
				seen[token] = struct{}{}
				result = append(result, token)
			}
		}
	}
	return result
}

func isMeaningfulCadenceValue(value string) bool {
	normalized := strings.TrimSpace(strings.ToLower(value))
	return normalized != "" && normalized != "none" && normalized != "(none)" && normalized != "无"
}
