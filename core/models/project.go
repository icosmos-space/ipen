package models

// LLMConfig 表示the configuration for an LLM provider。
type LLMConfig struct {
	Provider       string            `json:"provider"` // "anthropic", "openai", "custom"
	BaseURL        string            `json:"baseUrl"`
	APIKey         string            `json:"apiKey"`
	Model          string            `json:"model"`
	Temperature    float64           `json:"temperature"`
	MaxTokens      int               `json:"maxTokens"`
	ThinkingBudget int               `json:"thinkingBudget"`
	Extra          map[string]any    `json:"extra,omitempty"`
	Headers        map[string]string `json:"headers,omitempty"`
	APIFormat      string            `json:"apiFormat"` // "chat" or "responses"
	Stream         bool              `json:"stream"`
}

// DefaultLLMConfig 返回a LLMConfig with default values。
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
	Type       string   `json:"type"` // "telegram", "wechat-work", "feishu", "webhook"
	BotToken   string   `json:"botToken,omitempty"`
	ChatID     string   `json:"chatId,omitempty"`
	WebhookURL string   `json:"webhookUrl,omitempty"`
	Secret     string   `json:"secret,omitempty"`
	Events     []string `json:"events,omitempty"`
}

// DetectionConfig 表示AI detection configuration。
type DetectionConfig struct {
	Provider    string  `json:"provider"` // "gptzero", "originality", "custom"
	APIURL      string  `json:"apiUrl"`
	APIKeyEnv   string  `json:"apiKeyEnv"`
	Threshold   float64 `json:"threshold"`
	Enabled     bool    `json:"enabled"`
	AutoRewrite bool    `json:"autoRewrite"`
	MaxRetries  int     `json:"maxRetries"`
}

// DefaultDetectionConfig 返回a DetectionConfig with default values。
func DefaultDetectionConfig() DetectionConfig {
	return DetectionConfig{
		Provider:   "custom",
		Threshold:  0.5,
		MaxRetries: 3,
	}
}

// QualityGates 表示quality gate configurations。
type QualityGates struct {
	MaxAuditRetries               int     `json:"maxAuditRetries"`
	PauseAfterConsecutiveFailures int     `json:"pauseAfterConsecutiveFailures"`
	RetryTemperatureStep          float64 `json:"retryTemperatureStep"`
}

// DefaultQualityGates 返回QualityGates with default values。
func DefaultQualityGates() QualityGates {
	return QualityGates{
		MaxAuditRetries:               2,
		PauseAfterConsecutiveFailures: 3,
		RetryTemperatureStep:          0.1,
	}
}

// AgentLLMOverride 表示per-agent LLM override settings。
type AgentLLMOverride struct {
	Model     string  `json:"model"`
	Provider  *string `json:"provider,omitempty"`
	BaseURL   *string `json:"baseUrl,omitempty"`
	APIKeyEnv *string `json:"apiKeyEnv,omitempty"`
	Stream    *bool   `json:"stream,omitempty"`
}

// InputGovernanceMode 表示the input governance mode。
type InputGovernanceMode string

const (
	GovernanceModeLegacy InputGovernanceMode = "legacy"
	GovernanceModeV2     InputGovernanceMode = "v2"
)

// ScheduleConfig 表示daemon schedule configuration。
type ScheduleConfig struct {
	RadarCron string `json:"radarCron"`
	WriteCron string `json:"writeCron"`
}

// DaemonConfig 表示daemon configuration。
type DaemonConfig struct {
	Schedule               ScheduleConfig `json:"schedule"`
	MaxConcurrentBooks     int            `json:"maxConcurrentBooks"`
	ChaptersPerCycle       int            `json:"chaptersPerCycle"`
	RetryDelayMs           int            `json:"retryDelayMs"`
	CooldownAfterChapterMs int            `json:"cooldownAfterChapterMs"`
	MaxChaptersPerDay      int            `json:"maxChaptersPerDay"`
	QualityGates           QualityGates   `json:"qualityGates"`
}

// DefaultDaemonConfig 返回a DaemonConfig with default values。
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

// ProjectConfig 表示the overall project configuration。
type ProjectConfig struct {
	Name                string              `json:"name"`
	Version             string              `json:"version"`  // "0.1.0"
	Language            string              `json:"language"` // "zh" or "en"
	LLM                 LLMConfig           `json:"llm"`
	Notify              []NotifyChannel     `json:"notify"`
	Detection           *DetectionConfig    `json:"detection,omitempty"`
	ModelOverrides      map[string]any      `json:"modelOverrides,omitempty"`
	InputGovernanceMode InputGovernanceMode `json:"inputGovernanceMode"`
	Daemon              DaemonConfig        `json:"daemon"`
}

// DefaultProjectConfig 返回a ProjectConfig with default values。
func DefaultProjectConfig() ProjectConfig {
	return ProjectConfig{
		Version:             "0.1.0",
		Language:            "zh",
		InputGovernanceMode: GovernanceModeV2,
		Daemon:              DefaultDaemonConfig(),
	}
}
