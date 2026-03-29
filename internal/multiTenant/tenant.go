package multiTenant

import (
	"github.com/imysm/db-backup/internal/api/model"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

// Tenant 租户
type Tenant struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:128;notNull" json:"name"`            // 租户名称
	Code      string         `gorm:"size:64;uniqueIndex;notNull" json:"code"` // 租户代码（唯一标识）
	Status    bool           `gorm:"default:true" json:"status"`             // 是否启用
	Quota     TenantQuota    `gorm:"type:json" json:"quota"`                  // 资源配额(JSON)
	Plan      string         `gorm:"size:32;default:free" json:"plan"`        // 套餐: free/basic/pro/enterprise
	ExpireAt  *time.Time     `json:"expire_at"`                               // 过期时间
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Tenant) TableName() string {
	return "tenants"
}

// TenantQuota 资源配额
type TenantQuota struct {
	// 备份任务配额
	MaxJobs int `json:"max_jobs"` // 最大任务数，0表示无限制

	// 存储配额
	MaxStorageGB int `json:"max_storage_gb"` // 最大存储空间(GB)，0表示无限制
	MaxBackupCount int `json:"max_backup_count"` // 最大备份记录数

	// 计算配额
	MaxConcurrentBackups int `json:"max_concurrent_backups"` // 最大并发备份数

	// 功能开关
	EnablePITR bool `json:"enable_pitr"` // 是否启用PITR
	EnableMerge bool `json:"enable_merge"` // 是否启用合并
	EnableCluster bool `json:"enable_cluster"` // 是否启用集群备份

	// 通知配额
	MaxNotifyChannels int `json:"max_notify_channels"` // 最大通知渠道数
}

// Scan implements the sql.Scanner interface for TenantQuota
func (q *TenantQuota) Scan(value interface{}) error {
	if value == nil {
		*q = TenantQuota{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(bytes, q)
}

// Value implements the driver.Valuer interface for TenantQuota
func (q TenantQuota) Value() (driver.Value, error) {
	return json.Marshal(q)
}

const (
	PlanFree       = "free"
	PlanBasic       = "basic"
	PlanPro         = "pro"
	PlanEnterprise  = "enterprise"
)

// DefaultQuota 返回套餐默认配额
func DefaultQuota(plan string) TenantQuota {
	switch plan {
	case PlanEnterprise:
		return TenantQuota{
			MaxJobs:              0,   // 无限制
			MaxStorageGB:         0,   // 无限制
			MaxBackupCount:       0,
			MaxConcurrentBackups: 10,
			EnablePITR:           true,
			EnableMerge:          true,
			EnableCluster:        true,
			MaxNotifyChannels:    20,
		}
	case PlanPro:
		return TenantQuota{
			MaxJobs:              100,
			MaxStorageGB:         500,
			MaxBackupCount:       5000,
			MaxConcurrentBackups: 5,
			EnablePITR:           true,
			EnableMerge:          true,
			EnableCluster:        true,
			MaxNotifyChannels:    10,
		}
	case PlanBasic:
		return TenantQuota{
			MaxJobs:              20,
			MaxStorageGB:         100,
			MaxBackupCount:       1000,
			MaxConcurrentBackups: 2,
			EnablePITR:           false,
			EnableMerge:          true,
			EnableCluster:        false,
			MaxNotifyChannels:    5,
		}
	default: // free
		return TenantQuota{
			MaxJobs:              5,
			MaxStorageGB:         10,
			MaxBackupCount:       100,
			MaxConcurrentBackups: 1,
			EnablePITR:           false,
			EnableMerge:          false,
			EnableCluster:        false,
			MaxNotifyChannels:    1,
		}
	}
}

// CheckQuota 检查配额是否超限
func (t *Tenant) CheckQuota(quotaType string, currentValue int) error {
	quota := t.Quota
	if (quota == TenantQuota{}) {
		quota = DefaultQuota(t.Plan)
	}

	switch quotaType {
	case "jobs":
		if quota.MaxJobs > 0 && currentValue >= quota.MaxJobs {
			return ErrQuotaExceeded
		}
	case "storage":
		// currentValue 单位为 GB
		if quota.MaxStorageGB > 0 && currentValue >= quota.MaxStorageGB {
			return ErrQuotaExceeded
		}
	case "backups":
		if quota.MaxBackupCount > 0 && currentValue >= quota.MaxBackupCount {
			return ErrQuotaExceeded
		}
	case "concurrent":
		if quota.MaxConcurrentBackups > 0 && currentValue >= quota.MaxConcurrentBackups {
			return ErrQuotaExceeded
		}
	}
	return nil
}

// IsExpired 检查租户是否过期
func (t *Tenant) IsExpired() bool {
	if t.ExpireAt == nil {
		return false
	}
	return time.Now().After(*t.ExpireAt)
}

// CanUsePITR 检查是否能使用PITR
func (t *Tenant) CanUsePITR() bool {
	if t.IsExpired() {
		return false
	}
	quota := t.Quota
	if (quota == TenantQuota{}) {
		quota = DefaultQuota(t.Plan)
	}
	return quota.EnablePITR
}

// CanUseMerge 检查是否能使用合并
func (t *Tenant) CanUseMerge() bool {
	if t.IsExpired() {
		return false
	}
	quota := t.Quota
	if (quota == TenantQuota{}) {
		quota = DefaultQuota(t.Plan)
	}
	return quota.EnableMerge
}

// CanUseCluster 检查是否能使用集群备份
func (t *Tenant) CanUseCluster() bool {
	if t.IsExpired() {
		return false
	}
	quota := t.Quota
	if (quota == TenantQuota{}) {
		quota = DefaultQuota(t.Plan)
	}
	return quota.EnableCluster
}

// TenantResource 租户资源使用量（关联到租户的模型需实现此接口）
type TenantResource interface {
	GetTenantID() uint
}

// TenantResourceStat 租户资源统计
type TenantResourceStat struct {
	JobCount       int64 `json:"job_count"`
	StorageUsedGB  int64 `json:"storage_used_gb"`
	BackupCount    int64 `json:"backup_count"`
	ConcurrentNum  int   `json:"concurrent_num"`
}

// GetTenantStats 获取租户资源使用统计
func GetTenantStats(db *gorm.DB, tenantID uint) (*TenantResourceStat, error) {
	var stat TenantResourceStat

	// 统计任务数
	if err := db.Model(&model.BackupJob{}).Where("tenant_id = ?", tenantID).Count(&stat.JobCount).Error; err != nil {
		return nil, err
	}

	// 统计备份记录数
	if err := db.Model(&model.BackupRecord{}).
		Joins("JOIN backup_jobs ON backup_jobs.id = backup_records.job_id").
		Where("backup_jobs.tenant_id = ?", tenantID).
		Count(&stat.BackupCount).Error; err != nil {
		return nil, err
	}

	return &stat, nil
}
