package agents

import (
	"os"
	"strings"
)

const missingFilePlaceholder = "(文件尚未创建)"

func readFileWithFallback(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return missingFilePlaceholder, nil
		}
		return "", err
	}

	content := string(data)
	if strings.TrimSpace(content) == "" {
		return missingFilePlaceholder, nil
	}

	return content, nil
}

func truncateRunes(text string, max int) string {
	if max <= 0 {
		return ""
	}

	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	return string(runes[:max])
}

func extractFirstJSONObject(content string) string {
	start := strings.Index(content, "{")
	if start < 0 {
		return ""
	}

	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(content); i++ {
		ch := content[i]

		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return strings.TrimSpace(content[start : i+1])
			}
		}
	}

	return ""
}
