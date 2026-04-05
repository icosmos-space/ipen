package notify

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/icosmos-space/ipen/core/models"
)

// NotifyMessage 表示a notification message。
type NotifyMessage struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// DispatchNotification dispatches a notification to all channels
func DispatchNotification(ctx context.Context, channels []models.NotifyChannel, message NotifyMessage) error {
	fullText := fmt.Sprintf("**%s**\n\n%s", message.Title, message.Body)

	for _, channel := range channels {
		go func(ch models.NotifyChannel) {
			if err := dispatchToChannel(ctx, ch, fullText, message); err != nil {
				fmt.Fprintf(os.Stderr, "[notify] %s failed: %v\n", ch.Type, err)
			}
		}(channel)
	}

	return nil
}

func dispatchToChannel(ctx context.Context, channel models.NotifyChannel, fullText string, message NotifyMessage) error {
	switch channel.Type {
	case "telegram":
		return SendTelegram(ctx, TelegramConfig{
			BotToken: channel.BotToken,
			ChatID:   channel.ChatID,
		}, fullText)
	case "feishu":
		return SendFeishu(ctx, FeishuConfig{
			WebhookURL: channel.WebhookURL,
		}, message.Title, message.Body)
	case "wechat-work":
		return SendWechatWork(ctx, WechatWorkConfig{
			WebhookURL: channel.WebhookURL,
		}, fullText)
	case "webhook":
		return SendWebhook(ctx, WebhookConfig{
			URL:    channel.WebhookURL,
			Secret: channel.Secret,
			Events: channel.Events,
		}, WebhookPayload{
			Event:     "pipeline-complete",
			BookID:    "",
			Timestamp: time.Now().Format(time.RFC3339),
			Data: map[string]any{
				"title": message.Title,
				"body":  message.Body,
			},
		})
	default:
		return fmt.Errorf("unknown channel type: %s", channel.Type)
	}
}

// DispatchWebhookEvent dispatches a structured webhook event
func DispatchWebhookEvent(ctx context.Context, channels []models.NotifyChannel, payload WebhookPayload) error {
	var webhookChannels []models.NotifyChannel
	for _, ch := range channels {
		if ch.Type == "webhook" {
			webhookChannels = append(webhookChannels, ch)
		}
	}

	if len(webhookChannels) == 0 {
		return nil
	}

	for _, channel := range webhookChannels {
		go func(ch models.NotifyChannel) {
			if err := SendWebhook(ctx, WebhookConfig{
				URL:    ch.WebhookURL,
				Secret: ch.Secret,
				Events: ch.Events,
			}, payload); err != nil {
				fmt.Fprintf(os.Stderr, "[webhook] %s failed: %v\n", ch.WebhookURL, err)
			}
		}(channel)
	}

	return nil
}
