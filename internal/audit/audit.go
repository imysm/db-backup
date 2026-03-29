package audit

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// AuditLog 审计日志
type AuditLog struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	UserID       *uint          `gorm:"index" json:"user_id"`              // 操作用户ID
	Username     string         `gorm:"size:64" json:"username"`           // 操作用户名
	TenantID     *uint          `gorm:"index" json:"tenant_id"`            // 租户ID
	Action       string         `gorm:"size:64;notNull" json:"action"`      // 操作类型: CREATE, UPDATE, DELETE, EXECUTE, QUERY
	Resource     string         `gorm:"size:64;notNull" json:"resource"`   // 资源类型: job, record, template, restore, merge, user, tenant
	ResourceID   *uint          `gorm:"index" json:"resource_id"`         // 资源ID
	ResourceName string         `gorm:"size:256" json:"resource_name"`     // 资源名称
	Details      string         `gorm:"type:text" json:"details"`         // 详细信息(JSON)
	Result       string         `gorm:"size:20;notNull" json:"result"`    // 结果: success, failure, partial
	ErrorMessage string         `gorm:"type:text" json:"error_message"`   // 错误信息
	IPAddress    string         `gorm:"size:45" json:"ip_address"`        // IP地址(ipv4/ipv6)
	UserAgent    string         `gorm:"size:512" json:"user_agent"`       // 用户代理
	RequestID    string         `gorm:"size:64;index" json:"request_id"`   // 请求追踪ID
	Duration     int            `json:"duration"`                          // 操作耗时(毫秒)
	CreatedAt    time.Time      `gorm:"index" json:"created_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}

// 操作类型
const (
	ActionCreate  = "CREATE"
	ActionUpdate  = "UPDATE"
	ActionDelete  = "DELETE"
	ActionExecute = "EXECUTE"
	ActionQuery   = "QUERY"
	ActionLogin   = "LOGIN"
	ActionLogout  = "LOGOUT"
	ActionExport  = "EXPORT"
	ActionImport  = "IMPORT"
)

// 操作结果
const (
	ResultSuccess  = "success"
	ResultFailure  = "failure"
	ResultPartial  = "partial"
)

// ResourceType 资源类型
type ResourceType string

const (
	ResourceJob     ResourceType = "job"
	ResourceRecord ResourceType = "record"
	ResourceTemplate ResourceType = "template"
	ResourceRestore ResourceType = "restore"
	ResourceMerge   ResourceType = "merge"
	ResourceUser    ResourceType = "user"
	ResourceTenant  ResourceType = "tenant"
	ResourceRole    ResourceType = "role"
	ResourceSetting ResourceType = "setting"
)

// AuditDetails 审计详情
type AuditDetails struct {
	// 变更前后对比
	Before map[string]interface{} `json:"before,omitempty"`
	After  map[string]interface{} `json:"after,omitempty"`

	// 请求信息
	Method string `json:"method,omitempty"`
	Path   string `json:"path,omitempty"`
	Query  string `json:"query,omitempty"`

	// 操作信息
	Extra map[string]interface{} `json:"extra,omitempty"`
}

// NewAuditDetails 创建审计详情
func NewAuditDetails() *AuditDetails {
	return &AuditDetails{
		Before: make(map[string]interface{}),
		After:  make(map[string]interface{}),
		Extra:  make(map[string]interface{}),
	}
}

// ToJSON 序列化为JSON字符串
func (d *AuditDetails) ToJSON() string {
	data, err := json.Marshal(d)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// ParseDetails 解析JSON字符串为审计详情
func ParseDetails(jsonStr string) (*AuditDetails, error) {
	var details AuditDetails
	if jsonStr == "" {
		return &details, nil
	}
	err := json.Unmarshal([]byte(jsonStr), &details)
	if err != nil {
		return nil, err
	}
	return &details, nil
}
