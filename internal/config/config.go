// Package config 提供配置文件的加载和管理功能
package config

import (
	"fmt"
	"os"
	"time"

	"github.com/imysm/db-backup/internal/model"
	"gopkg.in/yaml.v3"
)

// Config 配置类型别名（兼容 CodeClaw 代码）
type Config = model.Config

// DefaultConfig 返回默认配置
func DefaultConfig() *model.Config {
	return &model.Config{
		Global: model.GlobalConfig{
			WorkDir:       "/tmp/db-backup",
			DefaultTZ:     "Asia/Shanghai",
			MaxConcurrent: 5,
			Timeout:       2 * time.Hour,
		},
		Log: model.LogConfig{
			Level:  "info",
			Format: "console",
		},
		Tasks: []model.BackupTask{},
	}
}

// Load 从文件加载配置
func Load(path string) (*model.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 设置默认值
	if err := setDefaults(cfg); err != nil {
		return nil, err
	}

	// 验证配置
	if err := Validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// LoadFromString 从字符串加载配置
func LoadFromString(content string) (*model.Config, error) {
	cfg := DefaultConfig()
	if err := yaml.Unmarshal([]byte(content), cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	if err := setDefaults(cfg); err != nil {
		return nil, err
	}

	if err := Validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save 保存配置到文件
func Save(cfg *model.Config, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// setDefaults 设置默认值
func setDefaults(cfg *model.Config) error {
	// 全局配置默认值
	if cfg.Global.WorkDir == "" {
		cfg.Global.WorkDir = "/tmp/db-backup"
	}
	if cfg.Global.DefaultTZ == "" {
		cfg.Global.DefaultTZ = "Asia/Shanghai"
	}
	if cfg.Global.MaxConcurrent == 0 {
		cfg.Global.MaxConcurrent = 5
	}
	if cfg.Global.Timeout == 0 {
		cfg.Global.Timeout = 2 * time.Hour
	}

	// 任务配置默认值
	for i := range cfg.Tasks {
		task := &cfg.Tasks[i]

		if task.Schedule.Timezone == "" {
			task.Schedule.Timezone = cfg.Global.DefaultTZ
		}
		if task.Storage.Type == "" {
			task.Storage.Type = "local"
		}
		if task.Storage.Path == "" {
			task.Storage.Path = cfg.Global.WorkDir
		}
		if task.Compression.Type == "" {
			task.Compression.Type = "gzip"
		}
		if task.Compression.Level == 0 {
			task.Compression.Level = 6
		}
	}

	return nil
}

// Validate 验证配置有效性
func Validate(cfg *model.Config) error {
	// 验证全局配置
	if cfg.Global.MaxConcurrent < 1 {
		return fmt.Errorf("max_concurrent 必须大于 0")
	}
	if cfg.Global.Timeout < time.Minute {
		return fmt.Errorf("timeout 必须大于等于 1 分钟")
	}

	// 验证任务配置
	taskIDs := make(map[string]bool)
	for i, task := range cfg.Tasks {
		if task.ID == "" {
			return fmt.Errorf("任务 %d: ID 不能为空", i)
		}
		if taskIDs[task.ID] {
			return fmt.Errorf("任务 ID 重复: %s", task.ID)
		}
		taskIDs[task.ID] = true

		if task.Name == "" {
			return fmt.Errorf("任务 %s: 名称不能为空", task.ID)
		}

		// 验证数据库类型
		switch task.Database.Type {
		case model.MySQL, model.PostgreSQL, model.MongoDB, model.SQLServer, model.Oracle:
			// 有效类型
		default:
			return fmt.Errorf("任务 %s: 不支持的数据库类型: %s", task.ID, task.Database.Type)
		}

		// 验证数据库连接
		if task.Database.Host == "" {
			return fmt.Errorf("任务 %s: 数据库主机不能为空", task.ID)
		}
		if task.Database.Port == 0 {
			return fmt.Errorf("任务 %s: 数据库端口不能为空", task.ID)
		}
		if task.Database.Port < 1 || task.Database.Port > 65535 {
			return fmt.Errorf("任务 %s: 数据库端口无效: %d", task.ID, task.Database.Port)
		}

		// 验证存储配置
		if task.Storage.Type != "local" && task.Storage.Type != "s3" && task.Storage.Type != "oss" && task.Storage.Type != "cos" {
			return fmt.Errorf("任务 %s: 不支持的存储类型: %s", task.ID, task.Storage.Type)
		}
		if task.Storage.Type == "s3" || task.Storage.Type == "oss" || task.Storage.Type == "cos" {
			if task.Storage.Endpoint == "" {
				return fmt.Errorf("任务 %s: S3/OSS/COS 存储需要指定 endpoint", task.ID)
			}
			if task.Storage.Bucket == "" {
				return fmt.Errorf("任务 %s: S3/OSS/COS 存储需要指定 bucket", task.ID)
			}
		}

		// 验证压缩配置
		if task.Compression.Enabled {
			switch task.Compression.Type {
			case "gzip", "zstd", "lz4":
				// 有效类型
			default:
				return fmt.Errorf("任务 %s: 不支持的压缩类型: %s", task.ID, task.Compression.Type)
			}
			if task.Compression.Level < 1 || task.Compression.Level > 9 {
				return fmt.Errorf("任务 %s: 压缩级别必须在 1-9 之间", task.ID)
			}
		}
	}

	return nil
}

// GetTaskByID 根据 ID 获取任务配置
func GetTaskByID(cfg *model.Config, id string) (*model.BackupTask, error) {
	for i := range cfg.Tasks {
		if cfg.Tasks[i].ID == id {
			return &cfg.Tasks[i], nil
		}
	}
	return nil, fmt.Errorf("任务不存在: %s", id)
}

// GetEnabledTasks 获取所有启用的任务
func GetEnabledTasks(cfg *model.Config) []model.BackupTask {
	var tasks []model.BackupTask
	for _, task := range cfg.Tasks {
		if task.Enabled {
			tasks = append(tasks, task)
		}
	}
	return tasks
}
