// Package util 提供通用工具函数
package util

import (
	"fmt"
	"regexp"
	"strings"
)

// dangerSQLKeywords SQL 注入危险关键字
var dangerSQLKeywords = []string{
	"DROP ", "DELETE FROM", "INSERT INTO", "UPDATE ", "ALTER ",
	"CREATE ", "EXEC ", "EXECUTE ", "xp_cmdshell", "sp_",
	"GRANT ", "REVOKE ", "TRUNCATE ", "SHUTDOWN",
}

// ValidateBackupPath 验证备份文件路径安全性
// 防止路径遍历和 SQL 注入
func ValidateBackupPath(path string) error {
	if path == "" {
		return fmt.Errorf("备份路径不能为空")
	}

	// 防止路径遍历
	if strings.Contains(path, "..") {
		return fmt.Errorf("备份路径不能包含路径遍历字符")
	}

	// 只允许安全字符
	re := regexp.MustCompile(`^[a-zA-Z0-9_\-/.\s]+$`)
	if !re.MatchString(path) {
		return fmt.Errorf("备份路径包含非法字符")
	}

	return nil
}

// ValidateDatabaseName 验证数据库名安全性
func ValidateDatabaseName(name string) error {
	if name == "" {
		return fmt.Errorf("数据库名不能为空")
	}

	// 只允许安全字符（字母、数字、下划线、连字符）
	re := regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)
	if !re.MatchString(name) {
		return fmt.Errorf("数据库名包含非法字符: %s", name)
	}

	return nil
}

// SanitizeParam 消毒用户输入参数，防止命令注入
// 只允许字母、数字、下划线、连字符、点、星号、空格、逗号、等号、冒号、斜杠、方括号、百分号、引号
// 用于 buildDumpArgs 中 cfg.Params 取出的值拼入命令行前的消毒
func SanitizeParam(input string) string {
	if input == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range input {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '_', r == '-', r == '.', r == '*', r == ' ', r == ',', r == '=', r == ':', r == '/', r == '[', r == ']', r == '%', r == '\'', r == '"':
			b.WriteRune(r)
		// 过滤所有 shell 元字符：; & | $ ` \n \r ( ) { } < > 以及其他控制字符
		default:
			// skip dangerous characters
		}
	}
	return b.String()
}

// SanitizeWhereClause 过滤 MySQL --where 参数中的危险关键字
func SanitizeWhereClause(where string) (string, error) {
	if where == "" {
		return "", nil
	}

	upper := strings.ToUpper(where)
	for _, kw := range dangerSQLKeywords {
		if strings.Contains(upper, kw) {
			return "", fmt.Errorf("--where 参数包含危险关键字: %s", strings.TrimSpace(kw))
		}
	}

	// 防止分号注入多语句
	if strings.Contains(where, ";") {
		return "", fmt.Errorf("--where 参数不能包含分号")
	}

	// 防止注释符
	if strings.Contains(where, "--") || strings.Contains(where, "/*") {
		return "", fmt.Errorf("--where 参数不能包含 SQL 注释符")
	}

	return where, nil
}
