package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// TelegramConfig 表示Telegram notification configuration。
type TelegramConfig struct {
	BotToken string `json:"botToken"`
	ChatID   string `json:"chatId"`
}

// SendTelegram sends a message via Telegram
func SendTelegram(ctx context.Context, config TelegramConfig, message string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", config.BotToken)

	payload := map[string]string{
		"chat_id":    config.ChatID,
		"text":       message,
		"parse_mode": "Markdown",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
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
		return fmt.Errorf("telegram send failed: %d %s", resp.StatusCode, string(respBody))
	}

	return nil
}
