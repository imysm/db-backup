package audit

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAuditDetails_ToJSON(t *testing.T) {
	details := &AuditDetails{
		Before: map[string]interface{}{
			"name": "old_name",
		},
		After: map[string]interface{}{
			"name": "new_name",
		},
		Extra: map[string]interface{}{
			"changed_by": "admin",
		},
	}

	jsonStr := details.ToJSON()
	if jsonStr == "" {
		t.Error("ToJSON() returned empty string")
	}

	// 验证可以解析回去
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Errorf("ToJSON() produced invalid JSON: %v", err)
	}
}

func TestParseDetails(t *testing.T) {
	tests := []struct {
		name    string
		jsonStr string
		wantErr bool
	}{
		{
			name:    "valid json",
			jsonStr: `{"before":{"name":"old"},"after":{"name":"new"}}`,
			wantErr: false,
		},
		{
			name:    "empty json",
			jsonStr: "",
			wantErr: false,
		},
		{
			name:    "invalid json",
			jsonStr: `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			details, err := ParseDetails(tt.jsonStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDetails() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && details == nil {
				t.Error("ParseDetails() returned nil for non-error case")
			}
		})
	}
}

func TestNewAuditDetails(t *testing.T) {
	details := NewAuditDetails()
	
	if details == nil {
		t.Fatal("NewAuditDetails() returned nil")
	}
	
	if details.Before == nil {
		t.Error("NewAuditDetails().Before is nil")
	}
	
	if details.After == nil {
		t.Error("NewAuditDetails().After is nil")
	}
	
	if details.Extra == nil {
		t.Error("NewAuditDetails().Extra is nil")
	}
}

func TestAuditDetails_EmptyJSON(t *testing.T) {
	details := &AuditDetails{}
	jsonStr := details.ToJSON()
	
	if jsonStr != "{}" {
		t.Errorf("Empty AuditDetails ToJSON() = %v, want {}", jsonStr)
	}
}

func TestAuditLog_TableName(t *testing.T) {
	log := AuditLog{}
	if name := log.TableName(); name != "audit_logs" {
		t.Errorf("AuditLog TableName() = %v, want audit_logs", name)
	}
}

func TestAuditConstants(t *testing.T) {
	// 验证操作类型常量
	if ActionCreate != "CREATE" {
		t.Errorf("ActionCreate = %v, want CREATE", ActionCreate)
	}
	if ActionUpdate != "UPDATE" {
		t.Errorf("ActionUpdate = %v, want UPDATE", ActionUpdate)
	}
	if ActionDelete != "DELETE" {
		t.Errorf("ActionDelete = %v, want DELETE", ActionDelete)
	}
	if ActionExecute != "EXECUTE" {
		t.Errorf("ActionExecute = %v, want EXECUTE", ActionExecute)
	}
	if ActionQuery != "QUERY" {
		t.Errorf("ActionQuery = %v, want QUERY", ActionQuery)
	}
	if ActionLogin != "LOGIN" {
		t.Errorf("ActionLogin = %v, want LOGIN", ActionLogin)
	}
	if ActionLogout != "LOGOUT" {
		t.Errorf("ActionLogout = %v, want LOGOUT", ActionLogout)
	}

	// 验证结果常量
	if ResultSuccess != "success" {
		t.Errorf("ResultSuccess = %v, want success", ResultSuccess)
	}
	if ResultFailure != "failure" {
		t.Errorf("ResultFailure = %v, want failure", ResultFailure)
	}
	if ResultPartial != "partial" {
		t.Errorf("ResultPartial = %v, want partial", ResultPartial)
	}
}

func TestResourceTypes(t *testing.T) {
	if ResourceJob != "job" {
		t.Errorf("ResourceJob = %v, want job", ResourceJob)
	}
	if ResourceRecord != "record" {
		t.Errorf("ResourceRecord = %v, want record", ResourceRecord)
	}
	if ResourceTemplate != "template" {
		t.Errorf("ResourceTemplate = %v, want template", ResourceTemplate)
	}
	if ResourceRestore != "restore" {
		t.Errorf("ResourceRestore = %v, want restore", ResourceRestore)
	}
	if ResourceMerge != "merge" {
		t.Errorf("ResourceMerge = %v, want merge", ResourceMerge)
	}
	if ResourceUser != "user" {
		t.Errorf("ResourceUser = %v, want user", ResourceUser)
	}
	if ResourceTenant != "tenant" {
		t.Errorf("ResourceTenant = %v, want tenant", ResourceTenant)
	}
	if ResourceSetting != "setting" {
		t.Errorf("ResourceSetting = %v, want setting", ResourceSetting)
	}
}

func TestAuditQuery_Fields(t *testing.T) {
	now := time.Now()
	userID := uint(1)
	tenantID := uint(2)
	resourceID := uint(3)

	query := &AuditQuery{
		UserID:     &userID,
		TenantID:   &tenantID,
		Action:     "CREATE",
		Resource:   "job",
		ResourceID: &resourceID,
		Result:     "success",
		StartTime:  &now,
		EndTime:    &now,
		Keyword:    "test",
		RequestID:  "req-123",
		Offset:     10,
		Limit:      20,
		OrderBy:    "created_at",
		OrderDesc:  true,
	}

	if query.UserID == nil || *query.UserID != userID {
		t.Error("AuditQuery.UserID not set correctly")
	}
	if query.Action != "CREATE" {
		t.Error("AuditQuery.Action not set correctly")
	}
	if query.Offset != 10 {
		t.Error("AuditQuery.Offset not set correctly")
	}
	if query.Limit != 20 {
		t.Error("AuditQuery.Limit not set correctly")
	}
}

func TestUserActivity_Fields(t *testing.T) {
	activity := &UserActivity{
		Total:        100,
		SuccessCount: 95,
		FailureCount: 5,
		Actions: map[string]int64{
			"CREATE": 50,
			"UPDATE": 30,
			"DELETE": 20,
		},
	}

	if activity.Total != 100 {
		t.Errorf("UserActivity.Total = %v, want 100", activity.Total)
	}
	if activity.SuccessCount != 95 {
		t.Errorf("UserActivity.SuccessCount = %v, want 95", activity.SuccessCount)
	}
	if activity.Actions["CREATE"] != 50 {
		t.Errorf("UserActivity.Actions[CREATE] = %v, want 50", activity.Actions["CREATE"])
	}
}
