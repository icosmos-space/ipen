package utils

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/icosmos-space/ipen/core/models"
)

// LengthLanguage 表示the length language。
type LengthLanguage string

const (
	LanguageZH LengthLanguage = "zh"
	LanguageEN LengthLanguage = "en"
)

const (
	ReferenceTarget = 2200
	SoftRangeDelta  = 300
	HardRangeDelta  = 600
)

// CountChapterLength 统计the length of chapter content。
func CountChapterLength(content string, countingMode models.LengthCountingMode) int {
	normalized := stripMarkdownMetadata(content)

	if countingMode == models.CountingModeENWords {
		words := regexp.MustCompile(`[A-Za-z0-9]+(?:'[A-Za-z0-9]+)?`).FindAllString(normalized, -1)
		return len(words)
	}

	count := 0
	for _, r := range normalized {
		if !unicode.IsSpace(r) {
			count++
		}
	}
	return count
}

// ResolveLengthCountingMode 解析the length counting mode from language。
func ResolveLengthCountingMode(language LengthLanguage) models.LengthCountingMode {
	if language == LanguageEN {
		return models.CountingModeENWords
	}
	return models.CountingModeZHChars
}

// FormatLengthCount 格式化a length count for display。
func FormatLengthCount(count int, countingMode models.LengthCountingMode) string {
	if countingMode == models.CountingModeENWords {
		return strconv.Itoa(count) + " words"
	}
	return strconv.Itoa(count) + "字"
}

// BuildLengthSpec 构建a length spec from target and language。
func BuildLengthSpec(target int, language LengthLanguage) models.LengthSpec {
	softDelta := scaleRangeDelta(target, SoftRangeDelta)
	hardDelta := int(math.Max(float64(softDelta), float64(scaleRangeDelta(target, HardRangeDelta))))
	softMin := int(math.Max(1, float64(target-softDelta)))
	softMax := target + softDelta
	hardMin := int(math.Max(1, float64(target-hardDelta)))
	hardMax := target + hardDelta

	return models.LengthSpec{
		Target:        target,
		SoftMin:       softMin,
		SoftMax:       softMax,
		HardMin:       hardMin,
		HardMax:       hardMax,
		CountingMode:  ResolveLengthCountingMode(language),
		NormalizeMode: models.NormalizeModeNone,
	}
}

func scaleRangeDelta(target int, referenceDelta int) int {
	return int(math.Max(1, math.Floor(float64(target*referenceDelta)/ReferenceTarget)))
}

// IsOutsideSoftRange 检查if count is outside soft range。
func IsOutsideSoftRange(count int, spec models.LengthSpec) bool {
	return count < spec.SoftMin || count > spec.SoftMax
}

// IsOutsideHardRange 检查if count is outside hard range。
func IsOutsideHardRange(count int, spec models.LengthSpec) bool {
	return count < spec.HardMin || count > spec.HardMax
}

// ChooseNormalizeMode 选择the normalize mode based on count。
func ChooseNormalizeMode(count int, spec models.LengthSpec) models.LengthNormalizeMode {
	if count < spec.SoftMin {
		return models.NormalizeModeExpand
	}
	if count > spec.SoftMax {
		return models.NormalizeModeCompress
	}
	return models.NormalizeModeNone
}

func stripMarkdownMetadata(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.TrimPrefix(content, "\uFEFF")
	lines := strings.Split(content, "\n")

	index := 0
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		index++
		for index < len(lines) && strings.TrimSpace(lines[index]) != "---" {
			index++
		}
		if index < len(lines) {
			index++
		}
	}

	proseLines := []string{}
	inFence := false
	for ; index < len(lines); index++ {
		line := lines[index]
		trimmed := strings.TrimSpace(line)

		if matched, _ := regexp.MatchString(`^(`+"```"+`|~~~)`, trimmed); matched {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if matched, _ := regexp.MatchString(`^#{1,6}\s+`, trimmed); matched {
			continue
		}
		if trimmed == "---" || trimmed == "..." {
			continue
		}
		proseLines = append(proseLines, line)
	}

	return strings.Join(proseLines, "\n")
}
