package agents

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/icosmos-space/ipen/core/models"
)

// PostWriteViolation 是a deterministic post-write rule violation。
type PostWriteViolation struct {
	Rule        string `json:"rule"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
}

var zhSurpriseMarkers = []string{"仿佛", "忽然", "竟然", "猛地", "不禁"}
var zhReportTerms = []string{"核心动机", "信息边界", "最优解", "策略优势", "关键结论"}
var zhSermonWords = []string{"显然", "不难看出", "众所周知"}

// ValidatePostWrite runs deterministic post-write validation rules.
func ValidatePostWrite(content string, genreProfile *models.GenreProfile, bookRules *models.BookRules, languageOverride string) []PostWriteViolation {
	language := strings.ToLower(strings.TrimSpace(languageOverride))
	if language == "" && genreProfile != nil {
		language = strings.ToLower(strings.TrimSpace(genreProfile.Language))
	}
	if language == "en" {
		return validatePostWriteEnglish(content, genreProfile, bookRules)
	}

	issues := []PostWriteViolation{}

	if regexp.MustCompile(`不是[^，。！？\n]{0,30}[，,:：]?\s*而是`).MatchString(content) {
		issues = append(issues, PostWriteViolation{
			Rule:        "forbidden-construction",
			Severity:    "error",
			Description: "Found '不是...而是...' construction in prose.",
			Suggestion:  "Rewrite as direct statement.",
		})
	}

	if strings.Contains(content, "——") {
		issues = append(issues, PostWriteViolation{
			Rule:        "forbidden-emdash",
			Severity:    "error",
			Description: "Found em dash in prose.",
			Suggestion:  "Use comma or sentence split instead.",
		})
	}

	totalMarkers := 0
	for _, marker := range zhSurpriseMarkers {
		totalMarkers += len(regexp.MustCompile(regexp.QuoteMeta(marker)).FindAllStringIndex(content, -1))
	}
	markerLimit := maxIntPWV(1, len([]rune(content))/3000)
	if totalMarkers > markerLimit {
		issues = append(issues, PostWriteViolation{
			Rule:        "surprise-marker-density",
			Severity:    "warning",
			Description: fmt.Sprintf("Surprise markers %d exceed limit %d", totalMarkers, markerLimit),
			Suggestion:  "Replace marker words with concrete action/sensory beats.",
		})
	}

	for _, term := range zhReportTerms {
		if strings.Contains(content, term) {
			issues = append(issues, PostWriteViolation{
				Rule:        "report-terminology",
				Severity:    "error",
				Description: "Found analytical/report term in prose: " + term,
				Suggestion:  "Use this only in planning notes.",
			})
		}
	}

	for _, word := range zhSermonWords {
		if strings.Contains(content, word) {
			issues = append(issues, PostWriteViolation{
				Rule:        "author-sermon",
				Severity:    "warning",
				Description: "Found sermon-like narration word: " + word,
				Suggestion:  "Let action imply conclusion.",
			})
		}
	}

	if bookRules != nil {
		for _, prohibition := range bookRules.Prohibitions {
			trimmed := strings.TrimSpace(prohibition)
			if len([]rune(trimmed)) >= 2 && len([]rune(trimmed)) <= 30 && strings.Contains(content, trimmed) {
				issues = append(issues, PostWriteViolation{
					Rule:        "book-prohibition",
					Severity:    "error",
					Description: "Found prohibited content: " + trimmed,
					Suggestion:  "Remove or rewrite this content.",
				})
			}
		}
	}

	fatigueWords := []string{}
	if bookRules != nil && len(bookRules.FatigueWordsOverride) > 0 {
		fatigueWords = append(fatigueWords, bookRules.FatigueWordsOverride...)
	} else if genreProfile != nil {
		fatigueWords = append(fatigueWords, genreProfile.FatigueWords...)
	}
	for _, word := range fatigueWords {
		trimmed := strings.TrimSpace(word)
		if trimmed == "" {
			continue
		}
		count := len(regexp.MustCompile(regexp.QuoteMeta(trimmed)).FindAllStringIndex(content, -1))
		if count > 1 {
			issues = append(issues, PostWriteViolation{
				Rule:        "fatigue-word",
				Severity:    "warning",
				Description: fmt.Sprintf("Fatigue word %q appears %d times", trimmed, count),
				Suggestion:  "Vary wording.",
			})
		}
	}

	issues = append(issues, DetectParagraphShapeWarnings(content, "zh")...)
	return issues
}

// DetectCrossChapterRepetition 检查repetitive phrase overlap with recent chapters。
func DetectCrossChapterRepetition(currentContent string, recentChaptersContent string, language string) []PostWriteViolation {
	if strings.TrimSpace(recentChaptersContent) == "" {
		return []PostWriteViolation{}
	}
	if strings.EqualFold(language, "en") {
		return detectCrossRepetitionEN(currentContent, recentChaptersContent)
	}
	return detectCrossRepetitionZH(currentContent, recentChaptersContent)
}

// DetectParagraphLengthDrift 检查paragraph-length drift compared with recent chapters。
func DetectParagraphLengthDrift(currentContent string, recentChaptersContent string, language string) []PostWriteViolation {
	if strings.TrimSpace(recentChaptersContent) == "" {
		return []PostWriteViolation{}
	}
	current := analyzeParagraphShape(currentContent, language)
	recent := analyzeParagraphShape(recentChaptersContent, language)
	if len(current.paragraphs) < 4 || len(recent.paragraphs) < 4 || current.averageLength <= 0 || recent.averageLength <= 0 {
		return []PostWriteViolation{}
	}

	shrinkRatio := current.averageLength / recent.averageLength
	shortRatioDelta := current.shortRatio - recent.shortRatio
	if shrinkRatio >= 0.6 || current.shortRatio < 0.5 || shortRatioDelta < 0.25 {
		return []PostWriteViolation{}
	}

	dropPercent := int((1 - shrinkRatio) * 100)
	if strings.EqualFold(language, "en") {
		return []PostWriteViolation{{
			Rule:        "paragraph-density-drift",
			Severity:    "warning",
			Description: fmt.Sprintf("Average paragraph length dropped from %d to %d chars (%d%% shorter)", int(recent.averageLength), int(current.averageLength), dropPercent),
			Suggestion:  "Merge related beats more often.",
		}}
	}
	return []PostWriteViolation{{
		Rule:        "段落密度漂移",
		Severity:    "warning",
		Description: fmt.Sprintf("平均段长从 %d 降到 %d（缩短 %d%%）", int(recent.averageLength), int(current.averageLength), dropPercent),
		Suggestion:  "适当合并碎段，恢复段落层次。",
	}}
}

// DetectParagraphShapeWarnings 检查fragmentation pressure。
func DetectParagraphShapeWarnings(content string, language string) []PostWriteViolation {
	shape := analyzeParagraphShape(content, language)
	if len(shape.paragraphs) < 4 {
		return []PostWriteViolation{}
	}

	issues := []PostWriteViolation{}
	if len(shape.shortParagraphs) >= 4 && shape.shortRatio >= 0.6 {
		if strings.EqualFold(language, "en") {
			issues = append(issues, PostWriteViolation{
				Rule:        "paragraph-fragmentation",
				Severity:    "warning",
				Description: fmt.Sprintf("%d/%d paragraphs are shorter than %d chars", len(shape.shortParagraphs), len(shape.paragraphs), shape.shortThreshold),
				Suggestion:  "Merge nearby action/observation/reaction beats.",
			})
		} else {
			issues = append(issues, PostWriteViolation{
				Rule:        "段落过碎",
				Severity:    "warning",
				Description: fmt.Sprintf("%d/%d 段低于 %d 字", len(shape.shortParagraphs), len(shape.paragraphs), shape.shortThreshold),
				Suggestion:  "合并相邻动作/观察/反应段，避免一行一段。",
			})
		}
	}

	if shape.maxConsecutiveShort >= 3 {
		if strings.EqualFold(language, "en") {
			issues = append(issues, PostWriteViolation{
				Rule:        "consecutive-short-paragraphs",
				Severity:    "warning",
				Description: fmt.Sprintf("%d short paragraphs appear consecutively", shape.maxConsecutiveShort),
				Suggestion:  "Break one-beat-per-paragraph rhythm.",
			})
		} else {
			issues = append(issues, PostWriteViolation{
				Rule:        "连续短段",
				Severity:    "warning",
				Description: fmt.Sprintf("连续出现 %d 个短段", shape.maxConsecutiveShort),
				Suggestion:  "重组碎段，至少让一段承载完整动作链。",
			})
		}
	}

	return issues
}

// DetectDuplicateTitle 检查exact or near title duplication。
func DetectDuplicateTitle(newTitle string, existingTitles []string) []PostWriteViolation {
	trimmed := strings.TrimSpace(newTitle)
	if trimmed == "" {
		return []PostWriteViolation{}
	}
	normalized := strings.ToLower(trimmed)
	for _, existing := range existingTitles {
		existingNorm := strings.ToLower(strings.TrimSpace(existing))
		if existingNorm == "" {
			continue
		}
		if normalized == existingNorm {
			return []PostWriteViolation{{Rule: "duplicate-title", Severity: "warning", Description: "Title duplicates existing chapter title", Suggestion: "Use a distinct title."}}
		}
		if stripPunctuation(normalized) == stripPunctuation(existingNorm) {
			return []PostWriteViolation{{Rule: "near-duplicate-title", Severity: "warning", Description: "Title is highly similar to existing title", Suggestion: "Rename to avoid collision."}}
		}
	}
	return []PostWriteViolation{}
}

// ResolveDuplicateTitle appends numeric suffix when duplicate title exists.
func ResolveDuplicateTitle(newTitle string, existingTitles []string, language string, _content string) (string, []PostWriteViolation) {
	issues := DetectDuplicateTitle(newTitle, existingTitles)
	if len(issues) == 0 {
		return newTitle, issues
	}
	base := strings.TrimSpace(newTitle)
	for i := 2; i < 100; i++ {
		candidate := fmt.Sprintf("%s (%d)", base, i)
		if len(DetectDuplicateTitle(candidate, existingTitles)) == 0 {
			return candidate, issues
		}
	}
	return newTitle, issues
}

func validatePostWriteEnglish(content string, genreProfile *models.GenreProfile, bookRules *models.BookRules) []PostWriteViolation {
	issues := []PostWriteViolation{}
	aiTellWords := []string{"delve", "tapestry", "testament", "intricate", "pivotal", "vibrant", "embark", "comprehensive", "nuanced"}
	for _, word := range aiTellWords {
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(word) + `\b`)
		hits := len(re.FindAllStringIndex(content, -1))
		if hits > maxIntPWV(1, len(content)/3000) {
			issues = append(issues, PostWriteViolation{Rule: "ai-tell-density", Severity: "warning", Description: fmt.Sprintf("%q appears %d times", word, hits), Suggestion: "Replace with more specific wording."})
		}
	}

	paragraphs := extractParagraphs(content)
	longCount := 0
	for _, paragraph := range paragraphs {
		if len([]rune(paragraph)) > 500 {
			longCount++
		}
	}
	if longCount >= 2 {
		issues = append(issues, PostWriteViolation{Rule: "paragraph-length", Severity: "warning", Description: fmt.Sprintf("%d paragraphs exceed 500 chars", longCount), Suggestion: "Break into shorter paragraphs."})
	}

	issues = append(issues, DetectParagraphShapeWarnings(content, "en")...)

	if bookRules != nil {
		for _, prohibition := range bookRules.Prohibitions {
			trimmed := strings.TrimSpace(prohibition)
			if len(trimmed) >= 2 && len(trimmed) <= 50 && strings.Contains(strings.ToLower(content), strings.ToLower(trimmed)) {
				issues = append(issues, PostWriteViolation{Rule: "book-prohibition", Severity: "error", Description: "Found banned content: " + trimmed, Suggestion: "Remove or rewrite this content."})
			}
		}
	}

	fatigueWords := []string{}
	if bookRules != nil && len(bookRules.FatigueWordsOverride) > 0 {
		fatigueWords = append(fatigueWords, bookRules.FatigueWordsOverride...)
	} else if genreProfile != nil {
		fatigueWords = append(fatigueWords, genreProfile.FatigueWords...)
	}
	for _, word := range fatigueWords {
		trimmed := strings.TrimSpace(word)
		if trimmed == "" {
			continue
		}
		hits := len(regexp.MustCompile(`(?i)\b`+regexp.QuoteMeta(trimmed)+`\b`).FindAllStringIndex(content, -1))
		if hits > 1 {
			issues = append(issues, PostWriteViolation{Rule: "fatigue-word", Severity: "warning", Description: fmt.Sprintf("%q appears %d times", trimmed, hits), Suggestion: "Vary vocabulary."})
		}
	}

	return issues
}

type paragraphShape struct {
	paragraphs          []string
	shortThreshold      int
	shortParagraphs     []string
	shortRatio          float64
	averageLength       float64
	maxConsecutiveShort int
}

func analyzeParagraphShape(content string, language string) paragraphShape {
	paragraphs := extractParagraphs(content)
	narrative := []string{}
	for _, paragraph := range paragraphs {
		if !isDialogueParagraph(paragraph) {
			narrative = append(narrative, paragraph)
		}
	}
	threshold := 35
	if strings.EqualFold(language, "en") {
		threshold = 120
	}
	short := []string{}
	totalLen := 0
	for _, paragraph := range paragraphs {
		totalLen += len([]rune(paragraph))
	}
	streak := 0
	maxStreak := 0
	for _, paragraph := range narrative {
		if len([]rune(paragraph)) < threshold {
			short = append(short, paragraph)
			streak++
			if streak > maxStreak {
				maxStreak = streak
			}
		} else {
			streak = 0
		}
	}
	avg := 0.0
	if len(paragraphs) > 0 {
		avg = float64(totalLen) / float64(len(paragraphs))
	}
	ratio := 0.0
	if len(narrative) > 0 {
		ratio = float64(len(short)) / float64(len(narrative))
	}
	return paragraphShape{paragraphs: paragraphs, shortThreshold: threshold, shortParagraphs: short, shortRatio: ratio, averageLength: avg, maxConsecutiveShort: maxStreak}
}

func extractParagraphs(content string) []string {
	chunks := regexp.MustCompile(`\n\s*\n`).Split(content, -1)
	result := []string{}
	for _, chunk := range chunks {
		trimmed := strings.TrimSpace(chunk)
		if trimmed == "" || trimmed == "---" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}

func isDialogueParagraph(paragraph string) bool {
	trimmed := strings.TrimSpace(paragraph)
	if trimmed == "" {
		return false
	}
	return regexp.MustCompile(`^(?:["鈥溾€濄€庛€宂|鈥?`).MatchString(trimmed)
}

func stripPunctuation(value string) string {
	return regexp.MustCompile(`[\p{P}\p{S}\s]`).ReplaceAllString(value, "")
}

func detectCrossRepetitionEN(currentContent string, recentContent string) []PostWriteViolation {
	tokens := strings.Fields(regexp.MustCompile(`[^\w\s']+`).ReplaceAllString(strings.ToLower(currentContent), ""))
	if len(tokens) < 3 {
		return []PostWriteViolation{}
	}
	counts := map[string]int{}
	for i := 0; i < len(tokens)-2; i++ {
		phrase := tokens[i] + " " + tokens[i+1] + " " + tokens[i+2]
		if len(phrase) > 8 {
			counts[phrase]++
		}
	}
	recentLower := strings.ToLower(recentContent)
	hits := []string{}
	for phrase, count := range counts {
		if count >= 2 && strings.Contains(recentLower, phrase) {
			hits = append(hits, fmt.Sprintf("%q(x%d)", phrase, count))
		}
	}
	if len(hits) >= 3 {
		return []PostWriteViolation{{Rule: "cross-chapter-repetition", Severity: "warning", Description: fmt.Sprintf("%d repeated phrases also appear in recent chapters", len(hits)), Suggestion: "Vary phrase patterns across chapters."}}
	}
	return []PostWriteViolation{}
}

func detectCrossRepetitionZH(currentContent string, recentContent string) []PostWriteViolation {
	runes := []rune(regexp.MustCompile(`[\s\n\r]+`).ReplaceAllString(currentContent, ""))
	if len(runes) < 6 {
		return []PostWriteViolation{}
	}
	counts := map[string]int{}
	for i := 0; i <= len(runes)-6; i++ {
		phrase := string(runes[i : i+6])
		if regexp.MustCompile(`^[\p{Han}]{6}$`).MatchString(phrase) {
			counts[phrase]++
		}
	}
	recentClean := regexp.MustCompile(`[\s\n\r]+`).ReplaceAllString(recentContent, "")
	hits := 0
	for phrase, count := range counts {
		if count >= 2 && strings.Contains(recentClean, phrase) {
			hits++
		}
	}
	if hits >= 3 {
		return []PostWriteViolation{{Rule: "跨章重复", Severity: "warning", Description: fmt.Sprintf("%d 个短语在近期章节里也出现", hits), Suggestion: "变换动作与场景表达，减少跨章重复。"}}
	}
	return []PostWriteViolation{}
}

func maxIntPWV(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
