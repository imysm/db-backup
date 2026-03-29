package dto

import "time"

// 创建备份任务请求
type CreateJobRequest struct {
	Name            string `json:"name" binding:"required"`
	DatabaseType    string `json:"database_type" binding:"required"`
	Host            string `json:"host" binding:"required"`
	Port            int    `json:"port" binding:"required"`
	Database        string `json:"database" binding:"required"`
	Username        string `json:"username"`
	Password        string `json:"password"`
	Schedule        string `json:"schedule" binding:"required"`
	RetentionDays   int    `json:"retention_days"`
	BackupType      string `json:"backup_type"`
	StorageType     string `json:"storage_type" binding:"required"`
	Compress        bool   `json:"compress"`
	Encrypt         bool   `json:"encrypt"`
	NotifyOnSuccess bool   `json:"notify_on_success"`
	NotifyOnFail    bool   `json:"notify_on_fail"`
}

// 更新备份任务请求
type UpdateJobRequest struct {
	Name            string `json:"name"`
	Schedule        string `json:"schedule"`
	RetentionDays   int    `json:"retention_days"`
	StorageType     string `json:"storage_type"`
	Compress        bool   `json:"compress"`
	Encrypt         bool   `json:"encrypt"`
	Enabled         *bool  `json:"enabled"`
	NotifyOnSuccess bool   `json:"notify_on_success"`
	NotifyOnFail    bool   `json:"notify_on_fail"`
}

// 备份任务响应
type JobResponse struct {
	ID            uint       `json:"id"`
	Name          string     `json:"name"`
	DatabaseType  string     `json:"database_type"`
	Host          string     `json:"host"`
	Port          int        `json:"port"`
	Database      string     `json:"database"`
	Schedule      string     `json:"schedule"`
	RetentionDays int        `json:"retention_days"`
	BackupType    string     `json:"backup_type"`
	StorageType   string     `json:"storage_type"`
	Compress      bool       `json:"compress"`
	Encrypt       bool       `json:"encrypt"`
	Enabled       bool       `json:"enabled"`
	LastRun       *time.Time `json:"last_run"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// 备份记录响应
type RecordResponse struct {
	ID           uint       `json:"id"`
	JobID        uint       `json:"job_id"`
	JobName      string     `json:"job_name"`
	StartedAt    time.Time  `json:"started_at"`
	FinishedAt   *time.Time `json:"finished_at"`
	Duration     int        `json:"duration"`
	FileSize     int64      `json:"file_size"`
	FilePath     string     `json:"file_path"`
	Checksum     string     `json:"checksum"`
	Status       string     `json:"status"`
	ErrorMessage string     `json:"error_message,omitempty"`
	Verified     bool       `json:"verified"`
	VerifiedAt   *time.Time `json:"verified_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

// 通用响应
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func Success(data interface{}) Response {
	return Response{Code: 0, Message: "success", Data: data}
}

func Error(message string) Response {
	return Response{Code: 1, Message: message}
}
