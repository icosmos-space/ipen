package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/icosmos-space/ipen/core/models"
	"github.com/joho/godotenv"
)

// GlobalConfigDir 是the global config directory。
var GlobalConfigDir = func() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".ipen"
	}
	return filepath.Join(home, ".ipen")
}()

// GlobalEnvPath 是the global env file path。
var GlobalEnvPath = filepath.Join(GlobalConfigDir, ".env")

// LoadProjectConfig 加载project config from ipen.json with .env overrides。
func LoadProjectConfig(root string) (*models.ProjectConfig, error) {
	// Load global .env
	_ = godotenv.Overload(GlobalEnvPath)

	// Load project .env
	projectEnvPath := filepath.Join(root, ".env")
	_ = godotenv.Overload(projectEnvPath)

	// Load ipen.json
	configPath := filepath.Join(root, "ipen.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("ipen.json 未在 %s 找到", root)
	}

	var config map[string]any
	if err = json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("ipen.json 不是合法的 JSON: %w", err)
	}

	// Apply .env overrides
	applyEnvOverrides(config)

	// Parse into ProjectConfig
	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	var projectConfig models.ProjectConfig
	if err = json.Unmarshal(configJSON, &projectConfig); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Apply defaults
	if projectConfig.Version == "" {
		projectConfig.Version = "0.1.0"
	}
	if projectConfig.Language == "" {
		projectConfig.Language = "zh"
	}

	return &projectConfig, nil
}

// IsApiKeyOptionalForEndpoint 检查if API key is optional for endpoint。
func IsApiKeyOptionalForEndpoint(provider string, baseURL string) bool {
	if provider == "anthropic" {
		return false
	}
	if baseURL == "" {
		return false
	}

	hostname := strings.ToLower(baseURL)
	// Simple hostname extraction
	if idx := strings.Index(hostname, "://"); idx != -1 {
		hostname = hostname[idx+3:]
	}
	if idx := strings.Index(hostname, "/"); idx != -1 {
		hostname = hostname[:idx]
	}

	return hostname == "localhost" ||
		hostname == "127.0.0.1" ||
		hostname == "::1" ||
		hostname == "0.0.0.0" ||
		strings.HasSuffix(hostname, ".local") ||
		isPrivateIPv4(hostname)
}

func isPrivateIPv4(hostname string) bool {
	return strings.HasPrefix(hostname, "192.168.") ||
		strings.HasPrefix(hostname, "10.") ||
		strings.HasPrefix(hostname, "172.16.")
}

func applyEnvOverrides(config map[string]any) {
	env := os.Environ()

	llmConfig := make(map[string]any)
	if llm, ok := config["llm"].(map[string]any); ok {
		llmConfig = llm
	}

	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		switch key {
		case "IPEN_LLM_PROVIDER":
			llmConfig["provider"] = value
		case "IPEN_LLM_BASE_URL":
			llmConfig["baseUrl"] = value
		case "IPEN_LLM_MODEL":
			llmConfig["model"] = value
		case "IPEN_LLM_TEMPERATURE":
			if temp, err := strconv.ParseFloat(value, 64); err == nil {
				llmConfig["temperature"] = temp
			}
		case "IPEN_LLM_MAX_TOKENS":
			if tokens, err := strconv.Atoi(value); err == nil {
				llmConfig["maxTokens"] = tokens
			}
		case "IPEN_LLM_THINKING_BUDGET":
			if budget, err := strconv.Atoi(value); err == nil {
				llmConfig["thinkingBudget"] = budget
			}
		}
	}

	config["llm"] = llmConfig
}
