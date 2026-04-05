package agents

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/icosmos-space/ipen/core/models"
)

var rhetoricalPatterns = []struct {
	Name  string
	Regex *regexp.Regexp
}{
	{Name: "simile", Regex: regexp.MustCompile(`(?:like|as if|as though|仿佛|好像)`)},
	{Name: "rhetorical-question", Regex: regexp.MustCompile(`(?:\?|难道|怎么可能|岂不是)`)},
	{Name: "hyperbole", Regex: regexp.MustCompile(`(?:惊天动地|天崩地裂|earth-shattering|never seen)`)},
}

// AnalyzeStyle 提取deterministic style profile from reference text。
func AnalyzeStyle(text string, sourceName string) models.StyleProfile {
	sentences := styleSplitSentences(text)
	paragraphs := styleSplitParagraphs(text)

	sentenceLengths := make([]float64, 0, len(sentences))
	for _, sentence := range sentences {
		sentenceLengths = append(sentenceLengths, float64(len([]rune(sentence))))
	}
	avgSentence := mean(sentenceLengths)
	stdSentence := stddev(sentenceLengths, avgSentence)

	paragraphLengths := make([]float64, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		paragraphLengths = append(paragraphLengths, float64(len([]rune(paragraph))))
	}
	avgParagraph := mean(paragraphLengths)
	minParagraph, maxParagraph := minMax(paragraphLengths)

	filteredChars := regexp.MustCompile(`[\s\p{P}\p{S}\d]+`).ReplaceAllString(strings.ToLower(text), "")
	runes := []rune(filteredChars)
	unique := map[rune]struct{}{}
	for _, r := range runes {
		unique[r] = struct{}{}
	}
	vocabularyDiversity := 0.0
	if len(runes) > 0 {
		vocabularyDiversity = float64(len(unique)) / float64(len(runes))
	}

	openingCounts := map[string]int{}
	for _, sentence := range sentences {
		r := []rune(strings.TrimSpace(sentence))
		if len(r) >= 2 {
			opening := string(r[:2])
			openingCounts[opening]++
		}
	}
	topPatterns := topOpenings(openingCounts, 5, 3)

	rhetoricalFeatures := []string{}
	for _, pattern := range rhetoricalPatterns {
		matches := pattern.Regex.FindAllStringIndex(text, -1)
		if len(matches) >= 2 {
			rhetoricalFeatures = append(rhetoricalFeatures, pattern.Name+"("+itoa(len(matches))+")")
		}
	}

	return models.StyleProfile{
		AvgSentenceLength:    round(avgSentence, 1),
		SentenceLengthStdDev: round(stdSentence, 1),
		AvgParagraphLength:   round(avgParagraph, 0),
		ParagraphLengthRange: models.LengthRange{Min: minParagraph, Max: maxParagraph},
		VocabularyDiversity:  round(vocabularyDiversity, 3),
		TopPatterns:          topPatterns,
		RhetoricalFeatures:   rhetoricalFeatures,
		SourceName:           sourceName,
		AnalyzedAt:           time.Now().UTC().Format(time.RFC3339),
	}
}

func styleSplitSentences(text string) []string {
	chunks := regexp.MustCompile(`[。！？?!\n]+`).Split(text, -1)
	result := []string{}
	for _, chunk := range chunks {
		trimmed := strings.TrimSpace(chunk)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func styleSplitParagraphs(text string) []string {
	chunks := regexp.MustCompile(`\n\s*\n`).Split(text, -1)
	result := []string{}
	for _, chunk := range chunks {
		trimmed := strings.TrimSpace(chunk)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}

func stddev(values []float64, m float64) float64 {
	if len(values) <= 1 {
		return 0
	}
	sum := 0.0
	for _, value := range values {
		delta := value - m
		sum += delta * delta
	}
	return math.Sqrt(sum / float64(len(values)))
}

func minMax(values []float64) (float64, float64) {
	if len(values) == 0 {
		return 0, 0
	}
	minValue := values[0]
	maxValue := values[0]
	for _, value := range values[1:] {
		if value < minValue {
			minValue = value
		}
		if value > maxValue {
			maxValue = value
		}
	}
	return minValue, maxValue
}

func topOpenings(counts map[string]int, limit int, minCount int) []string {
	type pair struct {
		key   string
		count int
	}
	pairs := make([]pair, 0, len(counts))
	for key, count := range counts {
		if count >= minCount {
			pairs = append(pairs, pair{key: key, count: count})
		}
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].count != pairs[j].count {
			return pairs[i].count > pairs[j].count
		}
		return pairs[i].key < pairs[j].key
	})
	if len(pairs) > limit {
		pairs = pairs[:limit]
	}
	result := make([]string, 0, len(pairs))
	for _, p := range pairs {
		result = append(result, p.key+"...("+itoa(p.count)+")")
	}
	return result
}

func round(value float64, digits int) float64 {
	factor := math.Pow(10, float64(digits))
	return math.Round(value*factor) / factor
}

func itoa(v int) string { return strconv.Itoa(v) }
