package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	coreutils "github.com/icosmos-space/ipen/core/utils"
	"github.com/spf13/cobra"
)

var knownConfigKeys = []string{
	"llm.provider",
	"llm.baseUrl",
	"llm.model",
	"llm.temperature",
	"llm.maxTokens",
	"llm.thinkingBudget",
	"llm.apiFormat",
	"llm.stream",
	"inputGovernanceMode",
	"daemon.schedule.radarCron",
	"daemon.schedule.writeCron",
	"daemon.maxConcurrentBooks",
	"daemon.chaptersPerCycle",
	"daemon.retryDelayMs",
	"daemon.cooldownAfterChapterMs",
	"daemon.maxChaptersPerDay",
}

var knownAgents = []string{
	"writer",
	"auditor",
	"reviser",
	"architect",
	"radar",
	"chapter-analyzer",
}

// ConfigCommand 管理项目配置与全局配置。
func ConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: T(TR.CmdConfigShort),
		Long:  T(TR.CmdConfigLong),
	}

	cmd.AddCommand(configSetCommand())
	cmd.AddCommand(configGetCommand())
	cmd.AddCommand(configSetGlobalCommand())
	cmd.AddCommand(configListCommand())
	cmd.AddCommand(configShowGlobalCommand())
	cmd.AddCommand(configShowCommand())
	cmd.AddCommand(configSetModelCommand())
	cmd.AddCommand(configRemoveModelCommand())
	cmd.AddCommand(configShowModelsCommand())

	return cmd
}

func configSetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: T(TR.CmdConfigSetShort),
		Long:  T(TR.CmdConfigSetLong),
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := findProjectRoot()
			if err != nil {
				return err
			}

			key := strings.TrimSpace(args[0])
			value := args[1]
			if validation := validateConfigKey(key); validation != "" {
				return fmt.Errorf("%s", validation)
			}

			configPath := filepath.Join(root, "ipen.json")
			config, err := readJSONMap(configPath)
			if err != nil {
				return fmt.Errorf("failed to read config: %w", err)
			}

			setNestedValue(config, key, coerceConfigValue(value))

			if err := writeJSONMap(configPath, config); err != nil {
				return fmt.Errorf("failed to write config: %w", err)
			}

			fmt.Printf("Set %s = %s\n", key, value)
			return nil
		},
	}
}

func configGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: T(TR.CmdConfigGetShort),
		Long:  T(TR.CmdConfigGetShort),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := findProjectRoot()
			if err != nil {
				return err
			}

			configPath := filepath.Join(root, "ipen.json")
			config, err := readJSONMap(configPath)
			if err != nil {
				return fmt.Errorf("failed to read config: %w", err)
			}

			value, ok := getNestedValue(config, args[0])
			if !ok {
				return fmt.Errorf("key %q not found", args[0])
			}

			switch typed := value.(type) {
			case map[string]any, []any:
				data, _ := json.MarshalIndent(typed, "", "  ")
				fmt.Println(string(data))
			default:
				fmt.Println(typed)
			}
			return nil
		},
	}
}

func configSetGlobalCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-global",
		Short: T(TR.CmdConfigSetShort),
		Long:  T(TR.CmdConfigSetLong),
		RunE: func(cmd *cobra.Command, args []string) error {
			provider, _ := cmd.Flags().GetString("provider")
			baseURL, _ := cmd.Flags().GetString("base-url")
			apiKey, _ := cmd.Flags().GetString("api-key")
			model, _ := cmd.Flags().GetString("model")
			temperature, _ := cmd.Flags().GetString("temperature")
			maxTokens, _ := cmd.Flags().GetString("max-tokens")
			thinkingBudget, _ := cmd.Flags().GetString("thinking-budget")
			apiFormat, _ := cmd.Flags().GetString("api-format")
			lang, _ := cmd.Flags().GetString("lang")

			if strings.TrimSpace(provider) == "" ||
				strings.TrimSpace(baseURL) == "" ||
				strings.TrimSpace(apiKey) == "" ||
				strings.TrimSpace(model) == "" {
				return fmt.Errorf("--provider, --base-url, --api-key and --model are required")
			}

			if err := os.MkdirAll(coreutils.GlobalConfigDir, 0755); err != nil {
				return err
			}

			lines := []string{
				"# iPen Global LLM Configuration",
				fmt.Sprintf("IPEN_LLM_PROVIDER=%s", provider),
				fmt.Sprintf("IPEN_LLM_BASE_URL=%s", baseURL),
				fmt.Sprintf("IPEN_LLM_API_KEY=%s", apiKey),
				fmt.Sprintf("IPEN_LLM_MODEL=%s", model),
			}
			if strings.TrimSpace(temperature) != "" {
				lines = append(lines, fmt.Sprintf("IPEN_LLM_TEMPERATURE=%s", temperature))
			}
			if strings.TrimSpace(maxTokens) != "" {
				lines = append(lines, fmt.Sprintf("IPEN_LLM_MAX_TOKENS=%s", maxTokens))
			}
			if strings.TrimSpace(thinkingBudget) != "" {
				lines = append(lines, fmt.Sprintf("IPEN_LLM_THINKING_BUDGET=%s", thinkingBudget))
			}
			if strings.TrimSpace(apiFormat) != "" {
				lines = append(lines, fmt.Sprintf("IPEN_LLM_API_FORMAT=%s", apiFormat))
			}
			if strings.TrimSpace(lang) != "" {
				lines = append(lines, fmt.Sprintf("IPEN_DEFAULT_LANGUAGE=%s", lang))
			}

			content := strings.Join(lines, "\n") + "\n"
			if err := os.WriteFile(coreutils.GlobalEnvPath, []byte(content), 0644); err != nil {
				return err
			}

			fmt.Printf("Global config saved to %s\n", coreutils.GlobalEnvPath)
			return nil
		},
	}

	cmd.Flags().String("provider", "", "LLM provider (openai / anthropic)")
	cmd.Flags().String("base-url", "", "API base URL")
	cmd.Flags().String("api-key", "", "API key")
	cmd.Flags().String("model", "", "Model name")
	cmd.Flags().String("temperature", "", "Temperature")
	cmd.Flags().String("max-tokens", "", "Max output tokens")
	cmd.Flags().String("thinking-budget", "", "Anthropic thinking budget")
	cmd.Flags().String("api-format", "", "API format (chat / responses)")
	cmd.Flags().String("lang", "", "Default language (zh / en)")

	return cmd
}

func configListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all project config keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := findProjectRoot()
			if err != nil {
				return err
			}

			config, err := readJSONMap(filepath.Join(root, "ipen.json"))
			if err != nil {
				return fmt.Errorf("failed to read config: %w", err)
			}

			items := flattenConfig(config, "")
			sort.Strings(items)
			for _, item := range items {
				fmt.Println(item)
			}
			return nil
		},
	}
}

func configShowGlobalCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show-global",
		Short: "Show global LLM config",
		Long:  "Show global LLM config from ~/.ipen/.env",
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := os.ReadFile(coreutils.GlobalEnvPath)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("No global config found. Run `ipen config set-global` first.")
					return nil
				}
				return err
			}

			fmt.Print(maskAPIKey(string(content)))
			return nil
		},
	}
}

func configShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current project configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := findProjectRoot()
			if err != nil {
				return err
			}

			config, err := readJSONMap(filepath.Join(root, "ipen.json"))
			if err != nil {
				return fmt.Errorf("failed to read config: %w", err)
			}

			if llmCfg, ok := config["llm"].(map[string]any); ok {
				if key, ok := llmCfg["apiKey"].(string); ok && key != "" {
					if len(key) > 12 {
						llmCfg["apiKey"] = key[:8] + "..." + key[len(key)-4:]
					} else {
						llmCfg["apiKey"] = "********"
					}
				}
			}

			data, _ := json.MarshalIndent(config, "", "  ")
			fmt.Println(string(data))
			return nil
		},
	}
}

func configSetModelCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-model <agent> <model>",
		Short: "Set model override for one agent",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			agent := args[0]
			model := args[1]
			if !containsString(knownAgents, agent) {
				return fmt.Errorf("unknown agent %q. valid: %s", agent, strings.Join(knownAgents, ", "))
			}

			baseURL, _ := cmd.Flags().GetString("base-url")
			provider, _ := cmd.Flags().GetString("provider")
			apiKeyEnv, _ := cmd.Flags().GetString("api-key-env")
			noStream, _ := cmd.Flags().GetBool("no-stream")

			if apiKeyEnv != "" && !regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString(apiKeyEnv) {
				return fmt.Errorf("--api-key-env must be an environment variable name")
			}

			root, err := findProjectRoot()
			if err != nil {
				return err
			}

			configPath := filepath.Join(root, "ipen.json")
			config, err := readJSONMap(configPath)
			if err != nil {
				return fmt.Errorf("failed to read config: %w", err)
			}

			overrides, _ := config["modelOverrides"].(map[string]any)
			if overrides == nil {
				overrides = map[string]any{}
			}

			hasRouting := strings.TrimSpace(baseURL) != "" ||
				strings.TrimSpace(provider) != "" ||
				strings.TrimSpace(apiKeyEnv) != "" ||
				noStream

			if !hasRouting {
				overrides[agent] = model
			} else {
				override := map[string]any{
					"model": model,
				}
				if baseURL != "" {
					override["baseUrl"] = baseURL
				}
				if provider != "" {
					override["provider"] = provider
				}
				if apiKeyEnv != "" {
					override["apiKeyEnv"] = apiKeyEnv
				}
				if noStream {
					override["stream"] = false
				}
				overrides[agent] = override
			}

			config["modelOverrides"] = overrides
			if err := writeJSONMap(configPath, config); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			fmt.Printf("Model override set: %s -> %s\n", agent, model)
			return nil
		},
	}

	cmd.Flags().String("base-url", "", "API base URL")
	cmd.Flags().String("provider", "", "Provider override")
	cmd.Flags().String("api-key-env", "", "API key env var name")
	cmd.Flags().Bool("stream", true, "Enable streaming")
	cmd.Flags().Bool("no-stream", false, "Disable streaming")

	return cmd
}

func configRemoveModelCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove-model <agent>",
		Short: "Remove model override for one agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agent := args[0]
			root, err := findProjectRoot()
			if err != nil {
				return err
			}

			configPath := filepath.Join(root, "ipen.json")
			config, err := readJSONMap(configPath)
			if err != nil {
				return fmt.Errorf("failed to read config: %w", err)
			}

			overrides, _ := config["modelOverrides"].(map[string]any)
			if overrides == nil {
				fmt.Printf("No model override for %q.\n", agent)
				return nil
			}

			if _, ok := overrides[agent]; !ok {
				fmt.Printf("No model override for %q.\n", agent)
				return nil
			}

			delete(overrides, agent)
			if len(overrides) == 0 {
				delete(config, "modelOverrides")
			} else {
				config["modelOverrides"] = overrides
			}

			if err := writeJSONMap(configPath, config); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			fmt.Printf("Removed model override for %s\n", agent)
			return nil
		},
	}
}

func configShowModelsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-models",
		Short: "Show model routing for all agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			asJSON, _ := cmd.Flags().GetBool("json")

			root, err := findProjectRoot()
			if err != nil {
				return err
			}

			config, err := readJSONMap(filepath.Join(root, "ipen.json"))
			if err != nil {
				return fmt.Errorf("failed to read config: %w", err)
			}

			defaultModel := "(not set)"
			if value, ok := getNestedValue(config, "llm.model"); ok {
				defaultModel = fmt.Sprintf("%v", value)
			}

			overrides, _ := config["modelOverrides"].(map[string]any)
			if overrides == nil {
				overrides = map[string]any{}
			}

			if asJSON {
				payload := map[string]any{
					"defaultModel": defaultModel,
					"overrides":    overrides,
				}
				data, _ := json.MarshalIndent(payload, "", "  ")
				fmt.Println(string(data))
				return nil
			}

			fmt.Printf("Default model: %s\n", defaultModel)
			if len(overrides) == 0 {
				fmt.Println("No agent-specific overrides.")
				return nil
			}

			fmt.Println("Agent overrides:")
			agents := make([]string, 0, len(overrides))
			for agent := range overrides {
				agents = append(agents, agent)
			}
			sort.Strings(agents)

			for _, agent := range agents {
				switch typed := overrides[agent].(type) {
				case string:
					fmt.Printf("  %s -> %s\n", agent, typed)
				case map[string]any:
					parts := []string{fmt.Sprintf("%v", typed["model"])}
					if baseURL, ok := typed["baseUrl"]; ok {
						parts = append(parts, fmt.Sprintf("@ %v", baseURL))
					}
					if stream, ok := typed["stream"].(bool); ok && !stream {
						parts = append(parts, "[no-stream]")
					}
					fmt.Printf("  %s -> %s\n", agent, strings.Join(parts, " "))
				default:
					fmt.Printf("  %s -> %v\n", agent, typed)
				}
			}

			return nil
		},
	}

	cmd.Flags().Bool("json", false, "Output as JSON")
	return cmd
}

func validateConfigKey(key string) string {
	if strings.HasPrefix(key, "llm.extra.") {
		return ""
	}
	if containsString(knownConfigKeys, key) {
		return ""
	}

	parts := strings.Split(key, ".")
	last := parts[len(parts)-1]

	samePrefix := []string{}
	for _, candidate := range knownConfigKeys {
		cParts := strings.Split(candidate, ".")
		if len(cParts) != len(parts) {
			continue
		}
		if strings.Join(cParts[:len(cParts)-1], ".") == strings.Join(parts[:len(parts)-1], ".") {
			samePrefix = append(samePrefix, candidate)
		}
	}

	best := ""
	bestDistance := 999
	for _, candidate := range samePrefix {
		cLast := strings.Split(candidate, ".")
		distance := levenshtein(last, cLast[len(cLast)-1])
		if distance < bestDistance {
			best = candidate
			bestDistance = distance
		}
	}

	if best != "" && bestDistance <= 3 {
		return fmt.Sprintf("unknown config key %q. did you mean %q?", key, best)
	}
	return fmt.Sprintf("unknown config key %q", key)
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func flattenConfig(value map[string]any, prefix string) []string {
	lines := []string{}
	keys := make([]string, 0, len(value))
	for k := range value {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch typed := value[key].(type) {
		case map[string]any:
			lines = append(lines, flattenConfig(typed, fullKey)...)
		default:
			lines = append(lines, fmt.Sprintf("%s=%v", fullKey, typed))
		}
	}
	return lines
}
