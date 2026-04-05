package agents

import (
	"context"

	"github.com/icosmos-space/ipen/core/llm"
	"github.com/icosmos-space/ipen/core/utils"
)

// AgentContext 表示the agent context。
type AgentContext struct {
	Client           *llm.LLMClient
	Model            string
	ProjectRoot      string
	BookID           string
	Logger           utils.Logger
	OnStreamProgress llm.OnStreamProgress
}

// BaseAgent 表示the base agent。
type BaseAgent struct {
	Ctx AgentContext
}

// NewBaseAgent 创建新的base agent。
func NewBaseAgent(ctx AgentContext) *BaseAgent {
	return &BaseAgent{Ctx: ctx}
}

// Log 返回the logger。
func (a *BaseAgent) Log() utils.Logger {
	return a.Ctx.Logger
}

// Chat 执行a chat with LLM。
func (a *BaseAgent) Chat(ctx context.Context, messages []llm.LLMMessage, options *llm.ChatOptions) (*llm.LLMResponse, error) {
	return llm.ChatCompletion(ctx, a.Ctx.Client, a.Ctx.Model, messages, options)
}

// ChatWithSearch 执行a chat with web search enable。
//
//	OpenAI: uses native web_search_options / web_search_preview.
//	Other providers: searches via Tavily API (TAVILY_API_KEY), injects results into prompt.
func (a *BaseAgent) ChatWithSearch(ctx context.Context, messages []llm.LLMMessage, options *llm.ChatOptions) (*llm.LLMResponse, error) {
	// OpenAI has native search
	if a.Ctx.Client.Provider == "openai" {
		if options == nil {
			options = &llm.ChatOptions{}
		}
		options.WebSearch = true
		return llm.ChatCompletion(ctx, a.Ctx.Client, a.Ctx.Model, messages, options)
	}

	// For non-OpenAI providers, fall back to regular chat for now.
	return a.Chat(ctx, messages, options)
}

// Name 返回the agent name (to be implemented by subclasses)。
func (a *BaseAgent) Name() string {
	return "base"
}
