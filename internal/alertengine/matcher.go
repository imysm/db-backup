package alertengine

import (
	"github.com/imysm/db-backup/internal/alertmodel"
)

// RuleMatcher 规则匹配器
type RuleMatcher struct {
	evaluator *ConditionEvaluator
}

// NewRuleMatcher 创建规则匹配器
func NewRuleMatcher() *RuleMatcher {
	return &RuleMatcher{
		evaluator: NewConditionEvaluator(),
	}
}

// MatchResult 匹配结果
type MatchResult struct {
	Rule     *alertmodel.AlertRule
	Matched  bool
	Reason   string
}

// MatchEvent 匹配事件到所有适用规则
func (m *RuleMatcher) MatchEvent(event *alertmodel.AlertEvent, rules []*alertmodel.AlertRule) []*MatchResult {
	results := make([]*MatchResult, 0, len(rules))

	for _, rule := range rules {
		result := &MatchResult{Rule: rule}

		// 检查规则是否启用
		if !rule.Enabled {
			result.Reason = "规则已禁用"
			results = append(results, result)
			continue
		}

		// 评估条件
		matched, err := m.evaluator.EvaluateRule(rule, event)
		if err != nil {
			result.Reason = "条件评估失败: " + err.Error()
			results = append(results, result)
			continue
		}

		if matched {
			result.Matched = true
			result.Reason = "条件匹配"
		} else {
			result.Reason = "条件不匹配"
		}
		results = append(results, result)
	}

	return results
}

// MatchEventByPriority 匹配事件并按优先级排序
func (m *RuleMatcher) MatchEventByPriority(event *alertmodel.AlertEvent, rules []*alertmodel.AlertRule) []*MatchResult {
	results := m.MatchEvent(event, rules)

	// 按优先级排序（从高到低）
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Rule.Priority > results[i].Rule.Priority {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results
}

// GetMatchedRules 获取所有匹配成功的规则
func (m *RuleMatcher) GetMatchedRules(event *alertmodel.AlertEvent, rules []*alertmodel.AlertRule) []*alertmodel.AlertRule {
	results := m.MatchEvent(event, rules)
	matched := make([]*alertmodel.AlertRule, 0)

	for _, r := range results {
		if r.Matched {
			matched = append(matched, r.Rule)
		}
	}

	return matched
}
