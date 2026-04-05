/**
 * Structural AI-tell detection - pure rule-based analysis (no LLM).
 *
 * Detects patterns common in AI-generated Chinese text:
 * - dim 20: Paragraph length uniformity (low variance)
 * - dim 21: Filler/hedge word density
 * - dim 22: Formulaic transition patterns
 * - dim 23: List-like structure (consecutive same-prefix sentences)
 */
package agents

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"unicode"
)

// AITellIssue 表示an AI-tell issue。
type AITellIssue struct {
	Severity    string `json:"severity"` // "warning" or "info"
	Category    string `json:"category"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
}

// AITellResult 表示AI-tell analysis result。
type AITellResult struct {
	Issues []AITellIssue `json:"issues"`
}

var hedgeWordsZH = []string{"似乎", "可能", "或许", "大概", "某种程度上", "一定程度上", "在某种意义上"}
var hedgeWordsEN = []string{"seems", "seemed", "perhaps", "maybe", "apparently", "in some ways", "to some extent"}

var transitionWordsZH = []string{"然而", "不过", "与此同时", "另一方面", "尽管如此", "话虽如此", "但值得注意的是"}
var transitionWordsEN = []string{"however", "meanwhile", "on the other hand", "nevertheless", "even so", "still"}

// AnalyzeAITells 分析text content for structural AI-tell patterns。
func AnalyzeAITells(content string, language string) AITellResult {
	var issues []AITellIssue
	isEnglish := language == "en"

	// Split into paragraphs
	paragraphs := splitParagraphs(content)

	// dim 20: Paragraph length uniformity (needs >=3 paragraphs)
	if len(paragraphs) >= 3 {
		paragraphLengths := make([]int, len(paragraphs))
		totalLen := 0
		for i, p := range paragraphs {
			paragraphLengths[i] = len([]rune(p))
			totalLen += paragraphLengths[i]
		}
		mean := float64(totalLen) / float64(len(paragraphs))
		if mean > 0 {
			variance := 0.0
			for _, l := range paragraphLengths {
				variance += (float64(l) - mean) * (float64(l) - mean)
			}
			variance /= float64(len(paragraphLengths))
			stdDev := math.Sqrt(variance)
			cv := stdDev / mean
			if cv < 0.15 {
				if isEnglish {
					issues = append(issues, AITellIssue{
						Severity:    "warning",
						Category:    "Paragraph uniformity",
						Description: fmt.Sprintf("Paragraph-length coefficient of variation is only %.3f (threshold <0.15), which suggests unnaturally uniform paragraph sizing", cv),
						Suggestion:  "Increase paragraph-length contrast: use shorter beats for impact and longer blocks for immersive detail",
					})
				} else {
					issues = append(issues, AITellIssue{
						Severity:    "warning",
						Category:    "段落等长",
						Description: fmt.Sprintf("段落长度变异系数仅为 %.3f（阈值 < 0.15），段落长度过于均匀，呈现 AI 生成特征", cv),
						Suggestion:  "增加段落长度差异：短段落用于节奏加速或冲击，长段落用于沉浸描写",
					})
				}
			}
		}
	}

	// dim 21: Hedge word density
	totalChars := len([]rune(content))
	if totalChars > 0 {
		hedgeWords := hedgeWordsZH
		if isEnglish {
			hedgeWords = hedgeWordsEN
		}
		hedgeCount := 0
		for _, word := range hedgeWords {
			count := strings.Count(content, word)
			hedgeCount += count
		}
		hedgeDensity := float64(hedgeCount) / (float64(totalChars) / 1000.0)
		if hedgeDensity > 3 {
			if isEnglish {
				issues = append(issues, AITellIssue{
					Severity:    "warning",
					Category:    "Hedge density",
					Description: fmt.Sprintf("Hedge-word density is %.1f per 1k characters (threshold >3), making the prose sound overly tentative", hedgeDensity),
					Suggestion:  "Replace hedges with firmer narration: remove vague qualifiers and use concrete detail instead",
				})
			} else {
				issues = append(issues, AITellIssue{
					Severity:    "warning",
					Category:    "套话密度",
					Description: fmt.Sprintf("套话词（似乎/可能/或许等）密度为 %.1f 次/千字（阈值 > 3），语气过于模糊犹豫", hedgeDensity),
					Suggestion:  "用确定性叙述替代模糊表达：去掉“似乎”，直接描述状态，用具体细节替代“可能”。",
				})
			}
		}
	}

	// dim 22: Formulaic transition repetition
	transitionWords := transitionWordsZH
	if isEnglish {
		transitionWords = transitionWordsEN
	}
	transitionCounts := make(map[string]int)
	for _, word := range transitionWords {
		count := strings.Count(strings.ToLower(content), strings.ToLower(word))
		if count > 0 {
			key := word
			if isEnglish {
				key = strings.ToLower(word)
			}
			transitionCounts[key] = count
		}
	}
	var repeatedTransitions []string
	for word, count := range transitionCounts {
		if count >= 3 {
			repeatedTransitions = append(repeatedTransitions, fmt.Sprintf("\"%s\"×%d", word, count))
		}
	}
	if len(repeatedTransitions) > 0 {
		joiner := "、"
		if isEnglish {
			joiner = ", "
		}
		detail := strings.Join(repeatedTransitions, joiner)
		if isEnglish {
			issues = append(issues, AITellIssue{
				Severity:    "warning",
				Category:    "Formulaic transitions",
				Description: fmt.Sprintf("Transition words repeat too often: %s. Reusing the same transition pattern 3+ times creates a formulaic AI texture", detail),
				Suggestion:  "Let scenes pivot through action, timing, or viewpoint shifts instead of repeating the same transitions",
			})
		} else {
			issues = append(issues, AITellIssue{
				Severity:    "warning",
				Category:    "公式化转折",
				Description: fmt.Sprintf("转折词重复使用：%s。同一转折模式≥3次，暴露 AI 生成痕迹", detail),
				Suggestion:  "用情节自然转折替代转折词，或换用不同的过渡手法（动作切入、时间跳跃、视角切换）",
			})
		}
	}

	// dim 23: List-like structure
	sentences := splitSentences(content, isEnglish)
	if len(sentences) >= 3 {
		consecutiveSamePrefix := 1
		maxConsecutive := 1
		for i := 1; i < len(sentences); i++ {
			prevPrefix := getSentencePrefix(sentences[i-1], isEnglish)
			currPrefix := getSentencePrefix(sentences[i], isEnglish)
			if prevPrefix == currPrefix {
				consecutiveSamePrefix++
				if consecutiveSamePrefix > maxConsecutive {
					maxConsecutive = consecutiveSamePrefix
				}
			} else {
				consecutiveSamePrefix = 1
			}
		}
		if maxConsecutive >= 3 {
			if isEnglish {
				issues = append(issues, AITellIssue{
					Severity:    "info",
					Category:    "List-like structure",
					Description: fmt.Sprintf("Detected %d consecutive sentences with the same opening pattern, creating a list-like generated cadence", maxConsecutive),
					Suggestion:  "Vary how sentences open: change subject, timing, or action entry to break the list effect",
				})
			} else {
				issues = append(issues, AITellIssue{
					Severity:    "info",
					Category:    "列表式结构",
					Description: fmt.Sprintf("检测到%d句连续以相同开头的句子，呈现列表式AI生成结构", maxConsecutive),
					Suggestion:  "变换句式开头：用不同主语、时间词、动作词开头，打破列表感。",
				})
			}
		}
	}

	return AITellResult{Issues: issues}
}

func splitParagraphs(content string) []string {
	// Split by double newlines or more
	re := regexp.MustCompile(`\n\s*\n`)
	paragraphs := re.Split(content, -1)
	var result []string
	for _, p := range paragraphs {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func splitSentences(content string, isEnglish bool) []string {
	var sentences []string
	var re *regexp.Regexp
	if isEnglish {
		re = regexp.MustCompile(`[.!?\n]+`)
	} else {
		re = regexp.MustCompile(`[。！？\n]+`)
	}
	parts := re.Split(content, -1)
	for _, s := range parts {
		trimmed := strings.TrimSpace(s)
		if len([]rune(trimmed)) > 2 {
			sentences = append(sentences, trimmed)
		}
	}
	return sentences
}

func getSentencePrefix(sentence string, isEnglish bool) string {
	if isEnglish {
		// Get first word
		fields := strings.Fields(sentence)
		if len(fields) > 0 {
			return strings.ToLower(fields[0])
		}
		return ""
	}
	// Get first 2 characters for Chinese
	runes := []rune(sentence)
	if len(runes) >= 2 {
		return string(runes[:2])
	}
	return sentence
}

// CountChineseChars 统计Chinese characters in text。
func countChineseChars(text string) int {
	count := 0
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			count++
		}
	}
	return count
}
