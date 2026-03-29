package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/imysm/db-backup/internal/api/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupStatsTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("创建数据库失败: %v", err)
	}
	db.AutoMigrate(&model.BackupJob{}, &model.BackupRecord{})
	return db
}

// TestStatsHandler_EmptyData 测试无数据时的空统计
func TestStatsHandler_EmptyData(t *testing.T) {
	db := setupStatsTestDB(t)
	h := NewStatsHandler(db)
	router := setupTestRouter()
	router.GET("/stats", h.GetStats)

	req := httptest.NewRequest("GET", "/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200, 实际 %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"].(float64) != 0 {
		t.Fatalf("期望 code=0, 实际 %v", resp["code"])
	}

	data := resp["data"].(map[string]interface{})
	// 无数据时应全部为零
	if data["total_tasks"].(float64) != 0 {
		t.Errorf("期望 total_tasks=0, 实际 %v", data["total_tasks"])
	}
	if data["enabled_tasks"].(float64) != 0 {
		t.Errorf("期望 enabled_tasks=0, 实际 %v", data["enabled_tasks"])
	}
	if data["week_success_rate"].(float64) != 0 {
		t.Errorf("期望 week_success_rate=0, 实际 %v", data["week_success_rate"])
	}
	if data["total_storage_bytes"].(float64) != 0 {
		t.Errorf("期望 total_storage_bytes=0, 实际 %v", data["total_storage_bytes"])
	}

	// daily_stats 应有7天
	dailyStats := data["daily_stats"].([]interface{})
	if len(dailyStats) != 7 {
		t.Errorf("期望 daily_stats 有7条, 实际 %d", len(dailyStats))
	}
}

// TestStatsHandler_WithData 测试有数据时的正确统计
func TestStatsHandler_WithData(t *testing.T) {
	db := setupStatsTestDB(t)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, now.Location())
	yesterday := today.Add(-24 * time.Hour)

	// 创建5个任务，然后把第5个设为禁用
	enabled := true
	for i := 0; i < 5; i++ {
		db.Create(&model.BackupJob{
			Name:         "job-" + string(rune('A'+i)),
			DatabaseType: model.DatabaseTypePostgres,
			Host:         "localhost",
			Port:         5432,
			Database:     "testdb",
			Schedule:     "0 2 * * *",
			StorageType:  model.StorageTypeLocal,
			Enabled:      enabled,
		})
	}
	// 将第5个设为禁用
	db.Model(&model.BackupJob{}).Where("id = ?", 5).Update("enabled", false)

	// 创建今日记录
	db.Create(&model.BackupRecord{JobID: 1, JobName: "job-A", StartedAt: today, Status: model.BackupStatusSuccess, FileSize: 1024 * 1024 * 100})
	db.Create(&model.BackupRecord{JobID: 2, JobName: "job-B", StartedAt: today, Status: model.BackupStatusSuccess, FileSize: 1024 * 1024 * 200})
	db.Create(&model.BackupRecord{JobID: 3, JobName: "job-C", StartedAt: today, Status: model.BackupStatusFailed, FileSize: 0})

	// 创建昨日记录
	db.Create(&model.BackupRecord{JobID: 1, JobName: "job-A", StartedAt: yesterday, Status: model.BackupStatusSuccess, FileSize: 1024 * 1024 * 150})
	db.Create(&model.BackupRecord{JobID: 2, JobName: "job-B", StartedAt: yesterday, Status: model.BackupStatusFailed, FileSize: 0})

	h := NewStatsHandler(db)
	router := setupTestRouter()
	router.GET("/stats", h.GetStats)

	req := httptest.NewRequest("GET", "/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200, 实际 %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})

	// 验证任务数
	if data["total_tasks"].(float64) != 5 {
		t.Errorf("期望 total_tasks=5, 实际 %v", data["total_tasks"])
	}
	if data["enabled_tasks"].(float64) != 4 {
		t.Errorf("期望 enabled_tasks=4, 实际 %v", data["enabled_tasks"])
	}

	// 验证今日统计
	if data["today_success"].(float64) != 2 {
		t.Errorf("期望 today_success=2, 实际 %v", data["today_success"])
	}
	if data["today_failed"].(float64) != 1 {
		t.Errorf("期望 today_failed=1, 实际 %v", data["today_failed"])
	}
	if data["today_total"].(float64) != 4 {
		t.Errorf("期望 today_total=4 (启用任务数), 实际 %v", data["today_total"])
	}

	// 验证存储量（3个成功记录：100+200+150 MB）
	expectedStorage := int64(1024*1024*100 + 1024*1024*200 + 1024*1024*150)
	if data["total_storage_bytes"].(float64) != float64(expectedStorage) {
		t.Errorf("期望 total_storage_bytes=%d, 实际 %v", expectedStorage, data["total_storage_bytes"])
	}

	// 验证成功率（总共5条，3条成功 → 60.0%）
	if data["week_success_rate"].(float64) != 60.0 {
		t.Errorf("期望 week_success_rate=60.0, 实际 %v", data["week_success_rate"])
	}
}

// TestStatsHandler_DateFilter 测试日期过滤逻辑
func TestStatsHandler_DateFilter(t *testing.T) {
	db := setupStatsTestDB(t)
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, now.Location())

	// 3天前的记录不应计入今日统计
	threeDaysAgo := today.Add(-3 * 24 * time.Hour)
	db.Create(&model.BackupRecord{JobID: 1, JobName: "job-A", StartedAt: threeDaysAgo, Status: model.BackupStatusSuccess, FileSize: 1024})

	// 今日记录
	db.Create(&model.BackupRecord{JobID: 1, JobName: "job-A", StartedAt: today, Status: model.BackupStatusSuccess, FileSize: 2048})

	h := NewStatsHandler(db)
	router := setupTestRouter()
	router.GET("/stats", h.GetStats)

	req := httptest.NewRequest("GET", "/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})

	// 今日应只有1条成功
	if data["today_success"].(float64) != 1 {
		t.Errorf("期望 today_success=1, 实际 %v", data["today_success"])
	}

	// daily_stats 中3天前的那天应有1条记录
	dailyStats := data["daily_stats"].([]interface{})
	found := false
	for _, ds := range dailyStats {
		entry := ds.(map[string]interface{})
		if entry["total"].(float64) == 1 && entry["success"].(float64) == 1 {
			found = true
		}
	}
	if !found {
		t.Error("未在 daily_stats 中找到3天前的记录")
	}

	// 今日那天也应有1条
	todayStr := today.Format("2006-01-02")
	for _, ds := range dailyStats {
		entry := ds.(map[string]interface{})
		if entry["date"].(string) == todayStr {
			if entry["total"].(float64) != 1 {
				t.Errorf("今日 daily_stats total 应为1, 实际 %v", entry["total"])
			}
		}
	}
}
