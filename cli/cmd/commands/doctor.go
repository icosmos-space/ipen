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

const OK = "✅->"
const NOT_OK = "⚠️->"

// DoctorCommand 检查环境与项目健康状态。
func DoctorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: T(TR.CmdDoctorShort),
		Long:  T(TR.CmdDoctorLong),
		RunE:  runDoctor,
	}
	cmd.Flags().Bool("check-api", false, "检查 LLM 连接状态")
	return cmd
}

func runDoctor(cmd *cobra.Command, args []string) error {
	root, err := findProjectRoot()
	if err != nil {
		return err
	}
	checkAPI, _ := cmd.Flags().GetBool("check-api")

	checks := make([]doctorCheck, 0, 12)

	if _, err = os.Stat(filepath.Join(root, "ipen.json")); err == nil {
		checks = append(checks, doctorCheck{Name: "ipen.json", OK: true, Detail: "找到"})
	} else {
		checks = append(checks, doctorCheck{Name: "ipen.json", OK: false, Detail: "未找到. 运行 ipen init"})
	}

	if _, err = os.Stat(filepath.Join(root, ".env")); err == nil {
		checks = append(checks, doctorCheck{Name: ".env", OK: true, Detail: "找到"})
	} else {
		checks = append(checks, doctorCheck{Name: ".env", OK: false, Detail: "未找到"})
	}

	globalConfigured := false
	if raw, errr := os.ReadFile(coreutils.GlobalEnvPath); errr == nil {
		text := string(raw)
		globalConfigured = strings.Contains(text, "IPEN_LLM_API_KEY=") && !strings.Contains(text, "your-api-key-here")
	}
	checks = append(checks, doctorCheck{
		Name: "全局配置",
		OK:   globalConfigured,
		Detail: map[bool]string{
			true:  "找到 (" + coreutils.GlobalEnvPath + ")",
			false: "未设置. 运行 ipen config set-global",
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
		Name:   "LLM API 密钥",
		OK:     hasAPIKey,
		Detail: map[bool]string{true: "已配置", false: "缺失"}[hasAPIKey],
	})

	if strings.TrimSpace(cfg.Provider) != "" || strings.TrimSpace(cfg.Model) != "" || strings.TrimSpace(cfg.BaseURL) != "" {
		checks = append(checks, doctorCheck{
			Name:   "LLM 配置",
			OK:     strings.TrimSpace(cfg.Model) != "",
			Detail: fmt.Sprintf("provider=%s model=%s baseUrl=%s", cfg.Provider, cfg.Model, cfg.BaseURL),
		})
	} else if cfgErr != nil {
		checks = append(checks, doctorCheck{
			Name:   "LLM 配置",
			OK:     false,
			Detail: cfgErr.Error(),
		})
	}

	sm := state.NewStateManager(root)
	books, _ := sm.ListBooks()
	checks = append(checks, doctorCheck{
		Name:   "书籍",
		OK:     true,
		Detail: fmt.Sprintf("%d 本书籍", len(books)),
	})

	legacyCount := 0
	for _, bookID := range books {
		if hint := getLegacyMigrationHint(root, bookID); hint != "" {
			legacyCount++
		}
	}
	if legacyCount > 0 {
		checks = append(checks, doctorCheck{
			Name:   "版本迁移检查",
			OK:     false,
			Detail: fmt.Sprintf("%d 本书籍仍使用旧状态布局", legacyCount),
		})
	} else if len(books) > 0 {
		checks = append(checks, doctorCheck{
			Name:   "版本迁移检查",
			OK:     true,
			Detail: "所有书籍都使用当前状态布局",
		})
	}

	if checkAPI {
		if !configLoaded {
			checks = append(checks, doctorCheck{
				Name:   "API连接状态",
				OK:     false,
				Detail: "项目配置缺失，无法测试 API连接状态",
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
					Name:   "API连接状态",
					OK:     false,
					Detail: err.Error(),
				})
			} else {
				checks = append(checks, doctorCheck{
					Name:   "API连接状态",
					OK:     true,
					Detail: fmt.Sprintf("OK (%d tokens)", resp.Usage.TotalTokens),
				})
			}
		}
	}

	fmt.Println("\tiPen Doctor")
	fmt.Println()
	failed := 0
	for _, check := range checks {
		icon := OK
		if !check.OK {
			icon = NOT_OK
			failed++
		}
		fmt.Printf("  %s %s: %s\n", icon, check.Name, check.Detail)
	}
	fmt.Println()
	if failed > 0 {
		fmt.Printf("找到 %d 异常.\n", failed)
	} else {
		fmt.Println("所有检查均通过")
	}
	return nil
}
