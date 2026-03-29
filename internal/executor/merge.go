package executor

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// MergeConfig 合并配置
type MergeConfig struct {
	OutputPath  string   // 输出路径
	SourceFiles []string // 源文件列表
	DeleteSource bool    // 合并后删除源文件
}

// MergeResult 合并结果
type MergeResult struct {
	OutputPath  string // 输出文件路径
	FileSize   int64  // 输出文件大小
	SourceCount int    // 合并文件数量
}

// MergeBackups 合并多个备份文件
func MergeBackups(config *MergeConfig) (*MergeResult, error) {
	if len(config.SourceFiles) == 0 {
		return nil, fmt.Errorf("no source files to merge")
	}

	if len(config.SourceFiles) == 1 {
		// 只有一个文件，直接复制
		return copyFile(config.SourceFiles[0], config.OutputPath)
	}

	// 打开输出文件
	outFile, err := os.Create(config.OutputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create output: %w", err)
	}
	defer outFile.Close()

	// 按文件名排序（确保顺序一致）
	files := make([]string, len(config.SourceFiles))
	copy(files, config.SourceFiles)
	sort.Strings(files)

	var totalSize int64
	for _, srcPath := range files {
		srcFile, err := os.Open(srcPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open %s: %w", srcPath, err)
		}

		written, err := io.Copy(outFile, srcFile)
		srcFile.Close()

		if err != nil {
			return nil, fmt.Errorf("failed to copy from %s: %w", srcPath, err)
		}

		totalSize += written
	}

	// 删除源文件（如果配置了）
	if config.DeleteSource {
		for _, srcPath := range files {
			os.Remove(srcPath)
		}
	}

	return &MergeResult{
		OutputPath:  config.OutputPath,
		FileSize:   totalSize,
		SourceCount: len(files),
	}, nil
}

// copyFile 复制文件
func copyFile(src, dst string) (*MergeResult, error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return nil, err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return nil, err
	}
	defer dstFile.Close()

	written, err := io.Copy(dstFile, srcFile)
	if err != nil {
		return nil, err
	}

	return &MergeResult{
		OutputPath:  dst,
		FileSize:   written,
		SourceCount: 1,
	}, nil
}

// ListBackupFiles 列出目录下的备份文件
func ListBackupFiles(dir, prefix string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, prefix) {
			files = append(files, filepath.Join(dir, name))
		}
	}

	sort.Strings(files)
	return files, nil
}

// MergeDirectory 合并目录下所有匹配前缀的文件
func MergeDirectory(dir, prefix, outputPath string, deleteSource bool) (*MergeResult, error) {
	files, err := ListBackupFiles(dir, prefix)
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no files found with prefix %s in %s", prefix, dir)
	}

	return MergeBackups(&MergeConfig{
		OutputPath:   outputPath,
		SourceFiles: files,
		DeleteSource: deleteSource,
	})
}
