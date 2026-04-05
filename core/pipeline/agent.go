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

// AGENT_TOOLS 是the tool registry exposed to external agent loops。
var AGENT_TOOLS = []llm.ToolDefinition{
	{Name: "write_draft", Description: "Write the next sequential chapter.", Parameters: map[string]any{"type": "object", "properties": map[string]any{"bookId": map[string]any{"type": "string"}, "guidance": map[string]any{"type": "string"}}, "required": []string{"bookId"}}},
	{Name: "plan_chapter", Description: "Generate chapter intent for the next chapter.", Parameters: map[string]any{"type": "object", "properties": map[string]any{"bookId": map[string]any{"type": "string"}, "guidance": map[string]any{"type": "string"}}, "required": []string{"bookId"}}},
	{Name: "compose_chapter", Description: "Build governed context package for next chapter.", Parameters: map[string]any{"type": "object", "properties": map[string]any{"bookId": map[string]any{"type": "string"}, "guidance": map[string]any{"type": "string"}}, "required": []string{"bookId"}}},
	{Name: "write_full_pipeline", Description: "Run full write pipeline for one or more chapters.", Parameters: map[string]any{"type": "object", "properties": map[string]any{"bookId": map[string]any{"type": "string"}, "count": map[string]any{"type": "number"}}, "required": []string{"bookId"}}},
	{Name: "update_author_intent", Description: "Overwrite story/author_intent.md.", Parameters: map[string]any{"type": "object", "properties": map[string]any{"bookId": map[string]any{"type": "string"}, "content": map[string]any{"type": "string"}}, "required": []string{"bookId", "content"}}},
	{Name: "update_current_focus", Description: "Overwrite story/current_focus.md.", Parameters: map[string]any{"type": "object", "properties": map[string]any{"bookId": map[string]any{"type": "string"}, "content": map[string]any{"type": "string"}}, "required": []string{"bookId", "content"}}},
	{Name: "get_book_status", Description: "Get summary status for one book.", Parameters: map[string]any{"type": "object", "properties": map[string]any{"bookId": map[string]any{"type": "string"}}, "required": []string{"bookId"}}},
	{Name: "read_truth_files", Description: "Read key story truth files.", Parameters: map[string]any{"type": "object", "properties": map[string]any{"bookId": map[string]any{"type": "string"}}, "required": []string{"bookId"}}},
	{Name: "list_books", Description: "List all books.", Parameters: map[string]any{"type": "object", "properties": map[string]any{}}},
	{Name: "write_truth_file", Description: "Replace one truth file content.", Parameters: map[string]any{"type": "object", "properties": map[string]any{"bookId": map[string]any{"type": "string"}, "fileName": map[string]any{"type": "string"}, "content": map[string]any{"type": "string"}}, "required": []string{"bookId", "fileName", "content"}}},
}

// AgentLoopOptions configures RunAgentLoop callbacks.
type AgentLoopOptions struct {
	OnToolCall   func(name string, args map[string]any)
	OnToolResult func(name string, result string)
	OnMessage    func(content string)
	MaxTurns     int
}

// RunAgentLoop runs an LLM tool loop against pipeline tools.
func RunAgentLoop(ctx context.Context, config PipelineConfig, instruction string, options *AgentLoopOptions) (string, error) {
	pipeline := NewPipelineRunner(config)
	stateManager := state.NewStateManager(config.ProjectRoot)

	messages := []llm.AgentMessage{
		{
			Role:    "system",
			Content: "You are the iPen writing assistant. Use tools to execute concrete book operations; keep outputs concise and actionable.",
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
			messages = append(messages, llm.AgentMessage{
				Role:       "tool",
				ToolCallID: toolCall.ID,
				Content:    toolResult,
			})
		}
	}

	return lastAssistantMessage, nil
}

// ExecuteAgentTool executes one tool call and returns JSON payload string.
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
