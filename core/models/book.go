package models

import (
	"time"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// Platform 表示the novel publishing platform。
type Platform string

const (
	PlatformTomato Platform = "tomato"
	PlatformFeilu  Platform = "feilu"
	PlatformQidian Platform = "qidian"
	PlatformOther  Platform = "other"
)

// Genre 表示the novel genre。
type Genre string

// BookStatus 表示the current status of a book。
type BookStatus string

const (
	StatusIncubating BookStatus = "incubating"
	StatusOutlining  BookStatus = "outlining"
	StatusActive     BookStatus = "active"
	StatusPaused     BookStatus = "paused"
	StatusCompleted  BookStatus = "completed"
	StatusDropped    BookStatus = "dropped"
)

// FanficMode 表示the fanfiction mode type。
type FanficMode string

const (
	FanficModeCanon FanficMode = "canon"
	FanficModeAU    FanficMode = "au"
	FanficModeOOC   FanficMode = "ooc"
	FanficModeCP    FanficMode = "cp"
)

// BookConfig 表示the configuration for a book。
type BookConfig struct {
	ID               string     `json:"id" validate:"required,min=1"`
	Title            string     `json:"title" validate:"required,min=1"`
	Platform         Platform   `json:"platform" validate:"required,oneof=tomato feilu qidian other"`
	Genre            Genre      `json:"genre" validate:"required,min=1"`
	Status           BookStatus `json:"status" validate:"required,oneof=incubating outlining active paused completed dropped"`
	TargetChapters   int        `json:"targetChapters" validate:"required,min=1" default:"200"`
	ChapterWordCount int        `json:"chapterWordCount" validate:"required,min=1000" default:"3000"`
	Language         string     `json:"language,omitempty" validate:"omitempty,oneof=zh en"`
	CreatedAt        time.Time  `json:"createdAt" validate:"required"`
	UpdatedAt        time.Time  `json:"updatedAt" validate:"required"`
	ParentBookID     string     `json:"parentBookId,omitempty"`
	FanficMode       FanficMode `json:"fanficMode,omitempty" validate:"omitempty,oneof=canon au ooc cp"`
}

// Validate 检查if the BookConfig is valid。
func (b *BookConfig) Validate() error {
	return validate.Struct(b)
}

// ValidationError 表示a validation error。
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
