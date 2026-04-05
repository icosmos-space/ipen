package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/icosmos-space/ipen/core/llm"
	"github.com/icosmos-space/ipen/core/models"
	"github.com/icosmos-space/ipen/core/pipeline"
	"github.com/icosmos-space/ipen/core/state"
	coreutils "github.com/icosmos-space/ipen/core/utils"
	"github.com/spf13/cobra"
)

func findProjectRoot() (string, error) {
	return os.Getwd()
}

func loadConfig(root string) (*models.ProjectConfig, error) {
	return coreutils.LoadProjectConfig(root)
}

func buildRunner(config *models.ProjectConfig, root string, quiet bool) *pipeline.PipelineRunner {
	return pipeline.NewPipelineRunner(buildPipelineConfig(config, root, quiet))
}

func buildLogger(quiet bool) coreutils.Logger {
	sinks := []coreutils.LogSink{&coreutils.NullSink{}}
	if !quiet {
		sinks = append(sinks, coreutils.NewStderrSink())
	}

	return coreutils.NewLogger("ipen", sinks, coreutils.InfoLevel)
}

func buildPipelineConfig(config *models.ProjectConfig, root string, quiet bool) pipeline.PipelineConfig {
	return pipeline.PipelineConfig{
		Client:              llm.NewLLMClient(config.LLM),
		Model:               config.LLM.Model,
		ProjectRoot:         root,
		DefaultLLMConfig:    &config.LLM,
		NotifyChannels:      config.Notify,
		ModelOverrides:      config.ModelOverrides,
		InputGovernanceMode: config.InputGovernanceMode,
		Logger:              buildLogger(quiet),
	}
}

func resolveBookID(root string, bookIDArg string) (string, error) {
	sm := state.NewStateManager(root)
	books, err := sm.ListBooks()
	if err != nil {
		return "", err
	}

	if bookIDArg != "" {
		for _, id := range books {
			if id == bookIDArg {
				return bookIDArg, nil
			}
		}
		available := "(none)"
		if len(books) > 0 {
			available = strings.Join(books, ", ")
		}
		return "", fmt.Errorf("book %q not found. available books: %s", bookIDArg, available)
	}

	switch len(books) {
	case 0:
		return "", fmt.Errorf("no books found. create one first with `ipen book create --title ...`")
	case 1:
		return books[0], nil
	default:
		return "", fmt.Errorf("multiple books found: %s. please specify a book id", strings.Join(books, ", "))
	}
}

func resolveContextInput(cmd *cobra.Command) (string, error) {
	contextText, _ := cmd.Flags().GetString("context")
	if strings.TrimSpace(contextText) != "" {
		return contextText, nil
	}

	contextFile, _ := cmd.Flags().GetString("context-file")
	if strings.TrimSpace(contextFile) != "" {
		data, err := os.ReadFile(filepath.Clean(contextFile))
		if err != nil {
			return "", err
		}
		return string(data), nil
	}

	stat, err := os.Stdin.Stat()
	if err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
		data, readErr := io.ReadAll(os.Stdin)
		if readErr != nil {
			return "", readErr
		}
		trimmed := strings.TrimSpace(string(data))
		if trimmed != "" {
			return trimmed, nil
		}
	}

	return "", nil
}

func parseOptionalIntFlag(cmd *cobra.Command, flagName string) (*int, error) {
	raw, _ := cmd.Flags().GetString(flagName)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid --%s value %q", flagName, raw)
	}
	return &value, nil
}

func sanitizeBookID(title string) string {
	base := strings.ToLower(strings.TrimSpace(title))
	re := regexp.MustCompile(`[^\p{Han}a-z0-9]+`)
	base = re.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")
	if base == "" {
		base = fmt.Sprintf("book-%d", time.Now().Unix())
	}

	runes := []rune(base)
	if len(runes) > 30 {
		base = strings.Trim(string(runes[:30]), "-")
	}
	if base == "" {
		base = fmt.Sprintf("book-%d", time.Now().Unix())
	}
	return base
}

func askForConfirmation(prompt string) (bool, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y", nil
}

func getLegacyMigrationHint(root, bookID string) string {
	sm := state.NewStateManager(root)
	stateDir := filepath.Join(sm.BookDir(bookID), "story", "state")
	info, err := os.Stat(stateDir)
	if err == nil && info.IsDir() {
		return ""
	}
	return fmt.Sprintf("Book %q uses a legacy state layout. The next write will auto-migrate state files.", bookID)
}

func maskAPIKey(value string) string {
	lines := strings.Split(value, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "IPEN_LLM_API_KEY=") {
			key := strings.TrimPrefix(line, "IPEN_LLM_API_KEY=")
			if len(key) <= 12 {
				lines[i] = "IPEN_LLM_API_KEY=********"
				continue
			}
			lines[i] = "IPEN_LLM_API_KEY=" + key[:8] + "..." + key[len(key)-4:]
		}
	}
	return strings.Join(lines, "\n")
}

func readJSONMap(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, err
	}
	return parsed, nil
}

func writeJSONMap(path string, value map[string]any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func getNestedValue(root map[string]any, dottedKey string) (any, bool) {
	parts := strings.Split(dottedKey, ".")
	current := any(root)
	for _, part := range parts {
		asMap, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		value, ok := asMap[part]
		if !ok {
			return nil, false
		}
		current = value
	}
	return current, true
}

func setNestedValue(root map[string]any, dottedKey string, value any) {
	parts := strings.Split(dottedKey, ".")
	current := root
	for i := 0; i < len(parts)-1; i++ {
		key := parts[i]
		next, ok := current[key]
		if !ok {
			child := map[string]any{}
			current[key] = child
			current = child
			continue
		}
		asMap, ok := next.(map[string]any)
		if !ok {
			child := map[string]any{}
			current[key] = child
			current = child
			continue
		}
		current = asMap
	}
	current[parts[len(parts)-1]] = value
}

func coerceConfigValue(raw string) any {
	trimmed := strings.TrimSpace(raw)
	if strings.EqualFold(trimmed, "true") {
		return true
	}
	if strings.EqualFold(trimmed, "false") {
		return false
	}
	if i, err := strconv.Atoi(trimmed); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return f
	}
	return raw
}

func levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	if a == "" {
		return len(b)
	}
	if b == "" {
		return len(a)
	}

	da := []rune(a)
	db := []rune(b)
	dp := make([][]int, len(da)+1)
	for i := range dp {
		dp[i] = make([]int, len(db)+1)
	}
	for i := 0; i <= len(da); i++ {
		dp[i][0] = i
	}
	for j := 0; j <= len(db); j++ {
		dp[0][j] = j
	}

	for i := 1; i <= len(da); i++ {
		for j := 1; j <= len(db); j++ {
			cost := 0
			if da[i-1] != db[j-1] {
				cost = 1
			}

			del := dp[i-1][j] + 1
			ins := dp[i][j-1] + 1
			sub := dp[i-1][j-1] + cost

			dp[i][j] = minInt(del, ins, sub)
		}
	}

	return dp[len(da)][len(db)]
}

func minInt(values ...int) int {
	if len(values) == 0 {
		return 0
	}
	best := values[0]
	for _, v := range values[1:] {
		if v < best {
			best = v
		}
	}
	return best
}
