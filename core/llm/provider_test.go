package llm

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/icosmos-space/ipen/core/models"
)

func TestNewLLMClient_OpenAI(t *testing.T) {
	config := models.LLMConfig{
		Provider:    "openai",
		APIKey:      "123456",
		BaseURL:     "http://localhost:5001/v1",
		Model:       "koboldcpp/qwen2.5-7b-instruct-q4_k_m",
		Temperature: 0.7,
		MaxTokens:   4096,
		Stream:      true,
	}

	client := NewLLMClient(config)

	if client.Provider != "openai" {
		t.Errorf("expected provider 'openai', got '%s'", client.Provider)
	}
	if client.Model != "koboldcpp/qwen2.5-7b-instruct-q4_k_m" {
		t.Errorf("expected model 'koboldcpp/qwen2.5-7b-instruct-q4_k_m', got '%s'", client.Model)
	}
	if client.OpenAI == nil {
		t.Error("expected OpenAI client to be initialized")
	}
	if !client.Stream {
		t.Error("expected stream to be enabled")
	}
	// opt := &ChatOptions{
	// 	OnStreamChunk: func(chunk StreamChunk) {
	// 		t.Logf("chunk: %q", chunk.Text)
	// 	},
	// 	OnStreamProgress: func(p StreamProgress) {
	// 		t.Logf("progress: %+v", p)
	// 	}}
	// ChatCompletion(
	// 	context.Background(),
	// 	client,
	// 	client.Model,
	// 	[]LLMMessage{
	// 		{
	// 			Role:    "user",
	// 			Content: "生命和生活的意义",
	// 		},
	// 	},
	// 	opt,
	// )
}

func TestChatCompletion_OpenAI_Stream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		if ct := r.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
			t.Fatalf("expected application/json request, got %q", ct)
		}

		w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")

		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("response writer does not support flushing")
		}

		events := []string{
			`data: {"id":"chatcmpl-1","object":"chat.completion.chunk","created":1,"model":"test-model","choices":[{"index":0,"delta":{"role":"assistant","content":"你"},"finish_reason":null}]}` + "\n\n",
			`data: {"id":"chatcmpl-1","object":"chat.completion.chunk","created":1,"model":"test-model","choices":[{"index":0,"delta":{"content":"好"},"finish_reason":null}]}` + "\n\n",
			`data: {"id":"chatcmpl-1","object":"chat.completion.chunk","created":1,"model":"test-model","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}` + "\n\n",
			"data: [DONE]\n\n",
		}

		for _, event := range events {
			if _, err := io.WriteString(w, event); err != nil {
				t.Fatalf("failed to write event: %v", err)
			}
			flusher.Flush()
		}
	}))
	defer server.Close()

	client := NewLLMClient(models.LLMConfig{
		Provider:    "openai",
		APIKey:      "test-key",
		BaseURL:     server.URL + "/v1",
		Model:       "test-model",
		Temperature: 0.7,
		MaxTokens:   128,
		Stream:      true,
	})

	var chunks []string
	var progressUpdates []StreamProgress

	resp, err := ChatCompletion(
		context.Background(),
		client,
		client.Model,
		[]LLMMessage{
			{
				Role:    "user",
				Content: "请只输出“你好”",
			},
		},
		&ChatOptions{
			StreamLabel: "test",
			OnStreamChunk: func(chunk StreamChunk) {
				t.Logf("chunk: %q", chunk.Text)
				chunks = append(chunks, chunk.Text)
			},
			OnStreamProgress: func(progress StreamProgress) {
				t.Logf("progress: %+v", progress)
				progressUpdates = append(progressUpdates, progress)
			},
		},
	)
	if err != nil {
		t.Fatalf("ChatCompletion returned error: %v", err)
	}

	if got := strings.Join(chunks, ""); got != "你好" {
		t.Fatalf("unexpected streamed content: %q", got)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.Content != "你好" {
		t.Fatalf("unexpected final content: %q", resp.Content)
	}
	if len(progressUpdates) == 0 {
		t.Fatal("expected progress callbacks")
	}
	if progressUpdates[len(progressUpdates)-1].Status != "done" {
		t.Fatalf("expected final progress status 'done', got %q", progressUpdates[len(progressUpdates)-1].Status)
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
	var chunks []StreamChunk
	var progressUpdates []StreamProgress

	monitor := createStreamMonitor(
		"writer",
		func(chunk StreamChunk) {
			chunks = append(chunks, chunk)
		},
		func(p StreamProgress) {
			progressUpdates = append(progressUpdates, p)
		},
	)

	monitor.onChunk("Hello")
	monitor.onChunk(" 你好")
	monitor.onChunk(" World")

	monitor.stop()

	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunk updates, got %d", len(chunks))
	}
	if chunks[0].Text != "Hello" {
		t.Errorf("expected first chunk text %q, got %q", "Hello", chunks[0].Text)
	}
	if chunks[1].Label != "writer" {
		t.Errorf("expected chunk label 'writer', got %q", chunks[1].Label)
	}
	if chunks[2].Status != "streaming" {
		t.Errorf("expected chunk status 'streaming', got %q", chunks[2].Status)
	}

	if len(progressUpdates) != 4 {
		t.Fatalf("expected 4 progress updates, got %d", len(progressUpdates))
	}
	for i := 0; i < len(progressUpdates)-1; i++ {
		if progressUpdates[i].Status != "streaming" {
			t.Errorf("expected streaming status before completion, got %q at index %d", progressUpdates[i].Status, i)
		}
	}

	lastUpdate := progressUpdates[len(progressUpdates)-1]
	if lastUpdate.Status != "done" {
		t.Errorf("expected status 'done', got '%s'", lastUpdate.Status)
	}
	if lastUpdate.TotalChars != 14 {
		t.Errorf("expected 14 total chars, got %d", lastUpdate.TotalChars)
	}
	if lastUpdate.ChineseChars != 2 {
		t.Errorf("expected 2 Chinese chars, got %d", lastUpdate.ChineseChars)
	}
	if lastUpdate.ElapsedMs < 0 {
		t.Errorf("expected non-negative elapsed time, got %d", lastUpdate.ElapsedMs)
	}
}

func TestStreamMonitorNilCallback(t *testing.T) {
	monitor := createStreamMonitor("", nil, nil)
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
	if opts.StreamLabel != "" {
		t.Errorf("expected empty stream label, got %q", opts.StreamLabel)
	}
	if opts.OnStreamChunk != nil {
		t.Error("expected nil stream chunk callback")
	}
	if opts.OnStreamProgress != nil {
		t.Error("expected nil stream progress callback")
	}
}
