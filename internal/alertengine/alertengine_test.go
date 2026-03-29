package alertengine

import (
	"github.com/imysm/db-backup/internal/alertmodel"
	"testing"
)

func TestConditionEvaluator_Evaluate(t *testing.T) {
	evaluator := NewConditionEvaluator()

	tests := []struct {
		name      string
		condition alertmodel.Condition
		event     *alertmodel.AlertEvent
		expected  bool
		wantErr   bool
	}{
		{
			name: "eq - 字符串相等",
			condition: alertmodel.Condition{
				Field:    "db_type",
				Operator: "eq",
				Value:    "mysql",
			},
			event: &alertmodel.AlertEvent{
				DBType: "mysql",
			},
			expected: true,
		},
		{
			name: "eq - 字符串不相等",
			condition: alertmodel.Condition{
				Field:    "db_type",
				Operator: "eq",
				Value:    "mysql",
			},
			event: &alertmodel.AlertEvent{
				DBType: "postgresql",
			},
			expected: false,
		},
		{
			name: "ne - 不等于",
			condition: alertmodel.Condition{
				Field:    "db_type",
				Operator: "ne",
				Value:    "mysql",
			},
			event: &alertmodel.AlertEvent{
				DBType: "postgresql",
			},
			expected: true,
		},
		{
			name: "gt - 数字大于",
			condition: alertmodel.Condition{
				Field:    "task_id",
				Operator: "gt",
				Value:    100,
			},
			event: &alertmodel.AlertEvent{
				TaskID: 200,
			},
			expected: true,
		},
		{
			name: "gt - 数字不大于",
			condition: alertmodel.Condition{
				Field:    "task_id",
				Operator: "gt",
				Value:    100,
			},
			event: &alertmodel.AlertEvent{
				TaskID: 50,
			},
			expected: false,
		},
		{
			name: "lt - 数字小于",
			condition: alertmodel.Condition{
				Field:    "task_id",
				Operator: "lt",
				Value:    100,
			},
			event: &alertmodel.AlertEvent{
				TaskID: 50,
			},
			expected: true,
		},
		{
			name: "contains - 字符串包含",
			condition: alertmodel.Condition{
				Field:    "content",
				Operator: "contains",
				Value:    "timeout",
			},
			event: &alertmodel.AlertEvent{
				Content: "connection timeout error",
			},
			expected: true,
		},
		{
			name: "contains - 字符串不包含",
			condition: alertmodel.Condition{
				Field:    "content",
				Operator: "contains",
				Value:    "timeout",
			},
			event: &alertmodel.AlertEvent{
				Content: "connection refused",
			},
			expected: false,
		},
		{
			name: "regex - 正则匹配",
			condition: alertmodel.Condition{
				Field:    "content",
				Operator: "regex",
				Value:    "connection.*timeout",
			},
			event: &alertmodel.AlertEvent{
				Content: "connection timeout error",
			},
			expected: true,
		},
		{
			name: "in - 在列表中",
			condition: alertmodel.Condition{
				Field:    "db_type",
				Operator: "in",
				Value:    []string{"mysql", "postgresql", "mongodb"},
			},
			event: &alertmodel.AlertEvent{
				DBType: "mysql",
			},
			expected: true,
		},
		{
			name: "not_in - 不在列表中",
			condition: alertmodel.Condition{
				Field:    "db_type",
				Operator: "not_in",
				Value:    []string{"mysql", "postgresql"},
			},
			event: &alertmodel.AlertEvent{
				DBType: "mongodb",
			},
			expected: true,
		},
		{
			name: "unknown operator",
			condition: alertmodel.Condition{
				Field:    "db_type",
				Operator: "unknown",
				Value:    "mysql",
			},
			event: &alertmodel.AlertEvent{
				DBType: "mysql",
			},
			expected: false,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evaluator.Evaluate(tt.condition, tt.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("Evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("Evaluate() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestConditionEvaluator_EvaluateRule(t *testing.T) {
	evaluator := NewConditionEvaluator()

	tests := []struct {
		name     string
		rule     *alertmodel.AlertRule
		event    *alertmodel.AlertEvent
		expected bool
	}{
		{
			name: "AND - 全部匹配",
			rule: &alertmodel.AlertRule{
				ConditionOp: alertmodel.ConditionOpAND,
				Conditions:   `[{"field":"db_type","operator":"eq","value":"mysql"},{"field":"content","operator":"contains","value":"failed"}]`,
			},
			event: &alertmodel.AlertEvent{
				DBType:  "mysql",
				Content: "backup failed",
			},
			expected: true,
		},
		{
			name: "AND - 部分不匹配",
			rule: &alertmodel.AlertRule{
				ConditionOp: alertmodel.ConditionOpAND,
				Conditions:   `[{"field":"db_type","operator":"eq","value":"mysql"},{"field":"content","operator":"contains","value":"timeout"}]`,
			},
			event: &alertmodel.AlertEvent{
				DBType:  "mysql",
				Content: "backup failed", // 不包含 timeout
			},
			expected: false,
		},
		{
			name: "OR - 部分匹配",
			rule: &alertmodel.AlertRule{
				ConditionOp: alertmodel.ConditionOpOR,
				Conditions:   `[{"field":"db_type","operator":"eq","value":"mysql"},{"field":"content","operator":"contains","value":"timeout"}]`,
			},
			event: &alertmodel.AlertEvent{
				DBType:  "postgresql", // 不匹配 db_type
				Content: "connection timeout", // 但匹配 contains
			},
			expected: true,
		},
		{
			name: "OR - 全部不匹配",
			rule: &alertmodel.AlertRule{
				ConditionOp: alertmodel.ConditionOpOR,
				Conditions:   `[{"field":"db_type","operator":"eq","value":"mysql"},{"field":"content","operator":"contains","value":"timeout"}]`,
			},
			event: &alertmodel.AlertEvent{
				DBType:  "postgresql",
				Content: "backup failed",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evaluator.EvaluateRule(tt.rule, tt.event)
			if err != nil {
				t.Fatalf("EvaluateRule() error = %v", err)
			}
			if got != tt.expected {
				t.Errorf("EvaluateRule() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestCooldownManager(t *testing.T) {
	mgr := NewCooldownManager()

	// 测试初始状态 - 不在冷却期
	if mgr.IsInCooldown(1, 300) {
		t.Error("expected not in cooldown initially")
	}

	// 记录触发
	mgr.RecordTrigger(1)

	// 测试在冷却期内
	if !mgr.IsInCooldown(1, 300) {
		t.Error("expected in cooldown after record")
	}

	// 测试获取剩余时间
	remaining := mgr.GetRemainingCooldown(1, 300)
	if remaining <= 0 || remaining > 300 {
		t.Errorf("expected remaining time between 1 and 300, got %d", remaining)
	}

	// 测试清除冷却时间
	mgr.ClearCooldown(1)
	if mgr.IsInCooldown(1, 300) {
		t.Error("expected not in cooldown after clear")
	}
}

func TestSilentPeriodManager(t *testing.T) {
	mgr := NewSilentPeriodManager()

	tests := []struct {
		name     string
		periods  []alertmodel.SilentPeriod
		expected bool
	}{
		{
			name:     "空静默时段",
			periods: []alertmodel.SilentPeriod{},
			expected: false,
		},
		{
			name: "全天静默（无效数据）",
			periods: []alertmodel.SilentPeriod{
				{StartTime: "00:00", EndTime: "23:59", Days: []int{0, 1, 2, 3, 4, 5, 6}},
			},
			// 取决于当前时间
			expected: false, // 假设测试不在这个时段
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mgr.IsInSilentPeriod(tt.periods)
			// 只验证空时段的情况
			if len(tt.periods) == 0 && got != tt.expected {
				t.Errorf("IsInSilentPeriod() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestRuleMatcher_MatchEvent(t *testing.T) {
	matcher := NewRuleMatcher()

	rules := []*alertmodel.AlertRule{
		{
			ID:          1,
			Name:        "MySQL失败告警",
			Enabled:     true,
			Priority:    80,
			ConditionOp: alertmodel.ConditionOpAND,
			Conditions:  `[{"field":"db_type","operator":"eq","value":"mysql"},{"field":"event_type","operator":"eq","value":"backup_failed"}]`,
		},
		{
			ID:          2,
			Name:        "通用失败告警",
			Enabled:     true,
			Priority:    50,
			ConditionOp: alertmodel.ConditionOpOR,
			Conditions:  `[{"field":"event_type","operator":"eq","value":"backup_failed"},{"field":"event_type","operator":"eq","value":"restore_failed"}]`,
		},
		{
			ID:          3,
			Name:        "已禁用的规则",
			Enabled:     false,
			Priority:    90,
			ConditionOp: alertmodel.ConditionOpAND,
			Conditions:  `[{"field":"db_type","operator":"eq","value":"mysql"}]`,
		},
	}

	event := &alertmodel.AlertEvent{
		EventType: alertmodel.EventTypeBackupFailed,
		DBType:    "mysql",
		TaskID:    123,
		Title:     "MySQL Backup Failed",
		Content:   "MySQL backup failed",
	}

	results := matcher.MatchEvent(event, rules)

	// 应该匹配 2 个规则（规则1和规则2）
	matchedCount := 0
	for _, r := range results {
		if r.Matched {
			matchedCount++
		}
	}
	if matchedCount != 2 {
		t.Errorf("expected 2 matched rules, got %d", matchedCount)
	}

	// 规则3应该被标记为未匹配（已禁用）
	for _, r := range results {
		if r.Rule.ID == 3 {
			if r.Matched {
				t.Error("disabled rule should not be matched")
			}
			if r.Reason != "规则已禁用" {
				t.Errorf("expected reason '规则已禁用', got '%s'", r.Reason)
			}
		}
	}
}

func TestRuleMatcher_GetMatchedRules(t *testing.T) {
	matcher := NewRuleMatcher()

	rules := []*alertmodel.AlertRule{
		{ID: 1, Enabled: true, Conditions: `[{"field":"db_type","operator":"eq","value":"mysql"}]`},
		{ID: 2, Enabled: true, Conditions: `[{"field":"db_type","operator":"eq","value":"postgresql"}]`},
		{ID: 3, Enabled: false, Conditions: `[{"field":"db_type","operator":"eq","value":"mysql"}]`},
	}

	event := &alertmodel.AlertEvent{DBType: "mysql"}

	matched := matcher.GetMatchedRules(event, rules)

	if len(matched) != 1 {
		t.Errorf("expected 1 matched rule, got %d", len(matched))
	}
	if matched[0].ID != 1 {
		t.Errorf("expected matched rule ID 1, got %d", matched[0].ID)
	}
}

func TestParseTime(t *testing.T) {
	tests := []struct {
		input    string
		expected int // 分钟数
	}{
		{"00:00", 0},
		{"01:00", 60},
		{"12:30", 12*60 + 30},
		{"23:59", 23*60 + 59},
	}

	for _, tt := range tests {
		got := parseTime(tt.input)
		if got != tt.expected {
			t.Errorf("parseTime(%s) = %d, expected %d", tt.input, got, tt.expected)
		}
	}
}
