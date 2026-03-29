// Package util 提供通用工具函数
package util

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ErrPathTraversal 路径遍历错误
type ErrPathTraversal struct {
	Path     string
	BaseDir  string
	Resolved string
}

func (e *ErrPathTraversal) Error() string {
	return fmt.Sprintf("路径遍历攻击被拒绝: 路径 %q 解析为 %q，不在允许的目录 %q 下", e.Path, e.Resolved, e.BaseDir)
}

// ValidateFilePath 验证文件路径是否在允许的目录下，防止路径遍历攻击。
//
// 验证逻辑：
//   - 如果 requestPath 是绝对路径，直接验证
//   - 如果 requestPath 是相对路径，基于 baseDir 解析
//   - filepath.Clean 清理 .. 和 .
//   - 检查解析后的路径是否在 baseDir 前缀下
//   - 确保前缀匹配是目录级别的（防止目录名前缀攻击）
func ValidateFilePath(requestPath, baseDir string) error {
	if requestPath == "" {
		return fmt.Errorf("文件路径不能为空")
	}
	if baseDir == "" {
		return fmt.Errorf("基础目录不能为空")
	}

	// 解析基础目录为绝对路径
	absBaseDir := filepath.Clean(baseDir)
	if !filepath.IsAbs(absBaseDir) {
		return fmt.Errorf("基础目录必须是绝对路径")
	}

	// 解析请求路径为绝对路径
	var absPath string
	if filepath.IsAbs(requestPath) {
		absPath = filepath.Clean(requestPath)
	} else {
		absPath = filepath.Clean(filepath.Join(absBaseDir, requestPath))
	}

	// 检查路径是否在允许的目录下
	// 确保前缀匹配是目录级别的（防止 /var/lib/backup2 匹配 /var/lib/backup）
	if !strings.HasPrefix(absPath, absBaseDir+string(filepath.Separator)) && absPath != absBaseDir {
		return &ErrPathTraversal{
			Path:     requestPath,
			BaseDir:  absBaseDir,
			Resolved: absPath,
		}
	}

	return nil
}
