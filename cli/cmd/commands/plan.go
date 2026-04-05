package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/icosmos-space/ipen/core/agents"
	"github.com/icosmos-space/ipen/core/state"
	"github.com/spf13/cobra"
)

// PlanCommand 执行章节规划相关操作。
func PlanCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan",
		Short: T(TR.CmdPlanShort),
		Long:  T(TR.CmdPlanLong),
	}
	cmd.AddCommand(planChapterCommand())
	return cmd
}

func planChapterCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chapter [book-id]",
		Short: T(TR.CmdPlanChapterShort),
		Long:  T(TR.CmdPlanChapterLong),
		Args:  cobra.MaximumNArgs(1),
		RunE:  runPlanChapter,
	}
	cmd.Flags().String("context", "", "Chapter guidance context")
	cmd.Flags().String("context-file", "", "Read chapter guidance from file")
	cmd.Flags().Bool("json", false, "Output JSON")
	cmd.Flags().BoolP("quiet", "q", false, "Suppress progress output")
	return cmd
}

func runPlanChapter(cmd *cobra.Command, args []string) error {
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

	contextText, err := resolveContextInput(cmd)
	if err != nil {
		return err
	}
	asJSON, _ := cmd.Flags().GetBool("json")
	quiet, _ := cmd.Flags().GetBool("quiet")

	sm := state.NewStateManager(root)
	book, err := sm.LoadBookConfig(bookID)
	if err != nil {
		return err
	}
	chapterNumber, err := sm.GetNextChapterNumber(bookID)
	if err != nil {
		return err
	}
	bookDir := sm.BookDir(bookID)

	planner := agents.NewPlannerAgent(agents.AgentContext{
		Client:      buildPipelineConfig(config, root, quiet).Client,
		Model:       config.LLM.Model,
		ProjectRoot: root,
		BookID:      bookID,
		Logger:      buildLogger(quiet),
	})
	result, err := planner.PlanChapter(context.Background(), agents.PlanChapterInput{
		Book:            book,
		BookDir:         bookDir,
		ChapterNumber:   chapterNumber,
		ExternalContext: contextText,
	})
	if err != nil {
		return err
	}

	intentPath, err := writeRuntimeArtifact(bookDir, result.RuntimePath, []byte(result.IntentMarkdown))
	if err != nil {
		return err
	}

	payload := map[string]any{
		"bookId":        bookID,
		"chapterNumber": chapterNumber,
		"goal":          result.Intent.Goal,
		"intentPath":    intentPath,
		"conflicts":     result.Intent.Conflicts,
		"plannedAt":     time.Now().Format(time.RFC3339),
	}

	if asJSON {
		data, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Planned chapter %d for %q\n", chapterNumber, bookID)
	fmt.Printf("Goal: %s\n", result.Intent.Goal)
	fmt.Printf("Intent: %s\n", intentPath)
	if len(result.Intent.Conflicts) > 0 {
		fmt.Println("Conflicts:")
		for _, conflict := range result.Intent.Conflicts {
			detail := ""
			if conflict.Detail != nil {
				detail = *conflict.Detail
			}
			if strings.TrimSpace(detail) == "" {
				detail = conflict.Resolution
			}
			fmt.Printf("  - %s: %s\n", conflict.Type, detail)
		}
	}
	return nil
}
