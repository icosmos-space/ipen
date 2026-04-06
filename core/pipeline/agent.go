package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/icosmos-space/ipen/core/llm"
	"github.com/icosmos-space/ipen/core/state"
	"github.com/icosmos-space/ipen/core/utils"
)

// AGENT_TOOLS agent循环定义。
var AGENT_TOOLS = []llm.ToolDefinition{
	{
		Name:        "write_draft",
		Description: "写【下一章】草稿。只能续写最新章之后的下一章，不能指定章节号，不能补历史空章。生成正文、更新状态卡/账本/伏笔池、保存章节文件。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"bookId":   map[string]any{"type": "string", "description": "书籍ID"},
				"guidance": map[string]any{"type": "string", "description": "本章创作指导（可选，自然语言）"}},
			"required": []string{"bookId"}}},
	{
		Name:        "plan_chapter",
		Description: "为下一章生成 chapter intent（章节目标、必须保留、冲突说明）。适合在正式写作前检查当前控制输入是否正确。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"bookId":   map[string]any{"type": "string", "description": "书籍ID"},
				"guidance": map[string]any{"type": "string", "description": "本章创作指导（可选，自然语言）"}},
			"required": []string{"bookId"}}},
	{
		Name:        "compose_chapter",
		Description: "为下一章生成 context/rule-stack/trace 运行时产物。适合在写作前确认系统实际会带哪些上下文和优先级。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"bookId":   map[string]any{"type": "string", "description": "书籍ID"},
				"guidance": map[string]any{"type": "string", "description": "本章创作指导（可选，自然语言）"}},
			"required": []string{"bookId"}}},
	{
		Name:        "audit_chapter",
		Description: "审计指定章节。检查连续性、OOC、数值、伏笔等问题。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"bookId":        map[string]any{"type": "string", "description": "书籍ID"},
				"chapterNumber": map[string]any{"type": "number", "description": "章节号（不填则审计最新章）"}},
			"required": []string{"bookId"}}},
	{
		Name:        "revise_chapter",
		Description: "修订指定章节的文字质量。根据审计问题做局部修正，不改变剧情走向。默认 local-fix（局部修复最小改动）；也支持 polish(润色)、rewrite(改写)、rework(重写)、anti-detect。注意：不能用来补缺失章节、不能改章节号、不能替代 write_draft。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"bookId":        map[string]any{"type": "string", "description": "书籍ID"},
				"chapterNumber": map[string]any{"type": "number", "description": "章节号（不填则修订最新章）"},
				"mode":          map[string]any{"type": "string", "enum": []string{"polish", "rewrite", "rework", "local-fix", "anti-detect"}, "description": "修订模式（默认 local-fix）"}},
			"required": []string{"bookId"}}},
	{
		Name:        "scan_market",
		Description: "扫描市场趋势。从平台排行榜获取实时数据并分析。",
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}},
	{
		Name:        "create_book",
		Description: "创建一本新书。生成世界观、卷纲、文风指南等基础设定。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"title":    map[string]any{"type": "string", "description": "书名"},
				"genre":    map[string]any{"type": "string", "enum": []string{"xuanhuan", "xianxia", "urban", "horror", "other"}, "description": "题材"},
				"platform": map[string]any{"type": "string", "enum": []string{"tomato", "feilu", "qidian", "other"}, "description": "目标平台"},
				"brief":    map[string]any{"type": "string", "description": "创作简述/需求（自然语言）"}},
			"required": []string{"title", "genre", "platform"}}},
	{
		Name:        "update_author_intent",
		Description: "更新书级长期意图文档 author_intent.md。用于修改这本书长期想成为什么。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"bookId":  map[string]any{"type": "string", "description": "书籍ID"},
				"content": map[string]any{"type": "string", "description": "author_intent.md 的完整新内容"},
			},
			"required": []string{"bookId", "content"},
		},
	},
	{
		Name:        "update_current_focus",
		Description: "更新当前关注点文档 current_focus.md。用于把最近几章的注意力拉回某条主线或冲突。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"bookId":  map[string]any{"type": "string", "description": "书籍ID"},
				"content": map[string]any{"type": "string", "description": "current_focus.md 的完整新内容"},
			},
			"required": []string{"bookId", "content"},
		},
	},

	{
		Name:        "get_book_status",
		Description: "获取书籍状态概览：章数、字数、最近章节审计情况。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"bookId": map[string]any{"type": "string", "description": "书籍ID"},
			},
			"required": []string{"bookId"},
		},
	},
	{
		Name:        "read_truth_files",
		Description: "读取书籍的长期记忆（状态卡、资源账本、伏笔池）+ 世界观和卷纲。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"bookId": map[string]any{"type": "string", "description": "书籍ID"},
			},
			"required": []string{"bookId"},
		},
	},
	{
		Name:        "list_books",
		Description: "列出所有书籍。",
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	},
	{
		Name:        "write_full_pipeline",
		Description: "完整管线：写草稿 → 审计 → 自动修订（如需要）。一键完成。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"bookId": map[string]any{"type": "string", "description": "书籍ID"},
				"count":  map[string]any{"type": "number", "description": "连续写几章（默认1）"},
			},
			"required": []string{"bookId"},
		},
	},
	{
		Name:        "web_fetch",
		Description: "抓取指定URL的文本内容。用于读取搜索结果中的详细页面。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url":      map[string]any{"type": "string", "description": "要抓取的URL"},
				"maxChars": map[string]any{"type": "number", "description": "最大返回字符数（默认8000）"},
			},
			"required": []string{"url"},
		},
	},

	{
		Name:        "import_style",
		Description: "从参考文本生成文风指南（统计 + LLM定性分析）。生成 style_profile.json 和 style_guide.md。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"bookId":        map[string]any{"type": "string", "description": "目标书籍ID"},
				"referenceText": map[string]any{"type": "string", "description": "参考文本（至少2000字）"}},
			"required": []string{"bookId", "referenceText"}}},
	{
		Name:        "import_canon",
		Description: "从正传导入正典参照，生成 parent_canon.md，启用番外写作和审计模式。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"targetBookId": map[string]any{"type": "string", "description": "番外书籍ID"},
				"parentBookId": map[string]any{"type": "string", "description": "正传书籍ID"}},
			"required": []string{"targetBookId", "parentBookId"}}},
	{
		Name:        "import_chapters",
		Description: "【整书重导】导入已有章节。从完整文本中自动分割所有章节，逐章分析并重建全部真相文件。这是整书级操作，不是补某一章的工具。导入后可用 write_draft 续写。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"bookId":       map[string]any{"type": "string", "description": "目标书籍ID"},
				"text":         map[string]any{"type": "string", "description": "包含多章的完整文本"},
				"splitPattern": map[string]any{"type": "string", "description": "章节分割正则（可选，默认匹配'第X章'）"}},
			"required": []string{"bookId", "text"}}},
	{
		Name:        "write_truth_file",
		Description: "【整文件覆盖】直接替换书的真相文件内容。用于扩展大纲、修改世界观、调整规则。注意：这是整文件覆盖写入，不是追加；不要用来改 current_state.md 的章节进度指针或 hack 章节号；不要用来补空章节。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"bookId":   map[string]any{"type": "string", "description": "书籍ID"},
				"fileName": map[string]any{"type": "string", "description": "文件名（如 volume_outline.md、story_bible.md、book_rules.md、current_state.md、pending_hooks.md）"},
				"content":  map[string]any{"type": "string", "description": "新的完整文件内容"},
			},
			"required": []string{"bookId", "fileName", "content"},
		},
	},
}

// AgentLoopOptions 配置 RunAgentLoop 回调.
type AgentLoopOptions struct {
	OnToolCall   func(name string, args map[string]any)
	OnToolResult func(name string, result string)
	OnMessage    func(content string)
	MaxTurns     int
}

// RunAgentLoop 使用管道工具运行一个 LLM 工具循环
func RunAgentLoop(ctx context.Context, config PipelineConfig, instruction string, options *AgentLoopOptions) (string, error) {
	pipeline := NewPipelineRunner(config)
	stateManager := state.NewStateManager(config.ProjectRoot)

	messages := []llm.AgentMessage{
		{
			Role: "system",
			Content: `你是 InkOS 小说写作 Agent。用户是小说作者，你帮他管理从建书到成稿的全过程。

## 工具

| 工具 | 作用 |
|------|------|
| list_books | 列出所有书 |
| get_book_status | 查看书的章数、字数、审计状态 |
| read_truth_files | 读取长期记忆（状态卡、资源账本、伏笔池）和设定（世界观、卷纲、本书规则） |
| create_book | 建书，生成世界观、卷纲、本书规则（自动加载题材 genre profile） |
| plan_chapter | 先生成 chapter intent，确认本章目标/冲突/优先级 |
| compose_chapter | 再生成 runtime context/rule stack，确认实际输入 |
| write_draft | 写【下一章】草稿（只能续写最新章之后，不能补历史章） |
| audit_chapter | 审计章节（32维度，按题材条件启用，含AI痕迹+敏感词检测） |
| revise_chapter | 修订章节文字质量（不能补空章/改章号，五种模式） |
| update_author_intent | 更新书级长期意图 author_intent.md |
| update_current_focus | 更新当前关注点 current_focus.md |
| write_full_pipeline | 完整管线：写 → 审 → 改（如需要） |
| scan_market | 扫描平台排行榜，分析市场趋势 |
| web_fetch | 抓取指定URL的文本内容 |
| import_style | 从参考文本生成文风指南（统计+LLM分析） |
| import_canon | 从正传导入正典参照，启用番外模式 |
| import_chapters | 【整书重导】导入全部已有章节并重建真相文件 |
| write_truth_file | 【整文件覆盖】替换真相文件内容，不能用来改章节进度 |

## 长期记忆

每本书有两层控制面：
- **author_intent.md** — 这本书长期想成为什么
- **current_focus.md** — 最近 1-3 章要把注意力拉回哪里

以及七个长期记忆文件，是 Agent 写作和审计的事实依据：
- **current_state.md** — 角色位置、关系、已知信息、当前冲突
- **particle_ledger.md** — 物品/资源账本，每笔增减有据可查
- **pending_hooks.md** — 已埋伏笔、推进状态、预期回收时机
- **chapter_summaries.md** — 每章压缩摘要（人物、事件、伏笔、情绪）
- **subplot_board.md** — 支线进度板
- **emotional_arcs.md** — 角色情感弧线
- **character_matrix.md** — 角色交互矩阵与信息边界

## 管线逻辑

- audit 返回 passed=true → 不需要 revise
- audit 返回 passed=false 且有 critical → 调 revise，改完可以再 audit
- write_full_pipeline 会自动走完 写→审→改，适合不需要中间干预的场景

## 规则

- 用户提供了题材/创意但没说要扫描市场 → 跳过 scan_market，直接 create_book
- 用户说了书名/bookId → 直接操作，不需要先 list_books
- 每完成一步，简要汇报进展
- 当用户要求“先把注意力拉回某条线”时，优先 update_current_focus，然后 plan_chapter / compose_chapter，再决定是否 write_draft 或 write_full_pipeline
- 仿写流程：用户提供参考文本 → import_style → 生成 style_guide.md，后续写作自动参照
- 番外流程：先 create_book 建番外书 → import_canon 导入正传正典 → 然后正常 write_draft
- 续写流程：用户提供已有章节 → import_chapters → 然后 write_draft 续写

## 禁止事项（严格遵守）

- 不要用 write_draft 补历史中间章节。write_draft 只能写【当前最新章之后的下一章】
- 不要用 import_chapters 修补某一个空章。import_chapters 是整书级重导工具
- 不要用 write_truth_file 修改 current_state.md 的章节进度来"骗"系统跳到某一章
- 不要用 revise_chapter 补缺失章节或改章节号。revise 只做文字质量修订
- 用户说"补第 N 章"或"第 N 章是空的"时，先用 get_book_status 和 read_truth_files 判断真实状态，再决定用哪个工具
- 不要在没有确认书籍状态的情况下直接调用写作工具`,
		},
		{Role: "user", Content: instruction},
	}

	maxTurns := 20
	if options != nil && options.MaxTurns > 0 {
		maxTurns = options.MaxTurns
	}
	lastAssistantMessage := ""

	for turn := 0; turn < maxTurns; turn++ {
		result, err := llm.ChatWithTools(ctx, config.Client, messages, AGENT_TOOLS, &llm.ChatOptions{})
		if err != nil {
			return lastAssistantMessage, err
		}

		// assistans 消息加入历史
		messages = append(messages, llm.AgentMessage{
			Role:      "assistant",
			Content:   result.Content,
			ToolCalls: result.ToolCalls,
		})

		if strings.TrimSpace(result.Content) != "" {
			lastAssistantMessage = result.Content
			if options != nil && options.OnMessage != nil {
				options.OnMessage(result.Content)
			}
		}
		if len(result.ToolCalls) == 0 {
			break
		}

		for _, toolCall := range result.ToolCalls {
			parsedArgs := map[string]any{}
			if strings.TrimSpace(toolCall.Arguments) != "" {
				_ = json.Unmarshal([]byte(toolCall.Arguments), &parsedArgs)
			}
			if options != nil && options.OnToolCall != nil {
				options.OnToolCall(toolCall.Name, parsedArgs)
			}

			toolResult, err := ExecuteAgentTool(ctx, pipeline, stateManager, config, toolCall.Name, parsedArgs)
			if err != nil {
				toolResult = mustJSON(map[string]any{"error": err.Error()})
			}
			if options != nil && options.OnToolResult != nil {
				options.OnToolResult(toolCall.Name, toolResult)
			}
			// tool消息加入历史
			messages = append(messages, llm.AgentMessage{
				Role:       "tool",
				ToolCallID: toolCall.ID,
				Content:    toolResult,
			})
		}
	}

	return lastAssistantMessage, nil
}

// ExecuteAgentTool 执行一个工具调用并返回一个 JSON 字符串
func ExecuteAgentTool(ctx context.Context, pipeline *PipelineRunner, stateManager *state.StateManager, config PipelineConfig, name string, args map[string]any) (string, error) {
	switch name {
	case "plan_chapter":
		bookID := asString(args["bookId"])
		bookConfig, err := stateManager.LoadBookConfig(bookID)
		if err != nil {
			return "", err
		}
		bookDir := stateManager.BookDir(bookID)
		nextChapter, err := stateManager.GetNextChapterNumber(bookID)
		if err != nil {
			return "", err
		}
		result, err := pipeline.planChapter(ctx, bookConfig, bookDir, nextChapter)
		if err != nil {
			return "", err
		}
		intentPath := normalizeRuntimePath(result.RuntimePath)
		return mustJSON(map[string]any{
			"chapterNumber":  nextChapter,
			"intentPath":     intentPath,
			"intentMarkdown": result.IntentMarkdown,
		}), nil

	case "compose_chapter":
		bookID := asString(args["bookId"])
		bookConfig, err := stateManager.LoadBookConfig(bookID)
		if err != nil {
			return "", err
		}
		bookDir := stateManager.BookDir(bookID)
		nextChapter, err := stateManager.GetNextChapterNumber(bookID)
		if err != nil {
			return "", err
		}
		planResult, err := pipeline.planChapter(ctx, bookConfig, bookDir, nextChapter)
		if err != nil {
			return "", err
		}
		composeResult, err := pipeline.composeChapter(ctx, bookConfig, bookDir, nextChapter, planResult)
		if err != nil {
			return "", err
		}
		return mustJSON(map[string]any{
			"chapterNumber": nextChapter,
			"contextPath":   normalizeRuntimePath(composeResult.ContextPath),
			"ruleStackPath": normalizeRuntimePath(composeResult.RuleStackPath),
			"tracePath":     normalizeRuntimePath(composeResult.TracePath),
		}), nil

	case "write_draft":
		bookID := asString(args["bookId"])
		if guardErr, err := getSequentialWriteGuardError(stateManager, bookID, "write_draft"); err != nil {
			return "", err
		} else if guardErr != "" {
			return mustJSON(map[string]any{"error": guardErr}), nil
		}
		nextChapter, err := stateManager.GetNextChapterNumber(bookID)
		if err != nil {
			return "", err
		}
		result, err := pipeline.RunChapterPipeline(ctx, bookID, nextChapter)
		if err != nil {
			return "", err
		}
		return mustJSON(result), nil

	case "write_full_pipeline":
		bookID := asString(args["bookId"])
		if guardErr, err := getSequentialWriteGuardError(stateManager, bookID, "write_full_pipeline"); err != nil {
			return "", err
		} else if guardErr != "" {
			return mustJSON(map[string]any{"error": guardErr}), nil
		}
		count := asInt(args["count"], 1)
		if count <= 0 {
			count = 1
		}
		results := make([]*ChapterPipelineResult, 0, count)
		for i := 0; i < count; i++ {
			nextChapter, err := stateManager.GetNextChapterNumber(bookID)
			if err != nil {
				return "", err
			}
			result, err := pipeline.RunChapterPipeline(ctx, bookID, nextChapter)
			if err != nil {
				return "", err
			}
			results = append(results, result)
			if result.Status != string(modelsStatusReadyForReview()) {
				break
			}
		}
		return mustJSON(results), nil

	case "update_author_intent":
		bookID := asString(args["bookId"])
		content := asString(args["content"])
		if err := stateManager.EnsureControlDocuments(bookID, ""); err != nil {
			return "", err
		}
		storyDir := filepath.Join(stateManager.BookDir(bookID), "story")
		if err := os.WriteFile(filepath.Join(storyDir, "author_intent.md"), []byte(content), 0644); err != nil {
			return "", err
		}
		return mustJSON(map[string]any{"bookId": bookID, "file": "story/author_intent.md", "written": true}), nil

	case "update_current_focus":
		bookID := asString(args["bookId"])
		content := asString(args["content"])
		if err := stateManager.EnsureControlDocuments(bookID, ""); err != nil {
			return "", err
		}
		storyDir := filepath.Join(stateManager.BookDir(bookID), "story")
		if err := os.WriteFile(filepath.Join(storyDir, "current_focus.md"), []byte(content), 0644); err != nil {
			return "", err
		}
		return mustJSON(map[string]any{"bookId": bookID, "file": "story/current_focus.md", "written": true}), nil

	case "get_book_status":
		bookID := asString(args["bookId"])
		bookConfig, err := stateManager.LoadBookConfig(bookID)
		if err != nil {
			return "", err
		}
		index, err := stateManager.LoadChapterIndex(bookID)
		if err != nil {
			return "", err
		}
		next, err := stateManager.GetNextChapterNumber(bookID)
		if err != nil {
			return "", err
		}
		lastChapter := 0
		if len(index) > 0 {
			lastChapter = index[len(index)-1].Number
		}
		return mustJSON(map[string]any{
			"bookId":         bookConfig.ID,
			"title":          bookConfig.Title,
			"status":         bookConfig.Status,
			"targetChapters": bookConfig.TargetChapters,
			"chapterCount":   len(index),
			"lastChapter":    lastChapter,
			"nextChapter":    next,
		}), nil

	case "read_truth_files":
		bookID := asString(args["bookId"])
		storyDir := filepath.Join(stateManager.BookDir(bookID), "story")
		files := []string{
			"story_bible.md", "volume_outline.md", "book_rules.md",
			"current_state.md", "particle_ledger.md", "pending_hooks.md",
			"chapter_summaries.md", "subplot_board.md", "emotional_arcs.md", "character_matrix.md",
		}
		result := map[string]string{}
		for _, name := range files {
			path := filepath.Join(storyDir, name)
			if raw, err := os.ReadFile(path); err == nil {
				result[name] = string(raw)
			} else {
				result[name] = ""
			}
		}
		return mustJSON(result), nil

	case "list_books":
		bookIDs, err := stateManager.ListBooks()
		if err != nil {
			return "", err
		}
		return mustJSON(bookIDs), nil

	case "write_truth_file":
		bookID := asString(args["bookId"])
		fileName := asString(args["fileName"])
		content := asString(args["content"])

		allowed := map[string]struct{}{
			"story_bible.md": {}, "volume_outline.md": {}, "book_rules.md": {},
			"current_state.md": {}, "particle_ledger.md": {}, "pending_hooks.md": {},
			"chapter_summaries.md": {}, "subplot_board.md": {}, "emotional_arcs.md": {},
			"character_matrix.md": {}, "style_guide.md": {},
		}
		if _, ok := allowed[fileName]; !ok {
			return mustJSON(map[string]any{"error": "file is not allowed to be overwritten"}), nil
		}
		if fileName == "current_state.md" && containsProgressManipulation(content) {
			return mustJSON(map[string]any{"error": "不允许通过 write_truth_file 修改 current_state.md 中的章节进度。章节进度由系统自动管理。"}), nil
		}
		storyDir := filepath.Join(stateManager.BookDir(bookID), "story")
		if err := os.MkdirAll(storyDir, 0755); err != nil {
			return "", err
		}
		path := filepath.Join(storyDir, fileName)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return "", err
		}
		return mustJSON(map[string]any{
			"bookId":  bookID,
			"file":    "story/" + fileName,
			"written": true,
			"size":    len(content),
		}), nil

	default:
		return mustJSON(map[string]any{"error": "Unknown tool: " + name}), nil
	}
}

func getSequentialWriteGuardError(stateManager *state.StateManager, bookID string, toolName string) (string, error) {
	nextNum, err := stateManager.GetNextChapterNumber(bookID)
	if err != nil {
		return "", err
	}
	index, err := stateManager.LoadChapterIndex(bookID)
	if err != nil {
		return "", err
	}
	if len(index) == 0 {
		return "", nil
	}
	lastIndexedChapter := index[len(index)-1].Number
	if lastIndexedChapter == nextNum-1 {
		return "", nil
	}
	return fmt.Sprintf("%s 只能续写下一章（当前应写第 %d 章）。检测到章节索引与运行时进度不一致，请先用 get_book_status 确认状态。", toolName, nextNum), nil
}

func containsProgressManipulation(content string) bool {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\blastAppliedChapter\b`),
		regexp.MustCompile(`(?i)\|\s*Current Chapter\s*\|\s*\d+\s*\|`),
		regexp.MustCompile(`\|\s*当前章节\s*\|\s*\d+\s*\|`),
		regexp.MustCompile(`(?i)\bCurrent Chapter\b\s*[:：]\s*\d+`),
		regexp.MustCompile(`当前章节\s*[:：]\s*\d+`),
		regexp.MustCompile(`(?i)\bprogress\b\s*[:：]\s*\d+`),
		regexp.MustCompile(`进度\s*[:：]\s*\d+`),
	}
	for _, p := range patterns {
		if p.MatchString(content) {
			return true
		}
	}
	return false
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		return ""
	}
}

func asInt(v any, fallback int) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case string:
		if parsed, err := strconv.Atoi(strings.TrimSpace(t)); err == nil {
			return parsed
		}
	}
	return fallback
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return `{"error":"json_marshal_failed"}`
	}
	return string(b)
}

func normalizeRuntimePath(path string) string {
	normalized := filepath.ToSlash(strings.TrimSpace(path))
	if normalized == "" {
		return ""
	}
	if strings.HasPrefix(normalized, "story/") {
		return normalized
	}
	if strings.HasPrefix(normalized, "runtime/") {
		return "story/" + normalized
	}
	if strings.Contains(normalized, "/story/") {
		idx := strings.Index(normalized, "/story/")
		return normalized[idx+1:]
	}
	return filepath.ToSlash(filepath.Join("story", "runtime", filepath.Base(normalized)))
}

func modelsStatusReadyForReview() string {
	return "ready-for-review"
}

var _ = utils.SplitChapters
