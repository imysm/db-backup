package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/imysm/db-backup/internal/api/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupRestoreTestDB 创建带 BackupHistory 的测试数据库
func setupRestoreTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: nil})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	db.AutoMigrate(&model.BackupJob{}, &model.BackupRecord{}, &model.BackupHistory{})
	return db
}

// TestRestoreHandler_ValidateGET_BackwardCompat 测试 GET 验证向后兼容
func TestRestoreHandler_ValidateGET_BackwardCompat(t *testing.T) {
	db := setupRestoreTestDB(t)
	handler := NewRestoreHandler(db)
	router := setupTestRouter()
	router.GET("/validate/:id", handler.Validate)

	// 创建测试任务和记录
	job := model.BackupJob{
		Name: "test-job", DatabaseType: model.DatabaseTypePostgres,
		Host: "localhost", Port: 5432, Database: "testdb",
		Schedule: "0 2 * * *", StorageType: model.StorageTypeLocal, Enabled: true,
	}
	db.Create(&job)

	// 创建临时文件
	tmpFile := filepath.Join(t.TempDir(), "backup.sql")
	os.WriteFile(tmpFile, []byte("test data"), 0644)

	record := model.BackupRecord{
		JobID: job.ID, JobName: "test-job", Status: model.BackupStatusSuccess,
		FilePath: tmpFile, FileSize: 9, Checksum: "abc123",
	}
	db.Create(&record)

	req := httptest.NewRequest("GET", "/validate/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["valid"] != true {
		t.Error("Expected valid=true")
	}
	if data["file_exists"] != true {
		t.Error("Expected file_exists=true")
	}
	if data["file_size"].(float64) != 9 {
		t.Errorf("Expected file_size=9, got %v", data["file_size"])
	}
}

// TestRestoreHandler_ValidateGET_FileNotFound 测试 GET 验证文件不存在
func TestRestoreHandler_ValidateGET_FileNotFound(t *testing.T) {
	db := setupRestoreTestDB(t)
	handler := NewRestoreHandler(db)
	router := setupTestRouter()
	router.GET("/validate/:id", handler.Validate)

	job := model.BackupJob{
		Name: "test-job", DatabaseType: model.DatabaseTypePostgres,
		Host: "localhost", Port: 5432, Database: "testdb",
		Schedule: "0 2 * * *", StorageType: model.StorageTypeLocal, Enabled: true,
	}
	db.Create(&job)
	record := model.BackupRecord{
		JobID: job.ID, JobName: "test-job", Status: model.BackupStatusSuccess,
		FilePath: "/nonexistent/file.sql", FileSize: 0,
	}
	db.Create(&record)

	req := httptest.NewRequest("GET", "/validate/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["valid"] != false {
		t.Error("Expected valid=false for missing file")
	}
}

// TestRestoreHandler_ValidatePOST 测试 POST 增强预检查
// TODO: 实现 RestoreHandler.ValidatePOST 和 ValidateRequest 后取消注释以下测试

/*
func TestRestoreHandler_ValidatePOST(t *testing.T) {
	db := setupRestoreTestDB(t)
	handler := NewRestoreHandler(db)
	router := setupTestRouter()
	router.POST("/validate/:id", handler.ValidatePOST)

	job := model.BackupJob{
		Name: "test-job", DatabaseType: model.DatabaseTypePostgres,
		Host: "localhost", Port: 5432, Database: "testdb",
		Schedule: "0 2 * * *", StorageType: model.StorageTypeLocal, Enabled: true,
	}
	db.Create(&job)

	tmpFile := filepath.Join(t.TempDir(), "backup.sql")
	os.WriteFile(tmpFile, []byte("test data for validate"), 0644)

	record := model.BackupRecord{
		JobID: job.ID, JobName: "test-job", Status: model.BackupStatusSuccess,
		FilePath: tmpFile, FileSize: 21, Checksum: "abc123",
	}
	db.Create(&record)

	// 发送带目标配置的 POST 请求
	body := ValidateRequest{TargetHost: "127.0.0.1", TargetPort: 80}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/validate/1", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["all_passed"] != true {
		t.Errorf("Expected all_passed=true, got %v", data["all_passed"])
	}

	checks := data["checks"].([]interface{})
	if len(checks) != 3 {
		t.Fatalf("Expected 3 checks, got %d", len(checks))
	}

	// 验证文件检查
	fileCheck := checks[0].(map[string]interface{})
	if fileCheck["passed"] != true {
		t.Error("File check should pass")
	}
	if fileCheck["name"] != "备份文件" {
		t.Errorf("Expected name '备份文件', got %v", fileCheck["name"])
	}
}
*/
/*

// TestRestoreHandler_ValidatePOST_EmptyBody 测试 POST 空请求体
func TestRestoreHandler_ValidatePOST_EmptyBody(t *testing.T) {
	db := setupRestoreTestDB(t)
	handler := NewRestoreHandler(db)
	router := setupTestRouter()
	router.POST("/validate/:id", handler.ValidatePOST)

	job := model.BackupJob{
		Name: "test-job", DatabaseType: model.DatabaseTypeMySQL,
		Host: "localhost", Port: 3306, Database: "testdb",
		Schedule: "0 2 * * *", StorageType: model.StorageTypeLocal, Enabled: true,
	}
	db.Create(&job)

	tmpFile := filepath.Join(t.TempDir(), "backup.sql")
	os.WriteFile(tmpFile, []byte("data"), 0644)

	record := model.BackupRecord{
		JobID: job.ID, JobName: "test-job", Status: model.BackupStatusSuccess,
		FilePath: tmpFile, FileSize: 4,
	}
	db.Create(&record)

	req := httptest.NewRequest("POST", "/validate/1", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	checks := data["checks"].([]interface{})
	// 空配置时数据库检查应跳过
	dbCheck := checks[1].(map[string]interface{})
	if dbCheck["passed"] != true {
		t.Error("DB check should pass (skipped) when no config provided")
	}
}
*/

/*
// TestRestoreHandler_GetDetail_Exists 测试获取恢复详情
func TestRestoreHandler_GetDetail_Exists(t *testing.T) {
	db := setupRestoreTestDB(t)
	handler := NewRestoreHandler(db)
	router := setupTestRouter()
	router.GET("/detail/:id", handler.GetDetail)

	job := model.BackupJob{
		Name: "test-job", DatabaseType: model.DatabaseTypePostgres,
		Host: "localhost", Port: 5432, Database: "testdb",
		Schedule: "0 2 * * *", StorageType: model.StorageTypeLocal, Enabled: true,
	}
	db.Create(&job)

	record := model.BackupRecord{
		JobID: job.ID, JobName: "test-job", Status: model.BackupStatusSuccess,
		FilePath: "/tmp/test.sql", FileSize: 100,
	}
	db.Create(&record)

	history := model.BackupHistory{
		JobID: job.ID, RecordID: record.ID, Action: "restored",
		Details: "恢复到 localhost:5432/mydb，耗时 2m 15s",
	}
	db.Create(&history)

	req := httptest.NewRequest("GET", "/detail/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["action"] != "restored" {
		t.Errorf("Expected action 'restored', got %v", data["action"])
	}
	if data["job_name"] != "test-job" {
		t.Errorf("Expected job_name 'test-job', got %v", data["job_name"])
	}
	if data["details"] != "恢复到 localhost:5432/mydb，耗时 2m 15s" {
		t.Errorf("Unexpected details: %v", data["details"])
	}
}*/
/*

// TestRestoreHandler_GetDetail_NotFound 测试获取不存在的恢复详情
func TestRestoreHandler_GetDetail_NotFound(t *testing.T) {
	db := setupRestoreTestDB(t)
	handler := NewRestoreHandler(db)
	router := setupTestRouter()
	router.GET("/detail/:id", handler.GetDetail)

	req := httptest.NewRequest("GET", "/detail/999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", w.Code)
	}
}
*/

// TestRestoreHandler_Restore_InvalidJSON 测试恢复无效 JSON
func TestRestoreHandler_Restore_InvalidJSON(t *testing.T) {
	db := setupRestoreTestDB(t)
	handler := NewRestoreHandler(db)
	router := setupTestRouter()
	router.POST("/restore", handler.Restore)

	req := httptest.NewRequest("POST", "/restore", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 for invalid JSON, got %d", w.Code)
	}
}

// TestRestoreHandler_Restore_RecordNotFound 测试恢复记录不存在
func TestRestoreHandler_Restore_RecordNotFound(t *testing.T) {
	db := setupRestoreTestDB(t)
	handler := NewRestoreHandler(db)
	router := setupTestRouter()
	router.POST("/restore", handler.Restore)

	body := map[string]interface{}{"record_id": 999}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/restore", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %d", w.Code)
	}
}

// TestRestoreHandler_List_Pagination 测试恢复列表分页
func TestRestoreHandler_List_Pagination(t *testing.T) {
	db := setupRestoreTestDB(t)
	handler := NewRestoreHandler(db)
	router := setupTestRouter()
	router.GET("/list", handler.List)

	job := model.BackupJob{
		Name: "test-job", DatabaseType: model.DatabaseTypePostgres,
		Host: "localhost", Port: 5432, Database: "testdb",
		Schedule: "0 2 * * *", StorageType: model.StorageTypeLocal, Enabled: true,
	}
	db.Create(&job)

	// 创建 5 条成功记录
	for i := 0; i < 5; i++ {
		record := model.BackupRecord{
			JobID: job.ID, JobName: "test-job", Status: model.BackupStatusSuccess,
			FilePath: "/tmp/test.sql", FileSize: int64(100 * (i + 1)),
		}
		db.Create(&record)
	}

	// 测试分页
	req := httptest.NewRequest("GET", "/list?page=1&page_size=3", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if int(data["total"].(float64)) != 5 {
		t.Errorf("Expected total=5, got %v", data["total"])
	}
	items := data["data"].([]interface{})
	if len(items) != 3 {
		t.Errorf("Expected 3 items, got %d", len(items))
	}
}

/*
// TestRestoreHandler_ValidatePOST_InvalidID 测试 POST 验证无效 ID
func TestRestoreHandler_ValidatePOST_InvalidID(t *testing.T) {
	db := setupRestoreTestDB(t)
	handler := NewRestoreHandler(db)
	router := setupTestRouter()
	router.POST("/validate/:id", handler.ValidatePOST)

	req := httptest.NewRequest("POST", "/validate/abc", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}
*/
