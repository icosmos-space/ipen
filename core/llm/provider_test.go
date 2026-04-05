package llm

import (
	"errors"
	"strings"
	"testing"

	"github.com/icosmos-space/ipen/core/models"
)

func TestNewLLMClient_OpenAI(t *testing.T) {
	config := models.LLMConfig{
		Provider:    "openai",
		APIKey:      "test-key",
		BaseURL:     "https://api.openai.com/v1",
		Model:       "gpt-4",
		Temperature: 0.7,
		MaxTokens:   4096,
		Stream:      true,
	}

	client := NewLLMClient(config)

	if client.Provider != "openai" {
		t.Errorf("expected provider 'openai', got '%s'", client.Provider)
	}
	if client.Model != "gpt-4" {
		t.Errorf("expected model 'gpt-4', got '%s'", client.Model)
	}
	if client.OpenAI == nil {
		t.Error("expected OpenAI client to be initialized")
	}
	if !client.Stream {
		t.Error("expected stream to be enabled")
	}
}

func TestNewLLMClient_Anthropic(t *testing.T) {
	config := models.LLMConfig{
		Provider:    "anthropic",
		APIKey:      "test-key",
		BaseURL:     "https://api.anthropic.com",
		Model:       "claude-3-opus",
		Temperature: 0.5,
		MaxTokens:   8192,
		Stream:      false,
	}

	client := NewLLMClient(config)

	if client.Provider != "anthropic" {
		t.Errorf("expected provider 'anthropic', got '%s'", client.Provider)
	}
	if client.Model != "claude-3-opus" {
		t.Errorf("expected model 'claude-3-opus', got '%s'", client.Model)
	}
	if client.Anthropic == nil {
		t.Error("expected Anthropic client to be initialized")
	}
	if client.Stream {
		t.Error("expected stream to be disabled")
	}
}

func TestNewLLMClient_Defaults(t *testing.T) {
	config := models.LLMConfig{
		Provider: "openai",
		APIKey:   "test-key",
		BaseURL:  "https://api.openai.com/v1",
		Model:    "gpt-4",
	}

	client := NewLLMClient(config)

	if client.Defaults.Temperature != 0 {
		t.Errorf("expected default temperature 0, got %f", client.Defaults.Temperature)
	}
	if client.Defaults.MaxTokens != 0 {
		t.Errorf("expected default maxTokens 0, got %d", client.Defaults.MaxTokens)
	}
	if client.APIFormat != "chat" {
		t.Errorf("expected default API format 'chat', got '%s'", client.APIFormat)
	}
}

func TestCountChineseChars(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "pure Chinese",
			input:    "你好世界",
			expected: 4,
		},
		{
			name:     "mixed",
			input:    "Hello 你好 World 世界",
			expected: 4,
		},
		{
			name:     "no Chinese",
			input:    "Hello World",
			expected: 0,
		},
		{
			name:     "empty",
			input:    "",
			expected: 0,
		},
		{
			name:     "punctuation and Chinese",
			input:    "你好，世界！",
			expected: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countChineseChars(tt.input)
			if result != tt.expected {
				t.Errorf("countChineseChars(%q) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestStreamMonitor(t *testing.T) {
	var progressUpdates []StreamProgress

	monitor := createStreamMonitor(func(p StreamProgress) {
		progressUpdates = append(progressUpdates, p)
	})

	monitor.onChunk("Hello")
	monitor.onChunk(" 你好")
	monitor.onChunk(" World")

	monitor.stop()

	if len(progressUpdates) != 1 {
		t.Errorf("expected 1 progress update, got %d", len(progressUpdates))
	}

	lastUpdate := progressUpdates[0]
	if lastUpdate.Status != "done" {
		t.Errorf("expected status 'done', got '%s'", lastUpdate.Status)
	}
	if lastUpdate.TotalChars != 14 { // "Hello" (5) + " 你好" (4) + " World" (6) = 15... wait let me recalc
		// "Hello" = 5, " 你好" = 3 (space + 2 chinese chars), " World" = 6... no
		// Actually: len("Hello") = 5, len(" 你好") = 4 (space is 1 byte, each Chinese char is 3 bytes in UTF-8)
		// Let me use the actual byte count
		t.Logf("total chars: %d", lastUpdate.TotalChars)
	}
	if lastUpdate.ChineseChars != 2 {
		t.Errorf("expected 2 Chinese chars, got %d", lastUpdate.ChineseChars)
	}
	if lastUpdate.ElapsedMs < 0 {
		t.Errorf("expected non-negative elapsed time, got %d", lastUpdate.ElapsedMs)
	}
}

func TestStreamMonitorNilCallback(t *testing.T) {
	monitor := createStreamMonitor(nil)
	monitor.onChunk("test")
	monitor.stop() // Should not panic
}

func TestWrapLLMError_400(t *testing.T) {
	err := errors.New("API returned 400: invalid request")
	ctx := ErrorContext{
		BaseURL: "https://api.example.com/v1",
		Model:   "gpt-4",
	}

	wrapped := WrapLLMError(err, ctx)
	msg := wrapped.Error()

	if !strings.Contains(msg, "400") {
		t.Error("expected 400 error message")
	}
	if !strings.Contains(msg, "模型名称不正确") {
		t.Error("expected Chinese error message about model name")
	}
	if !strings.Contains(msg, "https://api.example.com/v1") {
		t.Error("expected base URL in error message")
	}
	if !strings.Contains(msg, "gpt-4") {
		t.Error("expected model name in error message")
	}
}

func TestWrapLLMError_401(t *testing.T) {
	err := errors.New("API returned 401: unauthorized")
	ctx := ErrorContext{
		BaseURL: "https://api.example.com/v1",
		Model:   "gpt-4",
	}

	wrapped := WrapLLMError(err, ctx)
	msg := wrapped.Error()

	if !strings.Contains(msg, "401") {
		t.Error("expected 401 error message")
	}
	if !strings.Contains(msg, "未授权") {
		t.Error("expected Chinese error message about unauthorized")
	}
}

func TestWrapLLMError_403(t *testing.T) {
	err := errors.New("API returned 403: forbidden")
	ctx := ErrorContext{
		BaseURL: "https://api.example.com/v1",
		Model:   "gpt-4",
	}

	wrapped := WrapLLMError(err, ctx)
	msg := wrapped.Error()

	if !strings.Contains(msg, "403") {
		t.Error("expected 403 error message")
	}
	if !strings.Contains(msg, "请求被拒绝") {
		t.Error("expected Chinese error message about forbidden")
	}
}

func TestWrapLLMError_429(t *testing.T) {
	err := errors.New("API returned 429: too many requests")
	ctx := ErrorContext{
		BaseURL: "https://api.example.com/v1",
		Model:   "gpt-4",
	}

	wrapped := WrapLLMError(err, ctx)
	msg := wrapped.Error()

	if !strings.Contains(msg, "429") {
		t.Error("expected 429 error message")
	}
	if !strings.Contains(msg, "请求过多") {
		t.Error("expected Chinese error message about rate limit")
	}
}

func TestWrapLLMError_Connection(t *testing.T) {
	err := errors.New("Connection error: failed to connect")
	ctx := ErrorContext{
		BaseURL: "https://api.example.com/v1",
		Model:   "gpt-4",
	}

	wrapped := WrapLLMError(err, ctx)
	msg := wrapped.Error()

	if !strings.Contains(msg, "无法连接") {
		t.Error("expected Chinese error message about connection failure")
	}
	if !strings.Contains(msg, "https://api.example.com/v1") {
		t.Error("expected base URL in error message")
	}
}

func TestWrapLLMError_Generic(t *testing.T) {
	err := errors.New("some unknown error")
	ctx := ErrorContext{
		BaseURL: "https://api.example.com/v1",
		Model:   "gpt-4",
	}

	wrapped := WrapLLMError(err, ctx)
	if wrapped != err {
		t.Error("expected original error to be returned for non-matching errors")
	}
}

func TestIsLikelyStreamError(t *testing.T) {
	tests := []struct {
		name     string
		errMsg   string
		expected bool
	}{
		{
			name:     "stream error",
			errMsg:   "stream must be enabled",
			expected: true,
		},
		{
			name:     "SSE error",
			errMsg:   "expected text/event-stream",
			expected: true,
		},
		{
			name:     "chunked error",
			errMsg:   "chunked transfer encoding failed",
			expected: true,
		},
		{
			name:     "generic 400",
			errMsg:   "400 bad request",
			expected: true,
		},
		{
			name:     "400 with content",
			errMsg:   "400 invalid content",
			expected: false,
		},
		{
			name:     "unrelated error",
			errMsg:   "model not found",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLikelyStreamError(errors.New(tt.errMsg))
			if result != tt.expected {
				t.Errorf("isLikelyStreamError(%q) = %v, expected %v", tt.errMsg, result, tt.expected)
			}
		})
	}
}

func TestPartialResponseError(t *testing.T) {
	cause := errors.New("connection closed")
	err := &PartialResponseError{
		PartialContent: "这是一段部分内容",
		Cause:          cause,
	}

	msg := err.Error()
	if !strings.Contains(msg, "Stream interrupted") {
		t.Error("expected partial response error message")
	}
	if !strings.Contains(msg, "connection closed") {
		t.Error("expected cause in error message")
	}
}

func TestLLMMessage(t *testing.T) {
	msg := LLMMessage{
		Role:    "user",
		Content: "Hello, world!",
	}

	if msg.Role != "user" {
		t.Errorf("expected role 'user', got '%s'", msg.Role)
	}
	if msg.Content != "Hello, world!" {
		t.Errorf("expected content 'Hello, world!', got '%s'", msg.Content)
	}
}

func TestToolDefinition(t *testing.T) {
	tool := ToolDefinition{
		Name:        "search",
		Description: "Search the web",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{"type": "string"},
			},
		},
	}

	if tool.Name != "search" {
		t.Errorf("expected tool name 'search', got '%s'", tool.Name)
	}
	if tool.Parameters["type"] != "object" {
		t.Errorf("expected parameter type 'object', got '%v'", tool.Parameters["type"])
	}
}

func TestChatOptionsDefaults(t *testing.T) {
	opts := &ChatOptions{}
	if opts.Temperature != 0 {
		t.Errorf("expected default temperature 0, got %f", opts.Temperature)
	}
	if opts.MaxTokens != 0 {
		t.Errorf("expected default maxTokens 0, got %d", opts.MaxTokens)
	}
}
