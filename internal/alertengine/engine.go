package alertengine

import (
	"github.com/imysm/db-backup/internal/alertmodel"
	"github.com/imysm/db-backup/internal/alertstorage"
	"fmt"
	"log"
	"sync"
	"time"
)

// Engine 告警引擎
type Engine struct {
	// 存储层
	channelStorage *alertstorage.ChannelStorage
	ruleStorage   *alertstorage.RuleStorage
	recordStorage *alertstorage.RecordStorage

	// 组件
	evaluator     *ConditionEvaluator
	cooldown      *CooldownManager
	silentPeriod  *SilentPeriodManager
	ruleMatcher   *RuleMatcher

	// 配置
	checkInterval time.Duration

	// 状态
	mu      sync.RWMutex
	running bool
	stopCh  chan struct{}
}

// EngineConfig 引擎配置
type EngineConfig struct {
	CheckInterval time.Duration // 状态检查间隔
}

// NewEngine 创建告警引擎
func NewEngine(
	channelStorage *alertstorage.ChannelStorage,
	ruleStorage *alertstorage.RuleStorage,
	recordStorage *alertstorage.RecordStorage,
	config *EngineConfig,
) *Engine {
	if config == nil {
		config = &EngineConfig{
			CheckInterval: 1 * time.Minute,
		}
	}

	return &Engine{
		channelStorage: channelStorage,
		ruleStorage:   ruleStorage,
		recordStorage: recordStorage,
		evaluator:     NewConditionEvaluator(),
		cooldown:      NewCooldownManager(),
		silentPeriod:  NewSilentPeriodManager(),
		ruleMatcher:   NewRuleMatcher(),
		checkInterval: config.CheckInterval,
		stopCh:        make(chan struct{}),
	}
}

// ProcessEvent 处理告警事件
func (e *Engine) ProcessEvent(event *alertmodel.AlertEvent) error {
	// 1. 获取所有启用的规则
	ruleValues, err := e.ruleStorage.ListEnabled()
	if err != nil {
		return fmt.Errorf("获取规则失败: %w", err)
	}

	// 转换为指针切片
	rules := make([]*alertmodel.AlertRule, len(ruleValues))
	for i := range ruleValues {
		rules[i] = &ruleValues[i]
	}

	// 2. 匹配事件到规则
	matchedRules := e.ruleMatcher.GetMatchedRules(event, rules)
	if len(matchedRules) == 0 {
		log.Printf("[AlertEngine] 事件 %s 没有匹配的规则", event.EventType)
		return nil
	}

	// 3. 对每个匹配的规则进行处理
	for _, rule := range matchedRules {
		if err := e.processRule(rule, event); err != nil {
			log.Printf("[AlertEngine] 处理规则 %d 失败: %v", rule.ID, err)
			continue
		}
	}

	return nil
}

// processRule 处理单个规则
func (e *Engine) processRule(rule *alertmodel.AlertRule, event *alertmodel.AlertEvent) error {
	// 1. 检查静默时段
	silentPeriods, _ := rule.GetSilentPeriods()
	if e.silentPeriod.IsInSilentPeriod(silentPeriods) {
		log.Printf("[AlertEngine] 规则 %d 在静默时段内，跳过", rule.ID)
		return nil
	}

	// 2. 检查冷却时间
	if e.cooldown.IsInCooldown(rule.ID, rule.Cooldown) {
		remaining := e.cooldown.GetRemainingCooldown(rule.ID, rule.Cooldown)
		log.Printf("[AlertEngine] 规则 %d 在冷却期内，剩余 %d 秒", rule.ID, remaining)
		return nil
	}

	// 3. 获取通知渠道
	channelIDs, _ := rule.GetChannels()
	channels, err := e.channelStorage.GetByIDs(channelIDs)
	if err != nil || len(channels) == 0 {
		return fmt.Errorf("获取通知渠道失败或渠道为空")
	}

	// 4. 创建告警记录
	record := &alertmodel.AlertRecord{
		RuleID:      rule.ID,
		TaskID:      &event.TaskID,
		Level:       rule.Level,
		Title:       event.Title,
		Content:     event.Content,
		EventType:   event.EventType,
		TaskName:    event.TaskName,
		DBType:      event.DBType,
		Source:      event.Source,
		Status:      alertmodel.AlertStatusActive,
		TriggeredAt: time.Now(),
	}

	if err := e.recordStorage.Create(record); err != nil {
		return fmt.Errorf("创建告警记录失败: %w", err)
	}

	// 5. 记录冷却时间
	e.cooldown.RecordTrigger(rule.ID)

	// 6. 更新规则匹配计数
	if err := e.ruleStorage.IncrementMatchedCount(rule.ID); err != nil {
		log.Printf("[AlertEngine] 更新规则匹配计数失败: %v", err)
	}

	// 7. 为每个渠道创建发送记录
	for _, channel := range channels {
		notifRecord := &alertmodel.AlertNotificationRecord{
			AlertID:   record.ID,
			ChannelID: channel.ID,
			Status:    "pending",
		}
		if err := e.recordStorage.CreateNotificationRecord(notifRecord); err != nil {
			log.Printf("[AlertEngine] 创建通知记录失败: %v", err)
		}
	}

	log.Printf("[AlertEngine] 告警已创建: ID=%d, Rule=%s, Level=%s",
		record.ID, rule.Name, rule.Level)

	return nil
}

// CheckEscalations 检查并处理告警升级
func (e *Engine) CheckEscalations() error {
	// 获取所有活跃告警
	recordValues, _, err := e.recordStorage.List(&alertmodel.ListAlertsRequest{
		Status:   alertmodel.AlertStatusActive,
		Page:     1,
		PageSize: 100,
	})
	if err != nil {
		return err
	}

	// 转换为指针切片
	records := make([]*alertmodel.AlertRecord, len(recordValues))
	for i := range recordValues {
		records[i] = &recordValues[i]
	}

	// 获取所有规则
	ruleValues, err := e.ruleStorage.ListEnabled()
	if err != nil {
		return err
	}
	rulesMap := make(map[int64]*alertmodel.AlertRule)
	for i := range ruleValues {
		rulesMap[ruleValues[i].ID] = &ruleValues[i]
	}

	for _, record := range records {
		rule, exists := rulesMap[record.RuleID]
		if !exists || !rule.EscalateEnabled {
			continue
		}

		// 检查是否超过升级超时时间
		elapsed := time.Since(record.TriggeredAt)
		timeout := time.Duration(rule.EscalateTimeout) * time.Minute

		if elapsed >= timeout {
			if err := e.escalateAlert(record, rule); err != nil {
				log.Printf("[AlertEngine] 升级告警 %d 失败: %v", record.ID, err)
				continue
			}
			log.Printf("[AlertEngine] 告警 %d 已升级到 %s", record.ID, rule.EscalateLevel)
		}
	}

	return nil
}

// escalateAlert 升级告警
func (e *Engine) escalateAlert(record *alertmodel.AlertRecord, rule *alertmodel.AlertRule) error {
	// 更新告警状态
	if err := e.recordStorage.Escalate(record.ID, rule.EscalateLevel, "超时未确认，自动升级"); err != nil {
		return err
	}

	// 获取升级渠道
	channelIDs, _ := rule.GetEscalateChannels()
	if len(channelIDs) == 0 {
		return nil
	}

	// 为升级渠道创建发送记录
	for _, channelID := range channelIDs {
		notifRecord := &alertmodel.AlertNotificationRecord{
			AlertID:   record.ID,
			ChannelID: channelID,
			Status:    "pending",
		}
		if err := e.recordStorage.CreateNotificationRecord(notifRecord); err != nil {
			log.Printf("[AlertEngine] 创建升级通知记录失败: %v", err)
		}
	}

	return nil
}

// Start 启动引擎（用于定时检查升级等后台任务）
func (e *Engine) Start() {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return
	}
	e.running = true
	e.mu.Unlock()

	go func() {
		ticker := time.NewTicker(e.checkInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := e.CheckEscalations(); err != nil {
					log.Printf("[AlertEngine] 检查升级失败: %v", err)
				}
			case <-e.stopCh:
				return
			}
		}
	}()

	log.Printf("[AlertEngine] 引擎已启动")
}

// Stop 停止引擎
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return
	}

	close(e.stopCh)
	e.running = false
	log.Printf("[AlertEngine] 引擎已停止")
}

// GetCooldownManager 获取冷却管理器（供测试用）
func (e *Engine) GetCooldownManager() *CooldownManager {
	return e.cooldown
}

// GetSilentPeriodManager 获取静默时段管理器（供测试用）
func (e *Engine) GetSilentPeriodManager() *SilentPeriodManager {
	return e.silentPeriod
}
