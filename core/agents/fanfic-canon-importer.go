package agents

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/icosmos-space/ipen/core/llm"
	"github.com/icosmos-space/ipen/core/models"
)

// FanficCanonOutput 是the structured canon extraction package。
type FanficCanonOutput struct {
	WorldRules        string `json:"worldRules"`
	CharacterProfiles string `json:"characterProfiles"`
	KeyEvents         string `json:"keyEvents"`
	PowerSystem       string `json:"powerSystem"`
	WritingStyle      string `json:"writingStyle"`
	FullDocument      string `json:"fullDocument"`
}

var fanficModeLabels = map[models.FanficMode]string{
	models.FanficModeCanon: "canon (strictly preserve source canon)",
	models.FanficModeAU:    "AU (parallel world, controlled divergences)",
	models.FanficModeOOC:   "OOC (character behavior may intentionally diverge)",
	models.FanficModeCP:    "CP (relationship-centered mode)",
}

// FanficCanonImporter 提取reusable canon notes from source text。
type FanficCanonImporter struct {
	*BaseAgent
}

// NewFanficCanonImporter 创建importer。
func NewFanficCanonImporter(ctx AgentContext) *FanficCanonImporter {
	return &FanficCanonImporter{BaseAgent: NewBaseAgent(ctx)}
}

// Name 返回agent name。
func (a *FanficCanonImporter) Name() string { return "fanfic-canon-importer" }

// ImportFromText 提取structured canon sections from source text。
func (a *FanficCanonImporter) ImportFromText(ctx context.Context, sourceText string, sourceName string, fanficMode models.FanficMode) (*FanficCanonOutput, error) {
	maxLen := 50000
	truncated := len([]rune(sourceText)) > maxLen
	text := sourceText
	runes := []rune(sourceText)
	if len(runes) > maxLen {
		text = string(runes[:maxLen])
	}

	modeLabel := fanficModeLabels[fanficMode]
	if modeLabel == "" {
		modeLabel = string(fanficMode)
	}

	systemPrompt := strings.Join([]string{
		"You are a professional fanfic canon analyst.",
		"Extract structured canon references from source material for downstream writing agents.",
		"Fanfic mode: " + modeLabel,
		"",
		"Return sections split by these exact tags:",
		"=== SECTION: world_rules ===",
		"=== SECTION: character_profiles ===",
		"=== SECTION: key_events ===",
		"=== SECTION: power_system ===",
		"=== SECTION: writing_style ===",
		"",
		"Extraction rules:",
		"- Stay faithful to source. Do not invent unsupported facts.",
		"- Use '(not mentioned in source)' when data is unavailable.",
		"- Character voice markers and speech patterns should be precise when present.",
	}, "\n")
	if truncated {
		systemPrompt += "\nNote: source is truncated due to length; extract only from the provided portion."
	}

	response, err := a.Chat(ctx, []llm.LLMMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: fmt.Sprintf("Source %q:\n\n%s", sourceName, text)},
	}, &llm.ChatOptions{MaxTokens: 8192, Temperature: 0.3})
	if err != nil {
		return nil, err
	}

	content := response.Content
	extract := func(tag string) string {
		re := regexp.MustCompile(`(?s)===\s*SECTION:\s*` + regexp.QuoteMeta(tag) + `\s*===\s*(.*?)(?=(?:\n===\s*SECTION:)|$)`)
		match := re.FindStringSubmatch(content)
		if len(match) < 2 {
			return ""
		}
		return strings.TrimSpace(match[1])
	}

	worldRules := extract("world_rules")
	characterProfiles := extract("character_profiles")
	keyEvents := extract("key_events")
	powerSystem := extract("power_system")
	writingStyle := extract("writing_style")

	meta := strings.Join([]string{
		"---",
		"meta:",
		fmt.Sprintf("  sourceFile: %q", sourceName),
		fmt.Sprintf("  fanficMode: %q", fanficMode),
		fmt.Sprintf("  generatedAt: %q", time.Now().UTC().Format(time.RFC3339)),
	}, "\n")

	fullDocument := strings.Join([]string{
		fmt.Sprintf("# Fanfic Canon (%s)", sourceName),
		"",
		"## World Rules",
		emptyFallback(worldRules, "(not mentioned in source)"),
		"",
		"## Character Profiles",
		emptyFallback(characterProfiles, "(not mentioned in source)"),
		"",
		"## Key Events Timeline",
		emptyFallback(keyEvents, "(not mentioned in source)"),
		"",
		"## Power System",
		emptyFallback(powerSystem, "(not mentioned in source)"),
		"",
		"## Source Writing Style",
		emptyFallback(writingStyle, "(not mentioned in source)"),
		"",
		meta,
	}, "\n")

	return &FanficCanonOutput{
		WorldRules:        worldRules,
		CharacterProfiles: characterProfiles,
		KeyEvents:         keyEvents,
		PowerSystem:       powerSystem,
		WritingStyle:      writingStyle,
		FullDocument:      fullDocument,
	}, nil
}

func emptyFallback(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
