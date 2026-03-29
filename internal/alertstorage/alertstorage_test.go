package alertstorage

import (
	"github.com/imysm/db-backup/internal/alertmodel"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupTestDB 创建测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// 自动迁移
	err = db.AutoMigrate(
		&alertmodel.NotificationChannel{},
		&alertmodel.AlertRule{},
		&alertmodel.AlertRecord{},
		&alertmodel.AlertNotificationRecord{},
		&alertmodel.AlertNote{},
	)
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	return db
}

func TestChannelStorage_Create_GetByID(t *testing.T) {
	db := setupTestDB(t)
	storage := NewChannelStorage(db)

	channel := &alertmodel.NotificationChannel{
		Name:           "Test Channel",
		Type:           alertmodel.ChannelTypeFeishu,
		Config:         `{"webhook_url":"https://example.com","keyword":"test"}`,
		Enabled:        true,
		Priority:       10,
		HealthStatus:   alertmodel.HealthStatusUnknown,
		CreatedBy:      "test",
	}

	err := storage.Create(channel)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if channel.ID == 0 {
		t.Fatal("expected non-zero ID after create")
	}

	// 测试 GetByID
	result, err := storage.GetByID(channel.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Name != channel.Name {
		t.Errorf("expected Name '%s', got '%s'", channel.Name, result.Name)
	}
	if result.Type != channel.Type {
		t.Errorf("expected Type '%s', got '%s'", channel.Type, result.Type)
	}
}

func TestChannelStorage_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	storage := NewChannelStorage(db)

	result, err := storage.GetByID(999)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if result != nil {
		t.Error("expected nil result for non-existent ID")
	}
}

func TestChannelStorage_List(t *testing.T) {
	db := setupTestDB(t)
	storage := NewChannelStorage(db)

	// 创建多个渠道
	channels := []*alertmodel.NotificationChannel{
		{Name: "Channel 1", Type: alertmodel.ChannelTypeFeishu, Enabled: true, Priority: 10, CreatedBy: "test"},
		{Name: "Channel 2", Type: alertmodel.ChannelTypeWeCom, Enabled: true, Priority: 20, CreatedBy: "test"},
		{Name: "Channel 3", Type: alertmodel.ChannelTypeDingTalk, Enabled: false, Priority: 30, CreatedBy: "test"},
	}

	for _, ch := range channels {
		if err := storage.Create(ch); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// 测试 List - 全部
	results, total, err := storage.List(nil, nil, 1, 10)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	// 测试 List - 按类型过滤
	results, total, err = storage.List(ptrChannelType(alertmodel.ChannelTypeFeishu), nil, 1, 10)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}

	// 测试 List - 按启用状态过滤
	enabledVal := true
	results, total, err = storage.List(nil, &enabledVal, 1, 10)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// 测试分页
	results, total, err = storage.List(nil, nil, 1, 2)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestChannelStorage_ListEnabled(t *testing.T) {
	db := setupTestDB(t)
	storage := NewChannelStorage(db)

	channels := []*alertmodel.NotificationChannel{
		{Name: "Enabled 1", Type: alertmodel.ChannelTypeFeishu, Enabled: true, Priority: 10, CreatedBy: "test"},
		{Name: "Disabled", Type: alertmodel.ChannelTypeWeCom, Enabled: false, Priority: 20, CreatedBy: "test"},
		{Name: "Enabled 2", Type: alertmodel.ChannelTypeDingTalk, Enabled: true, Priority: 30, CreatedBy: "test"},
	}

	for _, ch := range channels {
		if err := storage.Create(ch); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	results, err := storage.ListEnabled()
	if err != nil {
		t.Fatalf("ListEnabled failed: %v", err)
	}
	// 验证只返回启用的渠道
	for _, ch := range results {
		if !ch.Enabled {
			t.Errorf("expected all channels to be enabled, got disabled: %s", ch.Name)
		}
	}
	// 验证按优先级排序
	if len(results) >= 2 && results[0].Priority < results[1].Priority {
		t.Error("expected results sorted by priority DESC")
	}
}

func TestChannelStorage_GetByIDs(t *testing.T) {
	db := setupTestDB(t)
	storage := NewChannelStorage(db)

	channels := []*alertmodel.NotificationChannel{
		{Name: "Channel 1", Type: alertmodel.ChannelTypeFeishu, Enabled: true, CreatedBy: "test"},
		{Name: "Channel 2", Type: alertmodel.ChannelTypeWeCom, Enabled: true, CreatedBy: "test"},
	}

	for _, ch := range channels {
		if err := storage.Create(ch); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	results, err := storage.GetByIDs([]int64{channels[0].ID, channels[1].ID})
	if err != nil {
		t.Fatalf("GetByIDs failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestChannelStorage_Update(t *testing.T) {
	db := setupTestDB(t)
	storage := NewChannelStorage(db)

	channel := &alertmodel.NotificationChannel{
		Name:      "Original Name",
		Type:      alertmodel.ChannelTypeFeishu,
		Enabled:   true,
		Priority:  10,
		CreatedBy: "test",
	}
	if err := storage.Create(channel); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// 更新
	newName := "Updated Name"
	channel.Name = newName
	channel.Priority = 20
	if err := storage.Update(channel); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// 验证
	result, err := storage.GetByID(channel.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if result.Name != newName {
		t.Errorf("expected Name '%s', got '%s'", newName, result.Name)
	}
	if result.Priority != 20 {
		t.Errorf("expected Priority 20, got %d", result.Priority)
	}
}

func TestChannelStorage_Delete(t *testing.T) {
	db := setupTestDB(t)
	storage := NewChannelStorage(db)

	channel := &alertmodel.NotificationChannel{
		Name:      "To Delete",
		Type:      alertmodel.ChannelTypeFeishu,
		Enabled:   true,
		CreatedBy: "test",
	}
	if err := storage.Create(channel); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// 删除
	if err := storage.Delete(channel.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// 验证 - 应该找不到
	result, err := storage.GetByID(channel.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if result != nil {
		t.Error("expected nil result after delete")
	}
}

func TestChannelStorage_UpdateHealthStatus(t *testing.T) {
	db := setupTestDB(t)
	storage := NewChannelStorage(db)

	channel := &alertmodel.NotificationChannel{
		Name:      "Health Check",
		Type:      alertmodel.ChannelTypeFeishu,
		Enabled:   true,
		CreatedBy: "test",
	}
	if err := storage.Create(channel); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// 更新健康状态
	err := storage.UpdateHealthStatus(channel.ID, alertmodel.HealthStatusHealthy, "")
	if err != nil {
		t.Fatalf("UpdateHealthStatus failed: %v", err)
	}

	// 验证
	result, err := storage.GetByID(channel.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if result.HealthStatus != alertmodel.HealthStatusHealthy {
		t.Errorf("expected HealthStatus '%s', got '%s'", alertmodel.HealthStatusHealthy, result.HealthStatus)
	}
}

func TestRuleStorage_Create_GetByID(t *testing.T) {
	db := setupTestDB(t)
	storage := NewRuleStorage(db)

	rule := &alertmodel.AlertRule{
		Name:        "Test Rule",
		Description: "Test description",
		Enabled:     true,
		Priority:    50,
		Level:       alertmodel.AlertLevelP1,
		ConditionOp: alertmodel.ConditionOpAND,
		Conditions:  `[{"field":"status","operator":"eq","value":"failed"}]`,
		Channels:    `[1]`,
		Cooldown:    300,
		CreatedBy:   "test",
	}

	err := storage.Create(rule)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if rule.ID == 0 {
		t.Fatal("expected non-zero ID after create")
	}

	// 测试 GetByID
	result, err := storage.GetByID(rule.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Name != rule.Name {
		t.Errorf("expected Name '%s', got '%s'", rule.Name, result.Name)
	}
}

func TestRuleStorage_List(t *testing.T) {
	db := setupTestDB(t)
	storage := NewRuleStorage(db)

	enabled := true
	rules := []*alertmodel.AlertRule{
		{Name: "Rule 1", Enabled: enabled, Level: alertmodel.AlertLevelP1, Conditions: `[]`, Channels: `[]`, CreatedBy: "test"},
		{Name: "Rule 2", Enabled: enabled, Level: alertmodel.AlertLevelP2, Conditions: `[]`, Channels: `[]`, CreatedBy: "test"},
		{Name: "Rule 3", Enabled: false, Level: alertmodel.AlertLevelP1, Conditions: `[]`, Channels: `[]`, CreatedBy: "test"},
	}

	for _, r := range rules {
		if err := storage.Create(r); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// 测试 List - 全部
	_, total, err := storage.List(nil, nil, 1, 10)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}

	// 测试按级别过滤
	_, total, err = storage.List(nil, ptrAlertLevel(alertmodel.AlertLevelP1), 1, 10)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
}

func TestRuleStorage_IncrementMatchedCount(t *testing.T) {
	db := setupTestDB(t)
	storage := NewRuleStorage(db)

	rule := &alertmodel.AlertRule{
		Name:       "Match Test",
		Enabled:    true,
		Level:      alertmodel.AlertLevelP1,
		Conditions: `[]`,
		Channels:   `[]`,
		CreatedBy:  "test",
	}
	if err := storage.Create(rule); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// 初始计数应该是 0
	if rule.MatchedCount != 0 {
		t.Errorf("expected initial MatchedCount 0, got %d", rule.MatchedCount)
	}

	// 增加计数
	err := storage.IncrementMatchedCount(rule.ID)
	if err != nil {
		t.Fatalf("IncrementMatchedCount failed: %v", err)
	}

	// 验证
	result, err := storage.GetByID(rule.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if result.MatchedCount != 1 {
		t.Errorf("expected MatchedCount 1, got %d", result.MatchedCount)
	}
	if result.LastMatchedAt == nil {
		t.Error("expected LastMatchedAt to be set")
	}
}

func TestRuleStorage_Copy(t *testing.T) {
	db := setupTestDB(t)
	storage := NewRuleStorage(db)

	rule := &alertmodel.AlertRule{
		Name:          "Original Rule",
		Description:   "Original description",
		Enabled:       true,
		Priority:      80,
		Level:         alertmodel.AlertLevelP1,
		ConditionOp:   alertmodel.ConditionOpAND,
		Conditions:    `[{"field":"status","operator":"eq","value":"failed"}]`,
		Channels:      `[1,2]`,
		Cooldown:      300,
		EscalateEnabled: true,
		EscalateTimeout: 30,
		CreatedBy:     "test",
	}
	if err := storage.Create(rule); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// 复制规则
	copy, err := storage.Copy(rule.ID, "Copied Rule", "tester")
	if err != nil {
		t.Fatalf("Copy failed: %v", err)
	}
	if copy == nil {
		t.Fatal("expected non-nil copy")
	}

	// 验证副本
	if copy.ID == rule.ID {
		t.Error("expected copy to have different ID")
	}
	if copy.Name != "Copied Rule" {
		t.Errorf("expected Name 'Copied Rule', got '%s'", copy.Name)
	}
	if copy.Enabled {
		t.Error("expected copy to be disabled")
	}
	if copy.Priority != rule.Priority {
		t.Errorf("expected Priority %d, got %d", rule.Priority, copy.Priority)
	}
	if copy.EscalateEnabled != rule.EscalateEnabled {
		t.Error("expected EscalateEnabled to be copied")
	}
}

func TestRuleStorage_Export(t *testing.T) {
	db := setupTestDB(t)
	storage := NewRuleStorage(db)

	rules := []*alertmodel.AlertRule{
		{Name: "Rule 1", Level: alertmodel.AlertLevelP1, Conditions: `[]`, Channels: `[]`, CreatedBy: "test"},
		{Name: "Rule 2", Level: alertmodel.AlertLevelP2, Conditions: `[]`, Channels: `[]`, CreatedBy: "test"},
	}

	for _, r := range rules {
		if err := storage.Create(r); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// 导出全部
	results, err := storage.Export(nil)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// 按 ID 导出
	results, err = storage.Export([]int64{rules[0].ID})
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestRecordStorage_Create_GetByID(t *testing.T) {
	db := setupTestDB(t)
	storage := NewRecordStorage(db)

	// 先创建关联的规则
	rule := &alertmodel.AlertRule{
		Name:       "Test Rule",
		Level:      alertmodel.AlertLevelP1,
		Conditions: `[]`,
		Channels:   `[]`,
		CreatedBy:  "test",
	}
	if err := NewRuleStorage(db).Create(rule); err != nil {
		t.Fatalf("Create rule failed: %v", err)
	}

	now := time.Now()
	record := &alertmodel.AlertRecord{
		RuleID:      rule.ID,
		Level:       alertmodel.AlertLevelP1,
		Title:       "Test Alert",
		Content:     "Test content",
		EventType:   alertmodel.EventTypeBackupFailed,
		Status:      alertmodel.AlertStatusActive,
		TriggeredAt: now,
	}

	err := storage.Create(record)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if record.ID == 0 {
		t.Fatal("expected non-zero ID after create")
	}

	// 测试 GetByID
	result, err := storage.GetByID(record.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Title != record.Title {
		t.Errorf("expected Title '%s', got '%s'", record.Title, result.Title)
	}
}

func TestRecordStorage_List(t *testing.T) {
	db := setupTestDB(t)
	storage := NewRecordStorage(db)

	// 先创建关联的规则
	rule := &alertmodel.AlertRule{
		Name:       "Test Rule",
		Level:      alertmodel.AlertLevelP1,
		Conditions: `[]`,
		Channels:   `[]`,
		CreatedBy:  "test",
	}
	if err := NewRuleStorage(db).Create(rule); err != nil {
		t.Fatalf("Create rule failed: %v", err)
	}

	records := []*alertmodel.AlertRecord{
		{RuleID: rule.ID, Level: alertmodel.AlertLevelP1, Title: "Alert 1", Content: "content", EventType: alertmodel.EventTypeBackupFailed, Status: alertmodel.AlertStatusActive, TriggeredAt: time.Now()},
		{RuleID: rule.ID, Level: alertmodel.AlertLevelP2, Title: "Alert 2", Content: "content", EventType: alertmodel.EventTypeBackupFailed, Status: alertmodel.AlertStatusActive, TriggeredAt: time.Now()},
		{RuleID: rule.ID, Level: alertmodel.AlertLevelP1, Title: "Alert 3", Content: "content", EventType: alertmodel.EventTypeBackupFailed, Status: alertmodel.AlertStatusResolved, TriggeredAt: time.Now()},
	}

	for _, r := range records {
		if err := storage.Create(r); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// 测试 List - 全部
	_, total, err := storage.List(&alertmodel.ListAlertsRequest{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 3 {
		t.Errorf("expected total 3, got %d", total)
	}

	// 测试按级别过滤
	_, total, err = storage.List(&alertmodel.ListAlertsRequest{Level: alertmodel.AlertLevelP1, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}

	// 测试按状态过滤
	_, total, err = storage.List(&alertmodel.ListAlertsRequest{Status: alertmodel.AlertStatusActive, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
}

func TestRecordStorage_CountActiveTotal(t *testing.T) {
	db := setupTestDB(t)
	storage := NewRecordStorage(db)

	// 先创建关联的规则
	rule := &alertmodel.AlertRule{
		Name:       "Test Rule",
		Level:      alertmodel.AlertLevelP1,
		Conditions: `[]`,
		Channels:   `[]`,
		CreatedBy:  "test",
	}
	if err := NewRuleStorage(db).Create(rule); err != nil {
		t.Fatalf("Create rule failed: %v", err)
	}

	records := []*alertmodel.AlertRecord{
		{RuleID: rule.ID, Level: alertmodel.AlertLevelP1, Title: "Alert 1", Content: "content", EventType: alertmodel.EventTypeBackupFailed, Status: alertmodel.AlertStatusActive, TriggeredAt: time.Now()},
		{RuleID: rule.ID, Level: alertmodel.AlertLevelP1, Title: "Alert 2", Content: "content", EventType: alertmodel.EventTypeBackupFailed, Status: alertmodel.AlertStatusActive, TriggeredAt: time.Now()},
		{RuleID: rule.ID, Level: alertmodel.AlertLevelP1, Title: "Alert 3", Content: "content", EventType: alertmodel.EventTypeBackupFailed, Status: alertmodel.AlertStatusResolved, TriggeredAt: time.Now()},
	}

	for _, r := range records {
		if err := storage.Create(r); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	count, err := storage.CountActiveTotal()
	if err != nil {
		t.Fatalf("CountActiveTotal failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}
}

func TestRecordStorage_CountActiveByLevel(t *testing.T) {
	db := setupTestDB(t)
	storage := NewRecordStorage(db)

	// 先创建关联的规则
	rule := &alertmodel.AlertRule{
		Name:       "Test Rule",
		Level:      alertmodel.AlertLevelP1,
		Conditions: `[]`,
		Channels:   `[]`,
		CreatedBy:  "test",
	}
	if err := NewRuleStorage(db).Create(rule); err != nil {
		t.Fatalf("Create rule failed: %v", err)
	}

	records := []*alertmodel.AlertRecord{
		{RuleID: rule.ID, Level: alertmodel.AlertLevelP0, Title: "Alert P0", Content: "content", EventType: alertmodel.EventTypeBackupFailed, Status: alertmodel.AlertStatusActive, TriggeredAt: time.Now()},
		{RuleID: rule.ID, Level: alertmodel.AlertLevelP1, Title: "Alert P1-1", Content: "content", EventType: alertmodel.EventTypeBackupFailed, Status: alertmodel.AlertStatusActive, TriggeredAt: time.Now()},
		{RuleID: rule.ID, Level: alertmodel.AlertLevelP1, Title: "Alert P1-2", Content: "content", EventType: alertmodel.EventTypeBackupFailed, Status: alertmodel.AlertStatusActive, TriggeredAt: time.Now()},
		{RuleID: rule.ID, Level: alertmodel.AlertLevelP2, Title: "Alert P2", Content: "content", EventType: alertmodel.EventTypeBackupFailed, Status: alertmodel.AlertStatusResolved, TriggeredAt: time.Now()},
	}

	for _, r := range records {
		if err := storage.Create(r); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	counts, err := storage.CountActiveByLevel()
	if err != nil {
		t.Fatalf("CountActiveByLevel failed: %v", err)
	}
	if counts[alertmodel.AlertLevelP0] != 1 {
		t.Errorf("expected P0 count 1, got %d", counts[alertmodel.AlertLevelP0])
	}
	if counts[alertmodel.AlertLevelP1] != 2 {
		t.Errorf("expected P1 count 2, got %d", counts[alertmodel.AlertLevelP1])
	}
	if counts[alertmodel.AlertLevelP2] != 0 {
		t.Errorf("expected P2 count 0, got %d", counts[alertmodel.AlertLevelP2])
	}
}

func TestRecordStorage_Acknowledge(t *testing.T) {
	db := setupTestDB(t)
	storage := NewRecordStorage(db)

	// 先创建关联的规则
	rule := &alertmodel.AlertRule{
		Name:       "Test Rule",
		Level:      alertmodel.AlertLevelP1,
		Conditions: `[]`,
		Channels:   `[]`,
		CreatedBy:  "test",
	}
	if err := NewRuleStorage(db).Create(rule); err != nil {
		t.Fatalf("Create rule failed: %v", err)
	}

	record := &alertmodel.AlertRecord{
		RuleID:      rule.ID,
		Level:       alertmodel.AlertLevelP1,
		Title:       "Alert to Ack",
		Content:     "content",
		EventType:   alertmodel.EventTypeBackupFailed,
		Status:      alertmodel.AlertStatusActive,
		TriggeredAt: time.Now(),
	}
	if err := storage.Create(record); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// 确认告警
	err := storage.Acknowledge(record.ID, "admin", "Looking into it")
	if err != nil {
		t.Fatalf("Acknowledge failed: %v", err)
	}

	// 验证
	result, err := storage.GetByID(record.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if result.Status != alertmodel.AlertStatusAcknowledged {
		t.Errorf("expected Status '%s', got '%s'", alertmodel.AlertStatusAcknowledged, result.Status)
	}
	if result.AcknowledgedBy != "admin" {
		t.Errorf("expected AcknowledgedBy 'admin', got '%s'", result.AcknowledgedBy)
	}
}

func TestRecordStorage_Resolve(t *testing.T) {
	db := setupTestDB(t)
	storage := NewRecordStorage(db)

	// 先创建关联的规则
	rule := &alertmodel.AlertRule{
		Name:       "Test Rule",
		Level:      alertmodel.AlertLevelP1,
		Conditions: `[]`,
		Channels:   `[]`,
		CreatedBy:  "test",
	}
	if err := NewRuleStorage(db).Create(rule); err != nil {
		t.Fatalf("Create rule failed: %v", err)
	}

	record := &alertmodel.AlertRecord{
		RuleID:      rule.ID,
		Level:       alertmodel.AlertLevelP1,
		Title:       "Alert to Resolve",
		Content:     "content",
		EventType:   alertmodel.EventTypeBackupFailed,
		Status:      alertmodel.AlertStatusActive,
		TriggeredAt: time.Now(),
	}
	if err := storage.Create(record); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// 解决告警
	err := storage.Resolve(record.ID, "admin", "Fixed by restarting")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// 验证
	result, err := storage.GetByID(record.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if result.Status != alertmodel.AlertStatusResolved {
		t.Errorf("expected Status '%s', got '%s'", alertmodel.AlertStatusResolved, result.Status)
	}
	if result.ResolvedBy != "admin" {
		t.Errorf("expected ResolvedBy 'admin', got '%s'", result.ResolvedBy)
	}
}

func TestRecordStorage_AddNote(t *testing.T) {
	db := setupTestDB(t)
	storage := NewRecordStorage(db)

	// 先创建关联的规则
	rule := &alertmodel.AlertRule{
		Name:       "Test Rule",
		Level:      alertmodel.AlertLevelP1,
		Conditions: `[]`,
		Channels:   `[]`,
		CreatedBy:  "test",
	}
	if err := NewRuleStorage(db).Create(rule); err != nil {
		t.Fatalf("Create rule failed: %v", err)
	}

	record := &alertmodel.AlertRecord{
		RuleID:      rule.ID,
		Level:       alertmodel.AlertLevelP1,
		Title:       "Alert for Notes",
		Content:     "content",
		EventType:   alertmodel.EventTypeBackupFailed,
		Status:      alertmodel.AlertStatusActive,
		TriggeredAt: time.Now(),
	}
	if err := storage.Create(record); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// 添加备注
	note, err := storage.AddNote(record.ID, "This is a test note", "tester")
	if err != nil {
		t.Fatalf("AddNote failed: %v", err)
	}
	if note.ID == 0 {
		t.Error("expected non-zero note ID")
	}

	// 获取备注列表
	notes, err := storage.GetNotes(record.ID)
	if err != nil {
		t.Fatalf("GetNotes failed: %v", err)
	}
	if len(notes) != 1 {
		t.Errorf("expected 1 note, got %d", len(notes))
	}
	if notes[0].Content != "This is a test note" {
		t.Errorf("expected content 'This is a test note', got '%s'", notes[0].Content)
	}
}

// Helper functions
func ptrChannelType(v alertmodel.ChannelType) *alertmodel.ChannelType {
	return &v
}

func ptrAlertLevel(v alertmodel.AlertLevel) *alertmodel.AlertLevel {
	return &v
}

func ptrBool(v bool) *bool {
	return &v
}
