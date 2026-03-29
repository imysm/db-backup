package approval

import (
	"encoding/json"
	"testing"
	"time"
)

func TestApprovalType_Values(t *testing.T) {
	tests := []struct {
		typ    ApprovalType
		method string
	}{
		{ApprovalTypeRestore, "restore"},
		{ApprovalTypeTaskDelete, "task_delete"},
		{ApprovalTypeConfigChange, "config_change"},
	}

	for _, tt := range tests {
		if string(tt.typ) != tt.method {
			t.Errorf("expected %s, got %s", tt.method, tt.typ)
		}
	}
}

func TestApprovalStatus_Values(t *testing.T) {
	statuses := []ApprovalStatus{
		ApprovalStatusPending,
		ApprovalStatusApproved,
		ApprovalStatusRejected,
		ApprovalStatusCancelled,
		ApprovalStatusExecuted,
	}

	expected := []string{"pending", "approved", "rejected", "cancelled", "executed"}

	for i, status := range statuses {
		if string(status) != expected[i] {
			t.Errorf("expected %s, got %s", expected[i], status)
		}
	}
}

func TestApproval_TableName(t *testing.T) {
	approval := Approval{}
	if approval.TableName() != "approvals" {
		t.Errorf("expected 'approvals', got '%s'", approval.TableName())
	}
}

func TestApproval_Fields(t *testing.T) {
	now := time.Now()
	approval := Approval{
		ID:             1,
		UUID:          "test-uuid-123",
		Type:          ApprovalTypeRestore,
		Status:        ApprovalStatusPending,
		ApplicantID:   100,
		ApplicantName: "admin",
		ResourceType:  "backup_record",
		ResourceID:    200,
		ResourceName:  "test-backup",
		Title:         "恢复审批",
		Content:       "需要恢复测试环境",
		AppliedAt:     now,
		CreatedAt:     now,
	}

	if approval.ID != 1 {
		t.Errorf("expected ID 1, got %d", approval.ID)
	}
	if approval.UUID != "test-uuid-123" {
		t.Errorf("expected UUID 'test-uuid-123', got '%s'", approval.UUID)
	}
	if approval.Type != ApprovalTypeRestore {
		t.Errorf("expected type 'restore', got '%s'", approval.Type)
	}
	if approval.Status != ApprovalStatusPending {
		t.Errorf("expected status 'pending', got '%s'", approval.Status)
	}
	if approval.ApplicantName != "admin" {
		t.Errorf("expected 'admin', got '%s'", approval.ApplicantName)
	}
	if approval.Title != "恢复审批" {
		t.Errorf("expected '恢复审批', got '%s'", approval.Title)
	}
}

func TestApproval_JSON(t *testing.T) {
	approval := Approval{
		ID:             1,
		UUID:          "test-uuid",
		Type:          ApprovalTypeRestore,
		Status:        ApprovalStatusPending,
		ApplicantID:   100,
		ApplicantName: "admin",
	}

	data, err := json.Marshal(approval)
	if err != nil {
		t.Errorf("failed to marshal: %v", err)
	}

	var unmarshaled Approval
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("failed to unmarshal: %v", err)
	}

	if unmarshaled.UUID != approval.UUID {
		t.Errorf("UUID mismatch")
	}
}

func TestApproval_StatusTransition(t *testing.T) {
	approval := Approval{Status: ApprovalStatusPending}

	// Test pending -> approved
	approval.Status = ApprovalStatusApproved
	if approval.Status != ApprovalStatusApproved {
		t.Errorf("expected 'approved', got '%s'", approval.Status)
	}

	// Test approved -> executed
	approval.Status = ApprovalStatusExecuted
	if approval.Status != ApprovalStatusExecuted {
		t.Errorf("expected 'executed', got '%s'", approval.Status)
	}
}

func TestApproval_ApproveNote(t *testing.T) {
	approval := Approval{
		ApproveNote: "同意恢复",
	}

	if approval.ApproveNote != "同意恢复" {
		t.Errorf("expected '同意恢复', got '%s'", approval.ApproveNote)
	}
}

func TestApproval_RejectReason(t *testing.T) {
	approval := Approval{
		RejectReason: "数据不可恢复",
	}

	if approval.RejectReason != "数据不可恢复" {
		t.Errorf("expected '数据不可恢复', got '%s'", approval.RejectReason)
	}
}

func TestApproval_TitleAndContent(t *testing.T) {
	approval := Approval{
		Title:   "紧急恢复审批",
		Content: "生产数据库需要紧急恢复",
	}

	if approval.Title != "紧急恢复审批" {
		t.Errorf("expected '紧急恢复审批', got '%s'", approval.Title)
	}
	if approval.Content != "生产数据库需要紧急恢复" {
		t.Errorf("expected '生产数据库需要紧急恢复', got '%s'", approval.Content)
	}
}
