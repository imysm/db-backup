package quota

import (
	"github.com/imysm/db-backup/internal/api/model"
	"log"
	"sync"
	"time"

	"gorm.io/gorm"
)

// Manager 配额管理器
type Manager struct {
	db    *gorm.DB
	cache *QuotaCache
	mu    sync.Mutex
}

// NewManager 创建配额管理器
func NewManager(db *gorm.DB) *Manager {
	return &Manager{
		db:    db,
		cache: NewQuotaCache(5 * time.Minute), // 5分钟缓存
	}
}

// CheckAndAlert 检查配额并触发告警
func (m *Manager) CheckAndAlert(scopeType, scopeID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 先检查缓存
	if status, ok := m.cache.Get(scopeType, scopeID); ok {
		return m.triggerAlertIfNeeded(status)
	}

	// 获取配额状态
	status, err := m.getQuotaStatus(scopeType, scopeID)
	if err != nil {
		return err
	}

	// 更新缓存
	m.cache.Set(status)

	// 触发告警
	return m.triggerAlertIfNeeded(status)
}


// CheckAllQuotas 检查所有配额
func (m *Manager) CheckAllQuotas() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	statuses, err := m.GetAllStatuses()
	if err != nil {
		return err
	}

	for _, status := range statuses {
		if err := m.triggerAlertIfNeeded(status); err != nil {
			log.Printf("[Quota] 触发告警失败: %v", err)
		}
	}

	return nil
}

// triggerAlertIfNeeded 必要时触发告警
func (m *Manager) triggerAlertIfNeeded(status *QuotaStatus) error {
	if status == nil || status.Status == "normal" || status.Status == "unlimited" {
		return nil
	}

	log.Printf("[Quota] 配额告警: scope=%s:%s, usage=%.1f%%, status=%s",
		status.ScopeType, status.ScopeID, status.UsagePercent, status.Status)

	return nil
}

// getQuotaStatus 获取配额状态
func (m *Manager) getQuotaStatus(scopeType, scopeID string) (*QuotaStatus, error) {
	storage := NewStorage(m.db)
	quota, err := storage.GetByScope(scopeType, scopeID)
	if err != nil {
		return nil, err
	}

	if quota == nil {
		return &QuotaStatus{
			ScopeType:   scopeType,
			ScopeID:     scopeID,
			QuotaBytes:  0,
			UsedBytes:   0,
			UsagePercent: 0,
			Status:      "unlimited",
		}, nil
	}

	return storage.GetQuotaStatus(quota)
}

// GetAllStatuses 获取所有配额状态
func (m *Manager) GetAllStatuses() ([]*QuotaStatus, error) {
	storage := NewStorage(m.db)
	quotas, err := storage.List("", "", nil)
	if err != nil {
		return nil, err
	}

	statuses := make([]*QuotaStatus, 0, len(quotas))
	for _, quota := range quotas {
		status, err := storage.GetQuotaStatus(&quota)
		if err != nil {
			continue
		}
		statuses = append(statuses, status)
	}

	return statuses, nil
}

// CheckBeforeBackup 备份前检查配额
func (m *Manager) CheckBeforeBackup(scopeType, scopeID string, requiredBytes int64) (*QuotaStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	status, err := m.getQuotaStatus(scopeType, scopeID)
	if err != nil {
		return nil, err
	}

	if status.Status == "exceeded" {
		return status, &QuotaExceededError{
			Status:  status,
			Message: "存储配额已超出，无法执行备份",
		}
	}

	if status.UsedBytes+requiredBytes > status.QuotaBytes {
		return status, &QuotaExceededError{
			Status:  status,
			Message: "备份后配额将超出限制",
		}
	}

	return status, nil
}

// UpdateCache 更新缓存
func (m *Manager) UpdateCache(scopeType, scopeID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cache.Invalidate(scopeType, scopeID)

	status, err := m.getQuotaStatus(scopeType, scopeID)
	if err == nil && status != nil {
		m.cache.Set(status)
	}
}

// InvalidateCache 使缓存失效
func (m *Manager) InvalidateCache(scopeType, scopeID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cache.Invalidate(scopeType, scopeID)
}

// ClearCache 清空缓存
func (m *Manager) ClearCache() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cache.Clear()
}

// QuotaExceededError 配额超出错误
type QuotaExceededError struct {
	Status  *QuotaStatus
	Message string
}

func (e *QuotaExceededError) Error() string {
	return e.Message
}

// ForecastUsage 预测存储使用趋势
type UsageForecast struct {
	ScopeType    string    `json:"scope_type"`
	ScopeID      string    `json:"scope_id"`
	CurrentUsage int64     `json:"current_usage"`
	DailyGrowth  float64   `json:"daily_growth_bytes"`
	DaysUntilFull int      `json:"days_until_full"`
	ForecastDate  time.Time `json:"forecast_date"`
}

// CalculateForecast 计算使用趋势
func (m *Manager) CalculateForecast(scopeType, scopeID string, days int) (*UsageForecast, error) {
	storage := NewStorage(m.db)

	// 获取最近 N 天的使用量
	type DailyUsage struct {
		Date  time.Time
		Bytes int64
	}

	var usages []DailyUsage
	err := m.db.Model(&model.BackupRecord{}).
		Select("DATE(created_at) as date, SUM(file_size) as bytes").
		Where("status = ? AND soft_deleted = ?", "success", false).
		Group("DATE(created_at)").
		Order("date DESC").
		Limit(days).
		Scan(&usages).Error

	if err != nil {
		return nil, err
	}

	if len(usages) < 2 {
		return &UsageForecast{
			ScopeType:    scopeType,
			ScopeID:      scopeID,
			DaysUntilFull: -1,
		}, nil
	}

	// 计算平均每日增长
	var totalGrowth int64
	for i := 0; i < len(usages)-1; i++ {
		if usages[i].Bytes > usages[i+1].Bytes {
			totalGrowth += usages[i].Bytes - usages[i+1].Bytes
		}
	}
	dailyGrowth := float64(totalGrowth) / float64(len(usages)-1)

	// 获取当前配额
	quota, err := storage.GetByScope(scopeType, scopeID)
	if err != nil || quota == nil {
		return nil, err
	}

	currentUsage := int64(0)
	if len(usages) > 0 {
		currentUsage = usages[0].Bytes
	}

	daysUntilFull := -1
	var forecastDate time.Time
	if dailyGrowth > 0 {
		remaining := quota.QuotaBytes - currentUsage
		if remaining > 0 {
			daysUntilFull = int(float64(remaining) / dailyGrowth)
			forecastDate = time.Now().AddDate(0, 0, daysUntilFull)
		}
	}

	return &UsageForecast{
		ScopeType:    scopeType,
		ScopeID:      scopeID,
		CurrentUsage: currentUsage,
		DailyGrowth:  dailyGrowth,
		DaysUntilFull: daysUntilFull,
		ForecastDate: forecastDate,
	}, nil
}
