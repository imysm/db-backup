// Package notify 提供备份通知功能
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/imysm/db-backup/internal/model"
	"github.com/imysm/db-backup/internal/util"
)

// Notifier 通知器
type Notifier struct {
	client *http.Client
}

// NewNotifier 创建通知器
func NewNotifier() *Notifier {
	return &Notifier{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Send 发送通知
func (n *Notifier) Send(ctx context.Context, cfg model.NotifyConfig, message string) error {
	if !cfg.Enabled {
		return nil
	}

	switch cfg.Type {
	case "webhook":
		return n.sendWebhook(ctx, cfg.Endpoint, message)
	case "dingtalk":
		return n.sendDingTalk(ctx, cfg.Endpoint, message)
	case "feishu":
		return n.sendFeishu(ctx, cfg.Endpoint, message)
	case "slack":
		return n.sendSlack(ctx, cfg.Endpoint, message)
	default:
		return n.sendWebhook(ctx, cfg.Endpoint, message)
	}
}

// NotifySuccess 发送成功通知
func (n *Notifier) NotifySuccess(ctx context.Context, cfg model.NotifyConfig, taskName string, result *model.BackupResult) error {
	if !cfg.Enabled {
		return nil
	}

	message := fmt.Sprintf(`✅ 备份成功
任务: %s
文件: %s
大小: %s
耗时: %v
时间: %s`,
		taskName,
		result.FilePath,
		util.FormatFileSize(result.FileSize),
		result.Duration.Round(time.Second),
		result.EndTime.Format("2006-01-02 15:04:05"),
	)

	return n.Send(ctx, cfg, message)
}

// NotifyFailure 发送失败通知
func (n *Notifier) NotifyFailure(ctx context.Context, cfg model.NotifyConfig, taskName string, err error) error {
	if !cfg.Enabled {
		return nil
	}

	message := fmt.Sprintf(`❌ 备份失败
任务: %s
错误: %v
时间: %s`,
		taskName,
		err,
		time.Now().Format("2006-01-02 15:04:05"),
	)

	return n.Send(ctx, cfg, message)
}

func (n *Notifier) sendWebhook(ctx context.Context, url, message string) error {
	if url == "" {
		return fmt.Errorf("webhook URL 为空")
	}

	payload := map[string]string{
		"text":    message,
		"content": message,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook 响应错误: %d", resp.StatusCode)
	}

	return nil
}

func (n *Notifier) sendDingTalk(ctx context.Context, url, message string) error {
	if url == "" {
		return fmt.Errorf("钉钉 webhook URL 为空")
	}

	payload := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": message,
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("钉钉请求失败: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

func (n *Notifier) sendFeishu(ctx context.Context, url, message string) error {
	if url == "" {
		return fmt.Errorf("飞书 webhook URL 为空")
	}

	payload := map[string]interface{}{
		"msg_type": "text",
		"content": map[string]string{
			"text": message,
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("飞书请求失败: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

func (n *Notifier) sendSlack(ctx context.Context, url, message string) error {
	if url == "" {
		return fmt.Errorf("Slack webhook URL 为空")
	}

	payload := map[string]string{
		"text": message,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("Slack 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("Slack 响应错误: HTTP %d", resp.StatusCode)
	}

	return nil
}
