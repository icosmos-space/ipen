package agents

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/icosmos-space/ipen/core/models"
)

// ArchitectOutput the generated foundation bundle。
type ArchitectOutput struct {
	StoryBible    string `json:"storyBible"`
	VolumeOutline string `json:"volumeOutline"`
	BookRules     string `json:"bookRules"`
	CurrentState  string `json:"currentState"`
	PendingHooks  string `json:"pendingHooks"`
}

// ArchitectAgent 生成书籍基础架构。
type ArchitectAgent struct {
	*BaseAgent
}

// NewArchitectAgent 创建ArchitectAgent。
func NewArchitectAgent(ctx AgentContext) *ArchitectAgent {
	return &ArchitectAgent{BaseAgent: NewBaseAgent(ctx)}
}

// Name 返回Agent名称。
func (a *ArchitectAgent) Name() string {
	return "architect"
}

// GenerateFoundation 生成基础架构。
func (a *ArchitectAgent) GenerateFoundation(ctx context.Context, book *models.BookConfig, externalContext string, reviewFeedback string) (*ArchitectOutput, error) {
	_ = ctx
	if book == nil {
		return nil, fmt.Errorf("book config is required")
	}
	language := strings.ToLower(strings.TrimSpace(book.Language))
	if language == "" {
		language = "zh"
	}
	isEnglish := language == "en"

	storyBible := buildArchitectStoryBible(book, externalContext, reviewFeedback, isEnglish)
	volumeOutline := buildArchitectVolumeOutline(book, externalContext, isEnglish)
	bookRules := buildArchitectBookRules(book, isEnglish)
	currentState := buildArchitectCurrentState(isEnglish)
	pendingHooks := buildArchitectPendingHooks(isEnglish)

	return &ArchitectOutput{
		StoryBible:    storyBible,
		VolumeOutline: volumeOutline,
		BookRules:     bookRules,
		CurrentState:  currentState,
		PendingHooks:  pendingHooks,
	}, nil
}

// WriteFoundationFiles 将基础架构文件写入故事目录。
func (a *ArchitectAgent) WriteFoundationFiles(bookDir string, output ArchitectOutput, numericalSystem bool, language string) error {
	storyDir := filepath.Join(bookDir, "story")
	if err := os.MkdirAll(storyDir, 0755); err != nil {
		return err
	}

	writes := map[string]string{
		"story_bible.md":      output.StoryBible,
		"volume_outline.md":   output.VolumeOutline,
		"book_rules.md":       output.BookRules,
		"current_state.md":    output.CurrentState,
		"pending_hooks.md":    output.PendingHooks,
		"subplot_board.md":    defaultSubplotBoard(language),
		"emotional_arcs.md":   defaultEmotionalArcs(language),
		"character_matrix.md": defaultCharacterMatrix(language),
	}

	if numericalSystem {
		writes["particle_ledger.md"] = defaultLedger(language)
	}

	for name, content := range writes {
		if err := os.WriteFile(filepath.Join(storyDir, name), []byte(content), 0644); err != nil {
			return err
		}
	}
	return nil
}

// GenerateFoundationFromImport 从导入的章节文本推断基础架构。
func (a *ArchitectAgent) GenerateFoundationFromImport(
	ctx context.Context,
	book *models.BookConfig,
	chaptersText string,
	externalContext string,
	reviewFeedback string,
	importMode string,
) (*ArchitectOutput, error) {
	_ = ctx
	if book == nil {
		return nil, fmt.Errorf("book config is required")
	}
	base, err := a.GenerateFoundation(context.Background(), book, externalContext, reviewFeedback)
	if err != nil {
		return nil, err
	}

	chapters := extractChapterNumbers(chaptersText)
	lastChapter := 0
	if len(chapters) > 0 {
		lastChapter = chapters[len(chapters)-1]
	}

	if strings.EqualFold(book.Language, "en") {
		base.CurrentState = strings.Replace(base.CurrentState, "| Current Chapter | 0 |", fmt.Sprintf("| Current Chapter | %d |", lastChapter), 1)
		base.VolumeOutline += "\n\n## Imported Continuation\n"
		if strings.EqualFold(importMode, "series") {
			base.VolumeOutline += "- Introduce a new conflict vector within 5 chapters.\n- Ensure at least half of key scenes are fresh situations."
		} else {
			base.VolumeOutline += "- Continue unresolved conflicts naturally from imported chapters."
		}
	} else {
		base.CurrentState = strings.Replace(base.CurrentState, "| 当前章节 | 0 |", fmt.Sprintf("| 当前章节 | %d |", lastChapter), 1)
		base.VolumeOutline += "\n\n## 导入续写方向\n"
		if strings.EqualFold(importMode, "series") {
			base.VolumeOutline += "- 5 章内引入新的冲突向量。\n- 至少一半关键场景使用全新情境。"
		} else {
			base.VolumeOutline += "- 从已导入章节结尾自然推进未解决冲突。"
		}
	}

	return base, nil
}

// GenerateFanficFoundation 生成同人基础架构。
func (a *ArchitectAgent) GenerateFanficFoundation(
	ctx context.Context,
	book *models.BookConfig,
	fanficCanon string,
	fanficMode models.FanficMode,
	reviewFeedback string,
) (*ArchitectOutput, error) {
	_ = ctx
	base, err := a.GenerateFoundation(context.Background(), book, "", reviewFeedback)
	if err != nil {
		return nil, err
	}

	isEnglish := strings.EqualFold(book.Language, "en")
	if isEnglish {
		base.StoryBible += "\n\n## Fanfic Canon Anchor\n" + truncateString(fanficCanon, 6000)
		base.BookRules = strings.Replace(base.BookRules, "fanficMode: \"\"", fmt.Sprintf("fanficMode: %q", fanficMode), 1)
		base.VolumeOutline += "\n\n## Fanfic Mode\n- Mode: " + string(fanficMode)
	} else {
		base.StoryBible += "\\n\\n## 同人正典锚点\\n" + truncateString(fanficCanon, 6000)
		base.BookRules = strings.Replace(base.BookRules, "fanficMode: \"\"", fmt.Sprintf("fanficMode: %q", fanficMode), 1)
		base.VolumeOutline += "\n\n## 同人模式\n- 模式：" + string(fanficMode)
	}

	return base, nil
}

func buildArchitectStoryBible(book *models.BookConfig, externalContext, reviewFeedback string, english bool) string {
	if english {
		parts := []string{
			"# Story Bible",
			"",
			"## 01_Worldview",
			"Define world rules, social frame, and high-level constraints.",
			"",
			"## 02_Protagonist",
			"Identity, edge, personality lock, and behavioral boundaries.",
			"",
			"## 03_Factions_and_Characters",
			"Main factions and key supporting characters with independent goals.",
			"",
			"## 04_Geography_and_Environment",
			"Core locations and scene characteristics.",
			"",
			"## 05_Title_and_Blurb",
			"Title: " + book.Title,
			"Blurb: concise conflict-forward product copy.",
		}
		if strings.TrimSpace(externalContext) != "" {
			parts = append(parts, "", "## External Context", externalContext)
		}
		if strings.TrimSpace(reviewFeedback) != "" {
			parts = append(parts, "", "## Review Feedback Applied", reviewFeedback)
		}
		return strings.Join(parts, "\n")
	}

	parts := []string{
		"# 故事圣经",
		"",
		"## 01_世界观",
		"定义世界规则、社会框架与硬性约束。",
		"",
		"## 02_主角",
		"身份、核心优势、性格锁与行为边界。",
		"",
		"## 03_势力与角色",
		"主要势力与关键配角（含独立目标）。",
		"",
		"## 04_地理与环境",
		"核心场景与空间特征。",
		"",
		"## 05_书名与简介",
		"书名：" + book.Title,
		"简介：冲突导向、可传播。",
	}
	if strings.TrimSpace(externalContext) != "" {
		parts = append(parts, "", "## 外部指令", externalContext)
	}
	if strings.TrimSpace(reviewFeedback) != "" {
		parts = append(parts, "", "## 已吸收评审反馈", reviewFeedback)
	}
	return strings.Join(parts, "\n")
}

func buildArchitectVolumeOutline(book *models.BookConfig, externalContext string, english bool) string {
	chapters := book.TargetChapters
	if chapters <= 0 {
		chapters = 200
	}
	if english {
		return fmt.Sprintf(`# Volume Outline

## Volume 1 (Ch.1-30)
- Core conflict ignition
- Hero leverage demonstration
- Short-term objective locking

## Volume 2 (Ch.31-70)
- Escalation and faction pressure

## Volume 3 (Ch.71-%d)
- Mid/late arc convergence and payoff

### Golden First Three Chapters
1. Throw core conflict immediately
2. Show protagonist leverage
3. Establish concrete short-term goal

%s`, chapters, strings.TrimSpace(externalContext))
	}
	return fmt.Sprintf(`# 卷纲

## 第一卷（1-30章）
- 点燃核心冲突
- 展示主角优势
- 锁定短期目标

## 第二卷（31-70章）
- 势力升级与外部压迫
## 第三卷（71-%d章）
- 中后期收束与回收

### 黄金前三章
1. 立刻抛出核心冲突
2. 展示主角应对抓手
3. 建立明确短期目标

%s`, chapters, strings.TrimSpace(externalContext))
}

func buildArchitectBookRules(book *models.BookConfig, english bool) string {
	genre := "other"
	if book != nil {
		genre = string(book.Genre)
	}
	if english {
		return fmt.Sprintf(`---
version: "1.0"
protagonist:
  name: ""
  personalityLock: []
  behavioralConstraints: []
genreLock:
  primary: %q
  forbidden: []
prohibitions: []
chapterTypesOverride: []
fatigueWordsOverride: []
additionalAuditDimensions: []
enableFullCastTracking: false
fanficMode: ""
allowedDeviations: []
---

## Narrative Perspective
Default close third-person.

## Core Conflict Driver
Conflict must escalate via action-consequence chain.`, genre)
	}
	return fmt.Sprintf(`---
version: "1.0"
protagonist:
  name: ""
  personalityLock: []
  behavioralConstraints: []
genreLock:
  primary: %q
  forbidden: []
prohibitions: []
chapterTypesOverride: []
fatigueWordsOverride: []
additionalAuditDimensions: []
enableFullCastTracking: false
fanficMode: ""
allowedDeviations: []
---

## 叙事视角
默认近距离第三人称。
## 核心冲突驱动
以“行动-后果-再决策”推进冲突。`, genre)
}

func buildArchitectCurrentState(english bool) string {
	if english {
		return `# Current State

| Field | Value |
| --- | --- |
| Current Chapter | 0 |
| Current Location | Starting point |
| Protagonist State | Initial condition |
| Current Goal | Initial goal |
| Current Constraint | Initial constraint |
| Current Alliances | TBD |
| Current Conflict | Core conflict ignition |`
	}
	return `# 当前状态

| 字段 | 值 |
| --- | --- |
| 当前章节 | 0 |
| 当前位置 | 起始地点 |
| 主角状态 | 初始状态 |
| 当前目标 | 初始目标 |
| 当前限制 | 初始限制 |
| 当前敌我 | 待展开 |
| 当前冲突 | 核心冲突点燃 |`
}

func buildArchitectPendingHooks(english bool) string {
	if english {
		return `# Pending Hooks

| hook_id | start_chapter | type | status | last_advanced_chapter | expected_payoff | payoff_timing | notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| opening-hook | 1 | mystery | open | 0 | Resolve opening tension | near-term | Seeded in volume opening |`
	}
	return `# Pending Hooks

| hook_id | 起始章节 | 类型 | 状态 | 最近推进 | 预期回收 | 回收节奏 | 备注 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| opening-hook | 1 | mystery | open | 0 | 回收开篇张力 | near-term | 开卷埋设 |`
}

func defaultLedger(language string) string {
	if strings.EqualFold(language, "en") {
		return "# Resource Ledger\n\n| Chapter | Opening Value | Source | Integrity | Delta | Closing Value | Evidence |\n| --- | --- | --- | --- | --- | --- | --- |\n| 0 | 0 | Initialization | - | 0 | 0 | Initial state |\n"
	}
	return "# 资源账本\n\n| 章节 | 期初值 | 来源 | 完整度 | 增量 | 期末值 | 依据 |\n| --- | --- | --- | --- | --- | --- | --- |\n| 0 | 0 | 初始化 | - | 0 | 0 | 开书初始 |\n"
}

func defaultSubplotBoard(language string) string {
	if strings.EqualFold(language, "en") {
		return "# Subplot Board\n\n| Subplot ID | Subplot | Related Characters | Start Chapter | Last Active Chapter | Chapters Since | Status | Progress Summary | Payoff ETA |\n| --- | --- | --- | --- | --- | --- | --- | --- | --- |\n"
	}
	return "# 支线进度板\n\n| 支线ID | 支线名 | 相关角色 | 起始章 | 最近活跃章 | 距今章数 | 状态 | 进度概述 | 回收ETA |\n| --- | --- | --- | --- | --- | --- | --- | --- | --- |\n"
}

func defaultEmotionalArcs(language string) string {
	if strings.EqualFold(language, "en") {
		return "# Emotional Arcs\n\n| Character | Chapter | Emotional State | Trigger Event | Intensity (1-10) | Arc Direction |\n| --- | --- | --- | --- | --- | --- |\n"
	}
	return "# 情感弧线\n\n| 角色 | 章节 | 情绪状态 | 触发事件 | 强度(1-10) | 弧线方向 |\n| --- | --- | --- | --- | --- | --- |\n"
}

func defaultCharacterMatrix(language string) string {
	if strings.EqualFold(language, "en") {
		return "# Character Matrix\n\n### Character Profiles\n| Character | Core Tags | Contrast Detail | Speech Style | Personality Core | Relationship to Protagonist | Core Motivation | Current Goal |\n| --- | --- | --- | --- | --- | --- | --- | --- |\n\n### Encounter Log\n| Character A | Character B | First Meeting Chapter | Latest Interaction Chapter | Relationship Type | Relationship Change |\n| --- | --- | --- | --- | --- | --- |\n\n### Information Boundaries\n| Character | Known Information | Unknown Information | Source Chapter |\n| --- | --- | --- | --- |\n"
	}
	return "# 角色交互矩阵\n\n### 角色档案\n| 角色 | 核心标签 | 反差细节 | 说话风格 | 性格底色 | 与主角关系 | 核心动机 | 当前目标 |\n| --- | --- | --- | --- | --- | --- | --- | --- |\n\n### 相遇记录\n| 角色A | 角色B | 首次相遇章 | 最近交互章 | 关系性质 | 关系变化 |\n| --- | --- | --- | --- | --- | --- |\n\n### 信息边界\n| 角色 | 已知信息 | 未知信息 | 信息来源章 |\n| --- | --- | --- | --- |\n"
}

func extractChapterNumbers(text string) []int {
	matches := regexp.MustCompile(`(?m)^(?:#\s*)?(?:第\s*(\d+)\s*章|Chapter\s+(\d+))`).FindAllStringSubmatch(text, -1)
	result := []int{}
	for _, match := range matches {
		candidate := ""
		if len(match) > 1 {
			candidate = match[1]
		}
		if candidate == "" && len(match) > 2 {
			candidate = match[2]
		}
		if candidate == "" {
			continue
		}
		if n, err := strconv.Atoi(candidate); err == nil {
			result = append(result, n)
		}
	}
	return result
}

func truncateString(value string, maxRunes int) string {
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes])
}
