package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// WechatWorkConfig 表示WeChat Work notification configuration。
type WechatWorkConfig struct {
	WebhookURL string `json:"webhookUrl"`
}

// SendWechatWork sends a message via WeChat Work
func SendWechatWork(ctx context.Context, config WechatWorkConfig, message string) error {
	payload := map[string]any{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": message,
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
		return fmt.Errorf("wechat-work send failed: %d %s", resp.StatusCode, string(respBody))
	}

	return nil
}
