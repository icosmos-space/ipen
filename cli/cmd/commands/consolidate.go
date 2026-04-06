package commands

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/icosmos-space/ipen/core/agents"
	"github.com/icosmos-space/ipen/core/state"
	"github.com/spf13/cobra"
)

// ConsolidateCommand 整合书籍章节摘要为卷级摘要，降低长篇上下文负担。
func ConsolidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "consolidate [book-id]",
		Short: T(TR.CmdConsolidateShort),
		Long:  T(TR.CmdConsolidateLong),
		Args:  cobra.MaximumNArgs(1),
		RunE:  runConsolidate,
	}
	cmd.Flags().Bool("json", false, "Output JSON")
	return cmd
}

func runConsolidate(cmd *cobra.Command, args []string) error {
	root, err := findProjectRoot()
	if err != nil {
		return err
	}
	config, err := loadConfig(root)
	if err != nil {
		return err
	}
	bookArg := ""
	if len(args) > 0 {
		bookArg = args[0]
	}
	bookID, err := resolveBookID(root, bookArg)
	if err != nil {
		return err
	}
	asJSON, _ := cmd.Flags().GetBool("json")

	sm := state.NewStateManager(root)
	consolidator := agents.NewConsolidatorAgent(agents.AgentContext{
		Client:      buildPipelineConfig(config, root, true).Client,
		Model:       config.LLM.Model,
		ProjectRoot: root,
		BookID:      bookID,
		Logger:      buildLogger(true),
	})
	result, err := consolidator.Consolidate(context.Background(), sm.BookDir(bookID))
	if err != nil {
		return err
	}

	if asJSON {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if result.ArchivedVolumes == 0 {
		fmt.Println("No completed volumes found to consolidate.")
		return nil
	}
	fmt.Printf("Consolidated %d volume(s).\n", result.ArchivedVolumes)
	fmt.Printf("Retained %d recent chapter summaries.\n", result.RetainedChapters)
	fmt.Println("Volume summaries: story/volume_summaries.md")
	fmt.Println("Archive dir: story/summaries_archive/")
	return nil
}
