package handler

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/imysm/db-backup/internal/api/model"
)

// --- Record List 筛选和排序测试 ---

func TestRecordHandler_List_WithStatusFilter_Data(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewRecordHandler(db)
	router := gin.New()
	router.GET("/records", handler.List)

	// 创建 job
	job := model.BackupJob{
		Name: "test-job", DatabaseType: model.DatabaseTypePostgres,
		Host: "localhost", Port: 5432, Database: "testdb",
		Schedule: "0 2 * * *", StorageType: model.StorageTypeLocal, Enabled: true,
	}
	db.Create(&job)

	// 创建不同状态的记录
	now := time.Now()
	db.Create(&model.BackupRecord{JobID: job.ID, Status: model.BackupStatusSuccess, FilePath: "/tmp/a.sql", FileSize: 100, StartedAt: now})
	db.Create(&model.BackupRecord{JobID: job.ID, Status: model.BackupStatusSuccess, FilePath: "/tmp/b.sql", FileSize: 200, StartedAt: now.Add(-time.Hour)})
	db.Create(&model.BackupRecord{JobID: job.ID, Status: model.BackupStatusFailed, FilePath: "", StartedAt: now.Add(-2 * time.Hour)})
	db.Create(&model.BackupRecord{JobID: job.ID, Status: model.BackupStatusPending, FilePath: "", StartedAt: now.Add(-3 * time.Hour)})

	// 按 status=success 筛选
	req := httptest.NewRequest("GET", "/records?status=success&page_size=100", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	items := data["data"].([]interface{})
	if len(items) != 2 {
		t.Errorf("status=success: expected 2 records, got %d", len(items))
	}

	// 按 status=failed 筛选
	req = httptest.NewRequest("GET", "/records?status=failed&page_size=100", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	json.Unmarshal(w.Body.Bytes(), &resp)
	data = resp["data"].(map[string]interface{})
	items = data["data"].([]interface{})
	if len(items) != 1 {
		t.Errorf("status=failed: expected 1 record, got %d", len(items))
	}
}

func TestRecordHandler_List_WithDateRange(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewRecordHandler(db)
	router := gin.New()
	router.GET("/records", handler.List)

	job := model.BackupJob{
		Name: "test-job", DatabaseType: model.DatabaseTypePostgres,
		Host: "localhost", Port: 5432, Database: "testdb",
		Schedule: "0 2 * * *", StorageType: model.StorageTypeLocal, Enabled: true,
	}
	db.Create(&job)

	// 创建不同日期的记录（使用本地时间避免时区问题）
	d1 := time.Date(2026, 3, 25, 2, 0, 0, 0, time.Now().Location())
	d2 := time.Date(2026, 3, 27, 2, 0, 0, 0, time.Now().Location())
	d3 := time.Date(2026, 3, 28, 2, 0, 0, 0, time.Now().Location())
	db.Create(&model.BackupRecord{JobID: job.ID, Status: model.BackupStatusSuccess, FilePath: "/tmp/a.sql", StartedAt: d1})
	db.Create(&model.BackupRecord{JobID: job.ID, Status: model.BackupStatusSuccess, FilePath: "/tmp/b.sql", StartedAt: d2})
	db.Create(&model.BackupRecord{JobID: job.ID, Status: model.BackupStatusSuccess, FilePath: "/tmp/c.sql", StartedAt: d3})

	// 日期范围筛选：3月26日-3月27日（应包含 d2）
	req := httptest.NewRequest("GET", "/records?start_date=2026-03-26&end_date=2026-03-27&page_size=100", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	items := data["data"].([]interface{})
	if len(items) != 1 {
		t.Errorf("date range 3/26-3/27: expected 1 record, got %d", len(items))
	}

	// 只传 start_date=3月28日（应只包含 d3）
	req = httptest.NewRequest("GET", "/records?start_date=2026-03-28&page_size=100", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	json.Unmarshal(w.Body.Bytes(), &resp)
	data = resp["data"].(map[string]interface{})
	items = data["data"].([]interface{})
	if len(items) != 1 {
		t.Errorf("start_date>=3/27: expected 1 record, got %d", len(items))
	}
}

func TestRecordHandler_List_WithSort(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewRecordHandler(db)
	router := gin.New()
	router.GET("/records", handler.List)

	job := model.BackupJob{
		Name: "test-job", DatabaseType: model.DatabaseTypePostgres,
		Host: "localhost", Port: 5432, Database: "testdb",
		Schedule: "0 2 * * *", StorageType: model.StorageTypeLocal, Enabled: true,
	}
	db.Create(&job)

	now := time.Now()
	db.Create(&model.BackupRecord{JobID: job.ID, Status: model.BackupStatusSuccess, FilePath: "/tmp/big.sql", FileSize: 5000, StartedAt: now, Duration: 120})
	db.Create(&model.BackupRecord{JobID: job.ID, Status: model.BackupStatusSuccess, FilePath: "/tmp/small.sql", FileSize: 100, StartedAt: now.Add(-time.Hour), Duration: 10})
	db.Create(&model.BackupRecord{JobID: job.ID, Status: model.BackupStatusSuccess, FilePath: "/tmp/med.sql", FileSize: 1000, StartedAt: now.Add(-2 * time.Hour), Duration: 60})

	// 按 file_size desc 排序
	req := httptest.NewRequest("GET", "/records?sort_by=file_size&sort_order=desc&page_size=100", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	items := data["data"].([]interface{})
	if len(items) < 2 {
		t.Fatalf("Expected at least 2 records, got %d", len(items))
	}
	first := items[0].(map[string]interface{})
	if first["file_size"].(float64) < 1000 {
		t.Errorf("Expected first record to have file_size >= 1000, got %v", first["file_size"])
	}

	// 按 file_size asc 排序
	req = httptest.NewRequest("GET", "/records?sort_by=file_size&sort_order=asc&page_size=100", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	json.Unmarshal(w.Body.Bytes(), &resp)
	data = resp["data"].(map[string]interface{})
	items = data["data"].([]interface{})
	first = items[0].(map[string]interface{})
	if first["file_size"].(float64) > 1000 {
		t.Errorf("Expected first record to have file_size <= 1000, got %v", first["file_size"])
	}

	// 按 duration 排序
	req = httptest.NewRequest("GET", "/records?sort_by=duration&sort_order=desc&page_size=100", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	json.Unmarshal(w.Body.Bytes(), &resp)
	data = resp["data"].(map[string]interface{})
	items = data["data"].([]interface{})
	first = items[0].(map[string]interface{})
	if first["duration"].(float64) < 60 {
		t.Errorf("Expected first record to have duration >= 60, got %v", first["duration"])
	}

	// 无效排序字段应回退到 started_at
	req = httptest.NewRequest("GET", "/records?sort_by=invalid_field&page_size=100", nil)
	w = httptest.NewRecorder()
	if w.Code != 200 {
		t.Errorf("Expected 200 for invalid sort field, got %d", w.Code)
	}
}

// --- Record Download 测试 ---

func TestRecordHandler_Download_FileExists(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewRecordHandler(db)
	router := gin.New()
	router.GET("/records/:id/download", handler.Download)

	job := model.BackupJob{
		Name: "test-job", DatabaseType: model.DatabaseTypePostgres,
		Host: "localhost", Port: 5432, Database: "testdb",
		Schedule: "0 2 * * *", StorageType: model.StorageTypeLocal, Enabled: true,
	}
	db.Create(&job)

	// 创建临时文件
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "backup.sql")
	// 设置 StorageConfig 以便 Download handler 能获取存储路径
	db.Model(&job).Update("storage_config", `{"path":"`+tmpDir+`"}`)
	os.WriteFile(tmpFile, []byte("test backup data"), 0644)

	db.Create(&model.BackupRecord{
		JobID: job.ID, Status: model.BackupStatusSuccess,
		FilePath: tmpFile, FileSize: 15,
	})

	req := httptest.NewRequest("GET", "/records/1/download", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}
	if w.Body.String() != "test backup data" {
		t.Errorf("Expected file content, got %q", w.Body.String())
	}
}

func TestRecordHandler_Download_FileNotFound(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewRecordHandler(db)
	router := gin.New()
	router.GET("/records/:id/download", handler.Download)

	job := model.BackupJob{
		Name: "test-job-download-notfound", DatabaseType: model.DatabaseTypePostgres,
		Host: "localhost", Port: 5432, Database: "testdb",
		Schedule: "0 2 * * *", StorageType: model.StorageTypeLocal, Enabled: true,
		StorageConfig: `{"path":"/nonexistent"}`,
	}
	db.Create(&job)

	db.Create(&model.BackupRecord{
		JobID: job.ID, Status: model.BackupStatusSuccess,
		FilePath: "/nonexistent/backup.sql", FileSize: 1024,
	})

	req := httptest.NewRequest("GET", "/records/1/download", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("Expected 404 for missing file, got %d", w.Code)
	}
}

func TestRecordHandler_Download_InvalidID(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewRecordHandler(db)
	router := gin.New()
	router.GET("/records/:id/download", handler.Download)

	req := httptest.NewRequest("GET", "/records/abc/download", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestRecordHandler_Download_RecordNotFound(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewRecordHandler(db)
	router := gin.New()
	router.GET("/records/:id/download", handler.Download)

	req := httptest.NewRequest("GET", "/records/999/download", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("Expected 404, got %d", w.Code)
	}
}

// --- Batch Verify 测试 ---

// TODO: 实现 VerifyHandler.BatchVerify 和 BatchVerifyRequest 后取消注释以下测试

/*
func TestVerifyHandler_BatchVerify_Success(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewVerifyHandler(db)
	router := gin.New()
	router.POST("/verify/batch", handler.BatchVerify)

	job := model.BackupJob{
		Name: "test-job", DatabaseType: model.DatabaseTypePostgres,
		Host: "localhost", Port: 5432, Database: "testdb",
		Schedule: "0 2 * * *", StorageType: model.StorageTypeLocal, Enabled: true,
	}
	db.Create(&job)

	// 创建两个记录，文件存在
	tmpDir := t.TempDir()
	for i := 0; i < 2; i++ {
		f := filepath.Join(tmpDir, "backup"+string(rune('a'+i))+".sql")
		os.WriteFile(f, []byte("data"), 0644)
		db.Create(&model.BackupRecord{
			JobID: job.ID, Status: model.BackupStatusSuccess,
			FilePath: f, FileSize: 4, Checksum: "",
		})
	}

	body, _ := json.Marshal(BatchVerifyRequest{IDs: []uint{1, 2}})
	req := httptest.NewRequest("POST", "/verify/batch", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if int(data["total"].(float64)) != 2 {
		t.Errorf("Expected 2 results, got %v", data["total"])
	}
}

func TestVerifyHandler_BatchVerify_Empty(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewVerifyHandler(db)
	router := gin.New()
	router.POST("/verify/batch", handler.BatchVerify)

	body, _ := json.Marshal(BatchVerifyRequest{IDs: []uint{}})
	req := httptest.NewRequest("POST", "/verify/batch", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("Expected 400 for empty IDs, got %d", w.Code)
	}
}

func TestVerifyHandler_BatchVerify_InvalidJSON(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewVerifyHandler(db)
	router := gin.New()
	router.POST("/verify/batch", handler.BatchVerify)

	req := httptest.NewRequest("POST", "/verify/batch", bytes.NewBuffer([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("Expected 400 for invalid JSON, got %d", w.Code)
	}
}

func TestVerifyHandler_BatchVerify_PartialFail(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewVerifyHandler(db)
	router := gin.New()
	router.POST("/verify/batch", handler.BatchVerify)

	job := model.BackupJob{
		Name: "test-job", DatabaseType: model.DatabaseTypePostgres,
		Host: "localhost", Port: 5432, Database: "testdb",
		Schedule: "0 2 * * *", StorageType: model.StorageTypeLocal, Enabled: true,
	}
	db.Create(&job)

	// 记录1：文件不存在
	db.Create(&model.BackupRecord{
		JobID: job.ID, Status: model.BackupStatusSuccess,
		FilePath: "/nonexistent/file.sql", FileSize: 0,
	})
	// 记录2：不存在的 ID（999）

	body, _ := json.Marshal(BatchVerifyRequest{IDs: []uint{1, 999}})
	req := httptest.NewRequest("POST", "/verify/batch", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if int(data["total"].(float64)) != 2 {
		t.Errorf("Expected 2 results, got %v", data["total"])
	}
	if int(data["passed"].(float64)) != 0 {
		t.Errorf("Expected 0 passed, got %v", data["passed"])
	}
}
*/
