package agents

import (
	"fmt"
	"strings"

	"github.com/icosmos-space/ipen/core/models"
)

// BuildEnglishCoreRules 返回universal English fiction craft rules。
func BuildEnglishCoreRules(_book *models.BookConfig) string {
	return `## Universal Writing Rules

1. Keep character behavior consistent with motivation and personality.
2. Show emotions through action and detail, not labels.
3. Distinguish character voices clearly.
4. End each chapter with a hook or pressure point.
5. Keep world rules stable and enforceable.
6. Make every major gain carry cost or consequence.
7. Avoid exposition dumps; layer information through scenes.
8. Keep pacing varied: intensity followed by breathing room.`
}

// BuildEnglishAntiAIRules 返回deterministic anti-AI prose constraints。
func BuildEnglishAntiAIRules() string {
	return `## Anti-AI Iron Laws

- Do not narrate conclusions the reader can infer.
- Avoid report language in prose ("core motivation", "optimal outcome", etc.).
- Rate-limit AI tell words (delve, tapestry, nuanced, pivotal, vibrant).
- Avoid repetitive metaphor cycling.
- Avoid planner terminology in chapter prose.
- Avoid overusing "Not X, but Y" constructions.`
}

// BuildEnglishCharacterMethod 返回internal planning rubric。
func BuildEnglishCharacterMethod() string {
	return `## Character Psychology Method

Before writing a character beat, check:
1. What the character knows right now.
2. What they want in this scene.
3. How personality shapes approach.
4. What action they take.
5. How others react.`
}

// BuildEnglishPreWriteChecklist 构建per-book checklist text。
func BuildEnglishPreWriteChecklist(book *models.BookConfig, gp *models.GenreProfile) string {
	chapterWords := 3000
	if book != nil && book.ChapterWordCount > 0 {
		chapterWords = book.ChapterWordCount
	}
	items := []string{
		"Outline anchor: what plot point this chapter advances",
		"POV consistency",
		"Next-chapter hook",
		"Character consistency",
		"Information boundary correctness",
		fmt.Sprintf("Length target: around %d words", chapterWords),
		"Show-don't-tell pass",
		"Conflict pressure pass",
	}
	if gp != nil && gp.PowerScaling {
		items = append(items, "Power scaling consistency")
	}
	if gp != nil && gp.NumericalSystem {
		items = append(items, "Numerical/resource ledger consistency")
	}
	lines := make([]string, 0, len(items))
	for i, item := range items {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, item))
	}
	return "## Pre-Write Checklist\n\n" + strings.Join(lines, "\n")
}

// BuildEnglishGenreIntro 返回English genre-specific intro。
func BuildEnglishGenreIntro(book *models.BookConfig, gp *models.GenreProfile) string {
	genreName := "web fiction"
	if gp != nil && strings.TrimSpace(gp.Name) != "" {
		genreName = gp.Name
	}
	chapterWords := 3000
	targetChapters := 200
	if book != nil {
		if book.ChapterWordCount > 0 {
			chapterWords = book.ChapterWordCount
		}
		if book.TargetChapters > 0 {
			targetChapters = book.TargetChapters
		}
	}
	return fmt.Sprintf(
		"You are a professional %s web-fiction author. Target around %d words/chapter for %d chapters. Write in natural English with varied sentence rhythm.",
		genreName,
		chapterWords,
		targetChapters,
	)
}
