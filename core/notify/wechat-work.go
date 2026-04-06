package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// WechatWorkConfig 企业微信配置
type WechatWorkConfig struct {
	WebhookURL string `json:"webhookUrl"`
}

// SendWechatWork 发送企业微信消息
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
		return fmt.Errorf("企业微信发送失败: %d %s", resp.StatusCode, string(respBody))
	}

	return nil
}
