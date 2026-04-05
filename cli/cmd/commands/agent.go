package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/icosmos-space/ipen/core/pipeline"
	"github.com/spf13/cobra"
)

// AgentCommand 进入自然语言智能体模式（LLM 通过 tool-use 编排任务）。
func AgentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "agent <instruction>",
		Short:   T(TR.CmdAgentShort),
		Long:    T(TR.CmdAgentLong),
		Args:    cobra.ExactArgs(1),
		Example: "  ipen agent \"Write one chapter for my active book\" --max-turns 10",
		RunE:    runAgent,
	}

	cmd.Flags().String("context", "", "Additional context text")
	cmd.Flags().String("context-file", "", "Read additional context from file")
	cmd.Flags().Int("max-turns", 20, "Maximum tool-use turns")
	cmd.Flags().Bool("json", false, "Output JSON")
	cmd.Flags().Bool("quiet", false, "Suppress tool logs")
	return cmd
}

func runAgent(cmd *cobra.Command, args []string) error {
	root, err := findProjectRoot()
	if err != nil {
		return err
	}
	config, err := loadConfig(root)
	if err != nil {
		return err
	}

	instruction := strings.TrimSpace(args[0])
	contextText, err := resolveContextInput(cmd)
	if err != nil {
		return err
	}
	if contextText != "" {
		instruction += "\n\nAdditional context:\n" + contextText
	}

	maxTurns, _ := cmd.Flags().GetInt("max-turns")
	asJSON, _ := cmd.Flags().GetBool("json")
	quiet, _ := cmd.Flags().GetBool("quiet")

	options := &pipeline.AgentLoopOptions{
		MaxTurns: maxTurns,
	}
	if !asJSON && !quiet {
		options.OnToolCall = func(name string, args map[string]any) {
			data, _ := json.Marshal(args)
			fmt.Printf("[tool] %s(%s)\n", name, string(data))
		}
		options.OnToolResult = func(name string, result string) {
			preview := strings.TrimSpace(result)
			if len(preview) > 220 {
				preview = preview[:220] + "..."
			}
			fmt.Printf("[result] %s => %s\n", name, preview)
		}
		options.OnMessage = func(content string) {
			if strings.TrimSpace(content) != "" {
				fmt.Printf("\n%s\n", strings.TrimSpace(content))
			}
		}
	}

	result, err := pipeline.RunAgentLoop(
		context.Background(),
		buildPipelineConfig(config, root, quiet),
		instruction,
		options,
	)
	if err != nil {
		return err
	}

	if asJSON {
		data, _ := json.MarshalIndent(map[string]any{
			"result": result,
		}, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if quiet && strings.TrimSpace(result) != "" {
		fmt.Println(result)
	}
	return nil
}
