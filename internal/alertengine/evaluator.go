package alertengine

import (
	"github.com/imysm/db-backup/internal/alertmodel"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// ConditionEvaluator 条件评估器
type ConditionEvaluator struct{}

// NewConditionEvaluator 创建条件评估器
func NewConditionEvaluator() *ConditionEvaluator {
	return &ConditionEvaluator{}
}

// Evaluate 评估单个条件是否匹配
func (e *ConditionEvaluator) Evaluate(condition alertmodel.Condition, event *alertmodel.AlertEvent) (bool, error) {
	// 获取事件字段值
	fieldValue, err := e.getFieldValue(event, condition.Field)
	if err != nil {
		return false, err
	}

	// 根据操作符进行评估
	switch condition.Operator {
	case "eq":
		return e.equals(fieldValue, condition.Value)
	case "ne":
		result, _ := e.equals(fieldValue, condition.Value)
		return !result, nil
	case "gt":
		return e.greaterThan(fieldValue, condition.Value)
	case "gte":
		result, _ := e.greaterThan(fieldValue, condition.Value)
		if result {
			return true, nil
		}
		return e.equals(fieldValue, condition.Value)
	case "lt":
		return e.lessThan(fieldValue, condition.Value)
	case "lte":
		result, _ := e.lessThan(fieldValue, condition.Value)
		if result {
			return true, nil
		}
		return e.equals(fieldValue, condition.Value)
	case "in":
		return e.inList(fieldValue, condition.Value)
	case "not_in":
		result, _ := e.inList(fieldValue, condition.Value)
		return !result, nil
	case "contains":
		return e.contains(fieldValue, condition.Value)
	case "regex":
		return e.regexMatch(fieldValue, condition.Value)
	default:
		return false, fmt.Errorf("未知的操作符: %s", condition.Operator)
	}
}

// EvaluateRule 评估规则的所有条件
func (e *ConditionEvaluator) EvaluateRule(rule *alertmodel.AlertRule, event *alertmodel.AlertEvent) (bool, error) {
	conditions, err := rule.GetConditions()
	if err != nil {
		return false, fmt.Errorf("获取规则条件失败: %w", err)
	}

	if len(conditions) == 0 {
		return false, nil
	}

	results := make([]bool, len(conditions))
	for i, cond := range conditions {
		result, err := e.Evaluate(cond, event)
		if err != nil {
			return false, err
		}
		results[i] = result
	}

	// 根据条件组合方式进行评估
	switch rule.ConditionOp {
	case alertmodel.ConditionOpAND:
		for _, r := range results {
			if !r {
				return false, nil
			}
		}
		return true, nil
	case alertmodel.ConditionOpOR:
		for _, r := range results {
			if r {
				return true, nil
			}
		}
		return false, nil
	default:
		// 默认 AND
		for _, r := range results {
			if !r {
				return false, nil
			}
		}
		return true, nil
	}
}

// getFieldValue 获取事件中指定字段的值
func (e *ConditionEvaluator) getFieldValue(event *alertmodel.AlertEvent, field string) (interface{}, error) {
	v := reflect.ValueOf(event).Elem()

	// 特殊字段映射
	fieldMap := map[string]string{
		"task_id":     "TaskID",
		"task_name":   "TaskName",
		"db_type":     "DBType",
		"source":      "Source",
		"level":       "Level",
		"event_type":  "EventType",
		"title":       "Title",
		"content":     "Content",
	}

	structFieldName := fieldMap[field]
	if structFieldName == "" {
		structFieldName = strings.Title(field)
	}

	f := v.FieldByName(structFieldName)
	if !f.IsValid() {
		return nil, fmt.Errorf("未知字段: %s", field)
	}

	return f.Interface(), nil
}

// equals 判断相等
func (e *ConditionEvaluator) equals(a, b interface{}) (bool, error) {
	// 转换为字符串比较
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)
	return aStr == bStr, nil
}

// greaterThan 判断大于
func (e *ConditionEvaluator) greaterThan(a, b interface{}) (bool, error) {
	// 尝试数值比较
	aNum, aIsNum := toFloat64(a)
	bNum, bIsNum := toFloat64(b)

	if aIsNum && bIsNum {
		return aNum > bNum, nil
	}

	// 字符串比较
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)
	return aStr > bStr, nil
}

// lessThan 判断小于
func (e *ConditionEvaluator) lessThan(a, b interface{}) (bool, error) {
	aNum, aIsNum := toFloat64(a)
	bNum, bIsNum := toFloat64(b)

	if aIsNum && bIsNum {
		return aNum < bNum, nil
	}

	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)
	return aStr < bStr, nil
}

// inList 判断是否在列表中
func (e *ConditionEvaluator) inList(value, list interface{}) (bool, error) {
	// value 转换为字符串
	valueStr := fmt.Sprintf("%v", value)

	// 尝试解析 list 为切片
	listValue := reflect.ValueOf(list)
	if listValue.Kind() != reflect.Slice {
		// 可能是逗号分隔的字符串
		if strList, ok := list.(string); ok {
			parts := strings.Split(strList, ",")
			for _, part := range parts {
				if strings.TrimSpace(part) == valueStr {
					return true, nil
				}
			}
			return false, nil
		}
		return false, fmt.Errorf("in 操作符需要列表值")
	}

	// 遍历切片
	for i := 0; i < listValue.Len(); i++ {
		item := listValue.Index(i).Interface()
		itemStr := fmt.Sprintf("%v", item)
		if itemStr == valueStr {
			return true, nil
		}
	}
	return false, nil
}

// contains 判断字符串包含
func (e *ConditionEvaluator) contains(value, substr interface{}) (bool, error) {
	valueStr := fmt.Sprintf("%v", value)
	substrStr := fmt.Sprintf("%v", substr)
	return strings.Contains(valueStr, substrStr), nil
}

// regexMatch 判断正则匹配
func (e *ConditionEvaluator) regexMatch(value, pattern interface{}) (bool, error) {
	valueStr := fmt.Sprintf("%v", value)
	patternStr := fmt.Sprintf("%v", pattern)

	matched, err := regexp.MatchString(patternStr, valueStr)
	if err != nil {
		return false, fmt.Errorf("无效的正则表达式: %s", err)
	}
	return matched, nil
}

// toFloat64 转换为 float64
func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case int:
		return float64(val), true
	case int8:
		return float64(val), true
	case int16:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case float32:
		return float64(val), true
	case float64:
		return val, true
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}
