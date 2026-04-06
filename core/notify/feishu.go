package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// FeishuConfig 飞书消息通知配置
type FeishuConfig struct {
	WebhookURL string `json:"webhookUrl"`
}

// SendFeishu 发送飞书消息通知
func SendFeishu(ctx context.Context, config FeishuConfig, title string, body string) error {
	payload := map[string]any{
		"msg_type": "interactive",
		"card": map[string]any{
			"header": map[string]any{
				"title": map[string]string{
					"tag":     "plain_text",
					"content": title,
				},
			},
			"elements": []map[string]any{
				{
					"tag":     "markdown",
					"content": body,
				},
			},
		},
	}

	reqBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", config.WebhookURL, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("飞书消息通知失败: %d %s", resp.StatusCode, string(respBody))
	}

	return nil
}
