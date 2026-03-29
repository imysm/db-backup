package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// BackupTotal 备份总数
	BackupTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "db_backup_total",
		Help: "Total number of database backups",
	}, []string{"database", "type", "status"})

	// BackupDuration 备份耗时
	BackupDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "db_backup_duration_seconds",
		Help:    "Duration of database backups in seconds",
		Buckets: prometheus.ExponentialBuckets(1, 2, 10), // 1s, 2s, 4s, ..., 512s
	}, []string{"database", "type"})

	// BackupSize 备份文件大小
	BackupSize = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "db_backup_size_bytes",
		Help: "Size of backup files in bytes",
	}, []string{"database", "type"})

	// StorageUsage 存储使用量
	StorageUsage = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "db_backup_storage_usage_bytes",
		Help: "Total storage usage in bytes",
	}, []string{"storage_type"})

	// JobStatus 任务状态
	JobStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "db_backup_job_status",
		Help: "Status of backup jobs (1=running, 0=stopped)",
	}, []string{"job_id", "database"})

	// RestoreTotal 恢复总数
	RestoreTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "db_backup_restore_total",
		Help: "Total number of database restores",
	}, []string{"database", "status"})

	// VerifyTotal 验证总数
	VerifyTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "db_backup_verify_total",
		Help: "Total number of backup verifications",
	}, []string{"database", "status"})

	// ActiveJobs 活跃任务数
	ActiveJobs = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "db_backup_active_jobs",
		Help: "Number of currently active backup jobs",
	})
)

// RecordBackup 记录备份指标
func RecordBackup(database, backupType, status string, durationSeconds float64, sizeBytes int64) {
	BackupTotal.WithLabelValues(database, backupType, status).Inc()
	BackupDuration.WithLabelValues(database, backupType).Observe(durationSeconds)
	BackupSize.WithLabelValues(database, backupType).Set(float64(sizeBytes))
}

// RecordRestore 记录恢复指标
func RecordRestore(database, status string) {
	RestoreTotal.WithLabelValues(database, status).Inc()
}

// RecordVerify 记录验证指标
func RecordVerify(database, status string) {
	VerifyTotal.WithLabelValues(database, status).Inc()
}

// UpdateJobStatus 更新任务状态
func UpdateJobStatus(jobID, database string, running bool) {
	status := float64(0)
	if running {
		status = 1
	}
	JobStatus.WithLabelValues(jobID, database).Set(status)
}

// UpdateStorageUsage 更新存储使用量
func UpdateStorageUsage(storageType string, usageBytes int64) {
	StorageUsage.WithLabelValues(storageType).Set(float64(usageBytes))
}

// SetActiveJobs 设置活跃任务数
func SetActiveJobs(count int) {
	ActiveJobs.Set(float64(count))
}
