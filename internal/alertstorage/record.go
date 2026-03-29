package alertstorage

import (
	"github.com/imysm/db-backup/internal/alertmodel"
	"errors"
	"time"

	"gorm.io/gorm"
)

// RecordStorage 告警记录存储访问
type RecordStorage struct {
	db *gorm.DB
}

// NewRecordStorage 创建记录存储访问
func NewRecordStorage(db *gorm.DB) *RecordStorage {
	return &RecordStorage{db: db}
}

// Create 创建告警记录
func (s *RecordStorage) Create(record *alertmodel.AlertRecord) error {
	now := time.Now()
	record.CreatedAt = now
	record.UpdatedAt = now
	return s.db.Create(record).Error
}

// GetByID 根据 ID 获取告警记录
func (s *RecordStorage) GetByID(id int64) (*alertmodel.AlertRecord, error) {
	var record alertmodel.AlertRecord
	err := s.db.Preload("Rule").First(&record, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &record, nil
}

// GetByIDWithDetails 获取告警记录（包含详情）
func (s *RecordStorage) GetByIDWithDetails(id int64) (*alertmodel.AlertRecord, error) {
	var record alertmodel.AlertRecord
	err := s.db.Preload("Rule").
		Preload("NotificationRecords", func(db *gorm.DB) *gorm.DB {
			return db.Preload("Channel")
		}).
		Preload("Notes").
		First(&record, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &record, nil
}

// List 列出告警记录
func (s *RecordStorage) List(req *alertmodel.ListAlertsRequest) ([]alertmodel.AlertRecord, int64, error) {
	var records []alertmodel.AlertRecord
	var total int64

	query := s.db.Model(&alertmodel.AlertRecord{})

	if req.Level != "" {
		query = query.Where("level = ?", req.Level)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.RuleID != nil {
		query = query.Where("rule_id = ?", *req.RuleID)
	}
	if req.TaskID != nil {
		query = query.Where("task_id = ?", *req.TaskID)
	}
	if req.StartTime != nil {
		query = query.Where("triggered_at >= ?", *req.StartTime)
	}
	if req.EndTime != nil {
		query = query.Where("triggered_at <= ?", *req.EndTime)
	}

	// 统计
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	if err := query.Preload("Rule").
		Order("triggered_at DESC, id DESC").
		Offset(offset).
		Limit(req.PageSize).
		Find(&records).Error; err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

// ListRecent 获取最近的告警记录
func (s *RecordStorage) ListRecent(status *alertmodel.AlertStatus, limit int) ([]alertmodel.AlertRecord, error) {
	var records []alertmodel.AlertRecord
	query := s.db.Model(&alertmodel.AlertRecord{})

	if status != nil {
		query = query.Where("status = ?", *status)
	}

	err := query.Preload("Rule").
		Order("triggered_at DESC").
		Limit(limit).
		Find(&records).Error
	return records, err
}

// CountByStatus 按状态统计
func (s *RecordStorage) CountByStatus() (map[alertmodel.AlertStatus]int64, error) {
	type Result struct {
		Status alertmodel.AlertStatus `json:"status"`
		Count  int64                 `json:"count"`
	}
	var results []Result
	err := s.db.Model(&alertmodel.AlertRecord{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Find(&results).Error
	if err != nil {
		return nil, err
	}

	counts := make(map[alertmodel.AlertStatus]int64)
	for _, r := range results {
		counts[r.Status] = r.Count
	}
	return counts, nil
}

// CountActiveByLevel 按级别统计活跃告警
func (s *RecordStorage) CountActiveByLevel() (map[alertmodel.AlertLevel]int64, error) {
	type Result struct {
		Level alertmodel.AlertLevel `json:"level"`
		Count int64                 `json:"count"`
	}
	var results []Result
	err := s.db.Model(&alertmodel.AlertRecord{}).
		Select("level, COUNT(*) as count").
		Where("status = ?", alertmodel.AlertStatusActive).
		Group("level").
		Find(&results).Error
	if err != nil {
		return nil, err
	}

	counts := make(map[alertmodel.AlertLevel]int64)
	for _, r := range results {
		counts[r.Level] = r.Count
	}
	return counts, nil
}

// CountActiveTotal 统计活跃告警总数
func (s *RecordStorage) CountActiveTotal() (int64, error) {
	var count int64
	err := s.db.Model(&alertmodel.AlertRecord{}).
		Where("status = ?", alertmodel.AlertStatusActive).
		Count(&count).Error
	return count, err
}

// Acknowledge 确认告警
func (s *RecordStorage) Acknowledge(id int64, acknowledgedBy, note string) error {
	updates := map[string]interface{}{
		"status":           alertmodel.AlertStatusAcknowledged,
		"acknowledged_at":   time.Now(),
		"acknowledged_by":  acknowledgedBy,
		"updated_at":       time.Now(),
	}
	if note != "" {
		updates["ack_note"] = note
	}
	return s.db.Model(&alertmodel.AlertRecord{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// Resolve 解决告警
func (s *RecordStorage) Resolve(id int64, resolvedBy, note string) error {
	updates := map[string]interface{}{
		"status":        alertmodel.AlertStatusResolved,
		"resolved_at":   time.Now(),
		"resolved_by":   resolvedBy,
		"updated_at":    time.Now(),
	}
	if note != "" {
		updates["resolve_note"] = note
	}
	return s.db.Model(&alertmodel.AlertRecord{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// Escalate 升级告警
func (s *RecordStorage) Escalate(id int64, newLevel alertmodel.AlertLevel, reason string) error {
	return s.db.Model(&alertmodel.AlertRecord{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":              alertmodel.AlertStatusEscalated,
			"escalated_at":        time.Now(),
			"escalated_to_level":  newLevel,
			"escalated_reason":    reason,
			"updated_at":          time.Now(),
		}).Error
}

// CreateNotificationRecord 创建通知发送记录
func (s *RecordStorage) CreateNotificationRecord(record *alertmodel.AlertNotificationRecord) error {
	now := time.Now()
	record.CreatedAt = now
	record.UpdatedAt = now
	return s.db.Create(record).Error
}

// UpdateNotificationRecord 更新通知发送记录
func (s *RecordStorage) UpdateNotificationRecord(record *alertmodel.AlertNotificationRecord) error {
	record.UpdatedAt = time.Now()
	return s.db.Save(record).Error
}

// GetNotificationRecords 获取通知发送记录列表
func (s *RecordStorage) GetNotificationRecords(alertID int64) ([]alertmodel.AlertNotificationRecord, error) {
	var records []alertmodel.AlertNotificationRecord
	err := s.db.Preload("Channel").
		Where("alert_id = ?", alertID).
		Order("id ASC").
		Find(&records).Error
	return records, err
}

// AddNote 添加备注
func (s *RecordStorage) AddNote(alertID int64, content, createdBy string) (*alertmodel.AlertNote, error) {
	note := &alertmodel.AlertNote{
		AlertID:   alertID,
		Content:   content,
		CreatedBy: createdBy,
		CreatedAt: time.Now(),
	}
	if err := s.db.Create(note).Error; err != nil {
		return nil, err
	}
	return note, nil
}

// GetNotes 获取备注列表
func (s *RecordStorage) GetNotes(alertID int64) ([]alertmodel.AlertNote, error) {
	var notes []alertmodel.AlertNote
	err := s.db.Where("alert_id = ?", alertID).
		Order("created_at ASC").
		Find(&notes).Error
	return notes, err
}

// DeleteOldRecords 删除旧记录（数据清理）
func (s *RecordStorage) DeleteOldRecords(before time.Time) (int64, error) {
	result := s.db.Where("triggered_at < ?", before).Delete(&alertmodel.AlertRecord{})
	return result.RowsAffected, result.Error
}

// GetAlertCountByLevel 获取按级别统计的告警数量
func (s *RecordStorage) GetAlertCountByLevel(startTime, endTime *time.Time) (map[alertmodel.AlertLevel]int64, error) {
	type Result struct {
		Level alertmodel.AlertLevel `json:"level"`
		Count int64                 `json:"count"`
	}
	var results []Result

	query := s.db.Model(&alertmodel.AlertRecord{}).
		Select("level, COUNT(*) as count")

	if startTime != nil {
		query = query.Where("triggered_at >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("triggered_at <= ?", *endTime)
	}

	err := query.Group("level").Find(&results).Error
	if err != nil {
		return nil, err
	}

	counts := make(map[alertmodel.AlertLevel]int64)
	for _, r := range results {
		counts[r.Level] = r.Count
	}
	return counts, nil
}
