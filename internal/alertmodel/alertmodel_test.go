package alertmodel

import (
	"testing"
	"time"
)

func TestNotificationChannel_TableName(t *testing.T) {
	ch := NotificationChannel{}
	if ch.TableName() != "alert_notification_channels" {
		t.Errorf("expected table name 'alert_notification_channels', got '%s'", ch.TableName())
	}
}

func TestNotificationChannel_GetSetConfig(t *testing.T) {
	ch := &NotificationChannel{}

	cfg := &ChannelConfig{
		WebhookURL: "https://example.com/webhook",
		Keyword:    "test",
		Secret:     "secret123",
	}

	// 测试设置配置
	err := ch.SetConfig(cfg)
	if err != nil {
		t.Fatalf("SetConfig failed: %v", err)
	}

	// 测试获取配置
	result, err := ch.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}

	if result.WebhookURL != cfg.WebhookURL {
		t.Errorf("expected WebhookURL '%s', got '%s'", cfg.WebhookURL, result.WebhookURL)
	}
	if result.Keyword != cfg.Keyword {
		t.Errorf("expected Keyword '%s', got '%s'", cfg.Keyword, result.Keyword)
	}
	if result.Secret != cfg.Secret {
		t.Errorf("expected Secret '%s', got '%s'", cfg.Secret, result.Secret)
	}
}

func TestNotificationChannel_GetConfig_Empty(t *testing.T) {
	ch := &NotificationChannel{Config: ""}
	cfg, err := ch.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
}

func TestNotificationChannel_IsHealthy(t *testing.T) {
	tests := []struct {
		status   HealthStatus
		expected bool
	}{
		{HealthStatusHealthy, true},
		{HealthStatusUnhealthy, false},
		{HealthStatusUnknown, false},
	}

	for _, tt := range tests {
		ch := &NotificationChannel{HealthStatus: tt.status}
		if ch.IsHealthy() != tt.expected {
			t.Errorf("IsHealthy() for status %s: expected %v, got %v", tt.status, tt.expected, ch.IsHealthy())
		}
	}
}

func TestNotificationChannel_IsEnabled(t *testing.T) {
	tests := []struct {
		enabled  bool
		expected bool
	}{
		{true, true},
		{false, false},
	}

	for _, tt := range tests {
		ch := &NotificationChannel{Enabled: tt.enabled}
		if ch.IsEnabled() != tt.expected {
			t.Errorf("IsEnabled() for enabled %v: expected %v, got %v", tt.enabled, tt.expected, ch.IsEnabled())
		}
	}
}

func TestNotificationChannel_ToResponse(t *testing.T) {
	now := time.Now()
	ch := &NotificationChannel{
		ID:            1,
		Name:          "Test Channel",
		Type:          ChannelTypeFeishu,
		Config:        `{"webhook_url":"https://example.com","keyword":"test","secret":"supersecret"}`,
		Enabled:       true,
		Priority:      10,
		Description:   "Test description",
		HealthStatus:  HealthStatusHealthy,
		LastSentAt:    &now,
		LastError:     "",
		SendCount:     100,
		FailedCount:   2,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// 测试脱敏
	resp, err := ch.ToResponse(true)
	if err != nil {
		t.Fatalf("ToResponse failed: %v", err)
	}

	if resp.ID != ch.ID {
		t.Errorf("expected ID %d, got %d", ch.ID, resp.ID)
	}
	if resp.Name != ch.Name {
		t.Errorf("expected Name '%s', got '%s'", ch.Name, resp.Name)
	}
	if resp.ConfigSummary == nil {
		t.Fatal("expected non-nil ConfigSummary")
	}
	if resp.ConfigSummary.Secret != "******" {
		t.Errorf("expected secret to be masked, got '%s'", resp.ConfigSummary.Secret)
	}

	// 测试不脱敏
	respNoMask, err := ch.ToResponse(false)
	if err != nil {
		t.Fatalf("ToResponse(false) failed: %v", err)
	}
	if respNoMask.ConfigSummary.Secret != "supersecret" {
		t.Errorf("expected secret not masked, got '%s'", respNoMask.ConfigSummary.Secret)
	}
}

func TestAlertRule_TableName(t *testing.T) {
	rule := AlertRule{}
	if rule.TableName() != "alert_rules" {
		t.Errorf("expected table name 'alert_rules', got '%s'", rule.TableName())
	}
}

func TestAlertRule_GetSetConditions(t *testing.T) {
	rule := &AlertRule{}

	conditions := []Condition{
		{Field: "db_type", Operator: "eq", Value: "mysql"},
		{Field: "status", Operator: "eq", Value: "failed"},
	}

	err := rule.SetConditions(conditions)
	if err != nil {
		t.Fatalf("SetConditions failed: %v", err)
	}

	result, err := rule.GetConditions()
	if err != nil {
		t.Fatalf("GetConditions failed: %v", err)
	}

	if len(result) != len(conditions) {
		t.Fatalf("expected %d conditions, got %d", len(conditions), len(result))
	}
	if result[0].Field != conditions[0].Field {
		t.Errorf("expected Field '%s', got '%s'", conditions[0].Field, result[0].Field)
	}
}

func TestAlertRule_GetSetChannels(t *testing.T) {
	rule := &AlertRule{}

	channels := []int64{1, 2, 3}

	err := rule.SetChannels(channels)
	if err != nil {
		t.Fatalf("SetChannels failed: %v", err)
	}

	result, err := rule.GetChannels()
	if err != nil {
		t.Fatalf("GetChannels failed: %v", err)
	}

	if len(result) != len(channels) {
		t.Fatalf("expected %d channels, got %d", len(channels), len(result))
	}
}

func TestAlertRule_GetSetSilentPeriods(t *testing.T) {
	rule := &AlertRule{}

	periods := []SilentPeriod{
		{StartTime: "00:00", EndTime: "06:00", Days: []int{0, 6}},
		{StartTime: "12:00", EndTime: "13:00", Days: []int{1, 2, 3, 4, 5}},
	}

	err := rule.SetSilentPeriods(periods)
	if err != nil {
		t.Fatalf("SetSilentPeriods failed: %v", err)
	}

	result, err := rule.GetSilentPeriods()
	if err != nil {
		t.Fatalf("GetSilentPeriods failed: %v", err)
	}

	if len(result) != len(periods) {
		t.Fatalf("expected %d periods, got %d", len(periods), len(result))
	}
}

func TestAlertRule_GetSetEscalateChannels(t *testing.T) {
	rule := &AlertRule{}

	channels := []int64{5, 6}

	err := rule.SetEscalateChannels(channels)
	if err != nil {
		t.Fatalf("SetEscalateChannels failed: %v", err)
	}

	result, err := rule.GetEscalateChannels()
	if err != nil {
		t.Fatalf("GetEscalateChannels failed: %v", err)
	}

	if len(result) != len(channels) {
		t.Fatalf("expected %d channels, got %d", len(channels), len(result))
	}
}

func TestAlertRule_ToResponse(t *testing.T) {
	now := time.Now()
	rule := &AlertRule{
		ID:            1,
		Name:          "Test Rule",
		Description:   "Test description",
		Enabled:       true,
		Priority:      80,
		Level:         AlertLevelP1,
		ConditionOp:   ConditionOpAND,
		Conditions:    `[{"field":"db_type","operator":"eq","value":"mysql"}]`,
		Channels:      `[1,2]`,
		Cooldown:      300,
		SilentPeriods:  `[{"start_time":"00:00","end_time":"06:00","days":[0,6]}]`,
		EscalateEnabled: true,
		EscalateTimeout: 30,
		EscalateLevel:   AlertLevelP0,
		EscalateChannels: `[3]`,
		MatchedCount:   15,
		LastMatchedAt:  &now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	resp, err := rule.ToResponse()
	if err != nil {
		t.Fatalf("ToResponse failed: %v", err)
	}

	if resp.ID != rule.ID {
		t.Errorf("expected ID %d, got %d", rule.ID, resp.ID)
	}
	if resp.Name != rule.Name {
		t.Errorf("expected Name '%s', got '%s'", rule.Name, resp.Name)
	}
	if resp.Level != rule.Level {
		t.Errorf("expected Level '%s', got '%s'", rule.Level, resp.Level)
	}
	if len(resp.Conditions) != 1 {
		t.Errorf("expected 1 condition, got %d", len(resp.Conditions))
	}
	if len(resp.Channels) != 2 {
		t.Errorf("expected 2 channels, got %d", len(resp.Channels))
	}
	if resp.Escalate == nil {
		t.Fatal("expected non-nil Escalate")
	}
	if resp.Escalate.TargetLevel != AlertLevelP0 {
		t.Errorf("expected Escalate.TargetLevel '%s', got '%s'", AlertLevelP0, resp.Escalate.TargetLevel)
	}
}

func TestAlertRecord_TableName(t *testing.T) {
	record := AlertRecord{}
	if record.TableName() != "alert_records" {
		t.Errorf("expected table name 'alert_records', got '%s'", record.TableName())
	}
}

func TestAlertRecord_ToResponse(t *testing.T) {
	now := time.Now()
	record := &AlertRecord{
		ID:        1,
		RuleID:    10,
		TaskID:    int64Ptr(100),
		Level:     AlertLevelP1,
		Status:    AlertStatusActive,
		Title:     "Test Alert",
		Content:   "Test content",
		EventType: EventTypeBackupFailed,
		TaskName:  "mysql_daily",
		DBType:    "mysql",
		Source:    "192.168.1.100:3306",
		TriggeredAt: now,
	}
	record.Rule = &AlertRule{Name: "Test Rule"}

	resp := record.ToResponse()

	if resp.ID != record.ID {
		t.Errorf("expected ID %d, got %d", record.ID, resp.ID)
	}
	if resp.RuleName != "Test Rule" {
		t.Errorf("expected RuleName 'Test Rule', got '%s'", resp.RuleName)
	}
	if *resp.TaskID != 100 {
		t.Errorf("expected TaskID 100, got %d", *resp.TaskID)
	}
}

func TestAlertNotificationRecord_TableName(t *testing.T) {
	rec := AlertNotificationRecord{}
	if rec.TableName() != "alert_notification_records" {
		t.Errorf("expected table name 'alert_notification_records', got '%s'", rec.TableName())
	}
}

func TestAlertNote_TableName(t *testing.T) {
	note := AlertNote{}
	if note.TableName() != "alert_notes" {
		t.Errorf("expected table name 'alert_notes', got '%s'", note.TableName())
	}
}

func TestAlertRuleStats_TableName(t *testing.T) {
	stats := AlertRuleStats{}
	if stats.TableName() != "alert_rule_stats" {
		t.Errorf("expected table name 'alert_rule_stats', got '%s'", stats.TableName())
	}
}

func TestAlertEvent_ToJSON_ParseAlertEvent(t *testing.T) {
	event := &AlertEvent{
		EventType: EventTypeBackupFailed,
		TaskID:    123,
		TaskName:  "mysql_backup",
		DBType:    "mysql",
		Source:    "192.168.1.100:3306",
		Level:     AlertLevelP1,
		Title:     "Backup Failed",
		Content:   "MySQL backup failed",
		Extra:     map[string]interface{}{"error": "timeout"},
	}

	// 序列化
	jsonStr, err := event.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// 反序列化
	parsed, err := ParseAlertEvent(jsonStr)
	if err != nil {
		t.Fatalf("ParseAlertEvent failed: %v", err)
	}

	if parsed.EventType != event.EventType {
		t.Errorf("expected EventType '%s', got '%s'", event.EventType, parsed.EventType)
	}
	if parsed.TaskID != event.TaskID {
		t.Errorf("expected TaskID %d, got %d", event.TaskID, parsed.TaskID)
	}
	if parsed.TaskName != event.TaskName {
		t.Errorf("expected TaskName '%s', got '%s'", event.TaskName, parsed.TaskName)
	}
}

// Helper function
func int64Ptr(i int64) *int64 {
	return &i
}
