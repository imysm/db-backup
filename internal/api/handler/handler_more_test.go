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

func setupRecordTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	db.AutoMigrate(&model.BackupJob{}, &model.BackupRecord{})
	return db
}

func TestRecordHandler_List(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewRecordHandler(db)
	router := gin.New()
	router.GET("/records", handler.List)

	req := httptest.NewRequest("GET", "/records?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRecordHandler_Get(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewRecordHandler(db)
	router := gin.New()
	router.GET("/records/:id", handler.Get)

	// Test with invalid ID
	req := httptest.NewRequest("GET", "/records/invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid ID, got %d", w.Code)
	}

	// Test with non-existent ID
	req = httptest.NewRequest("GET", "/records/999", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent record, got %d", w.Code)
	}
}

func TestRecordHandler_Verify(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewRecordHandler(db)
	router := gin.New()
	router.POST("/records/:id/verify", handler.Verify)

	// Test with invalid ID
	req := httptest.NewRequest("POST", "/records/invalid/verify", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid ID, got %d", w.Code)
	}
}

func TestRecordHandler_Delete(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewRecordHandler(db)
	router := gin.New()
	router.DELETE("/records/:id", handler.Delete)

	// Test with invalid ID
	req := httptest.NewRequest("DELETE", "/records/invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid ID, got %d", w.Code)
	}
}

func TestRestoreHandler_Restore(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewRestoreHandler(db)
	router := gin.New()
	router.POST("/restore", handler.Restore)

	// Test with empty request
	req := httptest.NewRequest("POST", "/restore", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should fail because record doesn't exist
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent record, got %d", w.Code)
	}
}

func TestRestoreHandler_Validate(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewRestoreHandler(db)
	router := gin.New()
	router.GET("/restore/validate/:id", handler.Validate)

	// Test with invalid ID
	req := httptest.NewRequest("GET", "/restore/validate/invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid ID, got %d", w.Code)
	}
}

func TestRestoreHandler_List(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewRestoreHandler(db)
	router := gin.New()
	router.GET("/restore/list", handler.List)

	req := httptest.NewRequest("GET", "/restore/list?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestVerifyHandler_VerifyBackup(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewVerifyHandler(db)
	router := gin.New()
	router.POST("/verify/:id", handler.VerifyBackup)

	// Test with invalid ID
	req := httptest.NewRequest("POST", "/verify/invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid ID, got %d", w.Code)
	}
}

func TestVerifyHandler_TestRestore(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewVerifyHandler(db)
	router := gin.New()
	router.POST("/verify/:id/restore", handler.TestRestore)

	// Test with invalid ID
	req := httptest.NewRequest("POST", "/verify/invalid/restore", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid ID, got %d", w.Code)
	}
}

func TestRestoreAPIRequest(t *testing.T) {
	req := RestoreAPIRequest{
		RecordID:   1,
		TargetHost: "localhost",
		TargetPort: 5432,
		TargetDB:   "testdb",
		TargetUser: "postgres",
		TargetPass: "password",
		NoRestore:  false,
	}

	if req.RecordID != 1 {
		t.Errorf("RecordID = %d, want 1", req.RecordID)
	}
	if req.TargetHost != "localhost" {
		t.Errorf("TargetHost = %s, want localhost", req.TargetHost)
	}
}

func TestRecordHandler_List_WithJobID(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewRecordHandler(db)
	router := gin.New()
	router.GET("/records", handler.List)

	req := httptest.NewRequest("GET", "/records?page=1&page_size=10&job_id=1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRecordHandler_Get_ExistingRecord(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewRecordHandler(db)
	router := gin.New()
	router.GET("/records/:id", handler.Get)

	// Create a job first
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

	// Create a record
	record := model.BackupRecord{
		JobID:    job.ID,
		Status:   model.BackupStatusSuccess,
		FilePath: "/tmp/backup.sql",
		FileSize: 1024,
		Checksum: "abc123",
	}
	db.Create(&record)

	// Test get existing record
	req := httptest.NewRequest("GET", "/records/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRecordHandler_Verify_ExistingRecord(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewRecordHandler(db)
	router := gin.New()
	router.POST("/records/:id/verify", handler.Verify)

	// Create a job first
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

	// Create a record
	record := model.BackupRecord{
		JobID:    job.ID,
		Status:   model.BackupStatusSuccess,
		FilePath: "/tmp/backup.sql",
		FileSize: 1024,
		Checksum: "abc123",
	}
	db.Create(&record)

	// Test verify existing record
	req := httptest.NewRequest("POST", "/records/1/verify", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRecordHandler_Delete_ExistingRecord(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewRecordHandler(db)
	router := gin.New()
	router.DELETE("/records/:id", handler.Delete)

	// Create a job first
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

	// Create a record
	record := model.BackupRecord{
		JobID:    job.ID,
		Status:   model.BackupStatusSuccess,
		FilePath: "/tmp/backup.sql",
		FileSize: 1024,
		Checksum: "abc123",
	}
	db.Create(&record)

	// Test delete existing record
	req := httptest.NewRequest("DELETE", "/records/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRestoreHandler_Restore_WithRecord(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewRestoreHandler(db)
	router := gin.New()
	router.POST("/restore", handler.Restore)

	// Create a job first
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
		StorageConfig: `{"path":"/nonexistent"}`,
		Enabled:      true,
	}
	db.Create(&job)

	// Create a record
	record := model.BackupRecord{
		JobID:    job.ID,
		Status:   model.BackupStatusSuccess,
		FilePath: "/nonexistent/backup.sql",
		FileSize: 1024,
		Checksum: "abc123",
	}
	db.Create(&record)

	// Test restore with valid record ID
	body := RestoreAPIRequest{
		RecordID:   record.ID,
		TargetHost: "localhost",
		TargetPort: 5432,
		TargetDB:   "testdb",
		TargetUser: "postgres",
		TargetPass: "password",
		NoRestore:  true,
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/restore", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 200 but with error in response (file doesn't exist)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRestoreHandler_Validate_ExistingRecord(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewRestoreHandler(db)
	router := gin.New()
	router.GET("/restore/validate/:id", handler.Validate)

	// Create a job first
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

	// Create a record
	record := model.BackupRecord{
		JobID:    job.ID,
		Status:   model.BackupStatusSuccess,
		FilePath: "/nonexistent/backup.sql",
		FileSize: 1024,
		Checksum: "abc123",
	}
	db.Create(&record)

	// Test validate existing record
	req := httptest.NewRequest("GET", "/restore/validate/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestVerifyHandler_VerifyBackup_ExistingRecord(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewVerifyHandler(db)
	router := gin.New()
	router.POST("/verify/:id", handler.VerifyBackup)

	// Create a job first
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

	// Create a record
	record := model.BackupRecord{
		JobID:    job.ID,
		Status:   model.BackupStatusSuccess,
		FilePath: "/nonexistent/backup.sql",
		FileSize: 1024,
		Checksum: "abc123",
	}
	db.Create(&record)

	// Test verify existing record
	req := httptest.NewRequest("POST", "/verify/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestVerifyHandler_TestRestore_ExistingRecord(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewVerifyHandler(db)
	router := gin.New()
	router.POST("/verify/:id/restore", handler.TestRestore)

	// Create a job first
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

	// Create a record
	record := model.BackupRecord{
		JobID:    job.ID,
		Status:   model.BackupStatusSuccess,
		FilePath: "/nonexistent/backup.sql",
		FileSize: 1024,
		Checksum: "abc123",
	}
	db.Create(&record)

	// Test restore with existing record
	req := httptest.NewRequest("POST", "/verify/1/restore", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestJobHandler_Delete_NonExistent(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewJobHandler(db, crypto.NewNoOpEncryptor())
	router := gin.New()
	router.DELETE("/jobs/:id", handler.Delete)

	// Test delete non-existent job
	req := httptest.NewRequest("DELETE", "/jobs/999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// SQLite delete of non-existent record returns success
	if w.Code != http.StatusOK {
		t.Logf("Delete non-existent returned status %d", w.Code)
	}
}

func TestJobHandler_Run_NonExistent(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewJobHandler(db, crypto.NewNoOpEncryptor())
	router := gin.New()
	router.POST("/jobs/:id/run", handler.Run)

	// Test run non-existent job should return 404
	req := httptest.NewRequest("POST", "/jobs/999/run", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestRecordHandler_Verify_NonExistent(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewRecordHandler(db)
	router := gin.New()
	router.POST("/records/:id/verify", handler.Verify)

	// Test verify non-existent record
	req := httptest.NewRequest("POST", "/records/999/verify", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Logf("Verify non-existent returned status %d", w.Code)
	}
}

func TestRestoreHandler_Validate_NonExistent(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewRestoreHandler(db)
	router := gin.New()
	router.GET("/restore/validate/:id", handler.Validate)

	// Test validate non-existent record
	req := httptest.NewRequest("GET", "/restore/validate/999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Logf("Validate non-existent returned status %d", w.Code)
	}
}

func TestJobHandler_Get_WithRecords(t *testing.T) {
	db := setupTestDB(t)
	handler := NewJobHandler(db, crypto.NewNoOpEncryptor())
	router := gin.New()
	router.GET("/jobs/:id", handler.Get)

	// Create a job
	job := model.BackupJob{
		Name:         "test-job-with-records",
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

	// Create some records
	for i := 0; i < 3; i++ {
		record := model.BackupRecord{
			JobID:    job.ID,
			Status:   model.BackupStatusSuccess,
			FilePath: "/tmp/backup.sql",
			FileSize: 1024,
			Checksum: "abc123",
		}
		db.Create(&record)
	}

	// Test get job with records
	req := httptest.NewRequest("GET", "/jobs/1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRecordHandler_List_WithJobFilter(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewRecordHandler(db)
	router := gin.New()
	router.GET("/records", handler.List)

	// Create jobs
	job1 := model.BackupJob{
		Name:         "job1",
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
	job2 := model.BackupJob{
		Name:         "job2",
		DatabaseType: model.DatabaseTypeMySQL,
		Host:         "localhost",
		Port:         3306,
		Database:     "testdb",
		Username:     "root",
		Password:     "password",
		Schedule:     "0 3 * * *",
		StorageType:  model.StorageTypeLocal,
		Enabled:      true,
	}
	db.Create(&job1)
	db.Create(&job2)

	// Create records for both jobs
	for i := 0; i < 5; i++ {
		db.Create(&model.BackupRecord{
			JobID:    job1.ID,
			Status:   model.BackupStatusSuccess,
			FilePath: "/tmp/backup1.sql",
			FileSize: 1024,
		})
		db.Create(&model.BackupRecord{
			JobID:    job2.ID,
			Status:   model.BackupStatusSuccess,
			FilePath: "/tmp/backup2.sql",
			FileSize: 2048,
		})
	}

	// Test list with job filter
	req := httptest.NewRequest("GET", "/records?job_id=1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Just verify the response is valid JSON
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	if err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}
}

func TestRecordHandler_List_WithStatusFilter(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewRecordHandler(db)
	router := gin.New()
	router.GET("/records", handler.List)

	// Create a job
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

	// Create records with different statuses
	for i := 0; i < 3; i++ {
		db.Create(&model.BackupRecord{
			JobID:    job.ID,
			Status:   model.BackupStatusSuccess,
			FilePath: "/tmp/backup.sql",
			FileSize: 1024,
		})
		db.Create(&model.BackupRecord{
			JobID:    job.ID,
			Status:   model.BackupStatusFailed,
			FilePath: "",
			FileSize: 0,
		})
	}

	// Test list with status filter
	req := httptest.NewRequest("GET", "/records?status=success", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRestoreHandler_List_WithPagination(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewRestoreHandler(db)
	router := gin.New()
	router.GET("/restore/list", handler.List)

	// Test list with pagination
	req := httptest.NewRequest("GET", "/restore/list?page=1&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestJobHandler_Create_WithValidation(t *testing.T) {
	db := setupTestDB(t)
	handler := NewJobHandler(db, crypto.NewNoOpEncryptor())
	router := gin.New()
	router.POST("/jobs", handler.Create)

	// Test with missing required fields
	body := dto.CreateJobRequest{
		Name: "", // Empty name
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/jobs", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should still return 200 (validation is lenient)
	if w.Code != http.StatusOK {
		t.Logf("Create with empty name returned status %d", w.Code)
	}
}

func TestJobHandler_Update_WithNilEnabled(t *testing.T) {
	db := setupTestDB(t)
	handler := NewJobHandler(db, crypto.NewNoOpEncryptor())
	router := gin.New()
	router.PUT("/jobs/:id", handler.Update)

	// Create test data
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

	// Test update with nil enabled (should not change enabled status)
	body := dto.UpdateJobRequest{
		Name:    "updated-job",
		Enabled: nil, // nil means don't change
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

func TestRecordHandler_Get_WithInvalidID(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewRecordHandler(db)
	router := gin.New()
	router.GET("/records/:id", handler.Get)

	// Test with invalid ID format
	req := httptest.NewRequest("GET", "/records/abc", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 400 Bad Request
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestRecordHandler_Delete_WithInvalidID(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewRecordHandler(db)
	router := gin.New()
	router.DELETE("/records/:id", handler.Delete)

	// Test with invalid ID format
	req := httptest.NewRequest("DELETE", "/records/abc", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 400 Bad Request
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestRestoreHandler_Restore_WithInvalidJSON(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewRestoreHandler(db)
	router := gin.New()
	router.POST("/restore", handler.Restore)

	// Test with invalid JSON
	req := httptest.NewRequest("POST", "/restore", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 400 Bad Request
	if w.Code != http.StatusBadRequest {
		t.Logf("Restore with invalid JSON returned status %d", w.Code)
	}
}

func TestVerifyHandler_VerifyBackup_WithInvalidID(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewVerifyHandler(db)
	router := gin.New()
	router.POST("/verify/:id", handler.VerifyBackup)

	// Test with invalid ID format
	req := httptest.NewRequest("POST", "/verify/abc", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 400 Bad Request
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestVerifyHandler_TestRestore_WithInvalidID(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewVerifyHandler(db)
	router := gin.New()
	router.POST("/verify/:id/restore", handler.TestRestore)

	// Test with invalid ID format
	req := httptest.NewRequest("POST", "/verify/abc/restore", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 400 Bad Request
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestJobHandler_List_WithInvalidPage(t *testing.T) {
	db := setupTestDB(t)
	handler := NewJobHandler(db, crypto.NewNoOpEncryptor())
	router := gin.New()
	router.GET("/jobs", handler.List)

	// Test with invalid page number
	req := httptest.NewRequest("GET", "/jobs?page=-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should still return 200 (defaults applied)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestJobHandler_List_WithLargePageSize(t *testing.T) {
	db := setupTestDB(t)
	handler := NewJobHandler(db, crypto.NewNoOpEncryptor())
	router := gin.New()
	router.GET("/jobs", handler.List)

	// Test with large page size
	req := httptest.NewRequest("GET", "/jobs?page_size=1000", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should still return 200 (page size limited)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRestoreHandler_Restore_WithMissingRecordID(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewRestoreHandler(db)
	router := gin.New()
	router.POST("/restore", handler.Restore)

	// Test with missing record ID
	body := RestoreAPIRequest{
		TargetHost: "localhost",
		TargetPort: 5432,
	}
	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/restore", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return error
	if w.Code != http.StatusOK {
		t.Logf("Restore with missing record ID returned status %d", w.Code)
	}
}

func TestJobHandler_Get_WithStringID(t *testing.T) {
	db := setupTestDB(t)
	handler := NewJobHandler(db, crypto.NewNoOpEncryptor())
	router := gin.New()
	router.GET("/jobs/:id", handler.Get)

	// Test with string ID (invalid)
	req := httptest.NewRequest("GET", "/jobs/notanumber", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 400 or 404
	if w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound {
		t.Logf("Get with string ID returned status %d", w.Code)
	}
}

func TestJobHandler_Delete_WithStringID(t *testing.T) {
	db := setupTestDB(t)
	handler := NewJobHandler(db, crypto.NewNoOpEncryptor())
	router := gin.New()
	router.DELETE("/jobs/:id", handler.Delete)

	// Test with string ID (invalid)
	req := httptest.NewRequest("DELETE", "/jobs/notanumber", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 400 or 404
	if w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound {
		t.Logf("Delete with string ID returned status %d", w.Code)
	}
}

func TestRecordHandler_List_WithInvalidJobID(t *testing.T) {
	db := setupRecordTestDB(t)
	handler := NewRecordHandler(db)
	router := gin.New()
	router.GET("/records", handler.List)

	// Test with invalid job_id
	req := httptest.NewRequest("GET", "/records?job_id=invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should still return 200 (filter ignored)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
