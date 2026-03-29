package audit

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AuditService 审计日志服务
type AuditService struct {
	db *gorm.DB
}

// NewAuditService 创建审计服务
func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{db: db}
}

// Log 记录审计日志
func (s *AuditService) Log(log *AuditLog) error {
	return s.db.Create(log).Error
}

// LogWithDetails 记录带详情的审计日志
func (s *AuditService) LogWithDetails(userID *uint, username string, tenantID *uint, action, resource string, resourceID *uint, resourceName string, details *AuditDetails, result, errorMsg string) error {
	log := &AuditLog{
		UserID:       userID,
		Username:     username,
		TenantID:     tenantID,
		Action:       action,
		Resource:     resource,
		ResourceID:   resourceID,
		ResourceName: resourceName,
		Details:      details.ToJSON(),
		Result:       result,
		ErrorMessage: errorMsg,
	}
	return s.db.Create(log).Error
}

// Query 查询审计日志
func (s *AuditService) Query(filter *AuditQuery) ([]AuditLog, int64, error) {
	var logs []AuditLog
	var total int64

	query := s.db.Model(&AuditLog{})

	// 应用过滤条件
	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	if filter.TenantID != nil {
		query = query.Where("tenant_id = ?", *filter.TenantID)
	}
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}
	if filter.Resource != "" {
		query = query.Where("resource = ?", filter.Resource)
	}
	if filter.ResourceID != nil {
		query = query.Where("resource_id = ?", *filter.ResourceID)
	}
	if filter.Result != "" {
		query = query.Where("result = ?", filter.Result)
	}
	if filter.StartTime != nil {
		query = query.Where("created_at >= ?", *filter.StartTime)
	}
	if filter.EndTime != nil {
		query = query.Where("created_at <= ?", *filter.EndTime)
	}
	if filter.Keyword != "" {
		query = query.Where("resource_name LIKE ? OR username LIKE ? OR details LIKE ?",
			"%"+filter.Keyword+"%", "%"+filter.Keyword+"%", "%"+filter.Keyword+"%")
	}
	if filter.RequestID != "" {
		query = query.Where("request_id = ?", filter.RequestID)
	}

	// 统计总数
	query.Count(&total)

	// 分页
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	} else {
		query = query.Limit(50) // 默认50条
	}

	// 排序
	orderBy := "created_at DESC"
	if filter.OrderBy != "" {
		orderBy = filter.OrderBy
		if filter.OrderDesc {
			orderBy += " DESC"
		} else {
			orderBy += " ASC"
		}
	}
	query = query.Order(clause.OrderByColumn{Column: clause.Column{Name: parseOrderColumn(orderBy)}, Desc: filter.OrderDesc})

	if err := query.Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetByID 根据ID获取审计日志
func (s *AuditService) GetByID(id uint) (*AuditLog, error) {
	var log AuditLog
	if err := s.db.First(&log, id).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

// GetByRequestID 根据请求ID获取审计日志
func (s *AuditService) GetByRequestID(requestID string) ([]AuditLog, error) {
	var logs []AuditLog
	if err := s.db.Where("request_id = ?", requestID).Order("created_at ASC").Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

// GetByResource 获取资源的审计历史
func (s *AuditService) GetByResource(resource string, resourceID uint, limit int) ([]AuditLog, error) {
	var logs []AuditLog
	query := s.db.Where("resource = ? AND resource_id = ?", resource, resourceID).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

// GetUserActivity 获取用户活动统计
func (s *AuditService) GetUserActivity(userID uint, startTime, endTime time.Time) (*UserActivity, error) {
	var activity UserActivity

	// 统计操作次数
	type Result struct {
		Action  string
		Count   int64
	}
	var results []Result

	if err := s.db.Model(&AuditLog{}).
		Select("action, COUNT(*) as count").
		Where("user_id = ? AND created_at >= ? AND created_at <= ?", userID, startTime, endTime).
		Group("action").
		Find(&results).Error; err != nil {
		return nil, err
	}

	activity.Actions = make(map[string]int64)
	for _, r := range results {
		activity.Actions[r.Action] = r.Count
		activity.Total++
	}

	// 统计成功/失败
	s.db.Model(&AuditLog{}).
		Where("user_id = ? AND created_at >= ? AND created_at <= ?", userID, startTime, endTime).
		Count(&activity.Total)

	var successCount, failureCount int64
	s.db.Model(&AuditLog{}).
		Where("user_id = ? AND created_at >= ? AND created_at <= ? AND result = ?", userID, startTime, endTime, ResultSuccess).
		Count(&successCount)
	s.db.Model(&AuditLog{}).
		Where("user_id = ? AND created_at >= ? AND created_at <= ? AND result = ?", userID, startTime, endTime, ResultFailure).
		Count(&failureCount)

	activity.SuccessCount = successCount
	activity.FailureCount = failureCount

	return &activity, nil
}

// CleanOldLogs 清理旧日志
func (s *AuditService) CleanOldLogs(days int) (int64, error) {
	before := time.Now().AddDate(0, 0, -days)
	result := s.db.Where("created_at < ?", before).Delete(&AuditLog{})
	return result.RowsAffected, result.Error
}

// AuditQuery 审计查询条件
type AuditQuery struct {
	UserID     *uint
	TenantID   *uint
	Action     string
	Resource   string
	ResourceID *uint
	Result     string
	StartTime  *time.Time
	EndTime    *time.Time
	Keyword    string
	RequestID  string
	Offset     int
	Limit      int
	OrderBy    string
	OrderDesc  bool
}

// UserActivity 用户活动统计
type UserActivity struct {
	Total        int64           `json:"total"`
	SuccessCount int64           `json:"success_count"`
	FailureCount int64           `json:"failure_count"`
	Actions      map[string]int64 `json:"actions"`
}

// parseOrderColumn 解析排序列
func parseOrderColumn(orderBy string) string {
	// 简单的列名提取
	parts := splitPath(orderBy)
	if len(parts) > 0 {
		return parts[0]
	}
	return "created_at"
}
