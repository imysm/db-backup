package approval

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Storage 审批存储访问
type Storage struct {
	db *gorm.DB
}

// NewStorage 创建审批存储访问
func NewStorage(db *gorm.DB) *Storage {
	return &Storage{db: db}
}

// Create 创建审批记录
func (s *Storage) Create(approval *Approval) error {
	now := time.Now()
	approval.UUID = uuid.New().String()
	approval.Status = ApprovalStatusPending
	approval.AppliedAt = now
	approval.CreatedAt = now
	approval.UpdatedAt = now
	return s.db.Create(approval).Error
}

// GetByID 根据 ID 获取审批记录
func (s *Storage) GetByID(id int64) (*Approval, error) {
	var approval Approval
	err := s.db.Where("id = ? AND deleted_at IS NULL", id).First(&approval).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &approval, nil
}

// GetByUUID 根据 UUID 获取审批记录
func (s *Storage) GetByUUID(uuid string) (*Approval, error) {
	var approval Approval
	err := s.db.Where("uuid = ? AND deleted_at IS NULL", uuid).First(&approval).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &approval, nil
}

// List 列出审批记录
func (s *Storage) List(req *ListApprovalsRequest, userID int64, role string) ([]Approval, int64, error) {
	var approvals []Approval
	var total int64

	query := s.db.Model(&Approval{}).Where("deleted_at IS NULL")

	// 按类型过滤
	if req.Type != "" {
		query = query.Where("type = ?", req.Type)
	}

	// 按状态过滤
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	// 按角色过滤
	switch role {
	case "applicant":
		query = query.Where("applicant_id = ?", userID)
	case "approver":
		query = query.Where("approver_id = ? OR approver_id IS NULL", userID)
	// "all" 不过滤
	}

	// 统计
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	if err := query.Order("applied_at DESC").Offset(offset).Limit(req.PageSize).Find(&approvals).Error; err != nil {
		return nil, 0, err
	}

	return approvals, total, nil
}

// ListPending 获取待审批列表（供审批人查看）
func (s *Storage) ListPending(approverID int64, page, pageSize int) ([]Approval, int64, error) {
	var approvals []Approval
	var total int64

	query := s.db.Model(&Approval{}).
		Where("status = ? AND deleted_at IS NULL", ApprovalStatusPending).
		Where("approver_id = ? OR approver_id IS NULL", approverID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("applied_at ASC").Offset(offset).Limit(pageSize).Find(&approvals).Error; err != nil {
		return nil, 0, err
	}

	return approvals, total, nil
}

// ListByApplicant 获取申请人发起的所有审批
func (s *Storage) ListByApplicant(applicantID int64, page, pageSize int) ([]Approval, int64, error) {
	var approvals []Approval
	var total int64

	query := s.db.Model(&Approval{}).Where("applicant_id = ? AND deleted_at IS NULL", applicantID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("applied_at DESC").Offset(offset).Limit(pageSize).Find(&approvals).Error; err != nil {
		return nil, 0, err
	}

	return approvals, total, nil
}

// Update 更新审批记录
func (s *Storage) Update(approval *Approval) error {
	approval.UpdatedAt = time.Now()
	return s.db.Save(approval).Error
}

// Approve 审批通过
func (s *Storage) Approve(id int64, approverID int64, approverName, note string) error {
	now := time.Now()
	return s.db.Model(&Approval{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       ApprovalStatusApproved,
			"approver_id":   approverID,
			"approver_name": approverName,
			"approve_note":  note,
			"approved_at":   now,
			"updated_at":    now,
		}).Error
}

// Reject 审批拒绝
func (s *Storage) Reject(id int64, approverID int64, approverName, reason string) error {
	now := time.Now()
	return s.db.Model(&Approval{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":        ApprovalStatusRejected,
			"approver_id":   approverID,
			"approver_name": approverName,
			"reject_reason": reason,
			"updated_at":    now,
		}).Error
}

// Cancel 取消审批
func (s *Storage) Cancel(id int64, userID int64) error {
	now := time.Now()
	return s.db.Model(&Approval{}).
		Where("id = ? AND applicant_id = ? AND status = ?", id, userID, ApprovalStatusPending).
		Updates(map[string]interface{}{
			"status":       ApprovalStatusCancelled,
			"cancelled_at": now,
			"updated_at":    now,
		}).Error
}

// MarkExecuted 标记为已执行
func (s *Storage) MarkExecuted(id int64) error {
	now := time.Now()
	return s.db.Model(&Approval{}).
		Where("id = ? AND status = ?", id, ApprovalStatusApproved).
		Updates(map[string]interface{}{
			"status":      ApprovalStatusExecuted,
			"executed_at": now,
			"updated_at":  now,
		}).Error
}

// CountPending 统计待审批数量
func (s *Storage) CountPending(approverID int64) (int64, error) {
	var count int64
	err := s.db.Model(&Approval{}).
		Where("status = ? AND deleted_at IS NULL", ApprovalStatusPending).
		Where("approver_id = ? OR approver_id IS NULL", approverID).
		Count(&count).Error
	return count, err
}

// CountByStatus 按状态统计
func (s *Storage) CountByStatus(userID int64, role string) (map[ApprovalStatus]int64, error) {
	type Result struct {
		Status ApprovalStatus `json:"status"`
		Count  int64        `json:"count"`
	}
	var results []Result

	query := s.db.Model(&Approval{}).Where("deleted_at IS NULL")

	switch role {
	case "applicant":
		query = query.Where("applicant_id = ?", userID)
	case "approver":
		query = query.Where("approver_id = ? OR approver_id IS NULL", userID)
	}

	err := query.Select("status, COUNT(*) as count").Group("status").Find(&results).Error
	if err != nil {
		return nil, err
	}

	counts := make(map[ApprovalStatus]int64)
	for _, r := range results {
		counts[r.Status] = r.Count
	}
	return counts, nil
}
