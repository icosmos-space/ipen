package llm

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/anthropics/anthropic-sdk-go"
	aoption "github.com/anthropics/anthropic-sdk-go/option"
	"github.com/icosmos-space/ipen/core/models"
	"github.com/openai/openai-go"
	ooption "github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
)

// Minimum chars to a partial responseable
const minSalvageableChars = 500

// StreamProgress 流响应进度
type StreamProgress struct {
	ElapsedMs    int64  `json:"elapsedMs"`
	TotalChars   int    `json:"totalChars"`
	ChineseChars int    `json:"chineseChars"`
	Status       string `json:"status"` // "streaming" or "done"
}

// StreamChunk 流响应文本分片
type StreamChunk struct {
	Text         string `json:"text"`
	Label        string `json:"label,omitempty"`
	ElapsedMs    int64  `json:"elapsedMs"`
	TotalChars   int    `json:"totalChars"`
	ChineseChars int    `json:"chineseChars"`
	Status       string `json:"status"` // "streaming"
}

// streamMonitor 管理流响应进度
type streamMonitor struct {
	// 总字符数
	totalChars int
	// 中文字符数
	chineseChars    int
	startTime       time.Time
	label           string
	onChunkCallback OnStreamChunk
	onProgress      OnStreamProgress
}

func createStreamMonitor(label string, onChunk OnStreamChunk, onProgress OnStreamProgress) *streamMonitor {
	monitor := &streamMonitor{
		startTime:       time.Now(),
		label:           label,
		onChunkCallback: onChunk,
		onProgress:      onProgress,
	}
	return monitor
}

func (m *streamMonitor) onChunk(text string) {
	if text == "" {
		return
	}
	m.totalChars += utf8.RuneCountInString(text)
	m.chineseChars += countChineseChars(text)
	elapsedMs := time.Since(m.startTime).Milliseconds()
	if m.onChunkCallback != nil {
		m.onChunkCallback(StreamChunk{
			Text:         text,
			Label:        m.label,
			ElapsedMs:    elapsedMs,
			TotalChars:   m.totalChars,
			ChineseChars: m.chineseChars,
			Status:       "streaming",
		})
	}
	if m.onProgress != nil {
		m.onProgress(StreamProgress{
			ElapsedMs:    elapsedMs,
			TotalChars:   m.totalChars,
			ChineseChars: m.chineseChars,
			Status:       "streaming",
		})
	}
}

func (m *streamMonitor) stop() {
	if m.onProgress != nil {
		m.onProgress(StreamProgress{
			ElapsedMs:    time.Since(m.startTime).Milliseconds(),
			TotalChars:   m.totalChars,
			ChineseChars: m.chineseChars,
			Status:       "done",
		})
	}
}

// ErrorContext provides context for error messages
type ErrorContext struct {
	BaseURL string
	Model   string
}

// OnStreamProgress 流响应进度回调函数
type OnStreamProgress func(progress StreamProgress)

// OnStreamChunk 流响应文本分片回调函数
type OnStreamChunk func(chunk StreamChunk)

// LLMResponse 表示an LLM response。
type LLMResponse struct {
	Content string            `json:"content"`
	Usage   models.TokenUsage `json:"usage"`
}

// LLMMessage 表示an LLM message。
type LLMMessage struct {
	Role    string `json:"role"` // "system", "user", "assistant"
	Content string `json:"content"`
}

// ToolDefinition 表示a tool definition。
type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// ToolCall 工具调用
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// AgentMessage Agent消息
type AgentMessage struct {
	// 角色
	Role string `json:"role"` // "system", "user", "assistant", "tool"
	// 内容
	Content string `json:"content,omitempty"`
	// 工具调用
	ToolCalls []ToolCall `json:"toolCalls,omitempty"`
	// 工具调用ID
	ToolCallID string `json:"toolCallId,omitempty"`
}

// ChatWithToolsResult 与工具调用的聊天结果
type ChatWithToolsResult struct {
	// 内容
	Content string `json:"content"`
	// 工具调用
	ToolCalls []ToolCall `json:"toolCalls"`
}

// PartialResponseError 表示a partial response error。
type PartialResponseError struct {
	PartialContent string
	Cause          error
}

func (e *PartialResponseError) Error() string {
	return fmt.Sprintf("Stream interrupted after %d chars: %v", utf8.RuneCountInString(e.PartialContent), e.Cause)
}

// ChatOptions 表示options for chat completion。
type ChatOptions struct {
	Temperature      float64
	MaxTokens        int
	WebSearch        bool
	StreamLabel      string
	OnStreamChunk    OnStreamChunk
	OnStreamProgress OnStreamProgress
}

// LLMClient 表示an LLM client using SDKs。
type LLMClient struct {
	Provider  string // "openai" or "anthropic"
	APIFormat string // "chat" or "responses"
	Stream    bool
	Defaults  LLMDefaults
	OpenAI    *openai.Client
	Anthropic *anthropic.Client
	Model     string
	BaseURL   string // Store base URL for error messages
}

// LLMDefaults 表示LLM defaults。
type LLMDefaults struct {
	Temperature    float64
	MaxTokens      int
	MaxTokensCap   *int
	ThinkingBudget int
	Extra          map[string]any
}

// NewLLMClient 创建新的LLM client from config using SDKs。
func NewLLMClient(config models.LLMConfig) *LLMClient {
	defaults := LLMDefaults{
		Temperature:    config.Temperature,
		MaxTokens:      config.MaxTokens,
		ThinkingBudget: config.ThinkingBudget,
		Extra:          config.Extra,
	}

	if config.MaxTokens > 0 {
		maxTokens := config.MaxTokens
		defaults.MaxTokensCap = &maxTokens
	}

	apiFormat := config.APIFormat
	if apiFormat == "" {
		apiFormat = "chat"
	}

	client := &LLMClient{
		Provider:  config.Provider,
		APIFormat: apiFormat,
		Stream:    config.Stream,
		Defaults:  defaults,
		Model:     config.Model,
		BaseURL:   config.BaseURL,
	}

	// Initialize SDK clients
	if config.Provider == "anthropic" {
		baseURL := strings.TrimSuffix(config.BaseURL, "/v1/")
		anthropicClient := anthropic.NewClient(
			aoption.WithBaseURL(baseURL),
			aoption.WithAPIKey(config.APIKey),
		)
		client.Anthropic = &anthropicClient
	} else {
		// openai or custom
		openaiClient := openai.NewClient(
			ooption.WithAPIKey(config.APIKey),
			ooption.WithBaseURL(config.BaseURL),
		)
		client.OpenAI = &openaiClient
	}

	return client
}

// ChatCompletion 执行a chat completion using SDK。
func ChatCompletion(
	ctx context.Context,
	client *LLMClient,
	model string,
	messages []LLMMessage,
	options *ChatOptions,
) (*LLMResponse, error) {
	if options == nil {
		options = &ChatOptions{}
	}

	temp := options.Temperature
	if temp == 0 {
		temp = client.Defaults.Temperature
	}
	maxTokens := options.MaxTokens
	if maxTokens == 0 {
		maxTokens = client.Defaults.MaxTokens
	}

	if client.Defaults.MaxTokensCap != nil && maxTokens > *client.Defaults.MaxTokensCap {
		maxTokens = *client.Defaults.MaxTokensCap
	}

	errorCtx := ErrorContext{
		BaseURL: getBaseURL(client),
		Model:   model,
	}

	// Try streaming first if enabled
	if client.Stream {
		if client.Provider == "anthropic" {
			resp, err := chatCompletionAnthropic(
				ctx,
				client,
				model,
				messages,
				temp,
				maxTokens,
				options.StreamLabel,
				options.OnStreamChunk,
				options.OnStreamProgress,
			)
			if err != nil {
				// If partial response, return it
				if partialErr, ok := err.(*PartialResponseError); ok {
					return &LLMResponse{
						Content: partialErr.PartialContent,
						Usage:   models.TokenUsage{},
					}, nil
				}
				// Try sync fallback for stream-related errors
				if isLikelyStreamError(err) {
					return chatCompletionAnthropicSync(ctx, client, model, messages, temp, maxTokens)
				}
				return nil, WrapLLMError(err, errorCtx)
			}
			return resp, nil
		}
		resp, err := chatCompletionOpenAI(
			ctx,
			client,
			model,
			messages,
			temp,
			maxTokens,
			options.WebSearch,
			options.StreamLabel,
			options.OnStreamChunk,
			options.OnStreamProgress,
		)
		if err != nil {
			// If partial response, return it
			if partialErr, ok := err.(*PartialResponseError); ok {
				return &LLMResponse{
					Content: partialErr.PartialContent,
					Usage:   models.TokenUsage{},
				}, nil
			}
			// Try sync fallback for stream-related errors
			if isLikelyStreamError(err) {
				return chatCompletionOpenAISync(ctx, client, model, messages, temp, maxTokens, options.WebSearch)
			}
			return nil, WrapLLMError(err, errorCtx)
		}
		return resp, nil
	}

	// Non-streaming mode
	if client.Provider == "anthropic" {
		return chatCompletionAnthropicSync(ctx, client, model, messages, temp, maxTokens)
	}

	return chatCompletionOpenAISync(ctx, client, model, messages, temp, maxTokens, options.WebSearch)
}

func getBaseURL(client *LLMClient) string {
	if client.BaseURL != "" {
		return client.BaseURL
	}
	return "(unknown)"
}

// ChatWithTools 执行a chat with tools using SDK。
func ChatWithTools(
	ctx context.Context,
	client *LLMClient,
	messages []AgentMessage,
	tools []ToolDefinition,
	options *ChatOptions,
) (*ChatWithToolsResult, error) {
	if options == nil {
		options = &ChatOptions{}
	}

	temp := options.Temperature
	if temp == 0 {
		temp = client.Defaults.Temperature
	}
	maxTokens := options.MaxTokens
	if maxTokens == 0 {
		maxTokens = client.Defaults.MaxTokens
	}

	if client.Provider == "anthropic" {
		return chatWithToolsAnthropic(ctx, client, messages, tools, temp, maxTokens)
	}

	return chatWithToolsOpenAI(ctx, client, messages, tools, temp, maxTokens)
}

// Anthropic chat completion using SDK (streaming)
func chatCompletionAnthropic(
	ctx context.Context,
	client *LLMClient,
	model string,
	messages []LLMMessage,
	temperature float64,
	maxTokens int,
	streamLabel string,
	onStreamChunk OnStreamChunk,
	onStreamProgress OnStreamProgress,
) (*LLMResponse, error) {
	// Extract system message
	var systemText string
	var nonSystemMessages []anthropic.MessageParam

	for _, msg := range messages {
		if msg.Role == "system" {
			systemText = msg.Content
		} else {
			nonSystemMessages = append(nonSystemMessages, anthropic.MessageParam{
				Role: anthropic.MessageParamRole(msg.Role),
				Content: []anthropic.ContentBlockParamUnion{
					{OfText: &anthropic.TextBlockParam{Text: msg.Content}},
				},
			})
		}
	}

	// Build chat request
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: int64(maxTokens),
		Messages:  nonSystemMessages,
	}

	if systemText != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: systemText},
		}
	}

	return streamAnthropic(ctx, client, params, streamLabel, onStreamChunk, onStreamProgress)
}

// Anthropic sync chat completion
func chatCompletionAnthropicSync(
	ctx context.Context,
	client *LLMClient,
	model string,
	messages []LLMMessage,
	temperature float64,
	maxTokens int,
) (*LLMResponse, error) {
	// Extract system message
	var systemText string
	var nonSystemMessages []anthropic.MessageParam

	for _, msg := range messages {
		if msg.Role == "system" {
			systemText = msg.Content
		} else {
			nonSystemMessages = append(nonSystemMessages, anthropic.MessageParam{
				Role: anthropic.MessageParamRole(msg.Role),
				Content: []anthropic.ContentBlockParamUnion{
					{OfText: &anthropic.TextBlockParam{Text: msg.Content}},
				},
			})
		}
	}

	// Build chat request
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: int64(maxTokens),
		Messages:  nonSystemMessages,
	}

	if systemText != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: systemText},
		}
	}

	// Non-streaming
	msg, err := client.Anthropic.Messages.New(ctx, params)
	if err != nil {
		return nil, err
	}

	if len(msg.Content) == 0 {
		return nil, fmt.Errorf("LLM returned empty response")
	}

	content := msg.Content[0].Text
	return &LLMResponse{
		Content: content,
		Usage: models.TokenUsage{
			PromptTokens:     int(msg.Usage.InputTokens),
			CompletionTokens: int(msg.Usage.OutputTokens),
			TotalTokens:      int(msg.Usage.InputTokens + msg.Usage.OutputTokens),
		},
	}, nil
}

// Stream Anthropic response
func streamAnthropic(
	ctx context.Context,
	client *LLMClient,
	params anthropic.MessageNewParams,
	streamLabel string,
	onStreamChunk OnStreamChunk,
	onStreamProgress OnStreamProgress,
) (*LLMResponse, error) {
	stream := client.Anthropic.Messages.NewStreaming(ctx, params)

	var content strings.Builder
	var inputTokens, outputTokens int64
	monitor := createStreamMonitor(streamLabel, onStreamChunk, onStreamProgress)

	defer monitor.stop()

	for stream.Next() {
		event := stream.Current()

		// Check if this is a content block delta event
		contentBlockDelta := event.AsContentBlockDelta()
		if contentBlockDelta.Delta.Text != "" {
			content.WriteString(contentBlockDelta.Delta.Text)
			monitor.onChunk(contentBlockDelta.Delta.Text)
		}

		// Track usage if available
		if event.Usage.InputTokens > 0 {
			inputTokens = event.Usage.InputTokens
		}
		if event.Usage.OutputTokens > 0 {
			outputTokens = event.Usage.OutputTokens
		}
	}

	if err := stream.Err(); err != nil {
		partial := content.String()
		if utf8.RuneCountInString(partial) >= minSalvageableChars {
			return nil, &PartialResponseError{PartialContent: partial, Cause: err}
		}
		return nil, err
	}

	result := content.String()
	if result == "" {
		return nil, fmt.Errorf("LLM returned empty response from stream")
	}

	return &LLMResponse{
		Content: result,
		Usage: models.TokenUsage{
			PromptTokens:     int(inputTokens),
			CompletionTokens: int(outputTokens),
			TotalTokens:      int(inputTokens + outputTokens),
		},
	}, nil
}

// OpenAI chat completion using SDK (streaming)
func chatCompletionOpenAI(
	ctx context.Context,
	client *LLMClient,
	model string,
	messages []LLMMessage,
	temperature float64,
	maxTokens int,
	webSearch bool,
	streamLabel string,
	onStreamChunk OnStreamChunk,
	onStreamProgress OnStreamProgress,
) (*LLMResponse, error) {
	// Convert messages
	var openaiMessages []openai.ChatCompletionMessageParamUnion
	for _, msg := range messages {
		switch msg.Role {
		case "system":
			openaiMessages = append(openaiMessages, openai.SystemMessage(msg.Content))
		case "user":
			openaiMessages = append(openaiMessages, openai.UserMessage(msg.Content))
		case "assistant":
			openaiMessages = append(openaiMessages, openai.AssistantMessage(msg.Content))
		}
	}

	params := openai.ChatCompletionNewParams{
		Model:       model,
		Messages:    openaiMessages,
		Temperature: param.NewOpt(temperature),
		MaxTokens:   param.NewOpt(int64(maxTokens)),
	}

	return streamOpenAI(ctx, client, params, streamLabel, onStreamChunk, onStreamProgress)
}

// OpenAI sync chat completion
func chatCompletionOpenAISync(
	ctx context.Context,
	client *LLMClient,
	model string,
	messages []LLMMessage,
	temperature float64,
	maxTokens int,
	webSearch bool,
) (*LLMResponse, error) {
	// Convert messages
	var openaiMessages []openai.ChatCompletionMessageParamUnion
	for _, msg := range messages {
		switch msg.Role {
		case "system":
			openaiMessages = append(openaiMessages, openai.SystemMessage(msg.Content))
		case "user":
			openaiMessages = append(openaiMessages, openai.UserMessage(msg.Content))
		case "assistant":
			openaiMessages = append(openaiMessages, openai.AssistantMessage(msg.Content))
		}
	}

	params := openai.ChatCompletionNewParams{
		Model:       model,
		Messages:    openaiMessages,
		Temperature: param.NewOpt(temperature),
		MaxTokens:   param.NewOpt(int64(maxTokens)),
	}

	// Non-streaming
	chat, err := client.OpenAI.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, err
	}

	if len(chat.Choices) == 0 {
		return nil, fmt.Errorf("LLM returned empty response")
	}

	content := chat.Choices[0].Message.Content
	if content == "" {
		return nil, fmt.Errorf("LLM returned empty response")
	}

	return &LLMResponse{
		Content: content,
		Usage: models.TokenUsage{
			PromptTokens:     int(chat.Usage.PromptTokens),
			CompletionTokens: int(chat.Usage.CompletionTokens),
			TotalTokens:      int(chat.Usage.TotalTokens),
		},
	}, nil
}

// Stream OpenAI response
func streamOpenAI(
	ctx context.Context,
	client *LLMClient,
	params openai.ChatCompletionNewParams,
	streamLabel string,
	onStreamChunk OnStreamChunk,
	onStreamProgress OnStreamProgress,
) (*LLMResponse, error) {
	stream := client.OpenAI.Chat.Completions.NewStreaming(ctx, params)
	var content strings.Builder
	var inputTokens, outputTokens int64
	monitor := createStreamMonitor(streamLabel, onStreamChunk, onStreamProgress)

	defer monitor.stop()

	for stream.Next() {
		event := stream.Current()
		if len(event.Choices) > 0 {
			delta := event.Choices[0].Delta
			if delta.Content != "" {
				content.WriteString(delta.Content)
				monitor.onChunk(delta.Content)
			}
		}

		if event.Usage.PromptTokens > 0 {
			inputTokens = event.Usage.PromptTokens
		}
		if event.Usage.CompletionTokens > 0 {
			outputTokens = event.Usage.CompletionTokens
		}
	}

	if err := stream.Err(); err != nil {
		partial := content.String()
		if utf8.RuneCountInString(partial) >= minSalvageableChars {
			return nil, &PartialResponseError{PartialContent: partial, Cause: err}
		}
		return nil, err
	}

	result := content.String()
	if result == "" {
		return nil, fmt.Errorf("LLM returned empty response from stream")
	}

	return &LLMResponse{
		Content: result,
		Usage: models.TokenUsage{
			PromptTokens:     int(inputTokens),
			CompletionTokens: int(outputTokens),
			TotalTokens:      int(inputTokens + outputTokens),
		},
	}, nil
}

// Chat with tools - Anthropic
func chatWithToolsAnthropic(
	ctx context.Context,
	client *LLMClient,
	messages []AgentMessage,
	tools []ToolDefinition,
	temperature float64,
	maxTokens int,
) (*ChatWithToolsResult, error) {
	// Convert tools to Anthropic format
	var toolParams []anthropic.ToolUnionParam
	for _, tool := range tools {
		toolParams = append(toolParams, anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        tool.Name,
				Description: anthropic.String(tool.Description),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: tool.Parameters,
					Type:       "object",
				},
			},
		})
	}

	// Convert messages
	var anthropicMessages []anthropic.MessageParam
	var systemText string
	for _, msg := range messages {
		if msg.Role == "system" {
			systemText = msg.Content
		} else if msg.Role == "user" {
			anthropicMessages = append(anthropicMessages, anthropic.MessageParam{
				Role: anthropic.MessageParamRoleUser,
				Content: []anthropic.ContentBlockParamUnion{
					{OfText: &anthropic.TextBlockParam{Text: msg.Content}},
				},
			})
		} else if msg.Role == "assistant" {
			anthropicMessages = append(anthropicMessages, anthropic.MessageParam{
				Role: anthropic.MessageParamRoleAssistant,
				Content: []anthropic.ContentBlockParamUnion{
					{OfText: &anthropic.TextBlockParam{Text: msg.Content}},
				},
			})
		}
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(client.Model),
		MaxTokens: int64(maxTokens),
		Messages:  anthropicMessages,
		Tools:     toolParams,
	}

	if systemText != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: systemText},
		}
	}

	msg, err := client.Anthropic.Messages.New(ctx, params)
	if err != nil {
		return nil, WrapLLMError(err, ErrorContext{BaseURL: getBaseURL(client), Model: client.Model})
	}

	var content string
	var toolCalls []ToolCall

	for _, block := range msg.Content {
		if block.Type == "text" {
			content += block.Text
		} else if block.Type == "tool_use" {
			toolCalls = append(toolCalls, ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: string(block.Input),
			})
		}
	}

	return &ChatWithToolsResult{
		Content:   content,
		ToolCalls: toolCalls,
	}, nil
}

// Chat with tools - OpenAI
func chatWithToolsOpenAI(
	ctx context.Context,
	client *LLMClient,
	messages []AgentMessage,
	tools []ToolDefinition,
	temperature float64,
	maxTokens int,
) (*ChatWithToolsResult, error) {
	// Convert tools to OpenAI format
	var openaiTools []openai.ChatCompletionToolParam
	for _, tool := range tools {
		openaiTools = append(openaiTools, openai.ChatCompletionToolParam{
			Function: openai.FunctionDefinitionParam{
				Name:        tool.Name,
				Description: param.NewOpt(tool.Description),
				Parameters:  tool.Parameters,
			},
		})
	}

	// Convert messages
	var openaiMessages []openai.ChatCompletionMessageParamUnion
	for _, msg := range messages {
		if msg.Role == "system" {
			openaiMessages = append(openaiMessages, openai.SystemMessage(msg.Content))
		} else if msg.Role == "user" {
			openaiMessages = append(openaiMessages, openai.UserMessage(msg.Content))
		} else if msg.Role == "assistant" {
			openaiMessages = append(openaiMessages, openai.AssistantMessage(msg.Content))
		}
	}

	params := openai.ChatCompletionNewParams{
		Model:       client.Model,
		Messages:    openaiMessages,
		Tools:       openaiTools,
		Temperature: param.NewOpt(temperature),
		MaxTokens:   param.NewOpt(int64(maxTokens)),
	}

	chat, err := client.OpenAI.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, WrapLLMError(err, ErrorContext{BaseURL: getBaseURL(client), Model: client.Model})
	}

	var content string
	var toolCalls []ToolCall

	if len(chat.Choices) > 0 {
		choice := chat.Choices[0]
		content = choice.Message.Content

		if choice.Message.ToolCalls != nil {
			for _, tc := range choice.Message.ToolCalls {
				toolCalls = append(toolCalls, ToolCall{
					ID:        tc.ID,
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				})
			}
		}
	}

	return &ChatWithToolsResult{
		Content:   content,
		ToolCalls: toolCalls,
	}, nil
}

// WrapLLMError wraps LLM errors with user-friendly messages
func WrapLLMError(err error, ctx ErrorContext) error {
	msg := err.Error()
	ctxLine := fmt.Sprintf("\n  (baseUrl: %s, model: %s)", ctx.BaseURL, ctx.Model)

	if strings.Contains(msg, "400") {
		return fmt.Errorf(
			"API 返回 400 (请求参数错误)。可能原因：\n"+
				"  1. 模型名称不正确（检查 IPEN_LLM_MODEL）\n"+
				"  2. 提供方不支持某些参数（如 max_tokens、stream）\n"+
				"  3. 消息格式不兼容（部分提供方不支持 system role）\n"+
				"  建议：检查提供方文档，确认该接口要求流式开启、流式关闭，还是根本不支持 stream %s",
			ctxLine,
		)
	}

	if strings.Contains(msg, "403") {
		return fmt.Errorf(
			"API 返回 403 (请求被拒绝)。可能原因：\n"+
				"  1. API Key 无效或过期\\n"+
				"  2. API 提供方的内容审查拦截了请求（公益/免费 API 常见）\n"+
				"  3. 账户余额不足\n"+
				"  建议：用 ipen doctor 测试 API 连通性，或换一个不限制内容的 API 提供方 %s",
			ctxLine,
		)
	}

	if strings.Contains(msg, "401") {
		return fmt.Errorf("API 返回 401 (未授权)。请检查 API Key 是否正确。%s", ctxLine)
	}

	if strings.Contains(msg, "429") {
		return fmt.Errorf("API 返回 429 (请求过多)。请稍后重试，或检查 API 配额。%s", ctxLine)
	}

	if strings.Contains(msg, "Connection error") ||
		strings.Contains(msg, "ECONNREFUSED") ||
		strings.Contains(msg, "ENOTFOUND") ||
		strings.Contains(msg, "fetch failed") {
		return fmt.Errorf(
			"无法连接到 API 服务。可能原因：\n"+
				"  1. baseUrl 地址不正确（当前：%s）\n"+
				"  2. 网络不通或被防火墙拦截\n"+
				"  3. API 服务暂时不可用\n"+
				"  建议：检查 IPEN_LLM_BASE_URL 是否包含完整路径（如 /v1）",
			ctx.BaseURL,
		)
	}

	return err
}

// isLikelyStreamError 检查if an error is likely related to streaming issues。
func isLikelyStreamError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "stream") ||
		strings.Contains(msg, "text/event-stream") ||
		strings.Contains(msg, "chunked") ||
		strings.Contains(msg, "unexpected end") ||
		strings.Contains(msg, "premature close") ||
		strings.Contains(msg, "terminated") ||
		strings.Contains(msg, "econnreset") ||
		(strings.Contains(msg, "400") && !strings.Contains(msg, "content"))
}

// countChineseChars 统计Chinese characters in text。
func countChineseChars(text string) int {
	count := 0
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			count++
		}
	}
	return count
}
