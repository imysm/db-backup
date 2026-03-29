// Package model 定义数据库备份系统的核心数据模型
package model

import (
	"time"
)

// DatabaseType 数据库类型
type DatabaseType string

const (
	MySQL      DatabaseType = "mysql"
	PostgreSQL DatabaseType = "postgres"
	MongoDB    DatabaseType = "mongodb"
	SQLServer  DatabaseType = "sqlserver"
	Oracle     DatabaseType = "oracle"
)

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending  TaskStatus = "pending"
	TaskStatusRunning  TaskStatus = "running"
	TaskStatusSuccess  TaskStatus = "success"
	TaskStatusFailed   TaskStatus = "failed"
	TaskStatusVerified TaskStatus = "verified"
)

// BackupTask 备份任务配置
type BackupTask struct {
	ID          string            `yaml:"id" json:"id" gorm:"primaryKey"`
	Name        string            `yaml:"name" json:"name" gorm:"size:100;not null"`
	Enabled     bool              `yaml:"enabled" json:"enabled" gorm:"default:true"`
	Database    DatabaseConfig    `yaml:"database" json:"database" gorm:"embedded;embeddedPrefix:db_"`
	Schedule    ScheduleConfig    `yaml:"schedule" json:"schedule" gorm:"embedded;embeddedPrefix:schedule_"`
	Storage     StorageConfig     `yaml:"storage" json:"storage" gorm:"embedded;embeddedPrefix:storage_"`
	Retention   RetentionConfig   `yaml:"retention" json:"retention" gorm:"embedded;embeddedPrefix:retention_"`
	Compression CompressionConfig `yaml:"compression" json:"compression" gorm:"embedded;embeddedPrefix:comp_"`
	Notify      NotifyConfig      `yaml:"notify" json:"notify" gorm:"embedded;embeddedPrefix:notify_"`
	CreatedAt   time.Time         `yaml:"-" json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time         `yaml:"-" json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName 指定表名
func (BackupTask) TableName() string {
	return "backup_tasks"
}

// DatabaseConfig 数据库连接配置
type DatabaseConfig struct {
	Type     DatabaseType      `yaml:"type" json:"type" gorm:"size:20;not null"`
	Host     string            `yaml:"host" json:"host" gorm:"size:100"`
	Port     int               `yaml:"port" json:"port"`
	Username string            `yaml:"username" json:"username" gorm:"size:50"`
	Password string            `yaml:"password" json:"password" gorm:"size:255"` // 建议加密存储
	Database string            `yaml:"database" json:"database" gorm:"size:100"`
	Params   map[string]string `yaml:"params" json:"params" gorm:"type:json"`
}

// ScheduleConfig 调度配置
type ScheduleConfig struct {
	Cron     string `yaml:"cron" json:"cron" gorm:"size:50"`
	Timezone string `yaml:"timezone" json:"timezone" gorm:"size:50;default:Asia/Shanghai"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	Type           string `yaml:"type" json:"type" gorm:"size:20;default:local"` // local/s3/oss/cos
	Path           string `yaml:"path" json:"path" gorm:"size:255"`
	Endpoint       string `yaml:"endpoint" json:"endpoint" gorm:"size:255"`
	Bucket         string `yaml:"bucket" json:"bucket" gorm:"size:100"`
	Region         string `yaml:"region" json:"region" gorm:"size:50"`
	AccessKey      string `yaml:"access_key" json:"access_key" gorm:"size:100"`
	SecretKey      string `yaml:"secret_key" json:"secret_key" gorm:"size:100"`
	ForcePathStyle bool   `yaml:"force_path_style" json:"force_path_style" gorm:"default:false"`
	// OSS 专用
	OSSEndpoint string `yaml:"oss_endpoint" json:"oss_endpoint" gorm:"size:255"`
	OSSBucket   string `yaml:"oss_bucket" json:"oss_bucket" gorm:"size:100"`
	// COS 专用
	COSEndpoint string `yaml:"cos_endpoint" json:"cos_endpoint" gorm:"size:255"`
	COSBucket   string `yaml:"cos_bucket" json:"cos_bucket" gorm:"size:100"`
	COSRegion   string `yaml:"cos_region" json:"cos_region" gorm:"size:50"`
	// 加密配置
	Encryption EncryptionConfig `yaml:"encryption" json:"encryption" gorm:"embedded;embeddedPrefix:enc_"`
}

// EncryptionConfig 加密配置
type EncryptionConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled" gorm:"default:false"`
	Type    string `yaml:"type" json:"type" gorm:"size:20;default:aes256"` // aes256
	Key     string `yaml:"key" json:"key" gorm:"size:255"`                 // 加密密钥（建议使用环境变量）
	KeyEnv  string `yaml:"key_env" json:"key_env" gorm:"size:50"`          // 从环境变量读取密钥
}

// RetentionConfig 保留策略
type RetentionConfig struct {
	KeepLast    int `yaml:"keep_last" json:"keep_last" gorm:"default:7"`
	KeepDays    int `yaml:"keep_days" json:"keep_days" gorm:"default:30"`
	KeepWeekly  int `yaml:"keep_weekly" json:"keep_weekly" gorm:"default:4"`
	KeepMonthly int `yaml:"keep_monthly" json:"keep_monthly" gorm:"default:3"`
}

// CompressionConfig 压缩配置
type CompressionConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled" gorm:"default:true"`
	Type    string `yaml:"type" json:"type" gorm:"size:20;default:gzip"` // gzip/zstd/lz4
	Level   int    `yaml:"level" json:"level" gorm:"default:6"`
}

// NotifyConfig 通知配置
type NotifyConfig struct {
	Enabled   bool     `yaml:"enabled" json:"enabled" gorm:"default:false"`
	Type      string   `yaml:"type" json:"type" gorm:"size:20"` // webhook/dingtalk/feishu/slack
	Endpoint  string   `yaml:"endpoint" json:"endpoint" gorm:"size:255"`
	Receivers []string `yaml:"receivers" json:"receivers" gorm:"type:json"`
}

// BackupRecord 备份执行记录
type BackupRecord struct {
	ID         uint       `yaml:"-" json:"id" gorm:"primaryKey;autoIncrement"`
	TaskID     string     `yaml:"task_id" json:"task_id" gorm:"size:36;index"`
	TraceID    string     `yaml:"trace_id" json:"trace_id" gorm:"size:32;uniqueIndex"`
	Status     TaskStatus `yaml:"status" json:"status" gorm:"size:20;default:pending;index"`
	StartTime  *time.Time `yaml:"start_time" json:"start_time"`
	EndTime    *time.Time `yaml:"end_time" json:"end_time"`
	DurationMs int64      `yaml:"duration_ms" json:"duration_ms"`
	FilePath   string     `yaml:"file_path" json:"file_path" gorm:"size:512"`
	FileSize   int64      `yaml:"file_size" json:"file_size"`
	Checksum   string     `yaml:"checksum" json:"checksum" gorm:"size:64"`
	ErrorMsg   string     `yaml:"error_msg" json:"error_msg" gorm:"type:text"`
	RawLog     string     `yaml:"raw_log" json:"raw_log" gorm:"type:text"`
	CreatedAt  time.Time  `yaml:"-" json:"created_at" gorm:"autoCreateTime;index"`
}

// TableName 指定表名
func (BackupRecord) TableName() string {
	return "backup_records"
}

// BackupResult 备份执行结果
type BackupResult struct {
	TaskID    string        `json:"task_id"`
	TraceID   string        `json:"trace_id"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
	Duration  time.Duration `json:"duration"`
	Status    TaskStatus    `json:"status"`
	FilePath  string        `json:"file_path"`
	FileSize  int64         `json:"file_size"`
	Checksum  string        `json:"checksum"`
	Error     string        `json:"error,omitempty"`
}

// GlobalConfig 全局配置
type GlobalConfig struct {
	WorkDir       string        `yaml:"work_dir" json:"work_dir"`
	DefaultTZ     string        `yaml:"default_tz" json:"default_tz"`
	MaxConcurrent int           `yaml:"max_concurrent" json:"max_concurrent"`
	Timeout       time.Duration `yaml:"timeout" json:"timeout"`
	EncryptionKey  string        `yaml:"encryption_key" json:"-"`   // 密码加密密钥，json:"-" 防止泄露
	APIKeys        []string      `yaml:"api_keys" json:"-"`         // API 认证密钥列表
	AllowedOrigins []string      `yaml:"allowed_origins" json:"allowed_origins"` // WebSocket 允许的 Origin 白名单
}

// Config 完整配置
type Config struct {
	Global GlobalConfig `yaml:"global" json:"global"`
	Tasks  []BackupTask `yaml:"tasks" json:"tasks"`
	Log    LogConfig    `yaml:"log" json:"log"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `yaml:"level" json:"level"`
	File   string `yaml:"file" json:"file"`
	Format string `yaml:"format" json:"format"`
}

// LogWriter 日志写入接口（用于实时日志推送）
type LogWriter interface {
	Write(p []byte) (n int, err error)
	WriteString(s string) (n int, err error)
	Close() error
}

// NilWriter 空日志写入器（用于不需要日志输出的场景）
type NilWriter struct{}

func (w *NilWriter) Write(p []byte) (n int, err error)       { return len(p), nil }
func (w *NilWriter) WriteString(s string) (n int, err error) { return len(s), nil }
func (w *NilWriter) Close() error                            { return nil }
