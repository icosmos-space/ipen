package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProjectConfig_LoadsDotenvAndProjectOverrides(t *testing.T) {
	tmpDir := t.TempDir()

	configJSON := `{
  "name": "test-project",
  "version": "0.1.0",
  "language": "zh",
  "llm": {
    "provider": "openai",
    "baseUrl": "https://api.example.com/v1",
    "apiKey": "",
    "model": "base-model"
  },
  "notify": [],
  "daemon": {
    "schedule": {
      "radarCron": "0 */6 * * *",
      "writeCron": "*/15 * * * *"
    },
    "maxConcurrentBooks": 3
  }
}`
	if err := os.WriteFile(filepath.Join(tmpDir, "ipen.json"), []byte(configJSON), 0644); err != nil {
		t.Fatalf("write ipen.json: %v", err)
	}

	globalEnv := filepath.Join(tmpDir, ".global.env")
	projectEnv := filepath.Join(tmpDir, ".env")

	globalContent := `IPEN_LLM_PROVIDER=openai
IPEN_LLM_BASE_URL="https://global.example.com/v1"
IPEN_LLM_MODEL=global-model
IPEN_LLM_API_KEY="global-api-key"
`
	projectContent := `IPEN_LLM_PROVIDER=custom
IPEN_LLM_BASE_URL="http://localhost:11434/v1"
IPEN_LLM_MODEL="project-model"
IPEN_LLM_API_KEY="project-api-key"
IPEN_LLM_TEMPERATURE=0.35
IPEN_LLM_MAX_TOKENS=4096
IPEN_LLM_THINKING_BUDGET=128
`
	if err := os.WriteFile(globalEnv, []byte(globalContent), 0644); err != nil {
		t.Fatalf("write global env: %v", err)
	}
	if err := os.WriteFile(projectEnv, []byte(projectContent), 0644); err != nil {
		t.Fatalf("write project env: %v", err)
	}

	originalGlobalEnvPath := GlobalEnvPath
	GlobalEnvPath = globalEnv
	t.Cleanup(func() {
		GlobalEnvPath = originalGlobalEnvPath
	})

	cfg, err := LoadProjectConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadProjectConfig failed: %v", err)
	}

	if cfg.LLM.Provider != "custom" {
		t.Fatalf("provider mismatch: got %q", cfg.LLM.Provider)
	}
	if cfg.LLM.BaseURL != "http://localhost:11434/v1" {
		t.Fatalf("baseUrl mismatch: got %q", cfg.LLM.BaseURL)
	}
	if cfg.LLM.Model != "project-model" {
		t.Fatalf("model mismatch: got %q", cfg.LLM.Model)
	}
	if cfg.LLM.Temperature != 0.35 {
		t.Fatalf("temperature mismatch: got %v", cfg.LLM.Temperature)
	}
	if cfg.LLM.MaxTokens != 4096 {
		t.Fatalf("maxTokens mismatch: got %d", cfg.LLM.MaxTokens)
	}
	if cfg.LLM.ThinkingBudget != 128 {
		t.Fatalf("thinkingBudget mismatch: got %d", cfg.LLM.ThinkingBudget)
	}
}
