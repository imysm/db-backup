package alertmodel

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// ChannelType 通知渠道类型
type ChannelType string

const (
	ChannelTypeFeishu   ChannelType = "feishu"
	ChannelTypeWeCom    ChannelType = "wecom"
	ChannelTypeDingTalk ChannelType = "dingtalk"
	ChannelTypeEmail    ChannelType = "email"
)

// HealthStatus 健康状态
type HealthStatus string

const (
	HealthStatusUnknown   HealthStatus = "unknown"
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// NotificationChannel 通知渠道配置表
type NotificationChannel struct {
	ID             int64         `json:"id" gorm:"primaryKey;autoIncrement"`
	Name           string        `json:"name" gorm:"size:100;not null"`
	Type           ChannelType   `json:"type" gorm:"size:20;not null;default:'feishu'"`
	Config         string        `json:"-" gorm:"type:text;not null"` // JSON 配置，加密存储
	ConfigData     *ChannelConfig `json:"config" gorm:"-"`              // 运行时解析
	ConfigEncrypted bool          `json:"config_encrypted" gorm:"default:1"`
	Enabled        bool          `json:"enabled"`
	Priority       int           `json:"priority" gorm:"default:1"`
	Description    string        `json:"description" gorm:"size:500"`

	// 健康状态
	HealthStatus HealthStatus `json:"health_status" gorm:"size:20;default:'unknown'"`
	LastSentAt   *time.Time   `json:"last_sent_at"`
	LastError    string       `json:"last_error" gorm:"size:1000"`

	// 统计
	SendCount  int `json:"send_count" gorm:"default:0"`
	FailedCount int `json:"failed_count" gorm:"default:0"`

	// 审计字段
	CreatedBy string    `json:"created_by" gorm:"size:100"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedBy string    `json:"updated_by" gorm:"size:100"`
	UpdatedAt time.Time `json:"updated_at"`

	// 软删除
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (NotificationChannel) TableName() string {
	return "alert_notification_channels"
}

// ChannelConfig 渠道配置（各渠道通用结构）
type ChannelConfig struct {
	// 通用 Webhook 配置
	WebhookURL string `json:"webhook_url,omitempty"`
	Keyword    string `json:"keyword,omitempty"`

	// 钉钉签名密钥
	Secret string `json:"secret,omitempty"`

	// 邮件配置
	SMTPHost  string   `json:"smtp_host,omitempty"`
	SMTPPort  int      `json:"smtp_port,omitempty"`
	Username  string   `json:"username,omitempty"`
	Password  string   `json:"password,omitempty"`
	From      string   `json:"from,omitempty"`
	To        []string `json:"to,omitempty"`
	UseTLS    bool     `json:"use_tls,omitempty"`
}

// GetConfig 解析配置 JSON
func (c *NotificationChannel) GetConfig() (*ChannelConfig, error) {
	if c.ConfigData != nil {
		return c.ConfigData, nil
	}
	if c.Config == "" {
		return &ChannelConfig{}, nil
	}
	var cfg ChannelConfig
	if err := json.Unmarshal([]byte(c.Config), &cfg); err != nil {
		return nil, err
	}
	c.ConfigData = &cfg
	return &cfg, nil
}

// SetConfig 设置配置 JSON
func (c *NotificationChannel) SetConfig(cfg *ChannelConfig) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	c.Config = string(data)
	c.ConfigData = cfg
	return nil
}

// IsHealthy 检查渠道是否健康
func (c *NotificationChannel) IsHealthy() bool {
	return c.HealthStatus == HealthStatusHealthy
}

// IsEnabled 检查渠道是否启用
func (c *NotificationChannel) IsEnabled() bool {
	return c.Enabled
}

// ChannelResponse 渠道响应结构（不包含敏感配置）
type ChannelResponse struct {
	ID            int64        `json:"id"`
	Name          string       `json:"name"`
	Type          ChannelType  `json:"type"`
	ConfigSummary *ChannelConfig `json:"config,omitempty"` // 展示用，脱敏
	Enabled       bool         `json:"enabled"`
	Priority      int          `json:"priority"`
	Description   string       `json:"description,omitempty"`
	HealthStatus  HealthStatus `json:"health_status"`
	LastSentAt    *time.Time   `json:"last_sent_at,omitempty"`
	LastError     string       `json:"last_error,omitempty"`
	SendCount     int          `json:"send_count"`
	FailedCount   int          `json:"failed_count"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

// ToResponse 转换为响应结构
func (c *NotificationChannel) ToResponse(maskSecret bool) (*ChannelResponse, error) {
	cfg, err := c.GetConfig()
	if err != nil {
		return nil, err
	}

	// 创建副本用于脱敏处理
	displayCfg := &ChannelConfig{}
	*displayCfg = *cfg

	// 脱敏处理
	if maskSecret {
		if displayCfg.Secret != "" {
			displayCfg.Secret = "******"
		}
		if displayCfg.Password != "" {
			displayCfg.Password = "******"
		}
	}

	return &ChannelResponse{
		ID:            c.ID,
		Name:          c.Name,
		Type:          c.Type,
		ConfigSummary: displayCfg,
		Enabled:       c.Enabled,
		Priority:      c.Priority,
		Description:   c.Description,
		HealthStatus:  c.HealthStatus,
		LastSentAt:    c.LastSentAt,
		LastError:     c.LastError,
		SendCount:     c.SendCount,
		FailedCount:   c.FailedCount,
		CreatedAt:     c.CreatedAt,
		UpdatedAt:     c.UpdatedAt,
	}, nil
}

// CreateChannelRequest 创建渠道请求
type CreateChannelRequest struct {
	Name        string        `json:"name" binding:"required,max=100"`
	Type        ChannelType   `json:"type" binding:"required,oneof=feishu wecom dingtalk email"`
	Config      *ChannelConfig `json:"config" binding:"required"`
	Enabled     *bool         `json:"enabled"`
	Priority    *int          `json:"priority"`
	Description string        `json:"description" binding:"max=500"`
}

// UpdateChannelRequest 更新渠道请求
type UpdateChannelRequest struct {
	Name        *string       `json:"name" binding:"omitempty,max=100"`
	Config      *ChannelConfig `json:"config"`
	Enabled     *bool         `json:"enabled"`
	Priority    *int          `json:"priority"`
	Description *string       `json:"description" binding:"omitempty,max=500"`
}

// TestChannelRequest 测试渠道请求
type TestChannelRequest struct {
	Message string `json:"message" binding:"required"`
}
