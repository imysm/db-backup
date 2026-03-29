package quota

import (
	"github.com/imysm/db-backup/internal/api/model"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
)

// StorageQuota 存储配额
type StorageQuota struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name     string    `json:"name" gorm:"size:100;not null"`

	// 配额范围
	ScopeType string `json:"scope_type" gorm:"size:20;not null"` // global / storage_bucket / job
	ScopeID   string `json:"scope_id" gorm:"size:100"`           // bucket_name 或 job_id

	// 配额限制
	QuotaBytes  int64 `json:"quota_bytes" gorm:"not null"`  // 配额上限（字节）
	WarnPercent int   `json:"warn_percent" gorm:"default:80"` // 告警百分比

	// 当前使用量（缓存）
	UsedBytes int64 `json:"used_bytes" gorm:"-"`

	// 是否启用
	Enabled bool `json:"enabled" gorm:"default:true"`

	// 审计字段
	CreatedBy string    `json:"created_by" gorm:"size:100"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (StorageQuota) TableName() string {
	return "storage_quotas"
}

// QuotaStatus 配额状态
type QuotaStatus struct {
	QuotaID     uint    `json:"quota_id"`
	ScopeType  string `json:"scope_type"`
	ScopeID    string `json:"scope_id"`
	UsedBytes  int64  `json:"used_bytes"`
	QuotaBytes int64  `json:"quota_bytes"`
	WarnPercent int    `json:"warn_percent"`
	UsagePercent float64 `json:"usage_percent"`
	Status     string `json:"status"` // normal / warning / exceeded / unlimited
}

// IsNormal 是否正常
func (s *QuotaStatus) IsNormal() bool {
	return s.Status == "normal"
}

// IsWarning 是否告警
func (s *QuotaStatus) IsWarning() bool {
	return s.Status == "warning"
}

// IsExceeded 是否超出
func (s *QuotaStatus) IsExceeded() bool {
	return s.Status == "exceeded"
}

// Storage 配额存储访问
type Storage struct {
	db *gorm.DB
}

// NewStorage 创建存储访问
func NewStorage(db *gorm.DB) *Storage {
	return &Storage{db: db}
}

// Create 创建配额
func (s *Storage) Create(quota *StorageQuota) error {
	return s.db.Create(quota).Error
}

// GetByID 获取配额
func (s *Storage) GetByID(id uint) (*StorageQuota, error) {
	var quota StorageQuota
	err := s.db.First(&quota, id).Error
	if err != nil {
		return nil, err
	}
	return &quota, nil
}

// List 列出配额
func (s *Storage) List(scopeType, scopeID string, enabled *bool) ([]StorageQuota, error) {
	var quotas []StorageQuota
	query := s.db.Model(&StorageQuota{})

	if scopeType != "" {
		query = query.Where("scope_type = ?", scopeType)
	}
	if scopeID != "" {
		query = query.Where("scope_id = ?", scopeID)
	}
	if enabled != nil {
		query = query.Where("enabled = ?", *enabled)
	}

	err := query.Order("id DESC").Find(&quotas).Error
	return quotas, err
}

// GetByScope 获取指定范围的配额
func (s *Storage) GetByScope(scopeType, scopeID string) (*StorageQuota, error) {
	var quota StorageQuota
	err := s.db.Where("scope_type = ? AND scope_id = ? AND enabled = ?", scopeType, scopeID, true).
		First(&quota).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &quota, nil
}

// Update 更新配额
func (s *Storage) Update(quota *StorageQuota) error {
	return s.db.Save(quota).Error
}

// Delete 删除配额
func (s *Storage) Delete(id uint) error {
	return s.db.Delete(&StorageQuota{}, id).Error
}

// CalculateUsage 计算存储使用量
func (s *Storage) CalculateUsage(scopeType, scopeID string) (int64, error) {
	var totalSize int64

	query := s.db.Model(&model.BackupRecord{}).
		Where("status = ? AND soft_deleted = ?", "success", false)

	switch scopeType {
	case "global":
		// 全局：所有记录
	case "storage_bucket":
		query = query.Where("storage_type = ?", scopeID)
	case "job":
		query = query.Where("job_id = ?", scopeID)
	}

	err := query.Select("COALESCE(SUM(file_size), 0)").Scan(&totalSize).Error
	return totalSize, err
}

// GetQuotaStatus 获取配额状态
func (s *Storage) GetQuotaStatus(quota *StorageQuota) (*QuotaStatus, error) {
	usedBytes, err := s.CalculateUsage(quota.ScopeType, quota.ScopeID)
	if err != nil {
		return nil, err
	}

	usagePercent := float64(0)
	if quota.QuotaBytes > 0 {
		usagePercent = float64(usedBytes) / float64(quota.QuotaBytes) * 100
	}

	status := "normal"
	if usagePercent >= 100 {
		status = "exceeded"
	} else if float64(quota.WarnPercent) > 0 && usagePercent >= float64(quota.WarnPercent) {
		status = "warning"
	}

	return &QuotaStatus{
		QuotaID:      quota.ID,
		ScopeType:   quota.ScopeType,
		ScopeID:     quota.ScopeID,
		UsedBytes:   usedBytes,
		QuotaBytes:  quota.QuotaBytes,
		WarnPercent: quota.WarnPercent,
		UsagePercent: usagePercent,
		Status:      status,
	}, nil
}

// StorageStat 存储统计
type StorageStat struct {
	ScopeType   string    `json:"scope_type"`
	ScopeID     string    `json:"scope_id"`
	TotalSize   int64     `json:"total_size"`
	RecordCount int       `json:"record_count"`
	OldestTime  time.Time `json:"oldest_time"`
	NewestTime  time.Time `json:"newest_time"`
}

// GetStorageStats 获取存储统计
func (s *Storage) GetStorageStats(scopeType, scopeID string) (*StorageStat, error) {
	stat := &StorageStat{
		ScopeType: scopeType,
		ScopeID:   scopeID,
	}

	query := s.db.Model(&model.BackupRecord{}).
		Where("status = ? AND soft_deleted = ?", "success", false)

	switch scopeType {
	case "global":
	case "storage_bucket":
		query = query.Where("storage_type = ?", scopeID)
	case "job":
		query = query.Where("job_id = ?", scopeID)
	}

	// 获取总数
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return nil, err
	}
	stat.RecordCount = int(count)

	// 获取总大小
	if err := query.Select("COALESCE(SUM(file_size), 0)").Scan(&stat.TotalSize).Error; err != nil {
		return nil, err
	}

	return stat, nil
}

// CachedQuota 缓存的配额状态
type CachedQuota struct {
	Status     *QuotaStatus
	LastUpdate time.Time
	mu         sync.RWMutex
}

// QuotaCache 配额缓存
type QuotaCache struct {
	cache map[string]*CachedQuota // key: scopeType:scopeID
	mu    sync.RWMutex
	ttl   time.Duration
}

// NewQuotaCache 创建配额缓存
func NewQuotaCache(ttl time.Duration) *QuotaCache {
	return &QuotaCache{
		cache: make(map[string]*CachedQuota),
		ttl:   ttl,
	}
}

// Get 获取缓存的配额状态
func (c *QuotaCache) Get(scopeType, scopeID string) (*QuotaStatus, bool) {
	key := scopeType + ":" + scopeID

	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, ok := c.cache[key]
	if !ok {
		return nil, false
	}

	// 检查是否过期
	if time.Since(cached.LastUpdate) > c.ttl {
		return nil, false
	}

	return cached.Status, true
}

// Set 设置缓存的配额状态
func (c *QuotaCache) Set(status *QuotaStatus) {
	key := status.ScopeType + ":" + status.ScopeID

	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[key] = &CachedQuota{
		Status:     status,
		LastUpdate: time.Now(),
	}
}

// Invalidate 使缓存失效
func (c *QuotaCache) Invalidate(scopeType, scopeID string) {
	key := scopeType + ":" + scopeID

	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, key)
}

// Clear 清空缓存
func (c *QuotaCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*CachedQuota)
}

// formatBytes 格式化字节数
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
