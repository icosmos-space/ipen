package models

// LLMConfig 表示LLM提供者的配置
// 用于存储LLM提供者的配置信息，包括URL、API密钥、模型、温度、最大令牌数、思考预算等。
type LLMConfig struct {
	//
	Provider string `json:"provider" validate:"oneof=anthropic openai custom"` // "anthropic", "openai", "custom"
	// 基础URL
	BaseURL string `json:"baseUrl"`
	// API密钥
	APIKey string `json:"apiKey"`
	// 模型ID
	Model string `json:"model" validate:"required"`
	// 温度
	Temperature float64 `json:"temperature"`
	// 最大令牌数
	MaxTokens int `json:"maxTokens" validate:"min=1024,max=8192"`
	// 思考预算
	ThinkingBudget int `json:"thinkingBudget"`
	// 额外参数
	Extra map[string]any `json:"extra,omitempty"`
	// 请求头
	Headers map[string]string `json:"headers,omitempty"`
	// API格式
	APIFormat string `json:"apiFormat" validate:"oneof=chat responses"` // "chat" or "responses"
	// 是否流式输出
	Stream bool `json:"stream"`
}

// DefaultLLMConfig 返回带有缺省值的LLM配置
func DefaultLLMConfig() LLMConfig {
	return LLMConfig{
		Temperature:    0.7,
		MaxTokens:      8192,
		ThinkingBudget: 0,
		APIFormat:      "chat",
		Stream:         true,
	}
}

// NotifyChannel 表示a notification channel configuration。
type NotifyChannel struct {
	// 通知类型
	Type string `json:"type"` // "telegram", "wechat-work", "feishu", "webhook"
	// 机器人Token
	BotToken string `json:"botToken,omitempty"`
	// 聊天ID
	ChatID string `json:"chatId,omitempty"`
	// Webhook URL
	WebhookURL string `json:"webhookUrl,omitempty"`
	// 密钥
	Secret string `json:"secret,omitempty"`
	// 事件
	Events []string `json:"events,omitempty"`
}

// DetectionConfig 表示AI detection configuration。
type DetectionConfig struct {
	Provider    string  `json:"provider" validate:"oneof=gptzero originality custom"` // "gptzero", "originality", "custom"
	APIURL      string  `json:"apiUrl"`
	APIKeyEnv   string  `json:"apiKeyEnv"`
	Threshold   float64 `json:"threshold"`
	Enabled     bool    `json:"enabled"`
	AutoRewrite bool    `json:"autoRewrite"`
	MaxRetries  int     `json:"maxRetries" validate:"min=1,max=10"`
}

// DefaultDetectionConfig 返回默认缺省值的AI检测配置
func DefaultDetectionConfig() DetectionConfig {
	return DetectionConfig{
		Provider:   "custom",
		Threshold:  0.5,
		MaxRetries: 3,
	}
}

// QualityGates 表示质量门控配置。
type QualityGates struct {
	// 最大重试次数
	MaxAuditRetries int `json:"maxAuditRetries"`
	// 连续失败次数
	PauseAfterConsecutiveFailures int `json:"pauseAfterConsecutiveFailures"`
	// 重试温度步长
	RetryTemperatureStep float64 `json:"retryTemperatureStep"`
}

// DefaultQualityGates 返回QualityGates with default values。
func DefaultQualityGates() QualityGates {
	return QualityGates{
		MaxAuditRetries:               2,
		PauseAfterConsecutiveFailures: 3,
		RetryTemperatureStep:          0.1,
	}
}

// AgentLLMOverride
type AgentLLMOverride struct {
	// 模型ID
	Model string `json:"model"`
	// 提供方
	Provider *string `json:"provider,omitempty" validate:"oneof=anthropic openai custom"`
	// 基础URL
	BaseURL *string `json:"baseUrl,omitempty"`
	// API密钥环境变量
	APIKeyEnv *string `json:"apiKeyEnv,omitempty"`
	// 是否流式输出
	Stream *bool `json:"stream,omitempty"`
}

// InputGovernanceMode 表示输入治理模式
type InputGovernanceMode string

const (
	GovernanceModeLegacy InputGovernanceMode = "legacy"
	GovernanceModeV2     InputGovernanceMode = "v2"
)

// ScheduleConfig 表示后台调度配置
type ScheduleConfig struct {
	// 雷达Cron
	RadarCron string `json:"radarCron"`
	// 写入Cron
	WriteCron string `json:"writeCron"`
}

// DaemonConfig 表示后台调度配置。
type DaemonConfig struct {
	// 调度配置
	Schedule ScheduleConfig `json:"schedule"`
	// 最大并发书籍数
	MaxConcurrentBooks int `json:"maxConcurrentBooks"`
	// 每个周期的章节数
	ChaptersPerCycle int `json:"chaptersPerCycle"`
	// 重试延迟时间
	RetryDelayMs int `json:"retryDelayMs"`
	// 冷却时间
	CooldownAfterChapterMs int `json:"cooldownAfterChapterMs"`
	// 最大章节数
	MaxChaptersPerDay int `json:"maxChaptersPerDay"`
	// 质量门控配置
	QualityGates QualityGates `json:"qualityGates"`
}

// DefaultDaemonConfig 返回默认缺省值的后台调度配置
func DefaultDaemonConfig() DaemonConfig {
	return DaemonConfig{
		Schedule: ScheduleConfig{
			RadarCron: "0 */6 * * *",
			WriteCron: "*/15 * * * *",
		},
		MaxConcurrentBooks:     3,
		ChaptersPerCycle:       1,
		RetryDelayMs:           30000,
		CooldownAfterChapterMs: 10000,
		MaxChaptersPerDay:      50,
		QualityGates:           DefaultQualityGates(),
	}
}

// ProjectConfig 表示项目配置。
type ProjectConfig struct {
	// 项目名称
	Name string `json:"name"`
	// 项目版本
	Version string `json:"version"` // "0.1.0"
	// 语言
	Language string `json:"language"` // "zh" or "en"
	// LLM 全局配置
	LLM LLMConfig `json:"llm"`
	// 通知渠道
	Notify []NotifyChannel `json:"notify"`
	// 检测配置
	Detection *DetectionConfig `json:"detection,omitempty"`
	// 模型覆写
	ModelOverrides map[string]any `json:"modelOverrides,omitempty"`
	// 输入治理模式
	InputGovernanceMode InputGovernanceMode `json:"inputGovernanceMode"`
	// 后台配置
	Daemon DaemonConfig `json:"daemon"`
}

// DefaultProjectConfig 返回默认缺省值的项目配置
func DefaultProjectConfig() ProjectConfig {
	return ProjectConfig{
		Version:             "0.1.0",
		Language:            "zh",
		InputGovernanceMode: GovernanceModeV2,
		Daemon:              DefaultDaemonConfig(),
	}
}
