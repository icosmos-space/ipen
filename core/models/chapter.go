package models

import "time"

// ChapterStatus 表示the status of a chapter。
type ChapterStatus string

const (
	StatusCardGenerated  ChapterStatus = "card-generated"
	StatusDrafting       ChapterStatus = "drafting"
	StatusDrafted        ChapterStatus = "drafted"
	StatusAuditing       ChapterStatus = "auditing"
	StatusAuditPassed    ChapterStatus = "audit-passed"
	StatusAuditFailed    ChapterStatus = "audit-failed"
	StatusStateDegraded  ChapterStatus = "state-degraded"
	StatusRevising       ChapterStatus = "revising"
	StatusReadyForReview ChapterStatus = "ready-for-review"
	StatusApproved       ChapterStatus = "approved"
	StatusRejected       ChapterStatus = "rejected"
	StatusPublished      ChapterStatus = "published"
	StatusImported       ChapterStatus = "imported"
)

// ChapterMeta 表示metadata for a chapter。
type ChapterMeta struct {
	Number            int              `json:"number"`
	Title             string           `json:"title"`
	Status            ChapterStatus    `json:"status"`
	WordCount         int              `json:"wordCount"`
	CreatedAt         time.Time        `json:"createdAt"`
	UpdatedAt         time.Time        `json:"updatedAt"`
	AuditIssues       []string         `json:"auditIssues"`
	LengthWarnings    []string         `json:"lengthWarnings"`
	ReviewNote        string           `json:"reviewNote,omitempty"`
	DetectionScore    *float64         `json:"detectionScore,omitempty"`
	DetectionProvider string           `json:"detectionProvider,omitempty"`
	DetectedAt        *time.Time       `json:"detectedAt,omitempty"`
	LengthTelemetry   *LengthTelemetry `json:"lengthTelemetry,omitempty"`
	TokenUsage        *TokenUsage      `json:"tokenUsage,omitempty"`
}

// TokenUsage 表示token usage for LLM calls。
type TokenUsage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}
