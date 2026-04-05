package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/icosmos-space/ipen/core/models"
	coreutils "github.com/icosmos-space/ipen/core/utils"
	"github.com/spf13/cobra"
)

// InitCommand 初始化 iPen 项目（默认当前目录）。
func InitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [name]",
		Short: T(TR.CmdInitShort),
		Long:  T(TR.CmdInitLong),
		Args:  cobra.MaximumNArgs(1),
		RunE:  runInit,
	}

	cmd.Flags().StringP("lang", "g", "zh", "Default writing language: zh or en")
	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	lang, _ := cmd.Flags().GetString("lang")
	if lang != "zh" && lang != "en" {
		return fmt.Errorf("unsupported --lang %q, must be zh or en", lang)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	projectDir := cwd
	if len(args) > 0 {
		projectDir = filepath.Join(cwd, args[0])
	}
	projectDir, err = filepath.Abs(projectDir)
	if err != nil {
		return err
	}

	projectName := filepath.Base(projectDir)
	configPath := filepath.Join(projectDir, "ipen.json")

	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("ipen.json already exists in %s", projectDir)
	}

	if err := os.MkdirAll(filepath.Join(projectDir, "books"), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(projectDir, "radar"), 0755); err != nil {
		return err
	}

	config := models.DefaultProjectConfig()
	config.Name = projectName
	config.Language = lang
	config.LLM.Provider = firstNonEmpty(os.Getenv("IPEN_LLM_PROVIDER"), "openai")
	config.LLM.BaseURL = os.Getenv("IPEN_LLM_BASE_URL")
	config.LLM.Model = os.Getenv("IPEN_LLM_MODEL")

	configMap := map[string]any{
		"name":     config.Name,
		"version":  config.Version,
		"language": config.Language,
		"llm": map[string]any{
			"provider": config.LLM.Provider,
			"baseUrl":  config.LLM.BaseURL,
			"model":    config.LLM.Model,
		},
		"notify": []any{},
		"daemon": map[string]any{
			"schedule": map[string]any{
				"radarCron": "0 */6 * * *",
				"writeCron": "*/15 * * * *",
			},
			"maxConcurrentBooks": 3,
		},
	}

	if err := writeJSONMap(configPath, configMap); err != nil {
		return err
	}

	_ = os.WriteFile(filepath.Join(projectDir, ".nvmrc"), []byte("22\n"), 0644)
	_ = os.WriteFile(filepath.Join(projectDir, ".node-version"), []byte("22\n"), 0644)

	if err := os.WriteFile(filepath.Join(projectDir, ".env"), []byte(defaultProjectEnv(hasGlobalConfig())), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(projectDir, ".gitignore"), []byte(".env\nnode_modules/\n.DS_Store\n"), 0644); err != nil {
		return err
	}

	fmt.Printf("Project initialized at %s\n\n", projectDir)
	if hasGlobalConfig() {
		fmt.Println("Global LLM config detected. Ready to go!")
	} else {
		fmt.Println("Next steps:")
		fmt.Println("  ipen config set-global --provider openai --base-url <api-url> --api-key <key> --model <model>")
		fmt.Println("  # or edit .env in this project")
	}
	if len(args) > 0 {
		fmt.Printf("  cd %s\n", args[0])
	}
	if lang == "en" {
		fmt.Println("  ipen book create --title \"My Novel\" --genre progression --platform qidian --lang en")
	} else {
		fmt.Println("  ipen book create --title \"我的小说\" --genre xuanhuan --platform tomato --lang zh")
	}
	fmt.Println("  ipen write next <book-id>")

	return nil
}

func hasGlobalConfig() bool {
	content, err := os.ReadFile(coreutils.GlobalEnvPath)
	if err != nil {
		return false
	}
	text := string(content)
	return strings.Contains(text, "IPEN_LLM_API_KEY=") && !strings.Contains(text, "your-api-key-here")
}

func defaultProjectEnv(globalExists bool) string {
	if globalExists {
		return strings.Join([]string{
			"# Project-level LLM overrides (optional)",
			"# Global config at ~/.ipen/.env will be used by default.",
			"# Uncomment below to override only for this project:",
			"# IPEN_LLM_PROVIDER=openai",
			"# IPEN_LLM_BASE_URL=",
			"# IPEN_LLM_API_KEY=",
			"# IPEN_LLM_MODEL=",
			"",
			"# Optional web search key:",
			"# TAVILY_API_KEY=tvly-xxxxx",
			"",
		}, "\n")
	}

	return strings.Join([]string{
		"# LLM Configuration",
		"# Tip: run `ipen config set-global` once for all projects.",
		"IPEN_LLM_PROVIDER=openai",
		"IPEN_LLM_BASE_URL=",
		"IPEN_LLM_API_KEY=",
		"IPEN_LLM_MODEL=",
		"",
		"# Optional settings:",
		"# IPEN_LLM_TEMPERATURE=0.7",
		"# IPEN_LLM_MAX_TOKENS=8192",
		"# IPEN_LLM_THINKING_BUDGET=0",
		"# IPEN_LLM_API_FORMAT=chat",
		"",
	}, "\n")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
