package alertengine

import (
	"github.com/imysm/db-backup/internal/alertmodel"
	"sync"
	"time"
)

// CooldownManager 冷却时间管理器
type CooldownManager struct {
	// ruleID -> 最后触发时间
	lastTriggered map[int64]time.Time
	mu           sync.RWMutex
}

// NewCooldownManager 创建冷却时间管理器
func NewCooldownManager() *CooldownManager {
	return &CooldownManager{
		lastTriggered: make(map[int64]time.Time),
	}
}

// IsInCooldown 检查规则是否在冷却期内
func (m *CooldownManager) IsInCooldown(ruleID int64, cooldownSeconds int) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	lastTriggered, exists := m.lastTriggered[ruleID]
	if !exists {
		return false
	}

	cooldownDuration := time.Duration(cooldownSeconds) * time.Second
	return time.Since(lastTriggered) < cooldownDuration
}

// RecordTrigger 记录规则触发
func (m *CooldownManager) RecordTrigger(ruleID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.lastTriggered[ruleID] = time.Now()
}

// GetRemainingCooldown 获取剩余冷却时间（秒）
func (m *CooldownManager) GetRemainingCooldown(ruleID int64, cooldownSeconds int) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	lastTriggered, exists := m.lastTriggered[ruleID]
	if !exists {
		return 0
	}

	cooldownDuration := time.Duration(cooldownSeconds) * time.Second
	elapsed := time.Since(lastTriggered)
	remaining := cooldownDuration - elapsed

	if remaining <= 0 {
		return 0
	}
	return int(remaining.Seconds())
}

// ClearCooldown 清除规则的冷却时间
func (m *CooldownManager) ClearCooldown(ruleID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.lastTriggered, ruleID)
}

// ClearAll 清除所有冷却时间
func (m *CooldownManager) ClearAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.lastTriggered = make(map[int64]time.Time)
}

// SilentPeriodManager 静默时段管理器
type SilentPeriodManager struct{}

// NewSilentPeriodManager 创建静默时段管理器
func NewSilentPeriodManager() *SilentPeriodManager {
	return &SilentPeriodManager{}
}

// IsInSilentPeriod 检查当前时间是否在静默时段内
func (m *SilentPeriodManager) IsInSilentPeriod(periods []alertmodel.SilentPeriod) bool {
	if len(periods) == 0 {
		return false
	}

	now := time.Now()
	currentTime := now.Format("15:04")
	currentDay := int(now.Weekday()) // 0=周日, 1=周一...

	for _, period := range periods {
		// 检查日期是否匹配
		dayMatch := false
		for _, day := range period.Days {
			if day == currentDay {
				dayMatch = true
				break
			}
		}
		if !dayMatch {
			continue
		}

		// 检查时间是否在范围内
		if m.isTimeInRange(currentTime, period.StartTime, period.EndTime) {
			return true
		}
	}

	return false
}

// isTimeInRange 检查时间是否在范围内（支持跨午夜）
func (m *SilentPeriodManager) isTimeInRange(current, start, end string) bool {
	// 解析时间
	c := parseTime(current)
	s := parseTime(start)
	e := parseTime(end)

	// 正常范围（如 09:00 - 17:00）
	if s <= e {
		return c >= s && c <= e
	}

	// 跨午夜范围（如 22:00 - 06:00）
	return c >= s || c <= e
}

// parseTime 解析 HH:mm 格式时间为分钟数
func parseTime(t string) int {
	parts := splitTime(t)
	if len(parts) != 2 {
		return 0
	}
	hour := parts[0]
	min := parts[1]
	return hour*60 + min
}

func splitTime(t string) [2]int {
	var result [2]int
	if len(t) < 5 {
		return result
	}
	// 处理 HH:mm 格式
	hourStr := t[0:2]
	minStr := t[3:5]
	hour := 0
	min := 0
	for _, c := range hourStr {
		if c >= '0' && c <= '9' {
			hour = hour*10 + int(c-'0')
		}
	}
	for _, c := range minStr {
		if c >= '0' && c <= '9' {
			min = min*10 + int(c-'0')
		}
	}
	result[0] = hour
	result[1] = min
	return result
}
