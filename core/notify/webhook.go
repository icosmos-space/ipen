package notify

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// WebhookConfig 表示webhook configuration。
type WebhookConfig struct {
	URL    string   `json:"url"`
	Secret string   `json:"secret,omitempty"`
	Events []string `json:"events,omitempty"`
}

// WebhookEvent 表示a webhook event type。
type WebhookEvent string

const (
	EventChapterComplete  WebhookEvent = "chapter-complete"
	EventAuditPassed      WebhookEvent = "audit-passed"
	EventAuditFailed      WebhookEvent = "audit-failed"
	EventRevisionComplete WebhookEvent = "revision-complete"
	EventPipelineComplete WebhookEvent = "pipeline-complete"
	EventPipelineError    WebhookEvent = "pipeline-error"
	EventDiagnosticAlert  WebhookEvent = "diagnostic-alert"
)

// WebhookPayload 表示a webhook payload。
type WebhookPayload struct {
	Event         WebhookEvent   `json:"event"`
	BookID        string         `json:"bookId"`
	ChapterNumber *int           `json:"chapterNumber,omitempty"`
	Timestamp     string         `json:"timestamp"`
	Data          map[string]any `json:"data,omitempty"`
}

// SendWebhook sends a webhook payload
func SendWebhook(ctx context.Context, config WebhookConfig, payload WebhookPayload) error {
	// Filter by subscribed events
	if len(config.Events) > 0 {
		subscribed := false
		for _, e := range config.Events {
			if string(payload.Event) == e {
				subscribed = true
				break
			}
		}
		if !subscribed {
			return nil
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", config.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	// HMAC-SHA256 signature if secret is configured
	if config.Secret != "" {
		signature := computeHMAC(config.Secret, body)
		req.Header.Set("X-iPen-Signature", "sha256="+signature)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook POST to %s failed: %d %s", config.URL, resp.StatusCode, string(respBody))
	}

	return nil
}

func computeHMAC(secret string, data []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}
