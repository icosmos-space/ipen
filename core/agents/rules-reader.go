package agents

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
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

var builtinGenresDir = "" // Will be set during initialization

func init() {
	builtinGenresDir = resolveBuiltinGenresDir()
}

// GetBuiltinGenresDir 返回the resolved built-in genres directory。
func GetBuiltinGenresDir() string {
	return getBuiltinGenresDir()
}

func getBuiltinGenresDir() string {
	if isValidBuiltinGenresDir(builtinGenresDir) {
		return builtinGenresDir
	}
	builtinGenresDir = resolveBuiltinGenresDir()
	return builtinGenresDir
}

func resolveBuiltinGenresDir() string {
	if fromEnv := strings.TrimSpace(os.Getenv("IPEN_BUILTIN_GENRES_DIR")); fromEnv != "" {
		if isValidBuiltinGenresDir(fromEnv) {
			return filepath.Clean(fromEnv)
		}
	}

	candidates := make([]string, 0, 12)
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		candidates = append(candidates,
			filepath.Join(execDir, "genres"),
			filepath.Join(execDir, "..", "genres"),
			filepath.Join(execDir, "..", "core", "genres"),
			filepath.Join(execDir, "..", "..", "core", "genres"),
		)
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "genres"),
			filepath.Join(cwd, "core", "genres"),
			filepath.Join(cwd, "..", "core", "genres"),
			filepath.Join(cwd, "..", "..", "core", "genres"),
		)
	}
	if _, sourceFile, _, ok := runtime.Caller(0); ok {
		agentsDir := filepath.Dir(sourceFile)
		candidates = append(candidates, filepath.Join(agentsDir, "..", "genres"))
	}

	for _, dir := range candidates {
		if isValidBuiltinGenresDir(dir) {
			return filepath.Clean(dir)
		}
	}
	return ""
}

func isValidBuiltinGenresDir(dir string) bool {
	if strings.TrimSpace(dir) == "" {
		return false
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return false
	}
	if _, err := os.Stat(filepath.Join(dir, "other.md")); err != nil {
		return false
	}
	return true
}

// ReadGenreProfile 加载genre profile. Lookup order:。
// 1. Project-level: {projectRoot}/genres/{genreId}.md
// 2. Built-in:     builtin_genres_dir/{genreId}.md
// 3. Fallback:     builtin_genres_dir/other.md
func ReadGenreProfile(projectRoot, genreID string) (*ParsedGenreProfile, error) {
	projectPath := filepath.Join(projectRoot, "genres", genreID+".md")
	builtinDir := getBuiltinGenresDir()
	builtinPath := filepath.Join(builtinDir, genreID+".md")
	fallbackPath := filepath.Join(builtinDir, "other.md")

	raw, err := tryReadFile(projectPath)
	if err != nil || raw == "" {
		raw, err = tryReadFile(builtinPath)
	}
	if err != nil || raw == "" {
		raw, err = tryReadFile(fallbackPath)
	}
	if err != nil || raw == "" {
		return nil, fmt.Errorf("genre profile not found for \"%s\" and fallback \"other.md\" is missing", genreID)
	}

	return parseGenreProfile(raw)
}

// ListAvailableGenres lists all available genre profiles
func ListAvailableGenres(projectRoot string) ([]GenreInfo, error) {
	results := make(map[string]GenreInfo)

	// Built-in genres first
	builtinDir := getBuiltinGenresDir()
	if builtinDir != "" {
		builtinFiles, err := readDir(builtinDir)
		if err == nil {
			for _, file := range builtinFiles {
				if !strings.HasSuffix(file, ".md") {
					continue
				}
				id := strings.TrimSuffix(file, ".md")
				raw, err := tryReadFile(filepath.Join(builtinDir, file))
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
