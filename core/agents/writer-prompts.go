package agents

import (
	"fmt"
	"strings"

	"github.com/icosmos-space/ipen/core/models"
	"github.com/icosmos-space/ipen/core/utils"
)

// FanficContext carries optional fanfic prompt context.
type FanficContext struct {
	FanficCanon       string
	FanficMode        models.FanficMode
	AllowedDeviations []string
}

const writerPromptMissingPlaceholder = "(文件尚未创建)"

// BuildWriterSystemPrompt 构建writer system prompt shared by writer/analyzer。
func BuildWriterSystemPrompt(
	book *models.BookConfig,
	genreProfile *models.GenreProfile,
	bookRules *models.BookRules,
	bookRulesBody string,
	genreBody string,
	styleGuide string,
	styleFingerprint *string,
	chapterNumber *int,
	mode string,
	fanficContext *FanficContext,
	languageOverride string,
	inputProfile string,
	lengthSpec *models.LengthSpec,
) string {
	resolvedLanguage := resolvePromptLanguage(genreProfile, languageOverride)
	isEnglish := resolvedLanguage == "en"
	governed := strings.EqualFold(strings.TrimSpace(inputProfile), "governed")
	if strings.TrimSpace(mode) == "" {
		mode = "full"
	}

	resolvedLengthSpec := lengthSpec
	if resolvedLengthSpec == nil {
		target := 3000
		if book != nil && book.ChapterWordCount > 0 {
			target = book.ChapterWordCount
		}
		lang := utils.LanguageZH
		if isEnglish {
			lang = utils.LanguageEN
		}
		spec := utils.BuildLengthSpec(target, lang)
		resolvedLengthSpec = &spec
	}

	if isEnglish {
		sections := []string{
			BuildEnglishGenreIntro(book, genreProfile),
			BuildEnglishCoreRules(book),
			buildGovernedInputContract("en", governed),
			buildLengthGuidance(*resolvedLengthSpec, "en"),
		}
		if !governed {
			sections = append(sections, BuildEnglishAntiAIRules(), BuildEnglishCharacterMethod())
		}
		sections = append(sections,
			buildGenreRulesEN(genreBody),
			buildBookRulesBodyEN(bookRulesBody),
			buildStyleGuideEN(styleGuide),
			buildStyleFingerprintEN(styleFingerprint),
		)
		if fanficContext != nil && strings.TrimSpace(fanficContext.FanficCanon) != "" {
			sections = append(sections,
				BuildFanficCanonSection(fanficContext.FanficCanon, fanficContext.FanficMode),
				BuildFanficModeInstructions(fanficContext.FanficMode, fanficContext.AllowedDeviations),
			)
			if voice := BuildCharacterVoiceProfiles(fanficContext.FanficCanon); strings.TrimSpace(voice) != "" {
				sections = append(sections, voice)
			}
		}
		if !governed {
			sections = append(sections, BuildEnglishPreWriteChecklist(book, genreProfile))
		}
		sections = append(sections, buildOutputFormatEN(mode, *resolvedLengthSpec))
		return joinNonEmpty(sections)
	}

	sections := []string{
		buildGenreIntroZH(book, genreProfile),
		buildCoreRulesZH(*resolvedLengthSpec),
		buildGovernedInputContract("zh", governed),
		buildLengthGuidance(*resolvedLengthSpec, "zh"),
	}
	if !governed {
		sections = append(sections,
			buildAntiAIExamplesZH(),
			buildCharacterPsychologyMethodZH(),
			buildReaderPsychologyMethodZH(),
			buildGoldenChaptersRulesZH(chapterNumber),
		)
	}
	if bookRules != nil && bookRules.EnableFullCastTracking {
		sections = append(sections, buildFullCastTrackingZH())
	}
	sections = append(sections,
		buildGenreRulesZH(genreBody),
		buildBookRulesBodyZH(bookRulesBody),
		buildStyleGuideZH(styleGuide),
		buildStyleFingerprintZH(styleFingerprint),
	)
	if fanficContext != nil && strings.TrimSpace(fanficContext.FanficCanon) != "" {
		sections = append(sections,
			BuildFanficCanonSection(fanficContext.FanficCanon, fanficContext.FanficMode),
			BuildFanficModeInstructions(fanficContext.FanficMode, fanficContext.AllowedDeviations),
		)
		if voice := BuildCharacterVoiceProfiles(fanficContext.FanficCanon); strings.TrimSpace(voice) != "" {
			sections = append(sections, voice)
		}
	}
	if !governed {
		sections = append(sections, buildPreWriteChecklistZH())
	}
	sections = append(sections, buildOutputFormatZH(mode, *resolvedLengthSpec))
	return joinNonEmpty(sections)
}

func resolvePromptLanguage(gp *models.GenreProfile, override string) string {
	if strings.EqualFold(strings.TrimSpace(override), "en") {
		return "en"
	}
	if strings.EqualFold(strings.TrimSpace(override), "zh") {
		return "zh"
	}
	if gp != nil && strings.EqualFold(strings.TrimSpace(gp.Language), "en") {
		return "en"
	}
	return "zh"
}

func joinNonEmpty(parts []string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			filtered = append(filtered, strings.TrimSpace(part))
		}
	}
	return strings.Join(filtered, "\n\n")
}

func buildGovernedInputContract(language string, governed bool) string {
	if !governed {
		return ""
	}
	if strings.EqualFold(language, "en") {
		return `## Input Governance Contract

- Chapter steering follows the provided chapter intent and context package.
- The outline is the default plan, not unconditional global supremacy.
- If an English Variance Brief is present, obey it and avoid listed repetitive patterns.
- In multi-character scenes, include at least one resistance-bearing exchange.`
	}
	return `## 输入治理契约

- 本章写作以提供的 chapter intent 与 context package 为准。
- 卷纲是默认规划，不是无条件的全局最高规则。
- 如存在显式 override，优先满足当前任务意图，再做局部调整。
- 只有硬护栏（设定、连续性事实、显式禁令）不可突破。`
}

func buildLengthGuidance(spec models.LengthSpec, language string) string {
	if strings.EqualFold(language, "en") {
		return fmt.Sprintf(`## Length Guidance

- Target length: %d words
- Acceptable range: %d-%d words
- Hard range: %d-%d words`, spec.Target, spec.SoftMin, spec.SoftMax, spec.HardMin, spec.HardMax)
	}
	return fmt.Sprintf(`## 字数治理

- 目标字数：%d
- 允许区间：%d-%d
- 硬区间：%d-%d`, spec.Target, spec.SoftMin, spec.SoftMax, spec.HardMin, spec.HardMax)
}

func buildGenreIntroZH(book *models.BookConfig, gp *models.GenreProfile) string {
	genreName := "网文"
	if gp != nil && strings.TrimSpace(gp.Name) != "" {
		genreName = gp.Name
	}
	platform := "平台"
	if book != nil && strings.TrimSpace(string(book.Platform)) != "" {
		platform = string(book.Platform)
	}
	return fmt.Sprintf("你是一位专业的%s作者，在%s写作。", genreName, platform)
}

func buildCoreRulesZH(spec models.LengthSpec) string {
	return fmt.Sprintf(`## 核心规则

1. 保持人物动机和行为一致。
2. 用行动与细节传递情绪，避免空泛总结。
3. 保持设定稳定，避免信息越界。
4. 每章结尾保留钩子。
5. 字数目标围绕 %d，允许波动在 %d-%d。`, spec.Target, spec.SoftMin, spec.SoftMax) + "\n\n" + `## 硬性禁令
- 不得输出分析腔、报告腔正文。
- 不得在正文中出现台账式结算数据。
- 不得出现破折号“——”。`
}

func buildAntiAIExamplesZH() string {
	return `## 去 AI 味对照
- 反例：他非常愤怒。
- 正例：他把杯沿捏出了裂纹。`
}

func buildCharacterPsychologyMethodZH() string {
	return `## 六步角色心理分析
1. 当前处境
2. 核心动机
3. 信息边界
4. 性格过滤
5. 行为选择
6. 情绪外化`
}

func buildReaderPsychologyMethodZH() string {
	return `## 读者心理学框架

- 预期管理
- 压力递增
- 兑现时机`
}

func buildGoldenChaptersRulesZH(chapterNumber *int) string {
	if chapterNumber == nil || *chapterNumber > 3 {
		return ""
	}
	return fmt.Sprintf("## 黄金三章规则\n\n当前第 %d 章：优先冲突、信息分层、强钩子。", *chapterNumber)
}

func buildFullCastTrackingZH() string {
	return `## 全员追踪

- POST_SETTLEMENT 追加出场角色状态变更。`
}

func buildGenreRulesZH(genreBody string) string {
	if strings.TrimSpace(genreBody) == "" {
		return ""
	}
	return "## 题材规则\n\n" + genreBody
}

func buildGenreRulesEN(genreBody string) string {
	if strings.TrimSpace(genreBody) == "" {
		return ""
	}
	return "## Genre Profile\n\n" + genreBody
}

func buildBookRulesBodyZH(body string) string {
	if strings.TrimSpace(body) == "" {
		return ""
	}
	return "## 本书规则\n\n" + body
}

func buildBookRulesBodyEN(body string) string {
	if strings.TrimSpace(body) == "" {
		return ""
	}
	return "## Book Rules\n\n" + body
}

func buildStyleGuideZH(styleGuide string) string {
	if strings.TrimSpace(styleGuide) == "" || strings.TrimSpace(styleGuide) == writerPromptMissingPlaceholder {
		return ""
	}
	return "## 文风指南\n\n" + styleGuide
}

func buildStyleGuideEN(styleGuide string) string {
	if strings.TrimSpace(styleGuide) == "" || strings.TrimSpace(styleGuide) == writerPromptMissingPlaceholder {
		return ""
	}
	return "## Style Guide\n\n" + styleGuide
}

func buildStyleFingerprintZH(styleFingerprint *string) string {
	if styleFingerprint == nil || strings.TrimSpace(*styleFingerprint) == "" {
		return ""
	}
	return "## 文风指纹\n\n" + strings.TrimSpace(*styleFingerprint)
}

func buildStyleFingerprintEN(styleFingerprint *string) string {
	if styleFingerprint == nil || strings.TrimSpace(*styleFingerprint) == "" {
		return ""
	}
	return "## Style Fingerprint\n\n" + strings.TrimSpace(*styleFingerprint)
}

func buildPreWriteChecklistZH() string {
	return `## 动笔前自检

1. 本章目标是否清晰。
2. 冲突是否可感。
3. 章尾钩子是否成立。`
}

func buildOutputFormatZH(mode string, spec models.LengthSpec) string {
	if strings.EqualFold(mode, "creative") {
		return fmt.Sprintf(`## 输出格式

=== PRE_WRITE_CHECK ===
(表格)

=== CHAPTER_TITLE ===
(标题)

=== CHAPTER_CONTENT ===
(正文，目标 %d，允许区间 %d-%d)`, spec.Target, spec.SoftMin, spec.SoftMax)
	}
	return fmt.Sprintf(`## 输出格式

=== PRE_WRITE_CHECK ===
(表格)

=== CHAPTER_TITLE ===
(标题)

=== CHAPTER_CONTENT ===
(正文，目标 %d，允许区间 %d-%d)

=== POST_SETTLEMENT ===
(结算)

=== UPDATED_STATE ===
(状态)

=== UPDATED_HOOKS ===
(伏笔池)`, spec.Target, spec.SoftMin, spec.SoftMax)
}

func buildOutputFormatEN(mode string, spec models.LengthSpec) string {
	if strings.EqualFold(mode, "creative") {
		return fmt.Sprintf(`## Output Format

=== PRE_WRITE_CHECK ===
(table)

=== CHAPTER_TITLE ===
(title)

=== CHAPTER_CONTENT ===
(prose, target %d, acceptable %d-%d)`, spec.Target, spec.SoftMin, spec.SoftMax)
	}
	return fmt.Sprintf(`## Output Format

=== PRE_WRITE_CHECK ===
(table)

=== CHAPTER_TITLE ===
(title)

=== CHAPTER_CONTENT ===
(prose, target %d, acceptable %d-%d)

=== POST_SETTLEMENT ===
(settlement)

=== UPDATED_STATE ===
(state)

=== UPDATED_HOOKS ===
(hooks)`, spec.Target, spec.SoftMin, spec.SoftMax)
}
