package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/icosmos-space/ipen/core/agents"
	"github.com/spf13/cobra"
)

// RadarCommand 执行市场雷达扫描。
func RadarCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "radar",
		Short: T(TR.CmdRadarShort),
		Long:  T(TR.CmdRadarLong),
		RunE:  runRadarScan,
	}
	cmd.Flags().Bool("json", false, "Output JSON")

	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan market opportunities",
		RunE:  runRadarScan,
	}
	scanCmd.Flags().Bool("json", false, "Output JSON")
	cmd.AddCommand(scanCmd)
	return cmd
}

func runRadarScan(cmd *cobra.Command, args []string) error {
	root, err := findProjectRoot()
	if err != nil {
		return err
	}
	config, err := loadConfig(root)
	if err != nil {
		return err
	}
	asJSON, _ := cmd.Flags().GetBool("json")

	radarAgent := agents.NewRadarAgent(agents.AgentContext{
		Client:      buildPipelineConfig(config, root, true).Client,
		Model:       config.LLM.Model,
		ProjectRoot: root,
		Logger:      buildLogger(true),
	})
	result, err := radarAgent.Scan(context.Background())
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(root, "radar"), 0755); err != nil {
		return err
	}
	timestamp := time.Now().Format("2006-01-02T15-04-05")
	filePath := filepath.Join(root, "radar", fmt.Sprintf("scan-%s.json", timestamp))
	body, _ := json.MarshalIndent(result, "", "  ")
	if err := os.WriteFile(filePath, body, 0644); err != nil {
		return err
	}

	if asJSON {
		payload := map[string]any{
			"recommendations": result.Recommendations,
			"marketSummary":   result.MarketSummary,
			"timestamp":       result.Timestamp,
			"savedTo":         filepath.ToSlash(filePath),
		}
		data, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Market summary:\n%s\n\n", result.MarketSummary)
	fmt.Println("Recommendations:")
	for _, rec := range result.Recommendations {
		fmt.Printf("  [%.0f%%] %s/%s\n", rec.Confidence*100, rec.Platform, rec.Genre)
		fmt.Printf("    Concept: %s\n", rec.Concept)
		fmt.Printf("    Reasoning: %s\n", rec.Reasoning)
		if len(rec.BenchmarkTitles) > 0 {
			fmt.Printf("    Benchmarks: %s\n", strings.Join(rec.BenchmarkTitles, ", "))
		}
	}
	fmt.Printf("\nSaved: %s\n", filepath.ToSlash(filePath))
	return nil
}
