package agents

import (
	"context"

	"github.com/icosmos-space/ipen/core/llm"
	"github.com/icosmos-space/ipen/core/utils"
)

type LlmAgentInterface interface {
}

// AgentContext agent上下文。
type AgentContext struct {
	// Client LLM客户端。
	Client *llm.LLMClient
	// Model LLM模型。
	Model string
	// ProjectRoot 项目根目录。
	ProjectRoot string
	// BookID 书籍ID。
	BookID string
	// Logger 日志记录器。
	Logger utils.Logger
	// OnStreamChunk 流式文本分片回调。
	OnStreamChunk llm.OnStreamChunk
	// OnStreamProgress 流式进度回调。
	OnStreamProgress llm.OnStreamProgress
}

// BaseAgent 基础agent。
type BaseAgent struct {
	Ctx AgentContext
}

// NewBaseAgent 创建新的基础agent。
func NewBaseAgent(ctx AgentContext) *BaseAgent {
	return &BaseAgent{Ctx: ctx}
}

// Log 返回日志记录器。
func (a *BaseAgent) Log() utils.Logger {
	return a.Ctx.Logger
}

func (a *BaseAgent) normalizeChatOptions(options *llm.ChatOptions) *llm.ChatOptions {
	if options == nil {
		options = &llm.ChatOptions{}
	} else {
		cloned := *options
		options = &cloned
	}

	if options.StreamLabel == "" {
		options.StreamLabel = a.Name()
	}
	if options.OnStreamChunk == nil {
		options.OnStreamChunk = a.Ctx.OnStreamChunk
	}
	if options.OnStreamProgress == nil {
		options.OnStreamProgress = a.Ctx.OnStreamProgress
	}

	return options
}

// Chat 执行一个LLM对话。
func (a *BaseAgent) Chat(ctx context.Context, messages []llm.LLMMessage, options *llm.ChatOptions) (*llm.LLMResponse, error) {
	return llm.ChatCompletion(ctx, a.Ctx.Client, a.Ctx.Model, messages, a.normalizeChatOptions(options))
}

// ChatWithSearch 执行一个开启web搜索的对话
//
//	OpenAI: 使用模型原生web_search_options / web_search_preview.
//	Other providers: 通过Tavily API (TAVILY_API_KEY), 注入结果到prompt中.
func (a *BaseAgent) ChatWithSearch(ctx context.Context, messages []llm.LLMMessage, options *llm.ChatOptions) (*llm.LLMResponse, error) {
	if a.Ctx.Client.Provider == "openai" {
		options = a.normalizeChatOptions(options)
		options.WebSearch = true
		return llm.ChatCompletion(ctx, a.Ctx.Client, a.Ctx.Model, messages, options)
	}

	// 对于非OpenAI providers, 回退到常规对话
	return a.Chat(ctx, messages, options)
}

// Name 返回agent名称。
func (a *BaseAgent) Name() string {
	return "base"
}
