package multiTenant

import (
	"github.com/imysm/db-backup/internal/api/model"
	"encoding/json"
	"errors"

	"gorm.io/gorm"
)

// TenantService 租户服务
type TenantService struct {
	db *gorm.DB
}

// NewTenantService 创建租户服务
func NewTenantService(db *gorm.DB) *TenantService {
	return &TenantService{db: db}
}

// Create 创建租户
func (s *TenantService) Create(tenant *Tenant) error {
	// 设置默认配额
	if (tenant.Quota == TenantQuota{}) {
		tenant.Quota = DefaultQuota(tenant.Plan)
	}
	return s.db.Create(tenant).Error
}

// GetByID 根据ID获取租户
func (s *TenantService) GetByID(id uint) (*Tenant, error) {
	var tenant Tenant
	if err := s.db.First(&tenant, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTenantNotFound
		}
		return nil, err
	}
	return &tenant, nil
}

// GetByCode 根据代码获取租户
func (s *TenantService) GetByCode(code string) (*Tenant, error) {
	var tenant Tenant
	if err := s.db.Where("code = ?", code).First(&tenant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTenantNotFound
		}
		return nil, err
	}
	return &tenant, nil
}

// Update 更新租户
func (s *TenantService) Update(tenant *Tenant) error {
	return s.db.Save(tenant).Error
}

// Delete 删除租户（软删除）
func (s *TenantService) Delete(id uint) error {
	return s.db.Delete(&Tenant{}, id).Error
}

// List 获取租户列表
func (s *TenantService) List(offset, limit int) ([]Tenant, int64, error) {
	var tenants []Tenant
	var total int64

	query := s.db.Model(&Tenant{})
	query.Count(&total)

	if err := query.Offset(offset).Limit(limit).Find(&tenants).Error; err != nil {
		return nil, 0, err
	}
	return tenants, total, nil
}

// UpdateQuota 更新配额
func (s *TenantService) UpdateQuota(id uint, quota TenantQuota) error {
	quotaJSON, err := json.Marshal(quota)
	if err != nil {
		return err
	}
	return s.db.Model(&Tenant{}).Where("id = ?", id).Updates(map[string]interface{}{
		"quota": string(quotaJSON),
	}).Error
}

// UpdatePlan 更新套餐
func (s *TenantService) UpdatePlan(id uint, plan string) error {
	quota := DefaultQuota(plan)
	quotaJSON, err := json.Marshal(quota)
	if err != nil {
		return err
	}
	return s.db.Model(&Tenant{}).Where("id = ?", id).Updates(map[string]interface{}{
		"plan": plan,
		"quota": string(quotaJSON),
	}).Error
}

// Enable 启用租户
func (s *TenantService) Enable(id uint) error {
	return s.db.Model(&Tenant{}).Where("id = ?", id).Update("status", true).Error
}

// Disable 禁用租户
func (s *TenantService) Disable(id uint) error {
	return s.db.Model(&Tenant{}).Where("id = ?", id).Update("status", false).Error
}

// CheckAndUpdateQuota 检查并更新配额使用
func (s *TenantService) CheckAndUpdateQuota(tenantID uint, resourceType string, increment int) error {
	tenant, err := s.GetByID(tenantID)
	if err != nil {
		return err
	}

	if !tenant.Status {
		return ErrTenantDisabled
	}

	if tenant.IsExpired() {
		return ErrTenantExpired
	}

	quota := tenant.Quota
	if (quota == TenantQuota{}) {
		quota = DefaultQuota(tenant.Plan)
	}

	switch resourceType {
	case "jobs":
		if quota.MaxJobs > 0 {
			var count int64
			s.db.Model(&model.BackupJob{}).Where("tenant_id = ?", tenantID).Count(&count)
			if count+int64(increment) > int64(quota.MaxJobs) {
				return ErrQuotaExceeded
			}
		}
	case "storage":
		if quota.MaxStorageGB > 0 {
			// TODO: 计算实际存储使用量
		}
	}
	return nil
}
