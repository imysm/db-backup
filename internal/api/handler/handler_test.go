package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/imysm/db-backup/internal/api/dto"
	"github.com/imysm/db-backup/internal/api/model"
	"github.com/imysm/db-backup/internal/crypto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: nil})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// 自动迁移
	db.AutoMigrate(&model.BackupJob{}, &model.BackupRecord{})

	return db
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestJobHandler_List(t *testing.T) {
	db := setupTestDB(t)
	handler := NewJobHandler(db, crypto.NewNoOpEncryptor())
	router := setupTestRouter()

	router.GET("/jobs", handler.List)

	// 创建测试数据
	job := model.BackupJob{
		Name:         "test-job",
		DatabaseType: model.DatabaseTypePostgres,
		Host:         "localhost",
		Port:         5432,
		Database:     "testdb",
		Username:     "postgres",
		Password:     "password",
		Schedule:     "0 2 * * *",
		StorageType:  model.StorageTypeLocal,
		Enabled:      true,
	}
	db.Create(&job)

	// 测试列表接口
	req := httptest.NewRequest("GET", "/jobs?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"].(float64) != 0 {
		t.Errorf("Expected code 0, got %v", resp["code"])
	}
}

func TestJobHandler_Get(t *testing.T) {
	db := setupTestDB(t)
	handler := NewJobHandler(db, crypto.NewNoOpEncryptor())
	router := setupTestRouter()

	router.GET("/jobs/:id", handler.Get)

	// 创建测试数据
	job := model.BackupJob{
		Name:         "test-job",
		DatabaseType: model.DatabaseTypePostgres,
		Host:         "localhost",
		Port:         5432,
		Database:     "testdb",
		Username:     "postgres",
		Password:     "password",
		Schedule:     "0 2 * * *",
		StorageType:  model.StorageTypeLocal,
		Enabled:      true,
	}
	db.Create(&job)

	// 测试获取接口
	req := httptest.NewRequest("GET", "/jobs/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"].(float64) != 0 {
		t.Errorf("Expected code 0, got %v", resp["code"])
	}
}

func TestJobHandler_Get_NotFound(t *testing.T) {
	db := setupTestDB(t)
	handler := NewJobHandler(db, crypto.NewNoOpEncryptor())
	router := setupTestRouter()

	router.GET("/jobs/:id", handler.Get)

	// 测试不存在的任务
	req := httptest.NewRequest("GET", "/jobs/999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestJobHandler_Create(t *testing.T) {
	db := setupTestDB(t)
	handler := NewJobHandler(db, crypto.NewNoOpEncryptor())
	router := setupTestRouter()

	router.POST("/jobs", handler.Create)

	// 测试创建接口
	body := dto.CreateJobRequest{
		Name:         "new-job",
		DatabaseType: "postgres",
		Host:         "localhost",
		Port:         5432,
		Database:     "testdb",
		Username:     "postgres",
		Password:     "password",
		Schedule:     "0 3 * * *",
		StorageType:  "local",
		Compress:     true,
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/jobs", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"].(float64) != 0 {
		t.Errorf("Expected code 0, got %v", resp["code"])
	}
}

func TestJobHandler_Create_InvalidRequest(t *testing.T) {
	db := setupTestDB(t)
	handler := NewJobHandler(db, crypto.NewNoOpEncryptor())
	router := setupTestRouter()

	router.POST("/jobs", handler.Create)

	// 测试无效请求
	req := httptest.NewRequest("POST", "/jobs", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestJobHandler_Update(t *testing.T) {
	db := setupTestDB(t)
	handler := NewJobHandler(db, crypto.NewNoOpEncryptor())
	router := setupTestRouter()

	router.PUT("/jobs/:id", handler.Update)

	// 创建测试数据
	job := model.BackupJob{
		Name:         "test-job",
		DatabaseType: model.DatabaseTypePostgres,
		Host:         "localhost",
		Port:         5432,
		Database:     "testdb",
		Username:     "postgres",
		Password:     "password",
		Schedule:     "0 2 * * *",
		StorageType:  model.StorageTypeLocal,
		Enabled:      true,
	}
	db.Create(&job)

	// 测试更新接口
	body := dto.UpdateJobRequest{
		Name:     "updated-job",
		Schedule: "0 4 * * *",
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("PUT", "/jobs/1", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestJobHandler_Delete(t *testing.T) {
	db := setupTestDB(t)
	handler := NewJobHandler(db, crypto.NewNoOpEncryptor())
	router := setupTestRouter()

	router.DELETE("/jobs/:id", handler.Delete)

	// 创建测试数据
	job := model.BackupJob{
		Name:         "test-job",
		DatabaseType: model.DatabaseTypePostgres,
		Host:         "localhost",
		Port:         5432,
		Database:     "testdb",
		Username:     "postgres",
		Password:     "password",
		Schedule:     "0 2 * * *",
		StorageType:  model.StorageTypeLocal,
		Enabled:      true,
	}
	db.Create(&job)

	// 测试删除接口
	req := httptest.NewRequest("DELETE", "/jobs/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestJobHandler_Run(t *testing.T) {
	db := setupTestDB(t)
	handler := NewJobHandler(db, crypto.NewNoOpEncryptor())
	router := setupTestRouter()

	router.POST("/jobs/:id/run", handler.Run)

	// 测试运行不存在的任务应返回 404
	req := httptest.NewRequest("POST", "/jobs/1/run", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestJobHandler_List_WithPagination(t *testing.T) {
	db := setupTestDB(t)
	handler := NewJobHandler(db, crypto.NewNoOpEncryptor())
	router := setupTestRouter()

	router.GET("/jobs", handler.List)

	// 创建多个任务
	for i := 0; i < 25; i++ {
		job := model.BackupJob{
			Name:         "test-job-" + string(rune(i)),
			DatabaseType: model.DatabaseTypePostgres,
			Host:         "localhost",
			Port:         5432,
			Database:     "testdb",
			Username:     "postgres",
			Password:     "password",
			Schedule:     "0 2 * * *",
			StorageType:  model.StorageTypeLocal,
			Enabled:      true,
		}
		db.Create(&job)
	}

	// 测试分页
	req := httptest.NewRequest("GET", "/jobs?page=2&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["page"].(float64) != 2 {
		t.Errorf("Expected page 2, got %v", data["page"])
	}
}

func TestJobHandler_Create_WithAllFields(t *testing.T) {
	db := setupTestDB(t)
	handler := NewJobHandler(db, crypto.NewNoOpEncryptor())
	router := setupTestRouter()

	router.POST("/jobs", handler.Create)

	// 测试创建接口（所有字段）
	body := dto.CreateJobRequest{
		Name:            "full-job",
		DatabaseType:    "postgres",
		Host:            "192.168.1.100",
		Port:            5432,
		Database:        "production_db",
		Username:        "backup_user",
		Password:        "secure_password",
		Schedule:        "0 3 * * *",
		StorageType:     "s3",
		RetentionDays:   30,
		BackupType:      "full",
		Compress:        true,
		Encrypt:         true,
		NotifyOnSuccess: true,
		NotifyOnFail:    true,
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/jobs", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestJobHandler_Update_WithAllFields(t *testing.T) {
	db := setupTestDB(t)
	handler := NewJobHandler(db, crypto.NewNoOpEncryptor())
	router := setupTestRouter()

	router.PUT("/jobs/:id", handler.Update)

	// 创建测试数据
	job := model.BackupJob{
		Name:         "test-job",
		DatabaseType: model.DatabaseTypePostgres,
		Host:         "localhost",
		Port:         5432,
		Database:     "testdb",
		Username:     "postgres",
		Password:     "password",
		Schedule:     "0 2 * * *",
		StorageType:  model.StorageTypeLocal,
		Enabled:      true,
	}
	db.Create(&job)

	// 测试更新所有字段
	enabled := false
	body := dto.UpdateJobRequest{
		Name:            "updated-job",
		Schedule:        "0 4 * * *",
		RetentionDays:   14,
		StorageType:     "s3",
		Compress:        false,
		Encrypt:         false,
		Enabled:         &enabled,
		NotifyOnSuccess: false,
		NotifyOnFail:    true,
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("PUT", "/jobs/1", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestJobHandler_Delete_Multiple(t *testing.T) {
	db := setupTestDB(t)
	handler := NewJobHandler(db, crypto.NewNoOpEncryptor())
	router := setupTestRouter()

	router.DELETE("/jobs/:id", handler.Delete)

	// 创建多个测试数据
	for i := 1; i <= 3; i++ {
		job := model.BackupJob{
			Name:         "test-job-" + string(rune(i)),
			DatabaseType: model.DatabaseTypePostgres,
			Host:         "localhost",
			Port:         5432,
			Database:     "testdb",
			Username:     "postgres",
			Password:     "password",
			Schedule:     "0 2 * * *",
			StorageType:  model.StorageTypeLocal,
			Enabled:      true,
		}
		db.Create(&job)
	}

	// 删除第一个
	req := httptest.NewRequest("DELETE", "/jobs/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// 删除第二个
	req = httptest.NewRequest("DELETE", "/jobs/2", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// 验证只剩一个
	var count int64
	db.Model(&model.BackupJob{}).Count(&count)
	if count != 1 {
		t.Errorf("Expected 1 job, got %d", count)
	}
}

// TestJobHandler_List_WithFilters 测试列表筛选功能
func TestJobHandler_List_WithFilters(t *testing.T) {
	db := setupTestDB(t)
	handler := NewJobHandler(db, crypto.NewNoOpEncryptor())
	router := setupTestRouter()

	router.GET("/jobs", handler.List)

	// 创建不同类型的测试数据
	jobs := []model.BackupJob{
		{Name: "mysql-prod", DatabaseType: model.DatabaseTypeMySQL, Host: "db1", Port: 3306, Database: "prod", Schedule: "0 2 * * *", StorageType: model.StorageTypeLocal, Enabled: true},
		{Name: "mysql-test", DatabaseType: model.DatabaseTypeMySQL, Host: "db2", Port: 3306, Database: "test", Schedule: "0 3 * * *", StorageType: model.StorageTypeLocal},
		{Name: "pg-prod", DatabaseType: model.DatabaseTypePostgres, Host: "db3", Port: 5432, Database: "prod", Schedule: "0 2 * * *", StorageType: model.StorageTypeLocal, Enabled: true},
	}
	for _, j := range jobs {
		db.Create(&j)
	}
	// 显式禁用 mysql-test（避免 GORM default:true 覆盖零值）
	db.Model(&model.BackupJob{}).Where("name = ?", "mysql-test").Update("enabled", false)

	// 测试 search 筛选
	req := httptest.NewRequest("GET", "/jobs?search=mysql", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if int(data["total"].(float64)) != 2 {
		t.Errorf("search=mysql: expected 2, got %v", data["total"])
	}

	// 测试 database_type 筛选
	req = httptest.NewRequest("GET", "/jobs?database_type=postgres", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	json.Unmarshal(w.Body.Bytes(), &resp)
	data = resp["data"].(map[string]interface{})
	if int(data["total"].(float64)) != 1 {
		t.Errorf("database_type=postgres: expected 1, got %v", data["total"])
	}

	// 测试 enabled 筛选
	req = httptest.NewRequest("GET", "/jobs?enabled=false", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	json.Unmarshal(w.Body.Bytes(), &resp)
	data = resp["data"].(map[string]interface{})
	if int(data["total"].(float64)) != 1 {
		t.Errorf("enabled=false: expected 1, got %v", data["total"])
	}

	// 测试组合筛选
	req = httptest.NewRequest("GET", "/jobs?search=mysql&enabled=true", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	json.Unmarshal(w.Body.Bytes(), &resp)
	data = resp["data"].(map[string]interface{})
	if int(data["total"].(float64)) != 1 {
		t.Errorf("combined filter: expected 1, got %v", data["total"])
	}
}

// TestJobHandler_TestConnection 测试连接功能
func TestJobHandler_TestConnection(t *testing.T) {
	db := setupTestDB(t)
	handler := NewJobHandler(db, crypto.NewNoOpEncryptor())
	router := setupTestRouter()

	router.POST("/jobs/:id/test-connection", handler.TestConnection)

	// 创建测试任务（使用 localhost 的随机高端口，连接应该成功）
	job := model.BackupJob{
		Name:         "test-conn",
		DatabaseType: model.DatabaseTypeMySQL,
		Host:         "127.0.0.1",
		Port:         80, // 假设 80 端口有服务
		Database:     "test",
		Schedule:     "0 2 * * *",
		StorageType:  model.StorageTypeLocal,
		Enabled:      true,
	}
	db.Create(&job)

	// 测试连接成功场景（使用 127.0.0.1:80 或其他可能有服务的端口）
	req := httptest.NewRequest("POST", "/jobs/1/test-connection", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	respData := resp["data"].(map[string]interface{})
	// 不管成功还是失败，都应返回正确的结构
	if _, ok := respData["success"]; !ok {
		t.Error("Response should contain 'success' field")
	}
	if _, ok := respData["latency_ms"]; !ok {
		t.Error("Response should contain 'latency_ms' field")
	}
}

// TestJobHandler_TestConnection_Failed 测试连接失败场景
func TestJobHandler_TestConnection_Failed(t *testing.T) {
	db := setupTestDB(t)
	handler := NewJobHandler(db, crypto.NewNoOpEncryptor())
	router := setupTestRouter()

	router.POST("/jobs/:id/test-connection", handler.TestConnection)

	// 创建测试任务（使用不存在的地址，连接应该失败）
	job := model.BackupJob{
		Name:         "test-conn-fail",
		DatabaseType: model.DatabaseTypeMySQL,
		Host:         "192.0.2.1", // TEST-NET，不可路由
		Port:         65534,
		Database:     "test",
		Schedule:     "0 2 * * *",
		StorageType:  model.StorageTypeLocal,
		Enabled:      true,
	}
	db.Create(&job)

	req := httptest.NewRequest("POST", "/jobs/1/test-connection", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	respData := resp["data"].(map[string]interface{})
	if respData["success"].(bool) != false {
		t.Error("Connection to 192.0.2.1:65534 should fail")
	}
}

// TestJobHandler_TestConnection_NotFound 测试连接-任务不存在
func TestJobHandler_TestConnection_NotFound(t *testing.T) {
	db := setupTestDB(t)
	handler := NewJobHandler(db, crypto.NewNoOpEncryptor())
	router := setupTestRouter()

	router.POST("/jobs/:id/test-connection", handler.TestConnection)

	req := httptest.NewRequest("POST", "/jobs/999/test-connection", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// TestJobHandler_NextRuns 测试下次执行时间
func TestJobHandler_NextRuns(t *testing.T) {
	db := setupTestDB(t)
	handler := NewJobHandler(db, crypto.NewNoOpEncryptor())
	router := setupTestRouter()

	router.GET("/jobs/:id/next-runs", handler.NextRuns)

	// 创建测试任务
	job := model.BackupJob{
		Name:         "test-cron",
		DatabaseType: model.DatabaseTypePostgres,
		Host:         "localhost",
		Port:         5432,
		Database:     "test",
		Schedule:     "0 2 * * *",
		StorageType:  model.StorageTypeLocal,
		Enabled:      true,
	}
	db.Create(&job)

	// 测试使用任务的 cron 表达式
	req := httptest.NewRequest("GET", "/jobs/1/next-runs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	respData := resp["data"].(map[string]interface{})
	runs := respData["next_runs"].([]interface{})
	if len(runs) != 5 {
		t.Errorf("Expected 5 next runs, got %d", len(runs))
	}
}

// TestJobHandler_NextRuns_WithCustomCron 测试自定义 cron 表达式
func TestJobHandler_NextRuns_WithCustomCron(t *testing.T) {
	db := setupTestDB(t)
	handler := NewJobHandler(db, crypto.NewNoOpEncryptor())
	router := setupTestRouter()

	router.GET("/jobs/:id/next-runs", handler.NextRuns)

	// 创建测试任务
	job := model.BackupJob{
		Name:         "test-cron2",
		DatabaseType: model.DatabaseTypePostgres,
		Host:         "localhost",
		Port:         5432,
		Database:     "test",
		Schedule:     "0 2 * * *",
		StorageType:  model.StorageTypeLocal,
		Enabled:      true,
	}
	db.Create(&job)

	// 使用自定义 cron 参数
	req := httptest.NewRequest("GET", "/jobs/1/next-runs", nil)
	q := req.URL.Query()
	q.Set("cron", "0 */6 * * *")
	req.URL.RawQuery = q.Encode()
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	respData := resp["data"].(map[string]interface{})
	runs := respData["next_runs"].([]interface{})
	if len(runs) != 5 {
		t.Errorf("Expected 5 next runs, got %d", len(runs))
	}
}

// TestJobHandler_NextRuns_InvalidCron 测试无效 cron 表达式
func TestJobHandler_NextRuns_InvalidCron(t *testing.T) {
	db := setupTestDB(t)
	handler := NewJobHandler(db, crypto.NewNoOpEncryptor())
	router := setupTestRouter()

	router.GET("/jobs/:id/next-runs", handler.NextRuns)

	job := model.BackupJob{
		Name:         "test-cron3",
		DatabaseType: model.DatabaseTypePostgres,
		Host:         "localhost",
		Port:         5432,
		Database:     "test",
		Schedule:     "invalid",
		StorageType:  model.StorageTypeLocal,
		Enabled:      true,
	}
	db.Create(&job)

	// 测试无效 cron
	req := httptest.NewRequest("GET", "/jobs/1/next-runs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	// 测试自定义无效 cron
	req = httptest.NewRequest("GET", "/jobs/1/next-runs?cron=invalid-cron", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}
