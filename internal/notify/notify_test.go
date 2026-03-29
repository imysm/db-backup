// Package notify 测试
package notify

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/imysm/db-backup/internal/model"
	"github.com/imysm/db-backup/internal/util"
)

func TestNewNotifier(t *testing.T) {
	notifier := NewNotifier()
	if notifier == nil {
		t.Fatal("NewNotifier() returned nil")
	}
	if notifier.client == nil {
		t.Error("notifier.client is nil")
	}
}

func TestNotifier_Send_Disabled(t *testing.T) {
	notifier := NewNotifier()
	cfg := model.NotifyConfig{
		Enabled: false,
	}

	err := notifier.Send(context.Background(), cfg, "test message")
	if err != nil {
		t.Errorf("Send() error = %v", err)
	}
}

func TestNotifier_Send_Webhook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewNotifier()
	cfg := model.NotifyConfig{
		Enabled:  true,
		Type:     "webhook",
		Endpoint: server.URL,
	}

	err := notifier.Send(context.Background(), cfg, "test message")
	if err != nil {
		t.Errorf("Send() error = %v", err)
	}
}

func TestNotifier_Send_Webhook_EmptyURL(t *testing.T) {
	notifier := NewNotifier()
	cfg := model.NotifyConfig{
		Enabled:  true,
		Type:     "webhook",
		Endpoint: "",
	}

	err := notifier.Send(context.Background(), cfg, "test message")
	if err == nil {
		t.Error("Send() expected error for empty URL")
	}
}

func TestNotifier_Send_Webhook_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	notifier := NewNotifier()
	cfg := model.NotifyConfig{
		Enabled:  true,
		Type:     "webhook",
		Endpoint: server.URL,
	}

	err := notifier.Send(context.Background(), cfg, "test message")
	if err == nil {
		t.Error("Send() expected error for HTTP 500")
	}
}

func TestNotifier_Send_DingTalk(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewNotifier()
	cfg := model.NotifyConfig{
		Enabled:  true,
		Type:     "dingtalk",
		Endpoint: server.URL,
	}

	err := notifier.Send(context.Background(), cfg, "test message")
	if err != nil {
		t.Errorf("Send() error = %v", err)
	}
}

func TestNotifier_Send_DingTalk_EmptyURL(t *testing.T) {
	notifier := NewNotifier()
	cfg := model.NotifyConfig{
		Enabled:  true,
		Type:     "dingtalk",
		Endpoint: "",
	}

	err := notifier.Send(context.Background(), cfg, "test message")
	if err == nil {
		t.Error("Send() expected error for empty URL")
	}
}

func TestNotifier_Send_Feishu(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewNotifier()
	cfg := model.NotifyConfig{
		Enabled:  true,
		Type:     "feishu",
		Endpoint: server.URL,
	}

	err := notifier.Send(context.Background(), cfg, "test message")
	if err != nil {
		t.Errorf("Send() error = %v", err)
	}
}

func TestNotifier_Send_Feishu_EmptyURL(t *testing.T) {
	notifier := NewNotifier()
	cfg := model.NotifyConfig{
		Enabled:  true,
		Type:     "feishu",
		Endpoint: "",
	}

	err := notifier.Send(context.Background(), cfg, "test message")
	if err == nil {
		t.Error("Send() expected error for empty URL")
	}
}

func TestNotifier_Send_Slack(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewNotifier()
	cfg := model.NotifyConfig{
		Enabled:  true,
		Type:     "slack",
		Endpoint: server.URL,
	}

	err := notifier.Send(context.Background(), cfg, "test message")
	if err != nil {
		t.Errorf("Send() error = %v", err)
	}
}

func TestNotifier_Send_Slack_EmptyURL(t *testing.T) {
	notifier := NewNotifier()
	cfg := model.NotifyConfig{
		Enabled:  true,
		Type:     "slack",
		Endpoint: "",
	}

	err := notifier.Send(context.Background(), cfg, "test message")
	if err == nil {
		t.Error("Send() expected error for empty URL")
	}
}

func TestNotifier_Send_DefaultToWebhook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewNotifier()
	cfg := model.NotifyConfig{
		Enabled:  true,
		Type:     "unknown", // 未知类型默认使用 webhook
		Endpoint: server.URL,
	}

	err := notifier.Send(context.Background(), cfg, "test message")
	if err != nil {
		t.Errorf("Send() error = %v", err)
	}
}

func TestNotifier_NotifySuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewNotifier()
	cfg := model.NotifyConfig{
		Enabled:  true,
		Type:     "webhook",
		Endpoint: server.URL,
	}

	result := &model.BackupResult{
		TaskID:   "test-001",
		FilePath: "/tmp/backup.sql",
		FileSize: 1024,
		EndTime:  time.Now(),
	}

	err := notifier.NotifySuccess(context.Background(), cfg, "Test Task", result)
	if err != nil {
		t.Errorf("NotifySuccess() error = %v", err)
	}
}

func TestNotifier_NotifySuccess_Disabled(t *testing.T) {
	notifier := NewNotifier()
	cfg := model.NotifyConfig{
		Enabled: false,
	}

	result := &model.BackupResult{}
	err := notifier.NotifySuccess(context.Background(), cfg, "Test Task", result)
	if err != nil {
		t.Errorf("NotifySuccess() error = %v", err)
	}
}

func TestNotifier_NotifyFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewNotifier()
	cfg := model.NotifyConfig{
		Enabled:  true,
		Type:     "webhook",
		Endpoint: server.URL,
	}

	err := notifier.NotifyFailure(context.Background(), cfg, "Test Task", errors.New("test error"))
	if err != nil {
		t.Errorf("NotifyFailure() error = %v", err)
	}
}

func TestNotifier_NotifyFailure_Disabled(t *testing.T) {
	notifier := NewNotifier()
	cfg := model.NotifyConfig{
		Enabled: false,
	}

	err := notifier.NotifyFailure(context.Background(), cfg, "Test Task", errors.New("test error"))
	if err != nil {
		t.Errorf("NotifyFailure() error = %v", err)
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		size int64
		want string
	}{
		{500, "500 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1048576, "1.00 MB"},
		{1572864, "1.50 MB"},
		{1073741824, "1.00 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := util.FormatFileSize(tt.size)
			if got != tt.want {
				t.Errorf("util.FormatFileSize() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestNotifier_Send_Timeout(t *testing.T) {
	// 创建一个延迟响应的服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 不响应，让客户端超时
		select {}
	}))
	defer server.Close()

	notifier := NewNotifier()
	cfg := model.NotifyConfig{
		Enabled:  true,
		Type:     "webhook",
		Endpoint: server.URL,
	}

	// 使用短超时的 context
	ctx, cancel := context.WithTimeout(context.Background(), 100)
	defer cancel()

	err := notifier.Send(ctx, cfg, "test message")
	if err == nil {
		t.Error("Send() expected timeout error")
	}
}
