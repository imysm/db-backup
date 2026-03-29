package alertmodel

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// AlertLevel 告警级别
type AlertLevel string

const (
	AlertLevelP0 AlertLevel = "P0"
	AlertLevelP1 AlertLevel = "P1"
	AlertLevelP2 AlertLevel = "P2"
	AlertLevelP3 AlertLevel = "P3"
)

// ConditionOperator 条件操作符
type ConditionOperator string

const (
	OpEQ     ConditionOperator = "eq"      // 等于
	OpNE     ConditionOperator = "ne"      // 不等于
	OpGT     ConditionOperator = "gt"      // 大于
	OpGTE    ConditionOperator = "gte"     // 大于等于
	OpLT     ConditionOperator = "lt"      // 小于
	OpLTE    ConditionOperator = "lte"     // 小于等于
	OpIn     ConditionOperator = "in"      // 包含在列表中
	OpNotIn  ConditionOperator = "not_in"  // 不在列表中
	OpContains ConditionOperator = "contains" // 字符串包含
	OpRegex  ConditionOperator = "regex"   // 正则匹配
)

// ConditionOp 条件组合方式
type ConditionOp string

const (
	ConditionOpAND ConditionOp = "AND"
	ConditionOpOR  ConditionOp = "OR"
)

// AlertRule 告警规则表
type AlertRule struct {
	ID          int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string    `json:"name" gorm:"size:200;not null"`
	Description string    `json:"description" gorm:"size:500"`
	Enabled     bool      `json:"enabled"`
	Priority    int       `json:"priority" gorm:"default:50"` // 1-100

	// 条件配置
	Level           AlertLevel   `json:"level" gorm:"size:10;not null;default:'P2'"`
	ConditionOp     ConditionOp  `json:"condition_op" gorm:"size:10;default:'AND'"`
	Conditions      string       `json:"-" gorm:"type:text;not null"` // JSON 条件列表
	ConditionsList  []Condition  `json:"conditions" gorm:"-"`          // 运行时解析

	// 通知配置
	Channels     string  `json:"-" gorm:"type:text;not null"` // JSON 渠道 ID 列表
	ChannelsList []int64 `json:"channels" gorm:"-"`           // 运行时解析
	Cooldown     int     `json:"cooldown" gorm:"default:300"` // 冷却时间（秒）

	// 静默时段配置
	SilentPeriods     string          `json:"-" gorm:"type:text"` // JSON 静默时段列表
	SilentPeriodsList []SilentPeriod `json:"silent_periods" gorm:"-"` // 运行时解析

	// 升级配置
	EscalateEnabled     bool        `json:"escalate_enabled" gorm:"default:0"`
	EscalateTimeout     int         `json:"escalate_timeout"`      // 超时时间（分钟）
	EscalateLevel       AlertLevel  `json:"escalate_level" gorm:"size:10"`
	EscalateChannels    string      `json:"-" gorm:"type:text"`    // JSON 升级渠道 ID 列表
	EscalateChannelsList []int64    `json:"escalate_channels" gorm:"-"`

	// 统计
	MatchedCount  int        `json:"matched_count" gorm:"default:0"`
	LastMatchedAt *time.Time `json:"last_matched_at"`

	// 审计字段
	CreatedBy string    `json:"created_by" gorm:"size:100"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedBy string    `json:"updated_by" gorm:"size:100"`
	UpdatedAt time.Time `json:"updated_at"`

	// 软删除
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (AlertRule) TableName() string {
	return "alert_rules"
}

// Condition 触发条件
type Condition struct {
	Field    string      `json:"field" binding:"required"`    // 字段名
	Operator string      `json:"operator" binding:"required"` // 操作符
	Value    interface{} `json:"value" binding:"required"`    // 比较值
}

// SilentPeriod 静默时段
type SilentPeriod struct {
	StartTime string `json:"start_time" binding:"required"` // HH:mm 格式
	EndTime   string `json:"end_time" binding:"required"`   // HH:mm 格式
	Days      []int  `json:"days" binding:"required"`      // 0=周日, 1=周一...
}

// GetConditions 解析条件列表
func (r *AlertRule) GetConditions() ([]Condition, error) {
	if len(r.ConditionsList) > 0 {
		return r.ConditionsList, nil
	}
	if r.Conditions == "" {
		return []Condition{}, nil
	}
	var conditions []Condition
	if err := json.Unmarshal([]byte(r.Conditions), &conditions); err != nil {
		return nil, err
	}
	r.ConditionsList = conditions
	return conditions, nil
}

// SetConditions 设置条件列表
func (r *AlertRule) SetConditions(conditions []Condition) error {
	data, err := json.Marshal(conditions)
	if err != nil {
		return err
	}
	r.Conditions = string(data)
	r.ConditionsList = conditions
	return nil
}

// GetChannels 解析渠道列表
func (r *AlertRule) GetChannels() ([]int64, error) {
	if len(r.ChannelsList) > 0 {
		return r.ChannelsList, nil
	}
	if r.Channels == "" {
		return []int64{}, nil
	}
	var channels []int64
	if err := json.Unmarshal([]byte(r.Channels), &channels); err != nil {
		return nil, err
	}
	r.ChannelsList = channels
	return channels, nil
}

// SetChannels 设置渠道列表
func (r *AlertRule) SetChannels(channels []int64) error {
	data, err := json.Marshal(channels)
	if err != nil {
		return err
	}
	r.Channels = string(data)
	r.ChannelsList = channels
	return nil
}

// GetSilentPeriods 解析静默时段列表
func (r *AlertRule) GetSilentPeriods() ([]SilentPeriod, error) {
	if len(r.SilentPeriodsList) > 0 {
		return r.SilentPeriodsList, nil
	}
	if r.SilentPeriods == "" {
		return []SilentPeriod{}, nil
	}
	var periods []SilentPeriod
	if err := json.Unmarshal([]byte(r.SilentPeriods), &periods); err != nil {
		return nil, err
	}
	r.SilentPeriodsList = periods
	return periods, nil
}

// SetSilentPeriods 设置静默时段列表
func (r *AlertRule) SetSilentPeriods(periods []SilentPeriod) error {
	data, err := json.Marshal(periods)
	if err != nil {
		return err
	}
	r.SilentPeriods = string(data)
	r.SilentPeriodsList = periods
	return nil
}

// GetEscalateChannels 解析升级渠道列表
func (r *AlertRule) GetEscalateChannels() ([]int64, error) {
	if len(r.EscalateChannelsList) > 0 {
		return r.EscalateChannelsList, nil
	}
	if r.EscalateChannels == "" {
		return []int64{}, nil
	}
	var channels []int64
	if err := json.Unmarshal([]byte(r.EscalateChannels), &channels); err != nil {
		return nil, err
	}
	r.EscalateChannelsList = channels
	return channels, nil
}

// SetEscalateChannels 设置升级渠道列表
func (r *AlertRule) SetEscalateChannels(channels []int64) error {
	data, err := json.Marshal(channels)
	if err != nil {
		return err
	}
	r.EscalateChannels = string(data)
	r.EscalateChannelsList = channels
	return nil
}

// AlertRuleResponse 告警规则响应
type AlertRuleResponse struct {
	ID          int64        `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	Enabled     bool         `json:"enabled"`
	Priority    int          `json:"priority"`
	Level       AlertLevel   `json:"level"`
	ConditionOp ConditionOp `json:"condition_op"`
	Conditions  []Condition  `json:"conditions"`
	Channels    []int64      `json:"channels"`
	Cooldown    int          `json:"cooldown"`
	SilentPeriods []SilentPeriod  `json:"silent_periods,omitempty"`
	Escalate    *EscalateConfig `json:"escalate,omitempty"`
	MatchedCount   int        `json:"matched_count"`
	LastMatchedAt  *time.Time `json:"last_matched_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// EscalateConfig 升级配置
type EscalateConfig struct {
	Enabled       bool        `json:"enabled"`
	Timeout       int         `json:"timeout"`
	TargetLevel   AlertLevel  `json:"target_level"`
	TargetChannels []int64    `json:"target_channels"`
}

// ToResponse 转换为响应结构
func (r *AlertRule) ToResponse() (*AlertRuleResponse, error) {
	conditions, err := r.GetConditions()
	if err != nil {
		return nil, err
	}
	channels, err := r.GetChannels()
	if err != nil {
		return nil, err
	}
	silentPeriods, err := r.GetSilentPeriods()
	if err != nil {
		return nil, err
	}

	var escalate *EscalateConfig
	if r.EscalateEnabled {
		escalateChannels, err := r.GetEscalateChannels()
		if err != nil {
			return nil, err
		}
		escalate = &EscalateConfig{
			Enabled:       r.EscalateEnabled,
			Timeout:       r.EscalateTimeout,
			TargetLevel:   r.EscalateLevel,
			TargetChannels: escalateChannels,
		}
	}

	return &AlertRuleResponse{
		ID:            r.ID,
		Name:          r.Name,
		Description:   r.Description,
		Enabled:       r.Enabled,
		Priority:      r.Priority,
		Level:         r.Level,
		ConditionOp:   r.ConditionOp,
		Conditions:    conditions,
		Channels:      channels,
		Cooldown:      r.Cooldown,
		SilentPeriods: silentPeriods,
		Escalate:     escalate,
		MatchedCount:  r.MatchedCount,
		LastMatchedAt: r.LastMatchedAt,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	}, nil
}

// CreateAlertRuleRequest 创建告警规则请求
type CreateAlertRuleRequest struct {
	Name         string          `json:"name" binding:"required,max=200"`
	Description  string          `json:"description" binding:"max=500"`
	Enabled      *bool           `json:"enabled"`
	Priority     *int            `json:"priority"`
	Level        AlertLevel      `json:"level" binding:"required,oneof=P0 P1 P2 P3"`
	ConditionOp  *ConditionOp    `json:"condition_op" binding:"omitempty,oneof=AND OR"`
	Conditions   []Condition     `json:"conditions" binding:"required,min=1"`
	Channels     []int64         `json:"channels" binding:"required,min=1"`
	Cooldown     *int            `json:"cooldown"`
	SilentPeriods []SilentPeriod `json:"silent_periods"`
	Escalate     *EscalateConfig `json:"escalate"`
}

// UpdateAlertRuleRequest 更新告警规则请求
type UpdateAlertRuleRequest struct {
	Name         *string          `json:"name" binding:"omitempty,max=200"`
	Description  *string          `json:"description" binding:"omitempty,max=500"`
	Enabled      *bool            `json:"enabled"`
	Priority     *int             `json:"priority"`
	Level        *AlertLevel      `json:"level" binding:"omitempty,oneof=P0 P1 P2 P3"`
	ConditionOp  *ConditionOp     `json:"condition_op" binding:"omitempty,oneof=AND OR"`
	Conditions   []Condition      `json:"conditions"`
	Channels     []int64          `json:"channels"`
	Cooldown     *int             `json:"cooldown"`
	SilentPeriods []SilentPeriod  `json:"silent_periods"`
	Escalate     *EscalateConfig  `json:"escalate"`
}

// ListAlertRulesRequest 列出规则请求
type ListAlertRulesRequest struct {
	Enabled *bool      `form:"enabled"`
	Level   AlertLevel `form:"level"`
	Page    int        `form:"page,default=1"`
	PageSize int       `form:"page_size,default=20"`
}
