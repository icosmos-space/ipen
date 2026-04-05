package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/icosmos-space/ipen/core/llm"
	"github.com/icosmos-space/ipen/core/state"
	coreutils "github.com/icosmos-space/ipen/core/utils"
	"github.com/spf13/cobra"
)

type doctorCheck struct {
	Name   string
	OK     bool
	Detail string
}

// DoctorCommand 检查环境与项目健康状态。
func DoctorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: T(TR.CmdDoctorShort),
		Long:  T(TR.CmdDoctorLong),
		RunE:  runDoctor,
	}
	cmd.Flags().Bool("repair-node-runtime", false, "Write .nvmrc and .node-version pinned to Node 22")
	cmd.Flags().Bool("check-api", false, "Run a live LLM connectivity check")
	return cmd
}

func runDoctor(cmd *cobra.Command, args []string) error {
	root, err := findProjectRoot()
	if err != nil {
		return err
	}
	repairNode, _ := cmd.Flags().GetBool("repair-node-runtime")
	checkAPI, _ := cmd.Flags().GetBool("check-api")

	checks := make([]doctorCheck, 0, 12)

	if repairNode {
		_ = os.WriteFile(filepath.Join(root, ".nvmrc"), []byte("22\n"), 0644)
		_ = os.WriteFile(filepath.Join(root, ".node-version"), []byte("22\n"), 0644)
		checks = append(checks, doctorCheck{
			Name:   "Node runtime pin files",
			OK:     true,
			Detail: "Wrote .nvmrc and .node-version (22)",
		})
	}

	if _, err := os.Stat(filepath.Join(root, "ipen.json")); err == nil {
		checks = append(checks, doctorCheck{Name: "ipen.json", OK: true, Detail: "Found"})
	} else {
		checks = append(checks, doctorCheck{Name: "ipen.json", OK: false, Detail: "Not found. Run `ipen init`"})
	}

	if _, err := os.Stat(filepath.Join(root, ".env")); err == nil {
		checks = append(checks, doctorCheck{Name: ".env", OK: true, Detail: "Found"})
	} else {
		checks = append(checks, doctorCheck{Name: ".env", OK: false, Detail: "Not found"})
	}

	globalConfigured := false
	if raw, err := os.ReadFile(coreutils.GlobalEnvPath); err == nil {
		text := string(raw)
		globalConfigured = strings.Contains(text, "IPEN_LLM_API_KEY=") && !strings.Contains(text, "your-api-key-here")
	}
	checks = append(checks, doctorCheck{
		Name: "Global Config",
		OK:   globalConfigured,
		Detail: map[bool]string{
			true:  "Found (" + coreutils.GlobalEnvPath + ")",
			false: "Not set. Run `ipen config set-global`",
		}[globalConfigured],
	})

	var configLoaded bool
	var cfgErr error
	var cfg = struct {
		Provider string
		BaseURL  string
		Model    string
		APIKey   string
	}{
		Provider: os.Getenv("IPEN_LLM_PROVIDER"),
		BaseURL:  os.Getenv("IPEN_LLM_BASE_URL"),
		Model:    os.Getenv("IPEN_LLM_MODEL"),
		APIKey:   os.Getenv("IPEN_LLM_API_KEY"),
	}
	projectConfig, err := loadConfig(root)
	if err == nil {
		configLoaded = true
		cfg.Provider = projectConfig.LLM.Provider
		cfg.BaseURL = projectConfig.LLM.BaseURL
		cfg.Model = projectConfig.LLM.Model
		cfg.APIKey = projectConfig.LLM.APIKey
	} else {
		cfgErr = err
	}

	apiKeyOptional := coreutils.IsApiKeyOptionalForEndpoint(cfg.Provider, cfg.BaseURL)
	hasAPIKey := apiKeyOptional || strings.TrimSpace(cfg.APIKey) != ""
	checks = append(checks, doctorCheck{
		Name:   "LLM API Key",
		OK:     hasAPIKey,
		Detail: map[bool]string{true: "Configured", false: "Missing"}[hasAPIKey],
	})

	if strings.TrimSpace(cfg.Provider) != "" || strings.TrimSpace(cfg.Model) != "" || strings.TrimSpace(cfg.BaseURL) != "" {
		checks = append(checks, doctorCheck{
			Name:   "LLM Config",
			OK:     strings.TrimSpace(cfg.Model) != "",
			Detail: fmt.Sprintf("provider=%s model=%s baseUrl=%s", cfg.Provider, cfg.Model, cfg.BaseURL),
		})
	} else if cfgErr != nil {
		checks = append(checks, doctorCheck{
			Name:   "LLM Config",
			OK:     false,
			Detail: cfgErr.Error(),
		})
	}

	sm := state.NewStateManager(root)
	books, _ := sm.ListBooks()
	checks = append(checks, doctorCheck{
		Name:   "Books",
		OK:     true,
		Detail: fmt.Sprintf("%d book(s)", len(books)),
	})

	legacyCount := 0
	for _, bookID := range books {
		if hint := getLegacyMigrationHint(root, bookID); hint != "" {
			legacyCount++
		}
	}
	if legacyCount > 0 {
		checks = append(checks, doctorCheck{
			Name:   "Version Migration",
			OK:     false,
			Detail: fmt.Sprintf("%d book(s) still use legacy state layout", legacyCount),
		})
	} else if len(books) > 0 {
		checks = append(checks, doctorCheck{
			Name:   "Version Migration",
			OK:     true,
			Detail: "All books use current state layout",
		})
	}

	if checkAPI {
		if !configLoaded {
			checks = append(checks, doctorCheck{
				Name:   "API Connectivity",
				OK:     false,
				Detail: "Project config is missing, cannot test API connectivity",
			})
		} else {
			client := llm.NewLLMClient(projectConfig.LLM)
			resp, err := llm.ChatCompletion(
				context.Background(),
				client,
				projectConfig.LLM.Model,
				[]llm.LLMMessage{{Role: "user", Content: "Say OK"}},
				&llm.ChatOptions{MaxTokens: 16, Temperature: 0},
			)
			if err != nil {
				checks = append(checks, doctorCheck{
					Name:   "API Connectivity",
					OK:     false,
					Detail: err.Error(),
				})
			} else {
				checks = append(checks, doctorCheck{
					Name:   "API Connectivity",
					OK:     true,
					Detail: fmt.Sprintf("OK (%d tokens)", resp.Usage.TotalTokens),
				})
			}
		}
	}

	fmt.Println("iPen Doctor")
	fmt.Println()
	failed := 0
	for _, check := range checks {
		icon := "[OK]"
		if !check.OK {
			icon = "[!!]"
			failed++
		}
		fmt.Printf("  %s %s: %s\n", icon, check.Name, check.Detail)
	}
	fmt.Println()
	if failed > 0 {
		fmt.Printf("%d issue(s) found.\n", failed)
	} else {
		fmt.Println("All checks passed.")
	}
	return nil
}
