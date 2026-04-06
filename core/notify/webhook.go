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

// WebhookConfig Webhook配置
type WebhookConfig struct {
	URL    string   `json:"url"`
	Secret string   `json:"secret,omitempty"`
	Events []string `json:"events,omitempty"`
}

// WebhookEvent Webhook事件类型
type WebhookEvent string

const (
	EventChapterComplete  WebhookEvent = "章节完成"
	EventAuditPassed      WebhookEvent = "审核通过"
	EventAuditFailed      WebhookEvent = "审核失败"
	EventRevisionComplete WebhookEvent = "修订完成"
	EventPipelineComplete WebhookEvent = "管道完成"
	EventPipelineError    WebhookEvent = "管道错误"
	EventDiagnosticAlert  WebhookEvent = "诊断警报"
)

// WebhookPayload Webhook负载
type WebhookPayload struct {
	// 事件
	Event WebhookEvent `json:"event"`
	// 书籍ID
	BookID string `json:"bookId"`
	// 章节编号
	ChapterNumber *int `json:"chapterNumber,omitempty"`
	// 时间戳
	Timestamp string `json:"timestamp"`
	// 数据
	Data map[string]any `json:"data,omitempty"`
}

// SendWebhook 发送Webhook负载到指定URL，使用HMAC-SHA256签名
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
		return fmt.Errorf("Webhook POST %s 失败: %d %s", config.URL, resp.StatusCode, string(respBody))
	}

	return nil
}

func computeHMAC(secret string, data []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}
