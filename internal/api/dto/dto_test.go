package dto

import "testing"

func TestSuccess(t *testing.T) {
	data := map[string]string{"key": "value"}
	resp := Success(data)

	if resp.Code != 0 {
		t.Errorf("Code = %d, want 0", resp.Code)
	}
	if resp.Message != "success" {
		t.Errorf("Message = %s, want success", resp.Message)
	}
	if resp.Data == nil {
		t.Error("Data should not be nil")
	}
}

func TestError(t *testing.T) {
	resp := Error("test error")

	if resp.Code != 1 {
		t.Errorf("Code = %d, want 1", resp.Code)
	}
	if resp.Message != "test error" {
		t.Errorf("Message = %s, want 'test error'", resp.Message)
	}
	if resp.Data != nil {
		t.Error("Data should be nil for error response")
	}
}

func TestResponse(t *testing.T) {
	resp := Response{
		Code:    0,
		Message: "test",
		Data:    "data",
	}

	if resp.Code != 0 {
		t.Errorf("Code = %d, want 0", resp.Code)
	}
	if resp.Message != "test" {
		t.Errorf("Message = %s, want test", resp.Message)
	}
}

func TestCreateJobRequest(t *testing.T) {
	req := CreateJobRequest{
		Name:            "test-job",
		DatabaseType:    "postgres",
		Host:            "localhost",
		Port:            5432,
		Database:        "testdb",
		Username:        "postgres",
		Password:        "password",
		Schedule:        "0 2 * * *",
		RetentionDays:   7,
		BackupType:      "full",
		StorageType:     "local",
		Compress:        true,
		Encrypt:         false,
		NotifyOnSuccess: true,
		NotifyOnFail:    true,
	}

	if req.Name != "test-job" {
		t.Errorf("Name = %s, want test-job", req.Name)
	}
	if req.DatabaseType != "postgres" {
		t.Errorf("DatabaseType = %s, want postgres", req.DatabaseType)
	}
	if req.Port != 5432 {
		t.Errorf("Port = %d, want 5432", req.Port)
	}
}

func TestUpdateJobRequest(t *testing.T) {
	enabled := true
	req := UpdateJobRequest{
		Name:            "updated-job",
		Schedule:        "0 3 * * *",
		RetentionDays:   14,
		StorageType:     "s3",
		Compress:        true,
		Encrypt:         false,
		Enabled:         &enabled,
		NotifyOnSuccess: false,
		NotifyOnFail:    true,
	}

	if req.Name != "updated-job" {
		t.Errorf("Name = %s, want updated-job", req.Name)
	}
	if req.Schedule != "0 3 * * *" {
		t.Errorf("Schedule = %s, want '0 3 * * *'", req.Schedule)
	}
	if req.Enabled == nil || *req.Enabled != true {
		t.Error("Enabled should be true")
	}
}

func TestJobResponse(t *testing.T) {
	resp := JobResponse{
		ID:            1,
		Name:          "test-job",
		DatabaseType:  "postgres",
		Host:          "localhost",
		Port:          5432,
		Database:      "testdb",
		Schedule:      "0 2 * * *",
		RetentionDays: 7,
		BackupType:    "full",
		StorageType:   "local",
		Compress:      true,
		Encrypt:       false,
		Enabled:       true,
	}

	if resp.ID != 1 {
		t.Errorf("ID = %d, want 1", resp.ID)
	}
	if resp.Name != "test-job" {
		t.Errorf("Name = %s, want test-job", resp.Name)
	}
}

func TestRecordResponse(t *testing.T) {
	resp := RecordResponse{
		ID:       1,
		JobID:    1,
		JobName:  "test-job",
		Status:   "success",
		FileSize: 1024,
		FilePath: "/tmp/backup.sql",
		Checksum: "abc123",
		Verified: true,
	}

	if resp.ID != 1 {
		t.Errorf("ID = %d, want 1", resp.ID)
	}
	if resp.Status != "success" {
		t.Errorf("Status = %s, want success", resp.Status)
	}
	if resp.FileSize != 1024 {
		t.Errorf("FileSize = %d, want 1024", resp.FileSize)
	}
}
