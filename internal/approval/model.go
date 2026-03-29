package approval

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// ApprovalType 审批类型
type ApprovalType string

const (
	ApprovalTypeRestore          ApprovalType = "restore"           // 恢复审批
	ApprovalTypeTaskDelete       ApprovalType = "task_delete"      // 任务删除审批
	ApprovalTypeConfigChange     ApprovalType = "config_change"   // 配置变更审批
)

// ApprovalStatus 审批状态
type ApprovalStatus string

const (
	ApprovalStatusPending    ApprovalStatus = "pending"     // 待审批
	ApprovalStatusApproved  ApprovalStatus = "approved"   // 已通过
	ApprovalStatusRejected  ApprovalStatus = "rejected"    // 已拒绝
	ApprovalStatusCancelled ApprovalStatus = "cancelled"   // 已取消
	ApprovalStatusExecuted  ApprovalStatus = "executed"   // 已执行
)

// Approval 审批记录表
type Approval struct {
	ID          int64         `json:"id" gorm:"primaryKey;autoIncrement"`
	UUID       string       `json:"uuid" gorm:"size:36;uniqueIndex;not null"` // 唯一标识
	Type       ApprovalType  `json:"type" gorm:"size:20;not null;index"`     // 审批类型
	Status     ApprovalStatus `json:"status" gorm:"size:20;default:'pending';index"` // 审批状态

	// 申请人信息
	ApplicantID   int64  `json:"applicant_id" gorm:"not null;index"`
	ApplicantName string `json:"applicant_name" gorm:"size:100"`

	// 审批人信息
	ApproverID   *int64 `json:"approver_id" gorm:"index"`
	ApproverName string `json:"approver_name" gorm:"size:100"`

	// 审批内容（JSON）
	ResourceType string `json:"resource_type" gorm:"size:50"`       // 资源类型：task, restore, config
	ResourceID   int64  `json:"resource_id" gorm:"index"`          // 资源 ID
	ResourceName string `json:"resource_name" gorm:"size:200"`       // 资源名称（冗余）

	// 申请详情
	Title    string `json:"title" gorm:"size:200;not null"`     // 申请标题
	Content  string `json:"content" gorm:"type:text"`            // 申请说明
	Details  string `json:"-" gorm:"type:json"`                  // 详细配置（JSON）
	DetailsMap map[string]interface{} `json:"details" gorm:"-"`   // 运行时解析

	// 审批意见
	ApproveNote string `json:"approve_note" gorm:"size:500"`     // 审批意见
	RejectReason string `json:"reject_reason" gorm:"size:500"`  // 拒绝原因

	// 时间戳
	AppliedAt   time.Time  `json:"applied_at" gorm:"autoCreateTime"`   // 申请时间
	ApprovedAt  *time.Time `json:"approved_at"`                       // 审批时间
	ExecutedAt   *time.Time `json:"executed_at"`                       // 执行时间
	CancelledAt  *time.Time `json:"cancelled_at"`                      // 取消时间

	// 过期时间（审批超时）
	ExpiredAt    *time.Time `json:"expired_at"`                       // 过期时间

	// 审计字段
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 软删除
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Approval) TableName() string {
	return "approvals"
}

// GetDetails 解析详情 JSON
func (a *Approval) GetDetails() (map[string]interface{}, error) {
	if len(a.DetailsMap) > 0 {
		return a.DetailsMap, nil
	}
	if a.Details == "" {
		return make(map[string]interface{}), nil
	}
	if err := json.Unmarshal([]byte(a.Details), &a.DetailsMap); err != nil {
		return nil, err
	}
	return a.DetailsMap, nil
}

// SetDetails 设置详情 JSON
func (a *Approval) SetDetails(details map[string]interface{}) error {
	data, err := json.Marshal(details)
	if err != nil {
		return err
	}
	a.Details = string(data)
	a.DetailsMap = details
	return nil
}

// ApprovalResponse 审批响应结构
type ApprovalResponse struct {
	ID            int64          `json:"id"`
	UUID          string         `json:"uuid"`
	Type          ApprovalType   `json:"type"`
	Status        ApprovalStatus `json:"status"`
	ApplicantID   int64         `json:"applicant_id"`
	ApplicantName string        `json:"applicant_name"`
	ApproverID    *int64        `json:"approver_id,omitempty"`
	ApproverName  string        `json:"approver_name,omitempty"`
	ResourceType  string        `json:"resource_type"`
	ResourceID    int64         `json:"resource_id"`
	ResourceName  string        `json:"resource_name"`
	Title         string        `json:"title"`
	Content       string        `json:"content,omitempty"`
	Details       map[string]interface{} `json:"details,omitempty"`
	ApproveNote   string        `json:"approve_note,omitempty"`
	RejectReason  string        `json:"reject_reason,omitempty"`
	AppliedAt     time.Time     `json:"applied_at"`
	ApprovedAt    *time.Time    `json:"approved_at,omitempty"`
	ExecutedAt    *time.Time    `json:"executed_at,omitempty"`
	CancelledAt   *time.Time    `json:"cancelled_at,omitempty"`
	ExpiredAt      *time.Time    `json:"expired_at,omitempty"`
}

// ToResponse 转换为响应结构
func (a *Approval) ToResponse() (*ApprovalResponse, error) {
	details, _ := a.GetDetails()
	return &ApprovalResponse{
		ID:           a.ID,
		UUID:         a.UUID,
		Type:         a.Type,
		Status:       a.Status,
		ApplicantID:   a.ApplicantID,
		ApplicantName: a.ApplicantName,
		ApproverID:   a.ApproverID,
		ApproverName: a.ApproverName,
		ResourceType: a.ResourceType,
		ResourceID:    a.ResourceID,
		ResourceName:  a.ResourceName,
		Title:        a.Title,
		Content:      a.Content,
		Details:      details,
		ApproveNote:  a.ApproveNote,
		RejectReason: a.RejectReason,
		AppliedAt:    a.AppliedAt,
		ApprovedAt:   a.ApprovedAt,
		ExecutedAt:   a.ExecutedAt,
		CancelledAt:  a.CancelledAt,
		ExpiredAt:    a.ExpiredAt,
	}, nil
}

// CreateApprovalRequest 创建审批请求
type CreateApprovalRequest struct {
	Type         ApprovalType              `json:"type" binding:"required,oneof=restore task_delete config_change"`
	ApproverID   *int64                    `json:"approver_id"`
	ResourceType string                    `json:"resource_type" binding:"required"`
	ResourceID   int64                     `json:"resource_id" binding:"required"`
	ResourceName string                    `json:"resource_name" binding:"required"`
	Title        string                    `json:"title" binding:"required,max=200"`
	Content      string                    `json:"content" binding:"max=1000"`
	Details      map[string]interface{}    `json:"details"`
}

// ListApprovalsRequest 列出审批请求
type ListApprovalsRequest struct {
	Type     ApprovalType   `form:"type"`
	Status   ApprovalStatus `form:"status"`
	Role     string        `form:"role"` // applicant/approver/all
	Page     int           `form:"page,default=1"`
	PageSize int           `form:"page_size,default=20"`
}

// ApproveRequest 审批通过请求
type ApproveRequest struct {
	Note string `json:"note" binding:"max=500"`
}

// RejectRequest 审批拒绝请求
type RejectRequest struct {
	Reason string `json:"reason" binding:"required,max=500"`
}
