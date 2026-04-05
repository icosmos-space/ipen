package utils

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// LongSpanFatigueIssue 是a warning produced by long-span fatigue checks。
type LongSpanFatigueIssue struct {
	Severity    string
	Category    string
	Description string
	Suggestion  string
}

// AnalyzeLongSpanFatigueInput holds inputs for long-span fatigue analysis.
type AnalyzeLongSpanFatigueInput struct {
	BookDir        string
	ChapterNumber  int
	ChapterContent string
	ChapterSummary string
	Language       string
}

// EnglishVarianceBrief summarizes repeated phrase/pattern risks for English prose.
type EnglishVarianceBrief struct {
	HighFrequencyPhrases    []string
	RepeatedOpeningPatterns []string
	RepeatedEndingShapes    []string
	SceneObligation         string
	Text                    string
}

type summaryRow struct {
	Chapter     int
	Title       string
	Mood        string
	ChapterType string
}

var chinesePunctuation = regexp.MustCompile(`[，。！？；：“”‘’（）《》\s\-—]`)
var englishPunctuation = regexp.MustCompile(`[^a-z0-9]+`)

// BuildEnglishVarianceBrief 构建an English-focused anti-repetition brief。
func BuildEnglishVarianceBrief(bookDir string, chapterNumber int) *EnglishVarianceBrief {
	chapterBodies := loadPreviousChapterBodies(bookDir, chapterNumber, CADENCE_WINDOW_DEFAULTS.EnglishVarianceLookback)
	if len(chapterBodies) < 2 {
		return nil
	}

	summaryRows := loadSummaryRows(filepath.Join(bookDir, "story", "chapter_summaries.md"))
	recentRows := []CadenceSummaryRow{}
	for _, row := range summaryRows {
		if row.Chapter < chapterNumber {
			recentRows = append(recentRows, CadenceSummaryRow{Chapter: row.Chapter, Title: row.Title, Mood: row.Mood, ChapterType: row.ChapterType})
		}
	}
	sort.Slice(recentRows, func(i, j int) bool { return recentRows[i].Chapter < recentRows[j].Chapter })
	if len(recentRows) > CADENCE_WINDOW_DEFAULTS.SummaryLookback {
		recentRows = recentRows[len(recentRows)-CADENCE_WINDOW_DEFAULTS.SummaryLookback:]
	}

	highFrequency := collectRepeatedEnglishPhrases(chapterBodies)
	repeatedOpenings := collectRepeatedBoundaryPatterns(chapterBodies, "opening")
	repeatedEndings := collectRepeatedBoundaryPatterns(chapterBodies, "ending")
	cadence := AnalyzeChapterCadence(recentRows, "en")
	sceneObligation := chooseSceneObligation(cadence, repeatedOpenings, repeatedEndings)

	lines := []string{
		"## English Variance Brief",
		"",
		"- High-frequency phrases to avoid: " + formatEnglishList(highFrequency),
		"- Repeated opening patterns to avoid: " + formatEnglishList(repeatedOpenings),
		"- Repeated ending patterns to avoid: " + formatEnglishList(repeatedEndings),
		"- Scene obligation: " + sceneObligation,
	}

	return &EnglishVarianceBrief{
		HighFrequencyPhrases:    highFrequency,
		RepeatedOpeningPatterns: repeatedOpenings,
		RepeatedEndingShapes:    repeatedEndings,
		SceneObligation:         sceneObligation,
		Text:                    strings.Join(lines, "\n"),
	}
}

// AnalyzeLongSpanFatigue 检测cadence monotony and repeated boundary-pattern issues。
func AnalyzeLongSpanFatigue(input AnalyzeLongSpanFatigueInput) []LongSpanFatigueIssue {
	language := input.Language
	if language == "" {
		language = "zh"
	}

	issues := []LongSpanFatigueIssue{}
	summaryRows := loadSummaryRows(filepath.Join(input.BookDir, "story", "chapter_summaries.md"))
	merged := mergeCurrentSummary(summaryRows, input.ChapterSummary)
	recentRows := []CadenceSummaryRow{}
	for _, row := range merged {
		if row.Chapter <= input.ChapterNumber {
			recentRows = append(recentRows, CadenceSummaryRow{Chapter: row.Chapter, Title: row.Title, Mood: row.Mood, ChapterType: row.ChapterType})
		}
	}
	sort.Slice(recentRows, func(i, j int) bool { return recentRows[i].Chapter < recentRows[j].Chapter })
	if len(recentRows) > CADENCE_WINDOW_DEFAULTS.SummaryLookback {
		recentRows = recentRows[len(recentRows)-CADENCE_WINDOW_DEFAULTS.SummaryLookback:]
	}
	cadence := AnalyzeChapterCadence(recentRows, language)

	if issue := buildChapterTypeIssue(cadence, language); issue != nil {
		issues = append(issues, *issue)
	}
	if issue := buildMoodIssue(cadence, language); issue != nil {
		issues = append(issues, *issue)
	}
	if issue := buildTitleIssue(cadence, language); issue != nil {
		issues = append(issues, *issue)
	}

	recentBodies := loadRecentChapterBodies(input.BookDir, input.ChapterNumber, input.ChapterContent)
	if issue := buildSentencePatternIssue(recentBodies, "opening", language); issue != nil {
		issues = append(issues, *issue)
	}
	if issue := buildSentencePatternIssue(recentBodies, "ending", language); issue != nil {
		issues = append(issues, *issue)
	}

	return issues
}

func loadSummaryRows(path string) []summaryRow {
	data, err := os.ReadFile(path)
	if err != nil {
		return []summaryRow{}
	}
	rows := []summaryRow{}
	for _, line := range strings.Split(string(data), "\n") {
		if row, ok := parseSummaryRow(line); ok {
			rows = append(rows, row)
		}
	}
	return rows
}

func loadPreviousChapterBodies(bookDir string, currentChapter int, limit int) []string {
	chaptersDir := filepath.Join(bookDir, "chapters")
	entries, err := os.ReadDir(chaptersDir)
	if err != nil {
		return []string{}
	}
	type chapterFile struct {
		Path    string
		Chapter int
	}
	files := []chapterFile{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") || len(entry.Name()) < 4 {
			continue
		}
		chapter, err := strconv.Atoi(entry.Name()[:4])
		if err != nil || chapter >= currentChapter {
			continue
		}
		files = append(files, chapterFile{Path: filepath.Join(chaptersDir, entry.Name()), Chapter: chapter})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Chapter < files[j].Chapter })
	if len(files) > limit {
		files = files[len(files)-limit:]
	}

	bodies := []string{}
	for _, file := range files {
		if body, err := os.ReadFile(file.Path); err == nil {
			bodies = append(bodies, string(body))
		}
	}
	return bodies
}

func mergeCurrentSummary(rows []summaryRow, currentSummary string) []summaryRow {
	current, ok := parseSummaryRow(currentSummary)
	if !ok {
		return append([]summaryRow(nil), rows...)
	}
	result := []summaryRow{}
	for _, row := range rows {
		if row.Chapter != current.Chapter {
			result = append(result, row)
		}
	}
	result = append(result, current)
	return result
}

func parseSummaryRow(line string) (summaryRow, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "|") || strings.Contains(trimmed, "---") || strings.Contains(trimmed, "章节 |") || strings.Contains(strings.ToLower(trimmed), "chapter |") {
		return summaryRow{}, false
	}
	cells := []string{}
	for _, part := range strings.Split(trimmed, "|") {
		part = strings.TrimSpace(part)
		if part != "" {
			cells = append(cells, part)
		}
	}
	if len(cells) < 8 {
		return summaryRow{}, false
	}
	chapter, err := strconv.Atoi(cells[0])
	if err != nil || chapter <= 0 {
		return summaryRow{}, false
	}
	return summaryRow{Chapter: chapter, Title: cells[1], Mood: cells[6], ChapterType: cells[7]}, true
}

func buildChapterTypeIssue(cadence ChapterCadenceAnalysis, language string) *LongSpanFatigueIssue {
	if cadence.ScenePressure == nil || cadence.ScenePressure.Pressure != "high" {
		return nil
	}
	repeatedType := cadence.ScenePressure.RepeatedType
	streak := cadence.ScenePressure.Streak
	if strings.EqualFold(language, "en") {
		return &LongSpanFatigueIssue{Severity: "warning", Category: "Pacing Monotony", Description: "The last " + strconv.Itoa(streak) + " chapter types stayed on " + repeatedType + ".", Suggestion: "Switch chapter function and rotate setup/payoff/reversal beats."}
	}
	return &LongSpanFatigueIssue{Severity: "warning", Category: "节奏单调", Description: "最近" + strconv.Itoa(streak) + " 章持续停留在" + repeatedType + "。", Suggestion: "下一章切换章节功能，不要继续重复同一种节拍。"}
}

func buildMoodIssue(cadence ChapterCadenceAnalysis, language string) *LongSpanFatigueIssue {
	if cadence.MoodPressure == nil || cadence.MoodPressure.Pressure != "high" {
		return nil
	}
	streak := cadence.MoodPressure.HighTensionStreak
	trail := strings.Join(cadence.MoodPressure.RecentMoods, " -> ")
	if strings.EqualFold(language, "en") {
		return &LongSpanFatigueIssue{Severity: "warning", Category: "Mood Monotony", Description: "High-tension mood locked for " + strconv.Itoa(streak) + " chapters (" + trail + ").", Suggestion: "Insert release beats before escalating again."}
	}
	return &LongSpanFatigueIssue{Severity: "warning", Category: "情绪单调", Description: "最近" + strconv.Itoa(streak) + " 章持续高压（" + trail + "）。", Suggestion: "安排一次情绪释放，再继续加压。"}
}

func buildTitleIssue(cadence ChapterCadenceAnalysis, language string) *LongSpanFatigueIssue {
	if cadence.TitlePressure == nil || cadence.TitlePressure.Pressure != "high" {
		return nil
	}
	token := cadence.TitlePressure.RepeatedToken
	count := cadence.TitlePressure.Count
	if strings.EqualFold(language, "en") {
		return &LongSpanFatigueIssue{Severity: "warning", Category: "Title Collapse", Description: "Recent titles collapse around '" + token + "' (" + strconv.Itoa(count) + " hits).", Suggestion: "Use a new image/action/consequence anchor in next title."}
	}
	return &LongSpanFatigueIssue{Severity: "warning", Category: "标题重复", Description: "近期标题持续围绕" + token + "（命中 " + strconv.Itoa(count) + " 次）。", Suggestion: "下一章标题换一个新焦点，不要重复同一关键词壳。"}
}

func loadRecentChapterBodies(bookDir string, currentChapter int, currentContent string) []string {
	prev := loadPreviousChapterBodies(bookDir, currentChapter, CADENCE_WINDOW_DEFAULTS.RecentBoundaryPatternBodies)
	if len(prev) < CADENCE_WINDOW_DEFAULTS.RecentBoundaryPatternBodies {
		return []string{}
	}
	return append(prev, currentContent)
}

func buildSentencePatternIssue(chapterBodies []string, boundary string, language string) *LongSpanFatigueIssue {
	if len(chapterBodies) < LONG_SPAN_FATIGUE_THRESHOLDS.BoundaryPatternMinBodies {
		return nil
	}
	sentences := []string{}
	for _, body := range chapterBodies {
		sentence := extractBoundarySentence(body, boundary)
		if sentence == "" {
			return nil
		}
		sentences = append(sentences, sentence)
	}
	normalized := []string{}
	for _, sentence := range sentences {
		norm := normalizeSentence(sentence, language)
		if len([]rune(norm)) < LONG_SPAN_FATIGUE_THRESHOLDS.BoundarySentenceMinLength {
			return nil
		}
		normalized = append(normalized, norm)
	}

	similarities := []float64{diceCoefficient(normalized[0], normalized[1]), diceCoefficient(normalized[1], normalized[2])}
	if minFloat(similarities[0], similarities[1]) < LONG_SPAN_FATIGUE_THRESHOLDS.BoundarySimilarityFloor {
		return nil
	}

	sample := summarizeSentence(sentences[2], language)
	pairText := fmtFloat(similarities[0]) + "/" + fmtFloat(similarities[1])
	if strings.EqualFold(language, "en") {
		category := "Opening Pattern Repetition"
		position := "openings"
		suggestion := "Change the next chapter opening vector."
		if boundary == "ending" {
			category = "Ending Pattern Repetition"
			position = "endings"
			suggestion = "Change the next chapter landing pattern."
		}
		return &LongSpanFatigueIssue{Severity: "warning", Category: category, Description: "The last 3 chapter " + position + " are highly similar (" + pairText + "). Signature: '" + sample + "'.", Suggestion: suggestion}
	}

	category := "开头同构"
	suggestion := "下一章换一个开篇入口，不要继续沿用同一种句式。"
	if boundary == "ending" {
		category = "结尾同构"
		suggestion = "下一章换一种收束方式，不要继续同样句法。"
	}
	return &LongSpanFatigueIssue{Severity: "warning", Category: category, Description: "最近 3 章" + mapBoundary(boundary) + "句式高度相似（" + pairText + "），当前句式近似" + sample + "。", Suggestion: suggestion}
}

func collectRepeatedEnglishPhrases(chapterBodies []string) []string {
	counts := map[string]int{}
	for _, body := range chapterBodies {
		tokens := strings.Fields(strings.ToLower(englishPunctuation.ReplaceAllString(body, " ")))
		seen := map[string]struct{}{}
		for i := 0; i+2 < len(tokens); i++ {
			if len(tokens[i]) < 3 || len(tokens[i+1]) < 3 || len(tokens[i+2]) < 3 {
				continue
			}
			phrase := tokens[i] + " " + tokens[i+1] + " " + tokens[i+2]
			seen[phrase] = struct{}{}
		}
		for phrase := range seen {
			counts[phrase]++
		}
	}
	return topRepeated(counts)
}

func collectRepeatedBoundaryPatterns(chapterBodies []string, boundary string) []string {
	counts := map[string]int{}
	for _, body := range chapterBodies {
		sentence := extractBoundarySentence(body, boundary)
		if sentence == "" {
			continue
		}
		tokens := strings.Fields(strings.ToLower(englishPunctuation.ReplaceAllString(sentence, " ")))
		if len(tokens) < 2 {
			continue
		}
		if len(tokens) > 4 {
			tokens = tokens[:4]
		}
		counts[strings.Join(tokens, " ")]++
	}
	return topRepeated(counts)
}

func chooseSceneObligation(cadence ChapterCadenceAnalysis, repeatedOpenings []string, repeatedEndings []string) string {
	if cadence.ScenePressure != nil && cadence.ScenePressure.Pressure == "high" {
		return "confrontation under pressure"
	}
	if len(repeatedEndings) > 0 {
		return "discovery under pressure"
	}
	if len(repeatedOpenings) > 0 {
		return "negotiation with withholding"
	}
	return "concealment with active pushback"
}

func extractBoundarySentence(content string, boundary string) string {
	parts := []string{}
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		parts = append(parts, trimmed)
	}
	flattened := strings.Join(parts, " ")
	sentences := regexp.MustCompile(`(?<=[。！？?!\.])\s+`).Split(flattened, -1)
	clean := []string{}
	for _, sentence := range sentences {
		trimmed := strings.TrimSpace(sentence)
		if trimmed != "" {
			clean = append(clean, trimmed)
		}
	}
	if len(clean) == 0 {
		return ""
	}
	if boundary == "opening" {
		return clean[0]
	}
	return clean[len(clean)-1]
}

func normalizeSentence(sentence string, language string) string {
	if strings.EqualFold(language, "en") {
		return strings.TrimSpace(englishPunctuation.ReplaceAllString(strings.ToLower(sentence), ""))
	}
	return strings.ToLower(chinesePunctuation.ReplaceAllString(sentence, ""))
}

func summarizeSentence(sentence string, language string) string {
	if strings.EqualFold(language, "en") {
		tokens := strings.Fields(strings.ToLower(englishPunctuation.ReplaceAllString(sentence, " ")))
		if len(tokens) > 6 {
			tokens = tokens[:6]
		}
		if len(tokens) > 0 {
			return strings.Join(tokens, " ")
		}
		runes := []rune(sentence)
		if len(runes) > 32 {
			return string(runes[:32])
		}
		return sentence
	}
	collapsed := chinesePunctuation.ReplaceAllString(sentence, "")
	runes := []rune(collapsed)
	if len(runes) > 12 {
		return string(runes[:12])
	}
	return collapsed
}

func formatEnglishList(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func diceCoefficient(left string, right string) float64 {
	if left == right {
		return 1
	}
	if len([]rune(left)) < 2 || len([]rune(right)) < 2 {
		return 0
	}
	leftBigrams := buildBigrams(left)
	rightBigrams := buildBigrams(right)
	overlap := 0
	for bigram, count := range leftBigrams {
		overlap += minInt(count, rightBigrams[bigram])
	}
	leftCount := 0
	for _, count := range leftBigrams {
		leftCount += count
	}
	rightCount := 0
	for _, count := range rightBigrams {
		rightCount += count
	}
	if leftCount+rightCount == 0 {
		return 0
	}
	return float64(2*overlap) / float64(leftCount+rightCount)
}

func buildBigrams(value string) map[string]int {
	runes := []rune(value)
	result := map[string]int{}
	for i := 0; i < len(runes)-1; i++ {
		bigram := string(runes[i : i+2])
		result[bigram]++
	}
	return result
}

func topRepeated(counts map[string]int) []string {
	type pair struct {
		Key   string
		Count int
	}
	pairs := []pair{}
	for key, count := range counts {
		if count >= 2 {
			pairs = append(pairs, pair{Key: key, Count: count})
		}
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].Count != pairs[j].Count {
			return pairs[i].Count > pairs[j].Count
		}
		return pairs[i].Key < pairs[j].Key
	})
	if len(pairs) > 3 {
		pairs = pairs[:3]
	}
	result := []string{}
	for _, pair := range pairs {
		result = append(result, pair.Key)
	}
	return result
}

func fmtFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', 2, 64)
}

func minFloat(a float64, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func mapBoundary(boundary string) string {
	if boundary == "opening" {
		return "开头"
	}
	return "结尾"
}
