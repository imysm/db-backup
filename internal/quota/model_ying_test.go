package quota

import (
	"testing"
)

func TestStorageQuota_TableName(t *testing.T) {
	q := StorageQuota{}
	if q.TableName() != "storage_quotas" {
		t.Errorf("expected 'storage_quotas', got '%s'", q.TableName())
	}
}

func TestStorageQuota_Fields(t *testing.T) {
	q := StorageQuota{
		ID:         1,
		Name:       "测试配额",
		ScopeType: "storage_bucket",
		ScopeID:   "bucket-1",
		QuotaBytes: 1024 * 1024 * 1024 * 100, // 100GB
		WarnPercent: 80,
		UsedBytes:  1024 * 1024 * 1024 * 50,  // 50GB
		Enabled:    true,
	}

	if q.ID != 1 {
		t.Errorf("expected ID 1, got %d", q.ID)
	}
	if q.ScopeType != "storage_bucket" {
		t.Errorf("expected 'storage_bucket', got '%s'", q.ScopeType)
	}
	if q.QuotaBytes != 1024*1024*1024*100 {
		t.Errorf("expected 100GB quota, got %d", q.QuotaBytes)
	}
}

func TestQuotaStatus_Fields(t *testing.T) {
	s := QuotaStatus{
		QuotaID:      1,
		ScopeType:   "storage_bucket",
		ScopeID:     "bucket-1",
		UsedBytes:   1024 * 1024 * 1024 * 80, // 80GB
		QuotaBytes:  1024 * 1024 * 1024 * 100, // 100GB
		WarnPercent: 80,
		Status:      "warning",
	}

	if s.QuotaID != 1 {
		t.Errorf("expected QuotaID 1, got %d", s.QuotaID)
	}
	if s.Status != "warning" {
		t.Errorf("expected status 'warning', got '%s'", s.Status)
	}
}

func TestQuotaStatus_CalculateUsagePercent(t *testing.T) {
	s := QuotaStatus{
		UsedBytes:   80,
		QuotaBytes:  100,
	}

	s.UsagePercent = float64(s.UsedBytes) / float64(s.QuotaBytes) * 100

	if s.UsagePercent != 80 {
		t.Errorf("expected 80%%, got %f%%", s.UsagePercent)
	}
}
