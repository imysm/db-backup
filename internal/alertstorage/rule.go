package alertstorage

import (
	"github.com/imysm/db-backup/internal/alertmodel"
	"errors"
	"time"

	"gorm.io/gorm"
)

// RuleStorage 告警规则存储访问
type RuleStorage struct {
	db *gorm.DB
}

// NewRuleStorage 创建规则存储访问
func NewRuleStorage(db *gorm.DB) *RuleStorage {
	return &RuleStorage{db: db}
}

// Create 创建规则
func (s *RuleStorage) Create(rule *alertmodel.AlertRule) error {
	now := time.Now()
	rule.CreatedAt = now
	rule.UpdatedAt = now
	return s.db.Create(rule).Error
}

// GetByID 根据 ID 获取规则
func (s *RuleStorage) GetByID(id int64) (*alertmodel.AlertRule, error) {
	var rule alertmodel.AlertRule
	err := s.db.Where("id = ? AND deleted_at IS NULL", id).First(&rule).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rule, nil
}

// List 列出规则
func (s *RuleStorage) List(enabled *bool, level *alertmodel.AlertLevel, page, pageSize int) ([]alertmodel.AlertRule, int64, error) {
	var rules []alertmodel.AlertRule
	var total int64

	query := s.db.Model(&alertmodel.AlertRule{}).Where("deleted_at IS NULL")

	if enabled != nil {
		query = query.Where("enabled = ?", *enabled)
	}
	if level != nil {
		query = query.Where("level = ?", *level)
	}

	// 统计
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Order("priority DESC, id ASC").Offset(offset).Limit(pageSize).Find(&rules).Error; err != nil {
		return nil, 0, err
	}

	return rules, total, nil
}

// ListEnabled 获取所有启用的规则（按优先级排序）
func (s *RuleStorage) ListEnabled() ([]alertmodel.AlertRule, error) {
	var rules []alertmodel.AlertRule
	err := s.db.Where("enabled = ? AND deleted_at IS NULL", true).
		Order("priority DESC, id ASC").
		Find(&rules).Error
	return rules, err
}

// GetByIDs 根据 ID 列表获取规则
func (s *RuleStorage) GetByIDs(ids []int64) ([]alertmodel.AlertRule, error) {
	var rules []alertmodel.AlertRule
	err := s.db.Where("id IN ? AND deleted_at IS NULL", ids).
		Order("priority DESC").
		Find(&rules).Error
	return rules, err
}

// Update 更新规则
func (s *RuleStorage) Update(rule *alertmodel.AlertRule) error {
	rule.UpdatedAt = time.Now()
	return s.db.Save(rule).Error
}

// Delete 删除规则（软删除）
func (s *RuleStorage) Delete(id int64) error {
	now := time.Now()
	return s.db.Model(&alertmodel.AlertRule{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"deleted_at": &now,
			"updated_at": now,
		}).Error
}

// IncrementMatchedCount 增加匹配计数
func (s *RuleStorage) IncrementMatchedCount(id int64) error {
	now := time.Now()
	return s.db.Model(&alertmodel.AlertRule{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"matched_count":    gorm.Expr("matched_count + 1"),
			"last_matched_at":  now,
			"updated_at":       now,
		}).Error
}

// GetStats 获取规则统计
func (s *RuleStorage) GetStats(ruleID int64, startDate, endDate string) (*alertmodel.AlertRuleStats, error) {
	var stats alertmodel.AlertRuleStats
	err := s.db.Where("rule_id = ? AND date >= ? AND date <= ?", ruleID, startDate, endDate).
		First(&stats).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &stats, nil
}

// UpsertStats 更新或创建规则统计
func (s *RuleStorage) UpsertStats(stats *alertmodel.AlertRuleStats) error {
	stats.UpdatedAt = time.Now()
	return s.db.Save(stats).Error
}

// Copy 复制规则
func (s *RuleStorage) Copy(id int64, newName, createdBy string) (*alertmodel.AlertRule, error) {
	original, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	if original == nil {
		return nil, nil
	}

	// 创建副本
	now := time.Now()
	copy := &alertmodel.AlertRule{
		Name:         newName,
		Description:  original.Description,
		Enabled:      false, // 副本默认禁用
		Priority:     original.Priority,
		Level:        original.Level,
		ConditionOp:  original.ConditionOp,
		Conditions:   original.Conditions,
		Channels:     original.Channels,
		Cooldown:     original.Cooldown,
		SilentPeriods: original.SilentPeriods,
		EscalateEnabled: original.EscalateEnabled,
		EscalateTimeout: original.EscalateTimeout,
		EscalateLevel:   original.EscalateLevel,
		EscalateChannels: original.EscalateChannels,
		CreatedBy:    createdBy,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.db.Create(copy).Error; err != nil {
		return nil, err
	}

	return copy, nil
}

// Export 导出规则
func (s *RuleStorage) Export(ids []int64) ([]alertmodel.AlertRule, error) {
	var rules []alertmodel.AlertRule
	query := s.db.Where("deleted_at IS NULL")
	if len(ids) > 0 {
		query = query.Where("id IN ?", ids)
	}
	err := query.Order("id ASC").Find(&rules).Error
	return rules, err
}

// Import 导入规则
func (s *RuleStorage) Import(rules []alertmodel.AlertRule, createdBy string) (int, int, []string, error) {
	imported := 0
	skipped := 0
	var errors []string

	for _, rule := range rules {
		// 检查同名规则是否已存在
		var existing alertmodel.AlertRule
		err := s.db.Where("name = ? AND deleted_at IS NULL", rule.Name).First(&existing).Error
		if err == nil && existing.ID > 0 {
			skipped++
			errors = append(errors, "规则已存在: "+rule.Name)
			continue
		}

		// 创建新规则
		now := time.Now()
		rule.ID = 0
		rule.Name = rule.Name + " (导入)"
		rule.Enabled = false
		rule.CreatedBy = createdBy
		rule.CreatedAt = now
		rule.UpdatedAt = now

		if err := s.db.Create(&rule).Error; err != nil {
			errors = append(errors, "导入失败: "+rule.Name+", "+err.Error())
			continue
		}
		imported++
	}

	return imported, skipped, errors, nil
}
