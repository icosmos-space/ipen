package agents

import (
	"fmt"
	"strings"

	"github.com/icosmos-space/ipen/core/models"
)

// BuildObserverSystemPrompt 构建observer-phase extraction prompt。
func BuildObserverSystemPrompt(_book *models.BookConfig, genreProfile *models.GenreProfile, language string) string {
	resolved := strings.ToLower(strings.TrimSpace(language))
	if resolved == "" && genreProfile != nil {
		resolved = strings.ToLower(strings.TrimSpace(genreProfile.Language))
	}
	if resolved == "en" {
		return `You are a fact extraction specialist.
Read the chapter and extract all observable fact changes.

Categories:
1. Character actions
2. Location changes
3. Resource changes
4. Relationship changes
5. Emotional shifts
6. Information flow
7. Plot threads
8. Time progression
9. Physical state

Output format:
=== OBSERVATIONS ===
[CHARACTERS] ...
[LOCATIONS] ...
[RESOURCES] ...
[RELATIONSHIPS] ...
[EMOTIONS] ...
[INFORMATION] ...
[PLOT_THREADS] ...
[TIME] ...
[PHYSICAL_STATE] ...`
	}

	return `你是章节事实提取器。请从正文中提取全部可观察变化。
输出格式：=== OBSERVATIONS ===
[角色行为]
[位置变化]
[资源变化]
[关系变化]
[情绪变化]
[信息流动]
[剧情线索]
[时间]
[身体状态]`
}

// BuildObserverUserPrompt 构建chapter-specific observer prompt。
func BuildObserverUserPrompt(chapterNumber int, title, content, language string) string {
	if strings.EqualFold(language, "en") {
		return fmt.Sprintf("Extract all facts from Chapter %d \"%s\":\n\n%s", chapterNumber, title, content)
	}
	return fmt.Sprintf("请提取第%d章《%s》中的全部事实变化：\n\n%s", chapterNumber, title, content)
}
