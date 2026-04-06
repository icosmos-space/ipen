package models

import "time"

// ChapterStatus 表示章节状态
type ChapterStatus string

const (
	StatusCardGenerated  ChapterStatus = "已生成"
	StatusDrafting       ChapterStatus = "草稿中"
	StatusDrafted        ChapterStatus = "已草稿"
	StatusAuditing       ChapterStatus = "审核中"
	StatusAuditPassed    ChapterStatus = "审核通过"
	StatusAuditFailed    ChapterStatus = "审核失败"
	StatusStateDegraded  ChapterStatus = "状态降级"
	StatusRevising       ChapterStatus = "修订中"
	StatusReadyForReview ChapterStatus = "待审核"
	StatusApproved       ChapterStatus = "已通过"
	StatusRejected       ChapterStatus = "已拒绝"
	StatusPublished      ChapterStatus = "已发布"
	StatusImported       ChapterStatus = "已导入"
)

// ChapterMeta 表示章节元数据
type ChapterMeta struct {
	//
	Number int `json:"number"`
	// 章节标题
	Title string `json:"title"`
	// 章节状态
	Status ChapterStatus `json:"status"`
	// 章节字数
	WordCount int `json:"wordCount"`
	// 创建时间
	CreatedAt time.Time `json:"createdAt"`
	// 更新时间
	UpdatedAt time.Time `json:"updatedAt"`
	// 审核问题
	AuditIssues []string `json:"auditIssues"`
	// 长度警告
	LengthWarnings []string `json:"lengthWarnings"`
	// 审核备注
	ReviewNote string `json:"reviewNote,omitempty"`
	// 检测分数
	DetectionScore *float64 `json:"detectionScore,omitempty"`
	// 检测工具
	DetectionProvider string `json:"detectionProvider,omitempty"`
	// 检测时间
	DetectedAt *time.Time `json:"detectedAt,omitempty"`
	// 长度测验数据
	LengthTelemetry *LengthTelemetry `json:"lengthTelemetry,omitempty"`
	// token使用量
	TokenUsage *TokenUsage `json:"tokenUsage,omitempty"`
}

// TokenUsage 表示LLM调用的token使用量
type TokenUsage struct {
	// 提示词使用Token量(llm输入)
	PromptTokens int `json:"promptTokens"`
	// 完成词使用Token量(llm输出)
	CompletionTokens int `json:"completionTokens"`
	// 总Token量
	TotalTokens int `json:"totalTokens"`
}

func (tu *TokenUsage) Validate() error {
	return validate.Struct(tu)
}
