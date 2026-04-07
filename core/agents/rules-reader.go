package agents

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/icosmos-space/ipen/core/data"
)

// GenreProfile 表示a genre profile。
type GenreProfile struct {
	Name        string `json:"name"`
	Language    string `json:"language"`
	Description string `json:"description"`
}

// ParsedGenreProfile 表示a parsed genre profile with raw body。
type ParsedGenreProfile struct {
	Profile GenreProfile `json:"profile"`
	Body    string       `json:"body"`
}

// ParsedBookRules 表示parsed book rules。
type ParsedBookRules struct {
	Rules map[string]any `json:"rules"`
	Raw   string         `json:"raw"`
}

// GenreInfo 表示available genre information。
type GenreInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Source string `json:"source"` // "project" or "builtin"
}

var builtinGenresDir = "" // Deprecated: now uses embedded filesystem

func init() {
	// Builtin genres are now embedded in the binary
	// This initialization is kept for backwards compatibility
	builtinGenresDir = getBuiltinGenresDir()
}

// GetBuiltinGenresDir 返回the resolved built-in genres directory.
// Deprecated: Builtin genres are now embedded. This function returns empty string.
func GetBuiltinGenresDir() string {
	return ""
}

func getBuiltinGenresDir() string {
	return ""
}

// ReadGenreProfile 加载genre profile. Lookup order:。
// 1. Project-level: {projectRoot}/genres/{genreId}.md
// 2. Built-in:     embedded genres/{genreId}.md
// 3. Fallback:     embedded genres/other.md
func ReadGenreProfile(projectRoot, genreID string) (*ParsedGenreProfile, error) {
	projectPath := filepath.Join(projectRoot, "genres", genreID+".md")

	// Try project level first
	raw, err := tryReadFile(projectPath)
	if err != nil || raw == "" {
		// Try embedded builtin
		raw, err = tryReadEmbeddedFile(genreID + ".md")
	}
	if err != nil || raw == "" {
		// Fallback to other.md
		raw, err = tryReadEmbeddedFile("other.md")
	}
	if err != nil || raw == "" {
		return nil, fmt.Errorf("genre profile not found for \"%s\" and fallback \"other.md\" is missing", genreID)
	}

	return parseGenreProfile(raw)
}

// ListAvailableGenres lists all available genre profiles
func ListAvailableGenres(projectRoot string) ([]GenreInfo, error) {
	results := make(map[string]GenreInfo)

	// Built-in genres from embedded filesystem
	embeddedGenres, err := listEmbeddedGenres()
	if err == nil {
		for _, id := range embeddedGenres {
			raw, err := tryReadEmbeddedFile(id + ".md")
			if err != nil || raw == "" {
				continue
			}
			parsed, err := parseGenreProfile(raw)
			if err != nil {
				continue
			}
			results[id] = GenreInfo{
				ID:     id,
				Name:   parsed.Profile.Name,
				Source: "builtin",
			}
		}
	}

	// Project-level genres override
	projectDir := filepath.Join(projectRoot, "genres")
	projectFiles, err := readDir(projectDir)
	if err == nil {
		for _, file := range projectFiles {
			if !strings.HasSuffix(file, ".md") {
				continue
			}
			id := strings.TrimSuffix(file, ".md")
			raw, err := tryReadFile(filepath.Join(projectDir, file))
			if err != nil || raw == "" {
				continue
			}
			parsed, err := parseGenreProfile(raw)
			if err != nil {
				continue
			}
			results[id] = GenreInfo{
				ID:     id,
				Name:   parsed.Profile.Name,
				Source: "project",
			}
		}
	}

	// Convert to sorted slice
	var result []GenreInfo
	for _, info := range results {
		result = append(result, info)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})

	return result, nil
}

// ReadBookRules 加载book_rules.md from the book's story directory。
func ReadBookRules(bookDir string) (*ParsedBookRules, error) {
	raw, err := tryReadFile(filepath.Join(bookDir, "story", "book_rules.md"))
	if err != nil || raw == "" {
		return nil, nil // File doesn't exist
	}
	return parseBookRules(raw)
}

// ReadBookLanguage 读取book.json and returns language if present。
func ReadBookLanguage(bookDir string) (string, error) {
	raw, err := tryReadFile(filepath.Join(bookDir, "book.json"))
	if err != nil || raw == "" {
		return "", nil
	}

	var bookConfig map[string]any
	if err := json.Unmarshal([]byte(raw), &bookConfig); err != nil {
		return "", nil
	}

	if lang, ok := bookConfig["language"].(string); ok {
		return lang, nil
	}

	return "", nil
}

// Helper functions
func tryReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil
	}
	return string(data), nil
}

// tryReadEmbeddedFile reads a file from the embedded genres filesystem
func tryReadEmbeddedFile(name string) (string, error) {
	data, err := fs.ReadFile(data.Genres, name)
	if err != nil {
		return "", nil
	}
	return string(data), nil
}

// listEmbeddedGenres lists all .md files from the embedded genres filesystem
func listEmbeddedGenres() ([]string, error) {
	entries, err := fs.ReadDir(data.Genres, ".")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			names = append(names, strings.TrimSuffix(entry.Name(), ".md"))
		}
	}
	return names, nil
}

func readDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	return names, nil
}

func parseGenreProfile(raw string) (*ParsedGenreProfile, error) {
	// Simple parsing - extract name from YAML frontmatter
	lines := strings.Split(raw, "\n")
	var profile GenreProfile
	var bodyLines []string
	inFrontmatter := false
	frontmatterEnded := false

	for _, line := range lines {
		if !frontmatterEnded {
			if strings.TrimSpace(line) == "---" {
				if !inFrontmatter {
					inFrontmatter = true
				} else {
					frontmatterEnded = true
				}
				continue
			}
			if inFrontmatter {
				if strings.HasPrefix(line, "name:") {
					profile.Name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
				}
				if strings.HasPrefix(line, "language:") {
					profile.Language = strings.TrimSpace(strings.TrimPrefix(line, "language:"))
				}
				if strings.HasPrefix(line, "description:") {
					profile.Description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
				}
			}
		} else {
			bodyLines = append(bodyLines, line)
		}
	}

	return &ParsedGenreProfile{
		Profile: profile,
		Body:    strings.Join(bodyLines, "\n"),
	}, nil
}

func parseBookRules(raw string) (*ParsedBookRules, error) {
	// Simple parsing - extract YAML frontmatter as rules
	lines := strings.Split(raw, "\n")
	var frontmatterLines []string
	inFrontmatter := false
	frontmatterEnded := false
	var bodyLines []string

	for _, line := range lines {
		if !frontmatterEnded {
			if strings.TrimSpace(line) == "---" {
				if !inFrontmatter {
					inFrontmatter = true
				} else {
					frontmatterEnded = true
				}
				continue
			}
			if inFrontmatter {
				frontmatterLines = append(frontmatterLines, line)
			}
		} else {
			bodyLines = append(bodyLines, line)
		}
	}

	// Parse frontmatter as simple key-value
	rules := make(map[string]any)
	for _, line := range frontmatterLines {
		if idx := strings.Index(line, ":"); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])
			rules[key] = value
		}
	}

	return &ParsedBookRules{
		Rules: rules,
		Raw:   raw,
	}, nil
}
