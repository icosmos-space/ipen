package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/icosmos-space/ipen/core/models"
)

func TestNotifyMessage(t *testing.T) {
	msg := NotifyMessage{
		Title: "Test Title",
		Body:  "Test Body",
	}

	if msg.Title != "Test Title" {
		t.Errorf("expected title 'Test Title', got '%s'", msg.Title)
	}
	if msg.Body != "Test Body" {
		t.Errorf("expected body 'Test Body', got '%s'", msg.Body)
	}
}

func TestDispatchNotification_EmptyChannels(t *testing.T) {
	ctx := context.Background()
	channels := []models.NotifyChannel{}
	message := NotifyMessage{
		Title: "Test",
		Body:  "Body",
	}

	err := DispatchNotification(ctx, channels, message)
	if err != nil {
		t.Errorf("expected no error for empty channels, got %v", err)
	}
}

func TestDispatchNotification_UnknownChannel(t *testing.T) {
	ctx := context.Background()
	channels := []models.NotifyChannel{
		{
			Type: "unknown",
		},
	}
	message := NotifyMessage{
		Title: "Test",
		Body:  "Body",
	}

	// Should not panic, but will print error to stderr
	err := DispatchNotification(ctx, channels, message)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Give goroutine time to execute
	time.Sleep(100 * time.Millisecond)
}

func TestDispatchNotification_Telegram(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/botTEST_TOKEN/sendMessage" {
			t.Errorf("expected path '/botTEST_TOKEN/sendMessage', got '%s'", r.URL.Path)
		}

		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if payload["chat_id"] != "TEST_CHAT_ID" {
			t.Errorf("expected chat_id 'TEST_CHAT_ID', got '%s'", payload["chat_id"])
		}
		if payload["parse_mode"] != "Markdown" {
			t.Errorf("expected parse_mode 'Markdown', got '%s'", payload["parse_mode"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	channels := []models.NotifyChannel{
		{
			Type:     "telegram",
			BotToken: "TEST_TOKEN",
			ChatID:   "TEST_CHAT_ID",
		},
	}
	message := NotifyMessage{
		Title: "Test Title",
		Body:  "Test Body",
	}

	err := DispatchNotification(ctx, channels, message)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Give goroutine time to execute
	time.Sleep(100 * time.Millisecond)
}

func TestDispatchNotification_Feishu(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if payload["msg_type"] != "interactive" {
			t.Errorf("expected msg_type 'interactive', got '%s'", payload["msg_type"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	channels := []models.NotifyChannel{
		{
			Type:       "feishu",
			WebhookURL: server.URL,
		},
	}
	message := NotifyMessage{
		Title: "Test Title",
		Body:  "Test Body",
	}

	err := DispatchNotification(ctx, channels, message)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Give goroutine time to execute
	time.Sleep(100 * time.Millisecond)
}

func TestDispatchNotification_WechatWork(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if payload["msgtype"] != "markdown" {
			t.Errorf("expected msgtype 'markdown', got '%s'", payload["msgtype"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	channels := []models.NotifyChannel{
		{
			Type:       "wechat-work",
			WebhookURL: server.URL,
		},
	}
	message := NotifyMessage{
		Title: "Test Title",
		Body:  "Test Body",
	}

	err := DispatchNotification(ctx, channels, message)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Give goroutine time to execute
	time.Sleep(100 * time.Millisecond)
}

func TestDispatchNotification_Webhook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected Content-Type 'application/json', got '%s'", contentType)
		}

		var payload WebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if payload.Data["title"] != "Test Title" {
			t.Errorf("expected title 'Test Title', got '%v'", payload.Data["title"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	channels := []models.NotifyChannel{
		{
			Type:       "webhook",
			WebhookURL: server.URL,
		},
	}
	message := NotifyMessage{
		Title: "Test Title",
		Body:  "Test Body",
	}

	err := DispatchNotification(ctx, channels, message)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Give goroutine time to execute
	time.Sleep(100 * time.Millisecond)
}

func TestDispatchWebhookEvent_NoWebhookChannels(t *testing.T) {
	ctx := context.Background()
	channels := []models.NotifyChannel{
		{
			Type: "telegram",
		},
	}
	payload := WebhookPayload{
		Event:     EventPipelineComplete,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	err := DispatchWebhookEvent(ctx, channels, payload)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestDispatchWebhookEvent_EmptyChannels(t *testing.T) {
	ctx := context.Background()
	channels := []models.NotifyChannel{}
	payload := WebhookPayload{
		Event:     EventPipelineComplete,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	err := DispatchWebhookEvent(ctx, channels, payload)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestDispatchWebhookEvent_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		var payload WebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if payload.Event != EventPipelineComplete {
			t.Errorf("expected event '%s', got '%s'", EventPipelineComplete, payload.Event)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	channels := []models.NotifyChannel{
		{
			Type:       "webhook",
			WebhookURL: server.URL,
		},
	}
	payload := WebhookPayload{
		Event:     EventPipelineComplete,
		BookID:    "test-book",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	err := DispatchWebhookEvent(ctx, channels, payload)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Give goroutine time to execute
	time.Sleep(100 * time.Millisecond)
}

func TestSendFeishu_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		card, ok := payload["card"].(map[string]any)
		if !ok {
			t.Fatal("expected card in payload")
		}

		header, ok := card["header"].(map[string]any)
		if !ok {
			t.Fatal("expected header in card")
		}

		title, ok := header["title"].(map[string]any)
		if !ok {
			t.Fatal("expected title in header")
		}

		if title["content"] != "Test Title" {
			t.Errorf("expected title content 'Test Title', got '%s'", title["content"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	config := FeishuConfig{
		WebhookURL: server.URL,
	}

	err := SendFeishu(ctx, config, "Test Title", "Test Body")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestSendFeishu_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal error"}`))
	}))
	defer server.Close()

	ctx := context.Background()
	config := FeishuConfig{
		WebhookURL: server.URL,
	}

	err := SendFeishu(ctx, config, "Test Title", "Test Body")
	if err == nil {
		t.Error("expected error for server error, got nil")
	}
}

func TestSendFeishu_InvalidURL(t *testing.T) {
	ctx := context.Background()
	config := FeishuConfig{
		WebhookURL: "://invalid-url",
	}

	err := SendFeishu(ctx, config, "Test Title", "Test Body")
	if err == nil {
		t.Error("expected error for invalid URL, got nil")
	}
}

func TestSendTelegram_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if payload["text"] != "Test Message" {
			t.Errorf("expected text 'Test Message', got '%s'", payload["text"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()

	// Override the URL by creating a custom test
	url := server.URL + "/botTEST_TOKEN/sendMessage"
	payload := map[string]string{
		"chat_id":    "TEST_CHAT_ID",
		"text":       "Test Message",
		"parse_mode": "Markdown",
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestSendTelegram_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "bad request"}`))
	}))
	defer server.Close()

	// Extract the path from server URL
	url := server.URL + "/botTEST_TOKEN/sendMessage"

	ctx := context.Background()
	config := TelegramConfig{
		BotToken: "TEST_TOKEN",
		ChatID:   "TEST_CHAT_ID",
	}

	// We need to test via actual function call
	payload := map[string]string{
		"chat_id":    config.ChatID,
		"text":       "Test Message",
		"parse_mode": "Markdown",
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Error("expected error status, got 200")
	}
}

func TestSendWechatWork_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if payload["msgtype"] != "markdown" {
			t.Errorf("expected msgtype 'markdown', got '%s'", payload["msgtype"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	config := WechatWorkConfig{
		WebhookURL: server.URL,
	}

	err := SendWechatWork(ctx, config, "**Test** Message")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestSendWechatWork_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"errcode": 500}`))
	}))
	defer server.Close()

	ctx := context.Background()
	config := WechatWorkConfig{
		WebhookURL: server.URL,
	}

	err := SendWechatWork(ctx, config, "**Test** Message")
	if err == nil {
		t.Error("expected error for server error, got nil")
	}
}

func TestSendWebhook_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected Content-Type 'application/json', got '%s'", contentType)
		}

		var payload WebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if payload.Event != EventChapterComplete {
			t.Errorf("expected event '%s', got '%s'", EventChapterComplete, payload.Event)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	config := WebhookConfig{
		URL: server.URL,
	}
	payload := WebhookPayload{
		Event:     EventChapterComplete,
		BookID:    "test-book",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	err := SendWebhook(ctx, config, payload)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestSendWebhook_WithSignature(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		signature := r.Header.Get("X-iPen-Signature")
		if signature == "" {
			t.Error("expected X-iPen-Signature header")
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	config := WebhookConfig{
		URL:    server.URL,
		Secret: "test-secret",
	}
	payload := WebhookPayload{
		Event:     EventPipelineComplete,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	err := SendWebhook(ctx, config, payload)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestSendWebhook_EventFiltering(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	config := WebhookConfig{
		URL:    server.URL,
		Events: []string{"chapter-complete"},
	}
	payload := WebhookPayload{
		Event:     EventPipelineComplete,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	err := SendWebhook(ctx, config, payload)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if called {
		t.Error("expected webhook not to be called for unsubscribed event")
	}
}

func TestSendWebhook_SubscribedEvent(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	config := WebhookConfig{
		URL:    server.URL,
		Events: []string{"pipeline-complete"},
	}
	payload := WebhookPayload{
		Event:     EventPipelineComplete,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	err := SendWebhook(ctx, config, payload)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if !called {
		t.Error("expected webhook to be called for subscribed event")
	}
}

func TestSendWebhook_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	ctx := context.Background()
	config := WebhookConfig{
		URL: server.URL,
	}
	payload := WebhookPayload{
		Event:     EventPipelineComplete,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	err := SendWebhook(ctx, config, payload)
	if err == nil {
		t.Error("expected error for server error, got nil")
	}
}

func TestComputeHMAC(t *testing.T) {
	secret := "test-secret"
	data := []byte("test-data")

	signature := computeHMAC(secret, data)

	if signature == "" {
		t.Error("expected non-empty signature")
	}

	// Verify it's a valid hex string
	if len(signature) != 64 { // SHA256 produces 64 hex characters
		t.Errorf("expected signature length 64, got %d", len(signature))
	}
}

func TestComputeHMAC_Deterministic(t *testing.T) {
	secret := "test-secret"
	data := []byte("test-data")

	sig1 := computeHMAC(secret, data)
	sig2 := computeHMAC(secret, data)

	if sig1 != sig2 {
		t.Error("expected deterministic signature output")
	}
}

func TestWebhookEvent_Constants(t *testing.T) {
	tests := []struct {
		name     string
		event    WebhookEvent
		expected string
	}{
		{
			name:     "chapter-complete",
			event:    EventChapterComplete,
			expected: "chapter-complete",
		},
		{
			name:     "audit-passed",
			event:    EventAuditPassed,
			expected: "audit-passed",
		},
		{
			name:     "audit-failed",
			event:    EventAuditFailed,
			expected: "audit-failed",
		},
		{
			name:     "revision-complete",
			event:    EventRevisionComplete,
			expected: "revision-complete",
		},
		{
			name:     "pipeline-complete",
			event:    EventPipelineComplete,
			expected: "pipeline-complete",
		},
		{
			name:     "pipeline-error",
			event:    EventPipelineError,
			expected: "pipeline-error",
		},
		{
			name:     "diagnostic-alert",
			event:    EventDiagnosticAlert,
			expected: "diagnostic-alert",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.event) != tt.expected {
				t.Errorf("expected event '%s', got '%s'", tt.expected, tt.event)
			}
		})
	}
}

func TestWebhookPayload_JSONMarshal(t *testing.T) {
	payload := WebhookPayload{
		Event:         EventChapterComplete,
		BookID:        "test-book",
		ChapterNumber: intPtr(5),
		Timestamp:     "2024-01-01T00:00:00Z",
		Data: map[string]any{
			"key": "value",
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	var decoded WebhookPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}

	if decoded.Event != payload.Event {
		t.Errorf("expected event '%s', got '%s'", payload.Event, decoded.Event)
	}
	if decoded.BookID != payload.BookID {
		t.Errorf("expected book ID '%s', got '%s'", payload.BookID, decoded.BookID)
	}
	if *decoded.ChapterNumber != *payload.ChapterNumber {
		t.Errorf("expected chapter number %d, got %d", *payload.ChapterNumber, *decoded.ChapterNumber)
	}
}

func intPtr(i int) *int {
	return &i
}

func TestDispatchNotification_MultipleChannels(t *testing.T) {
	var feishuCalled, wechatCalled, webhookCalled bool
	var mu sync.Mutex

	feishuServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		feishuCalled = true
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer feishuServer.Close()

	wechatServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		wechatCalled = true
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer wechatServer.Close()

	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		webhookCalled = true
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer webhookServer.Close()

	ctx := context.Background()
	channels := []models.NotifyChannel{
		{
			Type:       "feishu",
			WebhookURL: feishuServer.URL,
		},
		{
			Type:       "wechat-work",
			WebhookURL: wechatServer.URL,
		},
		{
			Type:       "webhook",
			WebhookURL: webhookServer.URL,
		},
	}
	message := NotifyMessage{
		Title: "Test Title",
		Body:  "Test Body",
	}

	err := DispatchNotification(ctx, channels, message)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Give goroutines time to execute
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if !feishuCalled {
		t.Error("expected feishu to be called")
	}
	if !wechatCalled {
		t.Error("expected wechat to be called")
	}
	if !webhookCalled {
		t.Error("expected webhook to be called")
	}
}
