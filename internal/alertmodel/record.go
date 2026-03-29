package alertmodel

import (
	"encoding/json"
	"time"
)

// AlertStatus 告警状态
type AlertStatus string

const (
	AlertStatusActive        AlertStatus = "active"        // 活跃
	AlertStatusAcknowledged AlertStatus = "acknowledged" // 已确认
	AlertStatusResolved     AlertStatus = "resolved"       // 已解决
	AlertStatusEscalated    AlertStatus = "escalated"     // 已升级
)

// EventType 事件类型
type EventType string

const (
	EventTypeBackupFailed        EventType = "backup_failed"
	EventTypeBackupTimeout       EventType = "backup_timeout"
	EventTypeBackupSlow         EventType = "backup_slow"
	EventTypeRestoreFailed       EventType = "restore_failed"
	EventTypeStorageFull        EventType = "storage_full"
	EventTypeEncryptionFailed    EventType = "encryption_failed"
	EventTypeCompressionFailed   EventType = "compression_failed"
	EventTypeUploadFailed        EventType = "upload_failed"
	EventTypeVerificationFailed  EventType = "verification_failed"
)

// AlertRecord 告警记录表
type AlertRecord struct {
	ID     int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	RuleID int64  `json:"rule_id" gorm:"not null;index"`
	TaskID *int64 `json:"task_id" gorm:"index"`

	// 告警内容
	Level     AlertLevel `json:"level" gorm:"size:10;not null;default:'P2'"`
	Title     string     `json:"title" gorm:"size:500;not null"`
	Content   string     `json:"content" gorm:"type:text;not null"`
	EventType EventType  `json:"event_type" gorm:"size:50;not null;index"`

	// 冗余存储的任务信息
	TaskName string `json:"task_name" gorm:"size:200"`
	DBType   string `json:"db_type" gorm:"size:20"`
	Source   string `json:"source" gorm:"size:255"`

	// 状态
	Status AlertStatus `json:"status" gorm:"size:20;default:'active';index"`

	// 确认信息
	AcknowledgedAt *time.Time `json:"acknowledged_at"`
	AcknowledgedBy string     `json:"acknowledged_by" gorm:"size:100"`
	AckNote        string     `json:"ack_note" gorm:"size:500"`

	// 解决信息
	ResolvedAt *time.Time `json:"resolved_at"`
	ResolvedBy string     `json:"resolved_by" gorm:"size:100"`
	ResolveNote string    `json:"resolve_note" gorm:"size:500"`

	// 升级信息
	EscalatedAt      *time.Time `json:"escalated_at"`
	EscalatedToLevel AlertLevel `json:"escalated_to_level" gorm:"size:10"`
	EscalatedReason  string     `json:"escalated_reason" gorm:"size:500"`

	// 触发时间
	TriggeredAt time.Time `json:"triggered_at" gorm:"index"`

	// 审计字段
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 关联
	Rule *AlertRule `json:"rule,omitempty" gorm:"foreignKey:RuleID"`
}

func (AlertRecord) TableName() string {
	return "alert_records"
}

// AlertNotificationRecord 通知发送记录表
type AlertNotificationRecord struct {
	ID        int64         `json:"id" gorm:"primaryKey;autoIncrement"`
	AlertID   int64         `json:"alert_id" gorm:"not null;uniqueIndex:uk_alert_channel"`
	ChannelID int64         `json:"channel_id" gorm:"not null;uniqueIndex:uk_alert_channel"`

	// 发送状态
	Status     string    `json:"status" gorm:"size:20;default:'pending';index"`
	SentAt     *time.Time `json:"sent_at"`
	Response   string    `json:"response" gorm:"size:1000"`
	ErrorMsg   string    `json:"error_message" gorm:"size:500"`
	RetryCount int       `json:"retry_count" gorm:"default:0"`

	// 审计字段
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 关联
	Alert   *AlertRecord         `json:"alert,omitempty" gorm:"foreignKey:AlertID"`
	Channel *NotificationChannel `json:"channel,omitempty" gorm:"foreignKey:ChannelID"`
}

func (AlertNotificationRecord) TableName() string {
	return "alert_notification_records"
}

// AlertNote 告警备注表
type AlertNote struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	AlertID   int64     `json:"alert_id" gorm:"not null;index"`
	Content   string    `json:"content" gorm:"size:1000;not null"`
	CreatedBy string    `json:"created_by" gorm:"size:100;not null"`
	CreatedAt time.Time `json:"created_at"`

	// 关联
	Alert *AlertRecord `json:"alert,omitempty" gorm:"foreignKey:AlertID"`
}

func (AlertNote) TableName() string {
	return "alert_notes"
}

// AlertRuleStats 告警规则统计表
type AlertRuleStats struct {
	ID     int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	RuleID int64  `json:"rule_id" gorm:"not null;uniqueIndex:uk_rule_date"`
	Date   string `json:"stat_date" gorm:"type:date;not null;uniqueIndex:uk_rule_date;index"` // YYYY-MM-DD

	// 触发统计
	TriggerCount int `json:"trigger_count" gorm:"default:0"`
	UniqueTasks  int `json:"unique_tasks" gorm:"default:0"`

	// 级别分布
	P0Count int `json:"p0_count" gorm:"default:0"`
	P1Count int `json:"p1_count" gorm:"default:0"`
	P2Count int `json:"p2_count" gorm:"default:0"`
	P3Count int `json:"p3_count" gorm:"default:0"`

	// 状态分布
	ActiveCount        int `json:"active_count" gorm:"default:0"`
	AcknowledgedCount  int `json:"acknowledged_count" gorm:"default:0"`
	ResolvedCount      int `json:"resolved_count" gorm:"default:0"`

	// 性能统计
	AvgResolutionMinutes *int `json:"avg_resolution_minutes"`

	// 审计字段
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (AlertRuleStats) TableName() string {
	return "alert_rule_stats"
}

// AlertRecordResponse 告警记录响应
type AlertRecordResponse struct {
	ID            int64                       `json:"id"`
	RuleID        int64                       `json:"rule_id"`
	RuleName      string                      `json:"rule_name,omitempty"`
	TaskID        *int64                      `json:"task_id,omitempty"`
	Level         AlertLevel                  `json:"level"`
	Status        AlertStatus                  `json:"status"`
	Title         string                      `json:"title"`
	Content       string                      `json:"content"`
	EventType     EventType                   `json:"event_type"`
	TaskName      string                      `json:"task_name,omitempty"`
	DBType        string                      `json:"db_type,omitempty"`
	Source        string                      `json:"source,omitempty"`
	TriggeredAt   time.Time                   `json:"triggered_at"`
	AcknowledgedAt *time.Time                  `json:"acknowledged_at,omitempty"`
	AcknowledgedBy string                      `json:"acknowledged_by,omitempty"`
	ResolvedAt    *time.Time                   `json:"resolved_at,omitempty"`
	ResolvedBy    string                      `json:"resolved_by,omitempty"`
	EscalatedAt   *time.Time                   `json:"escalated_at,omitempty"`
	EscalatedToLevel AlertLevel               `json:"escalated_to_level,omitempty"`
	NotificationRecords []NotificationRecordResponse `json:"notification_records,omitempty"`
	Notes         []AlertNoteResponse         `json:"notes,omitempty"`
}

// NotificationRecordResponse 通知发送记录响应
type NotificationRecordResponse struct {
	ChannelID   int64     `json:"channel_id"`
	ChannelName string    `json:"channel_name,omitempty"`
	ChannelType ChannelType `json:"channel_type,omitempty"`
	Status      string    `json:"status"`
	SentAt      *time.Time `json:"sent_at,omitempty"`
	Response    string    `json:"response,omitempty"`
	ErrorMsg    string    `json:"error_message,omitempty"`
}

// AlertNoteResponse 告警备注响应
type AlertNoteResponse struct {
	ID        int64     `json:"id"`
	Content   string    `json:"content"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

// ToResponse 转换为响应结构
func (a *AlertRecord) ToResponse() *AlertRecordResponse {
	resp := &AlertRecordResponse{
		ID:               a.ID,
		RuleID:           a.RuleID,
		TaskID:           a.TaskID,
		Level:            a.Level,
		Status:           a.Status,
		Title:            a.Title,
		Content:          a.Content,
		EventType:        a.EventType,
		TaskName:         a.TaskName,
		DBType:           a.DBType,
		Source:           a.Source,
		TriggeredAt:      a.TriggeredAt,
		AcknowledgedAt:   a.AcknowledgedAt,
		AcknowledgedBy:   a.AcknowledgedBy,
		ResolvedAt:       a.ResolvedAt,
		ResolvedBy:       a.ResolvedBy,
		EscalatedAt:      a.EscalatedAt,
		EscalatedToLevel: a.EscalatedToLevel,
	}
	if a.Rule != nil {
		resp.RuleName = a.Rule.Name
	}
	return resp
}

// ListAlertsRequest 列出告警请求
type ListAlertsRequest struct {
	Level     AlertLevel  `form:"level"`
	Status    AlertStatus `form:"status"`
	RuleID    *int64      `form:"rule_id"`
	TaskID    *int64      `form:"task_id"`
	StartTime *time.Time  `form:"start_time"`
	EndTime   *time.Time  `form:"end_time"`
	Page      int         `form:"page,default=1"`
	PageSize  int         `form:"page_size,default=20"`
}

// AcknowledgeAlertRequest 确认告警请求
type AcknowledgeAlertRequest struct {
	Note string `json:"note" binding:"max=500"`
}

// ResolveAlertRequest 解决告警请求
type ResolveAlertRequest struct {
	Note string `json:"note" binding:"max=500"`
}

// ReassignAlertRequest 转派告警请求
type ReassignAlertRequest struct {
	Assignee string `json:"assignee" binding:"required"`
	Reason   string `json:"reason" binding:"max=500"`
}

// AddNoteRequest 添加备注请求
type AddNoteRequest struct {
	Content string `json:"content" binding:"required,max=1000"`
}

// AlertEvent 告警事件（内部使用）
type AlertEvent struct {
	EventType EventType  `json:"event_type"`
	TaskID    int64      `json:"task_id"`
	TaskName  string     `json:"task_name"`
	DBType    string     `json:"db_type"`
	Source    string     `json:"source"`
	Level     AlertLevel `json:"level"`
	Title     string     `json:"title"`
	Content   string     `json:"content"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

// ToJSON 序列化为 JSON
func (e *AlertEvent) ToJSON() (string, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ParseAlertEvent 解析 JSON 为 AlertEvent
func ParseAlertEvent(data string) (*AlertEvent, error) {
	var event AlertEvent
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return nil, err
	}
	return &event, nil
}
