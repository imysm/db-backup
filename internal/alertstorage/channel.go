package alertstorage

import (
	"github.com/imysm/db-backup/internal/alertmodel"
	"errors"
	"time"

	"gorm.io/gorm"
)

// ChannelStorage 通知渠道存储访问
type ChannelStorage struct {
	db *gorm.DB
}

// NewChannelStorage 创建渠道存储访问
func NewChannelStorage(db *gorm.DB) *ChannelStorage {
	return &ChannelStorage{db: db}
}

// Create 创建渠道
func (s *ChannelStorage) Create(channel *alertmodel.NotificationChannel) error {
	now := time.Now()
	channel.CreatedAt = now
	channel.UpdatedAt = now
	return s.db.Create(channel).Error
}

// GetByID 根据 ID 获取渠道
func (s *ChannelStorage) GetByID(id int64) (*alertmodel.NotificationChannel, error) {
	var channel alertmodel.NotificationChannel
	err := s.db.Where("id = ? AND deleted_at IS NULL", id).First(&channel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &channel, nil
}

// List 列出渠道
func (s *ChannelStorage) List(channelType *alertmodel.ChannelType, enabled *bool, page, pageSize int) ([]alertmodel.NotificationChannel, int64, error) {
	var channels []alertmodel.NotificationChannel
	var total int64

	query := s.db.Model(&alertmodel.NotificationChannel{}).Where("deleted_at IS NULL")

	if channelType != nil {
		query = query.Where("type = ?", *channelType)
	}
	if enabled != nil {
		query = query.Where("enabled = ?", *enabled)
	}

	// 统计
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Order("priority DESC, id ASC").Offset(offset).Limit(pageSize).Find(&channels).Error; err != nil {
		return nil, 0, err
	}

	return channels, total, nil
}

// ListEnabled 获取所有启用的渠道（按优先级排序）
func (s *ChannelStorage) ListEnabled() ([]alertmodel.NotificationChannel, error) {
	var channels []alertmodel.NotificationChannel
	err := s.db.Where("enabled = ? AND deleted_at IS NULL", true).
		Order("priority DESC, id ASC").
		Find(&channels).Error
	return channels, err
}

// GetByIDs 根据 ID 列表获取渠道
func (s *ChannelStorage) GetByIDs(ids []int64) ([]alertmodel.NotificationChannel, error) {
	var channels []alertmodel.NotificationChannel
	err := s.db.Where("id IN ? AND deleted_at IS NULL", ids).
		Order("priority DESC").
		Find(&channels).Error
	return channels, err
}

// Update 更新渠道
func (s *ChannelStorage) Update(channel *alertmodel.NotificationChannel) error {
	channel.UpdatedAt = time.Now()
	return s.db.Save(channel).Error
}

// Delete 删除渠道（软删除）
func (s *ChannelStorage) Delete(id int64) error {
	now := time.Now()
	return s.db.Model(&alertmodel.NotificationChannel{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"deleted_at": &now,
			"updated_at": now,
		}).Error
}

// UpdateHealthStatus 更新健康状态
func (s *ChannelStorage) UpdateHealthStatus(id int64, status alertmodel.HealthStatus, lastError string) error {
	updates := map[string]interface{}{
		"health_status": status,
		"updated_at":    time.Now(),
	}
	if lastError != "" {
		updates["last_error"] = lastError
	}
	return s.db.Model(&alertmodel.NotificationChannel{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// UpdateLastSent 更新最后发送时间
func (s *ChannelStorage) UpdateLastSent(id int64) error {
	now := time.Now()
	return s.db.Model(&alertmodel.NotificationChannel{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_sent_at": now,
			"updated_at":   now,
			"send_count":   gorm.Expr("send_count + 1"),
		}).Error
}

// IncrementFailedCount 增加失败计数
func (s *ChannelStorage) IncrementFailedCount(id int64, lastError string) error {
	return s.db.Model(&alertmodel.NotificationChannel{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"failed_count": gorm.Expr("failed_count + 1"),
			"last_error":   lastError,
			"health_status": alertmodel.HealthStatusUnhealthy,
			"updated_at":   time.Now(),
		}).Error
}

// GetByType 根据类型获取单个启用的渠道
func (s *ChannelStorage) GetByType(channelType alertmodel.ChannelType) (*alertmodel.NotificationChannel, error) {
	var channel alertmodel.NotificationChannel
	err := s.db.Where("type = ? AND enabled = ? AND deleted_at IS NULL", channelType, true).
		Order("priority DESC").
		First(&channel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &channel, nil
}
