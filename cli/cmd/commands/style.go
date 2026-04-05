package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/icosmos-space/ipen/core/agents"
	"github.com/icosmos-space/ipen/core/llm"
	"github.com/icosmos-space/ipen/core/models"
	"github.com/icosmos-space/ipen/core/state"
	"github.com/spf13/cobra"
)

// StyleCommand 处理风格画像分析与风格指南导入。
func StyleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "style",
		Short: T(TR.CmdStyleShort),
		Long:  T(TR.CmdStyleLong),
	}
	cmd.AddCommand(styleAnalyzeCommand())
	cmd.AddCommand(styleImportCommand())
	return cmd
}

func styleAnalyzeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze <file>",
		Short: "Analyze a reference text file into style profile",
		Args:  cobra.ExactArgs(1),
		RunE:  runStyleAnalyze,
	}
	cmd.Flags().String("name", "", "Source name")
	cmd.Flags().Bool("json", false, "Output JSON only")
	return cmd
}

func runStyleAnalyze(cmd *cobra.Command, args []string) error {
	path, err := filepath.Abs(args[0])
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	if strings.TrimSpace(name) == "" {
		name = filepath.Base(path)
	}
	asJSON, _ := cmd.Flags().GetBool("json")

	profile := agents.AnalyzeStyle(string(raw), name)
	if asJSON {
		data, _ := json.MarshalIndent(profile, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Println("Style Profile:")
	fmt.Printf("  Source: %s\n", profile.SourceName)
	fmt.Printf("  Avg sentence length: %.1f\n", profile.AvgSentenceLength)
	fmt.Printf("  Sentence stddev: %.1f\n", profile.SentenceLengthStdDev)
	fmt.Printf("  Avg paragraph length: %.0f\n", profile.AvgParagraphLength)
	fmt.Printf("  Paragraph range: %.0f - %.0f\n", profile.ParagraphLengthRange.Min, profile.ParagraphLengthRange.Max)
	fmt.Printf("  Vocabulary diversity: %.3f\n", profile.VocabularyDiversity)
	if len(profile.TopPatterns) > 0 {
		fmt.Printf("  Top patterns: %s\n", strings.Join(profile.TopPatterns, ", "))
	}
	if len(profile.RhetoricalFeatures) > 0 {
		fmt.Printf("  Rhetorical features: %s\n", strings.Join(profile.RhetoricalFeatures, ", "))
	}
	return nil
}

func styleImportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import <file> [book-id]",
		Short: "Import style profile and generate style guide for a book",
		Args:  cobra.RangeArgs(1, 2),
		RunE:  runStyleImport,
	}
	cmd.Flags().String("name", "", "Source name")
	cmd.Flags().Bool("stats-only", false, "Skip style guide generation")
	cmd.Flags().Bool("json", false, "Output JSON")
	return cmd
}

func runStyleImport(cmd *cobra.Command, args []string) error {
	root, err := findProjectRoot()
	if err != nil {
		return err
	}
	filePath, err := filepath.Abs(args[0])
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	bookArg := ""
	if len(args) > 1 {
		bookArg = args[1]
	}
	bookID, err := resolveBookID(root, bookArg)
	if err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	if strings.TrimSpace(name) == "" {
		name = filepath.Base(filePath)
	}
	statsOnly, _ := cmd.Flags().GetBool("stats-only")
	asJSON, _ := cmd.Flags().GetBool("json")

	text := string(raw)
	profile := agents.AnalyzeStyle(text, name)

	sm := state.NewStateManager(root)
	book, err := sm.LoadBookConfig(bookID)
	if err != nil {
		return err
	}
	bookDir := sm.BookDir(bookID)
	storyDir := filepath.Join(bookDir, "story")
	if err := os.MkdirAll(storyDir, 0755); err != nil {
		return err
	}

	profileBytes, _ := json.MarshalIndent(profile, "", "  ")
	if err := os.WriteFile(filepath.Join(storyDir, "style_profile.json"), profileBytes, 0644); err != nil {
		return err
	}

	styleGuideRel := ""
	if !statsOnly {
		config, err := loadConfig(root)
		if err != nil {
			return err
		}
		guide, err := generateStyleGuideWithLLM(config.LLM, text, name, book.Language)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(storyDir, "style_guide.md"), []byte(strings.TrimSpace(guide)+"\n"), 0644); err != nil {
			return err
		}
		styleGuideRel = "story/style_guide.md"
	}

	payload := map[string]any{
		"bookId":       bookID,
		"file":         filepath.ToSlash(filePath),
		"statsProfile": "story/style_profile.json",
		"styleGuide":   styleGuideRel,
	}
	if asJSON {
		data, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Style profile imported to %q.\n", bookID)
	fmt.Println("Stats profile: story/style_profile.json")
	if styleGuideRel != "" {
		fmt.Println("Style guide: story/style_guide.md")
	}
	return nil
}

func generateStyleGuideWithLLM(cfg models.LLMConfig, referenceText string, sourceName string, language string) (string, error) {
	client := llm.NewLLMClient(cfg)
	prompt := "You are a fiction style analyst. Summarize a concise style guide for imitation in markdown."
	if strings.EqualFold(language, "zh") {
		prompt = "You are a fiction style analyst. Output a concise Chinese style guide in markdown for downstream imitation."
	}

	user := fmt.Sprintf("Source: %s\n\nReference text:\n%s", sourceName, truncateForStyleGuide(referenceText, 24000))
	resp, err := llm.ChatCompletion(
		context.Background(),
		client,
		cfg.Model,
		[]llm.LLMMessage{
			{Role: "system", Content: prompt},
			{Role: "user", Content: user},
		},
		&llm.ChatOptions{
			Temperature: 0.3,
			MaxTokens:   2000,
		},
	)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Content), nil
}

func truncateForStyleGuide(text string, maxRunes int) string {
	r := []rune(text)
	if len(r) <= maxRunes {
		return text
	}
	return string(r[:maxRunes])
}
