package agents

import (
	"fmt"
	"strings"

	"github.com/icosmos-space/ipen/core/models"
)

const codeFence = "```"

// BuildSettlerSystemPrompt 构建settle-stage system prompt。
func BuildSettlerSystemPrompt(
	book *models.BookConfig,
	genreProfile *models.GenreProfile,
	bookRules *models.BookRules,
	language string,
) string {
	isEnglish := strings.EqualFold(language, "en")
	if !isEnglish {
		isEnglish = genreProfile != nil && strings.EqualFold(genreProfile.Language, "en")
	}

	numericalBlock := "- This genre has no numerical system. Leave UPDATED_LEDGER empty."
	if genreProfile != nil && genreProfile.NumericalSystem {
		numericalBlock = "- This genre tracks numerical/resources systems. UPDATED_LEDGER must include every resource change in the chapter."
	}

	fullCastBlock := ""
	if bookRules != nil && bookRules.EnableFullCastTracking {
		fullCastBlock = "\n- Include cast appearance/relationship deltas in POST_SETTLEMENT."
	}

	if isEnglish {
		return fmt.Sprintf(`You are a state settlement analyst. Given chapter text and current truth files, produce updated truth artifacts.

Book: %s
Genre: %s
Platform: %s
%s%s

Output format (strict):
=== POST_SETTLEMENT ===
...

=== RUNTIME_STATE_DELTA ===
%sjson
{
  "chapter": 12,
  "currentStatePatch": {},
  "hookOps": {"upsert": [], "mention": [], "resolve": [], "defer": []},
  "newHookCandidates": [],
  "chapterSummary": null,
  "subplotOps": [],
  "emotionalArcOps": [],
  "characterMatrixOps": [],
  "notes": []
}
%s

Rules:
1. Output delta only; do not rewrite full files.
2. Use exact chapter numbers.
3. Existing hook updates go to hookOps.upsert/resolve/defer/mention.
4. Brand-new unresolved threads go to newHookCandidates, not fabricated hook ids.
5. If a hook is only mentioned without meaningful progress, keep it in mention only.`,
			bookTitle(book),
			genreName(genreProfile),
			bookPlatform(book),
			numericalBlock,
			fullCastBlock,
			codeFence,
			codeFence,
		)
	}

	return fmt.Sprintf(`你是状态结算分析器。给定章节正文和当前 truth files，输出更新结果。
书名：%s
题材：%s
平台：%s
%s%s

输出格式（严格）：
=== POST_SETTLEMENT ===
...

=== RUNTIME_STATE_DELTA ===
%sjson
{
  "chapter": 12,
  "currentStatePatch": {},
  "hookOps": {"upsert": [], "mention": [], "resolve": [], "defer": []},
  "newHookCandidates": [],
  "chapterSummary": null,
  "subplotOps": [],
  "emotionalArcOps": [],
  "characterMatrixOps": [],
  "notes": []
}
%s

规则：
1. 只输出增量，不重写整套文件。
2. 章节号必须是整数。
3. 已有伏笔更新放在 hookOps。
4. 全新未解线索放到 newHookCandidates，不直接发明 hookId。
5. 仅被提及但未推进的旧伏笔放到 mention。`,
		bookTitle(book),
		genreName(genreProfile),
		bookPlatform(book),
		numericalBlock,
		fullCastBlock,
		codeFence,
		codeFence,
	)
}

// SettlerUserPromptParams 分组settler user-prompt inputs。
type SettlerUserPromptParams struct {
	ChapterNumber         int
	Title                 string
	Content               string
	CurrentState          string
	Ledger                string
	Hooks                 string
	ChapterSummaries      string
	SubplotBoard          string
	EmotionalArcs         string
	CharacterMatrix       string
	VolumeOutline         string
	Observations          string
	SelectedEvidenceBlock string
	GovernedControlBlock  string
	ValidationFeedback    string
}

// BuildSettlerUserPrompt 构建settle-stage user prompt。
func BuildSettlerUserPrompt(params SettlerUserPromptParams) string {
	parts := []string{fmt.Sprintf("Please settle chapter %d: %s", params.ChapterNumber, params.Title)}
	if strings.TrimSpace(params.Observations) != "" {
		parts = append(parts, "\n## Observations\n"+params.Observations)
	}
	if strings.TrimSpace(params.ValidationFeedback) != "" {
		parts = append(parts, "\n## Validation Feedback\n"+params.ValidationFeedback)
	}
	parts = append(parts, "\n## Chapter Body\n"+params.Content)
	if strings.TrimSpace(params.GovernedControlBlock) != "" {
		parts = append(parts, "\n## Control Inputs\n"+params.GovernedControlBlock)
	}
	parts = append(parts, "\n## Current State\n"+params.CurrentState)
	if strings.TrimSpace(params.Ledger) != "" {
		parts = append(parts, "\n## Current Ledger\n"+params.Ledger)
	}
	parts = append(parts, "\n## Current Hooks\n"+params.Hooks)
	if strings.TrimSpace(params.SelectedEvidenceBlock) != "" {
		parts = append(parts, "\n## Selected Long-Range Evidence\n"+params.SelectedEvidenceBlock)
	}
	if strings.TrimSpace(params.ChapterSummaries) != "" {
		parts = append(parts, "\n## Chapter Summaries\n"+params.ChapterSummaries)
	}
	if strings.TrimSpace(params.SubplotBoard) != "" {
		parts = append(parts, "\n## Subplot Board\n"+params.SubplotBoard)
	}
	if strings.TrimSpace(params.EmotionalArcs) != "" {
		parts = append(parts, "\n## Emotional Arcs\n"+params.EmotionalArcs)
	}
	if strings.TrimSpace(params.CharacterMatrix) != "" {
		parts = append(parts, "\n## Character Matrix\n"+params.CharacterMatrix)
	}
	if strings.TrimSpace(params.GovernedControlBlock) == "" && strings.TrimSpace(params.VolumeOutline) != "" {
		parts = append(parts, "\n## Volume Outline\n"+params.VolumeOutline)
	}
	parts = append(parts, "\nReturn strictly with === TAG === blocks.")
	return strings.Join(parts, "\n")
}

func bookTitle(book *models.BookConfig) string {
	if book == nil || strings.TrimSpace(book.Title) == "" {
		return "(unknown)"
	}
	return book.Title
}

func bookPlatform(book *models.BookConfig) string {
	if book == nil {
		return "(unknown)"
	}
	return string(book.Platform)
}

func genreName(genreProfile *models.GenreProfile) string {
	if genreProfile == nil || strings.TrimSpace(genreProfile.Name) == "" {
		return "(unknown)"
	}
	return genreProfile.Name
}
