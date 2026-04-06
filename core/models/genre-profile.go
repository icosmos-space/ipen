package models

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// GenreProfile
type GenreProfile struct {
	// 名称
	Name string `json:"name"`
	// ID
	ID string `json:"id"`
	// 语言
	Language string `json:"language" validate:"required,oneof=zh en"` // "zh" or "en"
	// 章节类型
	ChapterTypes []string `json:"chapterTypes"`
	FatigueWords []string `json:"fatigueWords"`
	// 数字系统
	// 是否使用数字系统
	NumericalSystem   bool     `json:"numericalSystem"`
	PowerScaling      bool     `json:"powerScaling"`
	EraResearch       bool     `json:"eraResearch"`
	PacingRule        string   `json:"pacingRule"`
	SatisfactionTypes []string `json:"satisfactionTypes"`
	//
	AuditDimensions []int `json:"auditDimensions"`
}

// ParsedGenreProfile 表示a parsed genre profile with body。
type ParsedGenreProfile struct {
	Profile GenreProfile `json:"profile"`
	Body    string       `json:"body"`
}

func ParseGenreProfile(raw string) (ParsedGenreProfile, error) {
	re := regexp.MustCompile(`^---\s*\n([\s\S]*?)\n---\s*\n([\s\S]*)$`)
	fmMatch := re.FindStringSubmatch(raw)

	if fmMatch == nil {
		return ParsedGenreProfile{}, fmt.Errorf("genre profile missing YAML frontmatter (--- ... ---)")
	}

	frontmatter := fmMatch[1]
	body := strings.TrimSpace(fmMatch[2])

	var profile GenreProfile
	if err := yaml.Unmarshal([]byte(frontmatter), &profile); err != nil {
		return ParsedGenreProfile{}, fmt.Errorf("解析 YAML 格式失败: %w", err)
	}

	return ParsedGenreProfile{
		Profile: profile,
		Body:    body,
	}, nil
}
