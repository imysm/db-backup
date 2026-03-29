package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestRecordBackup(t *testing.T) {
	// 重置指标
	BackupTotal.Reset()
	BackupDuration.Reset()
	BackupSize.Reset()

	// 记录一次备份
	RecordBackup("testdb", "full", "success", 10.5, 1024)

	// 验证计数器增加
	count := testutil.ToFloat64(BackupTotal.WithLabelValues("testdb", "full", "success"))
	if count != 1 {
		t.Errorf("Expected backup count to be 1, got %f", count)
	}

	// 验证大小
	size := testutil.ToFloat64(BackupSize.WithLabelValues("testdb", "full"))
	if size != 1024 {
		t.Errorf("Expected backup size to be 1024, got %f", size)
	}
}

func TestRecordRestore(t *testing.T) {
	// 重置指标
	RestoreTotal.Reset()

	// 记录一次恢复
	RecordRestore("testdb", "success")

	// 验证计数器增加
	count := testutil.ToFloat64(RestoreTotal.WithLabelValues("testdb", "success"))
	if count != 1 {
		t.Errorf("Expected restore count to be 1, got %f", count)
	}
}

func TestRecordVerify(t *testing.T) {
	// 重置指标
	VerifyTotal.Reset()

	// 记录一次验证
	RecordVerify("testdb", "success")

	// 验证计数器增加
	count := testutil.ToFloat64(VerifyTotal.WithLabelValues("testdb", "success"))
	if count != 1 {
		t.Errorf("Expected verify count to be 1, got %f", count)
	}
}

func TestUpdateJobStatus(t *testing.T) {
	// 重置指标
	JobStatus.Reset()

	// 更新任务状态为运行中
	UpdateJobStatus("job-001", "testdb", true)
	status := testutil.ToFloat64(JobStatus.WithLabelValues("job-001", "testdb"))
	if status != 1 {
		t.Errorf("Expected job status to be 1 (running), got %f", status)
	}

	// 更新任务状态为停止
	UpdateJobStatus("job-001", "testdb", false)
	status = testutil.ToFloat64(JobStatus.WithLabelValues("job-001", "testdb"))
	if status != 0 {
		t.Errorf("Expected job status to be 0 (stopped), got %f", status)
	}
}

func TestUpdateStorageUsage(t *testing.T) {
	// 重置指标
	StorageUsage.Reset()

	// 更新存储使用量
	UpdateStorageUsage("local", 1024*1024*1024) // 1GB

	// 验证使用量
	usage := testutil.ToFloat64(StorageUsage.WithLabelValues("local"))
	if usage != 1024*1024*1024 {
		t.Errorf("Expected storage usage to be 1073741824, got %f", usage)
	}
}

func TestSetActiveJobs(t *testing.T) {
	// 设置活跃任务数
	SetActiveJobs(5)

	// 验证任务数
	count := testutil.ToFloat64(ActiveJobs)
	if count != 5 {
		t.Errorf("Expected active jobs to be 5, got %f", count)
	}
}
