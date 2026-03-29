package model

import (
	"time"
)

// DatabaseType 数据库类型
type DatabaseType string

const (
	DatabaseTypePostgres  DatabaseType = "postgres"
	DatabaseTypeMySQL     DatabaseType = "mysql"
	DatabaseTypeMongoDB   DatabaseType = "mongodb"
	DatabaseTypeSQLServer DatabaseType = "sqlserver"
	DatabaseTypeOracle    DatabaseType = "oracle"
)

// StorageType 存储类型
type StorageType string

const (
	StorageTypeLocal StorageType = "local"
	StorageTypeS3    StorageType = "s3"
	StorageTypeOSS   StorageType = "oss"
	StorageTypeCOS   StorageType = "cos"
)

// BackupType 备份类型
type BackupType string

const (
	BackupTypeFull        BackupType = "full"
	BackupTypeIncremental BackupType = "incremental"
)

// BackupStatus 备份状态
type BackupStatus string

const (
	BackupStatusPending BackupStatus = "pending"
	BackupStatusRunning BackupStatus = "running"
	BackupStatusSuccess BackupStatus = "success"
	BackupStatusFailed  BackupStatus = "failed"
)

// BackupJob 备份任务
type BackupJob struct {
	ID           uint         `gorm:"primaryKey" json:"id"`
	Name         string       `gorm:"size:128;notNull" json:"name"`
	DatabaseType DatabaseType `gorm:"size:20;notNull" json:"database_type"`
	Host         string       `gorm:"size:255;notNull" json:"host"`
	Port         int          `gorm:"notNull" json:"port"`
	Database     string       `gorm:"size:128;notNull" json:"database"`
	Username     string       `gorm:"size:128" json:"username"`
	Password     string       `gorm:"size:255" json:"-"` // 加密存储

	// 备份策略
	Schedule      string     `gorm:"size:64;notNull" json:"schedule"` // cron表达式
	RetentionDays int        `gorm:"default:7" json:"retention_days"`
	BackupType    BackupType `gorm:"size:20;default:full" json:"backup_type"`

	// 存储配置
	StorageType   StorageType `gorm:"size:20;notNull" json:"storage_type"`
	StorageConfig string      `gorm:"type:json" json:"storage_config"`

	// 压缩加密
	Compress   bool   `gorm:"default:true" json:"compress"`
	Encrypt    bool   `gorm:"default:false" json:"encrypt"`
	EncryptKey string `gorm:"size:255" json:"-"`

	// 通知配置
	NotifyOnSuccess bool   `gorm:"default:false" json:"notify_on_success"`
	NotifyOnFail    bool   `gorm:"default:true" json:"notify_on_fail"`
	NotifyChannels  string `gorm:"type:json" json:"notify_channels"`

	// 状态
	Enabled bool       `gorm:"default:true" json:"enabled"`
	LastRun *time.Time `json:"last_run"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (BackupJob) TableName() string {
	return "backup_jobs"
}

// BackupRecord 备份记录
type BackupRecord struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	JobID   uint   `gorm:"index;notNull" json:"job_id"`
	JobName string `gorm:"size:128" json:"job_name"`

	// 备份信息
	StartedAt  time.Time  `gorm:"notNull" json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
	Duration   int        `json:"duration"`  // seconds
	FileSize   int64      `json:"file_size"` // bytes
	FilePath   string     `gorm:"size:512" json:"file_path"`
	Checksum   string     `gorm:"size:64" json:"checksum"`

	// 状态
	Status       BackupStatus `gorm:"size:20;notNull" json:"status"`
	ErrorMessage string       `gorm:"type:text" json:"error_message"`

	// 验证
	Verified   bool       `gorm:"default:false" json:"verified"`
	VerifiedAt *time.Time `json:"verified_at"`

	CreatedAt time.Time `json:"created_at"`
}

func (BackupRecord) TableName() string {
	return "backup_records"
}

// BackupHistory 备份历史
type BackupHistory struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	JobID    uint   `gorm:"index" json:"job_id"`
	RecordID uint   `gorm:"index" json:"record_id"`
	Action   string `gorm:"size:20;notNull" json:"action"` // created, verified, restored, deleted
	Details  string `gorm:"type:text" json:"details"`

	CreatedAt time.Time `json:"created_at"`
}

func (BackupHistory) TableName() string {
	return "backup_history"
}

// StorageConfig 存储配置 (JSON)
type StorageConfig struct {
	// 本地存储
	BasePath string `json:"base_path,omitempty"`

	// S3/MinIO
	Endpoint       string `json:"endpoint,omitempty"`
	Region         string `json:"region,omitempty"`
	Bucket         string `json:"bucket,omitempty"`
	AccessKey      string `json:"access_key,omitempty"`
	SecretKey      string `json:"secret_key,omitempty"`
	ForcePathStyle bool   `json:"force_path_style,omitempty"`

	// OSS
	OSSEndpoint string `json:"oss_endpoint,omitempty"`
	OSSBucket   string `json:"oss_bucket,omitempty"`

	// COS
	COSEndpoint string `json:"cos_endpoint,omitempty"`
	COSBucket   string `json:"cos_bucket,omitempty"`
	COSRegion   string `json:"cos_region,omitempty"`
}

// NotifyConfig 通知配置 (JSON)
type NotifyConfig struct {
	Email    []string `json:"email,omitempty"`
	DingTalk []string `json:"dingtalk,omitempty"`
	Feishu   []string `json:"feishu,omitempty"`
	WeChat   []string `json:"wechat,omitempty"`
}
