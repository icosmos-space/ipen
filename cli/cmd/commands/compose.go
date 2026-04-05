package commands

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/icosmos-space/ipen/core/agents"
	"github.com/icosmos-space/ipen/core/state"
	"github.com/spf13/cobra"
)

// ComposeCommand 生成章节运行时产物（context/rule-stack/trace）。
func ComposeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compose [book-id]",
		Short: T(TR.CmdComposeShort),
		Long:  T(TR.CmdComposeLong),
		Args:  cobra.MaximumNArgs(1),
		RunE:  runComposeChapter,
	}
	cmd.Flags().String("context", "", "Chapter guidance context")
	cmd.Flags().String("context-file", "", "Read guidance from file")
	cmd.Flags().Bool("json", false, "Output JSON")
	cmd.Flags().BoolP("quiet", "q", false, "Suppress progress output")

	chapterCmd := &cobra.Command{
		Use:   "chapter [book-id]",
		Short: T(TR.CmdComposeShort),
		Long:  T(TR.CmdComposeLong),
		Args:  cobra.MaximumNArgs(1),
		RunE:  runComposeChapter,
	}
	chapterCmd.Flags().String("context", "", "Chapter guidance context")
	chapterCmd.Flags().String("context-file", "", "Read guidance from file")
	chapterCmd.Flags().Bool("json", false, "Output JSON")
	chapterCmd.Flags().BoolP("quiet", "q", false, "Suppress progress output")
	cmd.AddCommand(chapterCmd)

	return cmd
}

func runComposeChapter(cmd *cobra.Command, args []string) error {
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

	agentCtx := agents.AgentContext{
		Client:      buildPipelineConfig(config, root, quiet).Client,
		Model:       config.LLM.Model,
		ProjectRoot: root,
		BookID:      bookID,
		Logger:      buildLogger(quiet),
	}

	planner := agents.NewPlannerAgent(agentCtx)
	planResult, err := planner.PlanChapter(context.Background(), agents.PlanChapterInput{
		Book:            book,
		BookDir:         bookDir,
		ChapterNumber:   chapterNumber,
		ExternalContext: contextText,
	})
	if err != nil {
		return err
	}

	composer := agents.NewComposerAgent(agentCtx)
	composeResult, err := composer.ComposeChapter(context.Background(), agents.ComposeChapterInput{
		Book:          book,
		BookDir:       bookDir,
		ChapterNumber: chapterNumber,
		Plan:          *planResult,
	})
	if err != nil {
		return err
	}

	intentPath, err := writeRuntimeArtifact(bookDir, planResult.RuntimePath, []byte(planResult.IntentMarkdown))
	if err != nil {
		return err
	}
	contextPayload, _ := json.MarshalIndent(composeResult.ContextPackage, "", "  ")
	contextPath, err := writeRuntimeArtifact(bookDir, composeResult.ContextPath, contextPayload)
	if err != nil {
		return err
	}
	rulePayload, _ := json.MarshalIndent(composeResult.RuleStack, "", "  ")
	ruleStackPath, err := writeRuntimeArtifact(bookDir, composeResult.RuleStackPath, rulePayload)
	if err != nil {
		return err
	}
	tracePayload, _ := json.MarshalIndent(composeResult.Trace, "", "  ")
	tracePath, err := writeRuntimeArtifact(bookDir, composeResult.TracePath, tracePayload)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"bookId":        bookID,
		"chapterNumber": chapterNumber,
		"intentPath":    intentPath,
		"contextPath":   contextPath,
		"ruleStackPath": ruleStackPath,
		"tracePath":     tracePath,
	}

	if asJSON {
		data, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Composed chapter %d for %q\n", chapterNumber, bookID)
	fmt.Printf("Intent: %s\n", intentPath)
	fmt.Printf("Context: %s\n", contextPath)
	fmt.Printf("Rule stack: %s\n", ruleStackPath)
	fmt.Printf("Trace: %s\n", tracePath)
	return nil
}
