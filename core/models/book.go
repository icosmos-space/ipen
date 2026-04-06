package models

import (
	"time"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// Platform 小说发布平台。
type Platform string

const (
	PlatformTomato Platform = "番茄"
	PlatformFeilu  Platform = "飞书"
	PlatformQidian Platform = "起点"
	PlatformOther  Platform = "其他"
)

// Genre 小说类型。
type Genre string

// BookStatus 书籍状态。
type BookStatus string

const (
	StatusIncubating BookStatus = "孵化中"
	StatusOutlining  BookStatus = "大纲中"
	StatusActive     BookStatus = "进行中"
	StatusPaused     BookStatus = "暂停中"
	StatusCompleted  BookStatus = "已完成"
	StatusDropped    BookStatus = "已废弃"
)

// FanficMode 同人小说模式。
type FanficMode string

const (
	FanficModeCanon FanficMode = "正典延续"
	FanficModeAU    FanficMode = "架空世界"
	FanficModeOOC   FanficMode = "性格重塑"
	FanficModeCP    FanficMode = "CP向"
)

// BookConfig 书籍配置
type BookConfig struct {
	// 书籍ID
	ID string `json:"id" validate:"required,min=1"`
	// 书籍标题
	Title string `json:"title" validate:"required,min=1"`
	// 小说发布平台
	Platform Platform `json:"platform" validate:"required,oneof=番茄 飞书 起点 其他"`
	// 小说类型
	Genre Genre `json:"genre" validate:"required,min=1"`
	// 书籍状态
	Status BookStatus `json:"status" validate:"required"`
	// 目标章节数
	TargetChapters int `json:"targetChapters" validate:"required,min=1" default:"200"`
	// 每章字数
	ChapterWordCount int `json:"chapterWordCount" validate:"required,min=1000" default:"3000"`
	// 书籍语言
	Language string `json:"language,omitempty" validate:"omitempty,oneof=zh en"`
	// 创建时间
	CreatedAt time.Time `json:"createdAt" validate:"required"`
	// 更新时间
	UpdatedAt time.Time `json:"updatedAt" validate:"required"`
	// 父书籍ID
	ParentBookID string `json:"parentBookId,omitempty"`
	// 同人小说模式
	// 正典延续：保持与原小说的正典风格，继续发展。
	// 架空世界：创建一个全新的世界，角色和事件发生在这个世界中。
	// 性格重塑：改变角色的性格和行为，适应新的环境。
	// CP向：主要关注角色CP的发展，而不是角色的个人发展。
	FanficMode FanficMode `json:"fanficMode,omitempty" validate:"omitempty,oneof=canon au ooc cp"`
}

// Validate 检查书籍配置是否合法。
func (b *BookConfig) Validate() error {
	return validate.Struct(b)
}

// ValidationError 验证错误。
type ValidationError struct {
	// 字段
	Field string
	// 错误消息
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
