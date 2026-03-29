package audit

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	host := getEnv("TEST_PG_HOST", "localhost")
	port := getEnv("TEST_PG_PORT", "5432")
	user := getEnv("TEST_PG_USER", "malert")
	password := getEnv("TEST_PG_PASSWORD", "MalertP@ssw0rd")
	dbname := getEnv("TEST_PG_DBNAME", "malert_backup")
	schema := getEnv("TEST_PG_SCHEMA", "test")

	dsn := "host=" + host + " port=" + port + " user=" + user + " password=" + password + " dbname=" + dbname + " options=-csearch_path=" + schema + " sslmode=disable"

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Create schema if not exists
	db.Exec("CREATE SCHEMA IF NOT EXISTS " + schema)

	err = db.AutoMigrate(&AuditLog{})
	require.NoError(t, err)

	// Clean up before each test
	db.Exec("DELETE FROM audit_logs")

	return db
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func TestAuditService_Log(t *testing.T) {
	if os.Getenv("RUN_PG_TESTS") != "true" {
		t.Skip("Skipping PostgreSQL test. Set RUN_PG_TESTS=true to run")
	}

	db := setupTestDB(t)
	svc := NewAuditService(db)

	log := &AuditLog{
		Username:     "testuser",
		Action:       ActionCreate,
		Resource:     "job",
		ResourceName: "test-job",
		Result:       ResultSuccess,
	}

	err := svc.Log(log)
	assert.NoError(t, err)
	assert.NotZero(t, log.ID)

	// Verify it was saved
	var saved AuditLog
	err = db.First(&saved, log.ID).Error
	assert.NoError(t, err)
	assert.Equal(t, "testuser", saved.Username)
}

func TestAuditService_LogWithDetails(t *testing.T) {
	if os.Getenv("RUN_PG_TESTS") != "true" {
		t.Skip("Skipping PostgreSQL test. Set RUN_PG_TESTS=true to run")
	}

	db := setupTestDB(t)
	svc := NewAuditService(db)

	userID := uint(1)
	tenantID := uint(2)
	resourceID := uint(3)
	details := NewAuditDetails()
	details.Before["name"] = "old"
	details.After["name"] = "new"

	err := svc.LogWithDetails(
		&userID, "admin", &tenantID,
		ActionUpdate, "job", &resourceID, "my-job",
		details, ResultSuccess, "",
	)

	assert.NoError(t, err)

	// Verify
	var saved AuditLog
	err = db.First(&saved).Error
	assert.NoError(t, err)
	assert.Equal(t, userID, *saved.UserID)
	assert.Equal(t, "admin", saved.Username)
	assert.Equal(t, ActionUpdate, saved.Action)
	assert.NotEmpty(t, saved.Details)
}

func TestAuditService_GetByID(t *testing.T) {
	if os.Getenv("RUN_PG_TESTS") != "true" {
		t.Skip("Skipping PostgreSQL test. Set RUN_PG_TESTS=true to run")
	}

	db := setupTestDB(t)
	svc := NewAuditService(db)

	// Create a log
	log := &AuditLog{
		Username:     "test",
		Action:       ActionCreate,
		Resource:     "job",
		ResourceName: "test",
		Result:       ResultSuccess,
	}
	db.Create(log)

	// Get by ID
	found, err := svc.GetByID(log.ID)
	assert.NoError(t, err)
	assert.Equal(t, log.ID, found.ID)
	assert.Equal(t, "test", found.Username)

	// Non-existent ID
	_, err = svc.GetByID(99999)
	assert.Error(t, err)
}

func TestAuditService_GetByRequestID(t *testing.T) {
	if os.Getenv("RUN_PG_TESTS") != "true" {
		t.Skip("Skipping PostgreSQL test. Set RUN_PG_TESTS=true to run")
	}

	db := setupTestDB(t)
	svc := NewAuditService(db)

	requestID := "req-123-abc"
	for i := 0; i < 3; i++ {
		db.Create(&AuditLog{
			RequestID:    requestID,
			Username:     "test",
			Action:       ActionCreate,
			Resource:     "job",
			ResourceName: "test",
			Result:       ResultSuccess,
		})
	}

	logs, err := svc.GetByRequestID(requestID)
	assert.NoError(t, err)
	assert.Len(t, logs, 3)
}

func TestAuditService_GetByResource(t *testing.T) {
	if os.Getenv("RUN_PG_TESTS") != "true" {
		t.Skip("Skipping PostgreSQL test. Set RUN_PG_TESTS=true to run")
	}

	db := setupTestDB(t)
	svc := NewAuditService(db)

	jobID := uint(1)
	// Create logs for same resource
	for i := 0; i < 3; i++ {
		db.Create(&AuditLog{
			Resource:     "job",
			ResourceID:   &jobID,
			Username:     "test",
			Action:       ActionUpdate,
			ResourceName: "test",
			Result:       ResultSuccess,
		})
	}
	// Create logs for different resource
	db.Create(&AuditLog{
		Resource:     "record",
		ResourceID:   uintPtr(2),
		Username:     "test",
		Action:       ActionCreate,
		ResourceName: "test",
		Result:       ResultSuccess,
	})

	logs, err := svc.GetByResource("job", jobID, 0)
	assert.NoError(t, err)
	assert.Len(t, logs, 3)

	// With limit
	logs, err = svc.GetByResource("job", jobID, 2)
	assert.NoError(t, err)
	assert.Len(t, logs, 2)
}

func TestAuditService_GetUserActivity(t *testing.T) {
	if os.Getenv("RUN_PG_TESTS") != "true" {
		t.Skip("Skipping PostgreSQL test. Set RUN_PG_TESTS=true to run")
	}

	db := setupTestDB(t)
	svc := NewAuditService(db)

	userID := uint(1)
	now := time.Now()

	// Create logs
	db.Create(&AuditLog{
		UserID:   &userID,
		Action:   ActionCreate,
		Resource: "job",
		Result:   ResultSuccess,
	})
	db.Create(&AuditLog{
		UserID:   &userID,
		Action:   ActionCreate,
		Resource: "job",
		Result:   ResultSuccess,
	})
	db.Create(&AuditLog{
		UserID:   &userID,
		Action:   ActionDelete,
		Resource: "job",
		Result:   ResultFailure,
	})

	activity, err := svc.GetUserActivity(userID, now.Add(-time.Hour), now.Add(time.Hour))
	assert.NoError(t, err)
	assert.Equal(t, int64(3), activity.Total)
	assert.Equal(t, int64(2), activity.SuccessCount)
	assert.Equal(t, int64(1), activity.FailureCount)
	assert.Equal(t, int64(2), activity.Actions[ActionCreate])
	assert.Equal(t, int64(1), activity.Actions[ActionDelete])
}

func TestAuditService_CleanOldLogs(t *testing.T) {
	if os.Getenv("RUN_PG_TESTS") != "true" {
		t.Skip("Skipping PostgreSQL test. Set RUN_PG_TESTS=true to run")
	}

	db := setupTestDB(t)
	svc := NewAuditService(db)

	// Create old logs (2 days ago)
	oldTime := time.Now().AddDate(0, 0, -2)
	db.Create(&AuditLog{
		Username:     "old",
		Action:       ActionCreate,
		Resource:     "job",
		ResourceName: "old",
		Result:       ResultSuccess,
		CreatedAt:    oldTime,
	})

	// Create new logs (today)
	db.Create(&AuditLog{
		Username:     "new",
		Action:       ActionCreate,
		Resource:     "job",
		ResourceName: "new",
		Result:       ResultSuccess,
	})

	// Clean logs older than 1 day
	deleted, err := svc.CleanOldLogs(1)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	// Verify only new log remains
	var count int64
	db.Model(&AuditLog{}).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestParseOrderColumn(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"created_at", "created_at"},
		{"username", "username"},
		{"invalid/path", "invalid"}, // splitPath splits by /
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseOrderColumn(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func uintPtr(v uint) *uint {
	return &v
}
