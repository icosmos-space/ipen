package notify

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/icosmos-space/ipen/core/models"
)

// NotifyMessage 通知消息
type NotifyMessage struct {
	// 消息标题
	Title string `json:"title"`
	// 消息内容
	Body string `json:"body"`
}

// DispatchNotification 通知通知消息
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
		return fmt.Errorf("未知消息渠道类型: %s", channel.Type)
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
				fmt.Fprintf(os.Stderr, "[webhook] %s 失败: %v\n", ch.WebhookURL, err)
			}
		}(channel)
	}

	return nil
}
