# Phase 5: 多存储目标 + 备份窗口 + 文件浏览器 — 产品设计文档

> 版本: v1.0 | 日期: 2026-03-27 | 状态: 设计中

## 1. 产品概述

Phase 5 从**存储灵活性**、**执行精细化控制**和**文件可管理性**三个维度提升系统能力：

| 模块 | 解决的问题 | 核心价值 |
|------|-----------|---------|
| **多存储目标** | 单一存储存在单点故障风险 | 一份数据多地存储，提升容灾能力 |
| **备份窗口** | 备份占用业务资源、无法精确控制执行时间 | 精确控制备份执行时段和资源消耗 |
| **文件浏览器** | 备份文件不可见、不可管理 | 在线浏览和管理所有备份文件 |

---

## 2. 多存储目标

### 2.1 功能描述

支持一个备份任务同时写入多个存储目标（如本地 + S3、S3 + OSS），实现异地容灾和存储冗余。

**典型场景**：

| 场景 | 存储组合 | 说明 |
|------|---------|------|
| 本地快速恢复 + 云端异地容灾 | local + s3 | 本地用于快速恢复，S3 用于异地灾备 |
| 跨云容灾 | s3 + oss | 阿里云 OSS + AWS S3 双云冗余 |
| 三副本高可靠 | local + s3 + oss | 金融级数据安全 |
| 分级存储 | local + s3 | 热数据本地，冷数据云端 |

### 2.2 架构设计

**主存储 + 镜像存储模式**：

```
┌──────────────────────────────────────────────────────────┐
│                    备份执行器                              │
│                                                          │
│  ┌──────────┐    ┌──────────────────────────────────┐    │
│  │ 数据库   │───▶│        备份文件 (本地临时)         │    │
│  │ dumper   │    └──────────┬───────────────────────┘    │
│  └──────────┘               │                            │
│                             ▼                            │
│              ┌──────────────────────────┐                │
│              │     多存储写入协调器       │                │
│              │                          │                │
│              │  写入策略: 至少一个成功     │                │
│              └──┬───────┬───────┬───────┘                │
│                 │       │       │                         │
│                 ▼       ▼       ▼                         │
│         ┌───────┐ ┌───────┐ ┌───────┐                   │
│         │ Local │ │  S3   │ │  OSS  │                   │
│         │(主存储)│ │(镜像1) │ │(镜像2) │                   │
│         └───────┘ └───────┘ └───────┘                   │
│              │       │       │                         │
│              ▼       ▼       ▼                         │
│         [成功✓]  [成功✓]  [失败✗]                      │
│                                                          │
│         最终状态: 部分成功 (2/3)                          │
└──────────────────────────────────────────────────────────┘
```

**写入策略**：

| 策略 | 说明 | 适用场景 |
|------|------|---------|
| `at_least_one` | 至少一个存储写入成功即视为成功 | 默认策略，兼顾可用性和容灾 |
| `all` | 全部存储写入成功才视为成功 | 金融级场景，要求绝对一致 |
| `primary_only` | 仅主存储成功即可，镜像异步同步 | 主存储优先，镜像作为增强 |

**并发写入流程**：

```
开始备份
    │
    ▼
数据库 dump 到临时文件
    │
    ▼
计算校验和 (SHA256)
    │
    ▼
并发写入所有存储目标 ──────┐
    │                     │
    ├─▶ 写入 Local ◀──────┤ goroutine 1
    │   [成功/失败]        │
    │                     │
    ├─▶ 写入 S3    ◀──────┤ goroutine 2
    │   [成功/失败]        │
    │                     │
    └─▶ 写入 OSS   ◀──────┘ goroutine 3
        [成功/失败]
    │
    ▼
汇总结果，判断最终状态
    │
    ├── at_least_one: 任一成功 → success
    ├── all: 全部成功 → success
    └── 否则 → failed
    │
    ▼
记录每个存储的写入结果
清理临时文件
```

### 2.3 数据库设计变更

**新增 `backup_storages` 关联表**：

```sql
-- 备份任务存储配置关联表
CREATE TABLE backup_storages (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    task_id         VARCHAR(36)    NOT NULL,
    storage_name    VARCHAR(100)   NOT NULL COMMENT '存储名称（如：主存储、异地灾备）',
    storage_type    VARCHAR(20)    NOT NULL COMMENT '存储类型: local/s3/oss/cos',
    is_primary      BOOLEAN        NOT NULL DEFAULT FALSE COMMENT '是否为主存储',
    priority        INT            NOT NULL DEFAULT 0 COMMENT '优先级，越小越优先',
    write_strategy  VARCHAR(20)    NOT NULL DEFAULT 'at_least_one' COMMENT '写入策略',
    
    -- 存储配置（JSON，复用 StorageConfig 结构）
    config          JSON           NOT NULL COMMENT '存储连接配置',
    
    enabled         BOOLEAN        NOT NULL DEFAULT TRUE,
    created_at      DATETIME       DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME       DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    UNIQUE KEY uk_task_name (task_id, storage_name),
    INDEX idx_task_id (task_id),
    INDEX idx_task_enabled (task_id, enabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='备份任务存储配置';
```

**新增 `backup_storage_results` 记录表**：

```sql
-- 每次备份执行时各存储的写入结果
CREATE TABLE backup_storage_results (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    record_id       BIGINT UNSIGNED NOT NULL COMMENT '关联 backup_records.id',
    task_id         VARCHAR(36)    NOT NULL,
    storage_id      BIGINT UNSIGNED NOT NULL COMMENT '关联 backup_storages.id',
    storage_name    VARCHAR(100)   NOT NULL,
    storage_type    VARCHAR(20)    NOT NULL,
    
    status          VARCHAR(20)    NOT NULL DEFAULT 'pending' COMMENT 'pending/uploading/success/failed/skipped',
    file_path       VARCHAR(512)   DEFAULT NULL COMMENT '该存储上的文件路径',
    file_size       BIGINT         DEFAULT NULL COMMENT '该存储上的文件大小',
    checksum        VARCHAR(64)    DEFAULT NULL COMMENT '该存储上的文件校验和',
    error_msg       TEXT           DEFAULT NULL,
    
    started_at      DATETIME       DEFAULT NULL,
    completed_at    DATETIME       DEFAULT NULL,
    duration_ms     BIGINT         DEFAULT NULL,
    
    created_at      DATETIME       DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_record_id (record_id),
    INDEX idx_task_id (task_id),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='备份存储写入结果';
```

**`backup_tasks` 表变更**：

```sql
-- 新增写入策略字段
ALTER TABLE backup_tasks 
    ADD COLUMN write_strategy VARCHAR(20) NOT NULL DEFAULT 'at_least_one' 
    COMMENT '多存储写入策略: at_least_one/all/primary_only' 
    AFTER compression;

-- 原有 storage_* 嵌入字段保留兼容，新任务优先使用 backup_storages 关联表
```

### 2.4 API 设计

**创建任务时指定多存储**：

```
POST /api/v1/tasks
```

请求体变更 — `storages` 从单对象改为数组：

```json
{
  "name": "production-mysql-daily",
  "database": {
    "type": "mysql",
    "host": "10.0.0.1",
    "port": 3306,
    "username": "backup",
    "password": "***",
    "database": "app_production"
  },
  "schedule": {
    "cron": "0 2 * * *",
    "timezone": "Asia/Shanghai"
  },
  "write_strategy": "at_least_one",
  "storages": [
    {
      "storage_name": "本地存储",
      "storage_type": "local",
      "is_primary": true,
      "priority": 1,
      "config": {
        "type": "local",
        "path": "/data/backups/mysql"
      }
    },
    {
      "storage_name": "S3 异地灾备",
      "storage_type": "s3",
      "is_primary": false,
      "priority": 2,
      "config": {
        "type": "s3",
        "endpoint": "https://s3.us-west-2.amazonaws.com",
        "bucket": "my-db-backups",
        "region": "us-west-2",
        "access_key": "AKIA***",
        "secret_key": "***"
      }
    },
    {
      "storage_name": "OSS 备份",
      "storage_type": "oss",
      "is_primary": false,
      "priority": 3,
      "config": {
        "type": "oss",
        "oss_endpoint": "https://oss-cn-hangzhou.aliyuncs.com",
        "oss_bucket": "my-db-backups",
        "access_key": "LTAI***",
        "secret_key": "***"
      }
    }
  ],
  "retention": {
    "keep_last": 7,
    "keep_days": 30
  }
}
```

**获取任务详情 — 包含存储列表**：

```
GET /api/v1/tasks/:id
```

响应：

```json
{
  "id": "task-uuid-001",
  "name": "production-mysql-daily",
  "write_strategy": "at_least_one",
  "storages": [
    {
      "id": 1,
      "storage_name": "本地存储",
      "storage_type": "local",
      "is_primary": true,
      "priority": 1,
      "enabled": true,
      "last_status": "success",
      "last_backup_time": "2026-03-27T02:00:00+08:00"
    },
    {
      "id": 2,
      "storage_name": "S3 异地灾备",
      "storage_type": "s3",
      "is_primary": false,
      "priority": 2,
      "enabled": true,
      "last_status": "success",
      "last_backup_time": "2026-03-27T02:00:05+08:00"
    }
  ]
}
```

**获取备份记录 — 包含各存储写入结果**：

```
GET /api/v1/records/:id
```

响应新增 `storage_results` 字段：

```json
{
  "id": 1024,
  "task_id": "task-uuid-001",
  "status": "success",
  "file_path": "/data/backups/mysql/app_production_20260327.sql.gz",
  "file_size": 1073741824,
  "storage_results": [
    {
      "storage_id": 1,
      "storage_name": "本地存储",
      "storage_type": "local",
      "status": "success",
      "file_path": "/data/backups/mysql/app_production_20260327.sql.gz",
      "file_size": 1073741824,
      "checksum": "sha256:abc123...",
      "duration_ms": 5200
    },
    {
      "storage_id": 2,
      "storage_name": "S3 异地灾备",
      "storage_type": "s3",
      "status": "success",
      "file_path": "s3://my-db-backups/mysql/app_production_20260327.sql.gz",
      "file_size": 1073741824,
      "checksum": "sha256:abc123...",
      "duration_ms": 15200
    }
  ]
}
```

**存储管理 API**：

```
POST   /api/v1/tasks/:id/storages          # 添加存储目标
PUT    /api/v1/tasks/:id/storages/:sid      # 更新存储配置
DELETE /api/v1/tasks/:id/storages/:sid      # 删除存储目标
PUT    /api/v1/tasks/:id/storages/reorder   # 重新排序存储优先级
POST   /api/v1/tasks/:id/storages/:sid/sync # 同步已有备份到该存储
POST   /api/v1/tasks/:id/storages/:sid/health-check # 存储健康检查
```

### 2.5 后端实现方案

**多存储写入协调器**：

```go
// internal/storage/coordinator.go
type StorageCoordinator struct {
    storages     []TargetStorage
    strategy     WriteStrategy
    maxRetries   int
    retryDelay   time.Duration
}

type WriteStrategy string
const (
    StrategyAtLeastOne WriteStrategy = "at_least_one"
    StrategyAll        WriteStrategy = "all"
    StrategyPrimary    WriteStrategy = "primary_only"
)

// WriteResult 单个存储的写入结果
type WriteResult struct {
    StorageID   uint64
    StorageName string
    Status      string // success/failed
    FilePath    string
    FileSize    int64
    Checksum    string
    Duration    time.Duration
    Error       error
}

// Write 并发写入所有存储
func (c *StorageCoordinator) Write(ctx context.Context, localPath, remotePath string) ([]WriteResult, error) {
    var wg sync.WaitGroup
    results := make([]WriteResult, len(c.storages))
    
    for i, s := range c.storages {
        wg.Add(1)
        go func(idx int, s TargetStorage) {
            defer wg.Done()
            results[idx] = c.writeWithRetry(ctx, s, localPath, remotePath)
        }(i, s)
    }
    wg.Wait()
    
    return results, c.evaluateStrategy(results)
}

func (c *StorageCoordinator) evaluateStrategy(results []WriteResult) error {
    successCount := 0
    primarySuccess := false
    
    for _, r := range results {
        if r.Status == "success" {
            successCount++
            // 检查是否是主存储
        }
    }
    
    switch c.strategy {
    case StrategyAtLeastOne:
        if successCount == 0 {
            return fmt.Errorf("所有存储写入失败")
        }
    case StrategyAll:
        if successCount != len(results) {
            return fmt.Errorf("部分存储写入失败 (%d/%d)", successCount, len(results))
        }
    }
    return nil
}
```

**存储同步**：

```go
// SyncBackups 同步已有备份到新增存储
func (c *StorageCoordinator) SyncBackups(ctx context.Context, records []BackupRecord) []SyncResult {
    // 1. 获取源存储（主存储）的备份文件列表
    // 2. 逐个文件从源存储下载 → 上传到目标存储
    // 3. 验证校验和
    // 4. 记录同步结果
    // 支持断点续传和并发同步（可配置并发数）
}
```

### 2.6 前端交互

**任务创建/编辑页面 — 存储配置区域**：

```
┌──────────────────────────────────────────────────────────────┐
│  存储配置                                    [+ 添加存储]    │
│                                                              │
│  写入策略:  ○ 至少一个成功  ● 全部成功  ○ 仅主存储           │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐  │
│  │ ≡  📁 本地存储                    [主存储] [✏️] [🗑️]   │  │
│  │    类型: 本地存储  |  路径: /data/backups/mysql        │  │
│  │    上次备份: 2026-03-27 02:00  |  状态: ✅ 成功        │  │
│  ├────────────────────────────────────────────────────────┤  │
│  │ ≡  ☁️ S3 异地灾备                  [镜像] [✏️] [🗑️]    │  │
│  │    类型: AWS S3  |  Bucket: my-db-backups              │  │
│  │    上次备份: 2026-03-27 02:00  |  状态: ✅ 成功        │  │
│  ├────────────────────────────────────────────────────────┤  │
│  │ ≡  ☁️ OSS 备份                      [镜像] [✏️] [🗑️]    │  │
│  │    类型: 阿里云 OSS  |  Bucket: my-db-backups          │  │
│  │    上次备份: 2026-03-27 02:00  |  状态: ❌ 失败        │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                              │
│  💡 拖拽卡片可调整存储优先级，排名第一的自动设为主存储        │
└──────────────────────────────────────────────────────────────┘
```

**存储同步对话框**：

```
┌──────────────────────────────────────────┐
│  同步备份到 S3 异地灾备                   │
│                                          │
│  将已有 15 个备份文件同步到该存储          │
│  预计数据量: 12.8 GB                      │
│                                          │
│  并发数:  [3 ▼]                           │
│  □ 跳过已存在的文件                        │
│  □ 同步完成后验证校验和                    │
│                                          │
│  [取消]  [开始同步]                       │
│                                          │
│  同步进度: 8/15 (53%)                     │
│  ████████████░░░░░░░░░░  7.2/12.8 GB     │
│  预计剩余时间: 3 分钟                     │
└──────────────────────────────────────────┘
```

### 2.7 存储健康检查

**检查内容**：

| 检查项 | 说明 | 频率 |
|--------|------|------|
| 连通性 | 验证存储连接是否正常 | 每次备份前 |
| 空间可用 | 检查存储剩余空间 | 每 6 小时 |
| 文件完整性 | 校验最近 N 个备份的 checksum | 每日 |
| 权限验证 | 验证读写权限 | 每次备份前 |

**健康检查 API**：

```
POST /api/v1/tasks/:id/storages/:sid/health-check
```

响应：

```json
{
  "storage_id": 2,
  "storage_name": "S3 异地灾备",
  "checks": {
    "connectivity": { "status": "ok", "latency_ms": 45 },
    "space": { "status": "ok", "used_gb": 68.3, "total_gb": 100, "usage_percent": 68.3 },
    "integrity": { "status": "warning", "checked": 15, "passed": 14, "failed": 1, "failed_files": ["app_production_20260320.sql.gz"] },
    "permissions": { "status": "ok", "read": true, "write": true }
  },
  "overall": "warning",
  "checked_at": "2026-03-27T10:00:00+08:00"
}
```

---

## 3. 备份窗口管理

### 3.1 功能描述

精确控制备份任务的执行时段，避免在业务高峰期占用系统资源。

### 3.2 配置设计

**两级配置**：

| 级别 | 说明 | 优先级 |
|------|------|--------|
| **全局级别** | 适用于所有任务的默认备份窗口 | 低 |
| **任务级别** | 单个任务的自定义备份窗口，覆盖全局配置 | 高 |

**备份窗口配置模型**：

```go
type BackupWindow struct {
    Enabled      bool     `json:"enabled" gorm:"default:false"`
    Weekdays     []int    `json:"weekdays" gorm:"type:json"`       // 0=周日, 1=周一, ..., 6=周六
    StartTime    string   `json:"start_time" gorm:"size:5"`        // "02:00"
    EndTime      string   `json:"end_time" gorm:"size:5"`          // "06:00"
    Timezone     string   `json:"timezone" gorm:"size:50;default:Asia/Shanghai"`
    MaxDuration  int      `json:"max_duration" gorm:"default:0"`   // 最大执行时长(分钟), 0=不限
    MaxBandwidth int      `json:"max_bandwidth" gorm:"default:0"`  // 最大带宽限制(MB/s), 0=不限
    MaxIO        int      `json:"max_io" gorm:"default:0"`         // 最大 I/O 限制(MB/s), 0=不限
}
```

### 3.3 数据库设计变更

**`backup_tasks` 表新增字段**：

```sql
ALTER TABLE backup_tasks
    ADD COLUMN backup_window JSON DEFAULT NULL 
    COMMENT '备份窗口配置（JSON 格式的 BackupWindow）' 
    AFTER write_strategy;
```

**`global_settings` 表新增全局备份窗口**：

```sql
INSERT INTO global_settings (`key`, `value`, `description`) VALUES
('backup_window', '{"enabled":false,"weekdays":[0,1,2,3,4,5,6],"start_time":"00:00","end_time":"23:59","timezone":"Asia/Shanghai","max_duration":0,"max_bandwidth":0,"max_io":0}', '全局默认备份窗口配置');
```

### 3.4 API 设计

**获取/更新全局备份窗口**：

```
GET /api/v1/settings/backup-window
PUT /api/v1/settings/backup-window
```

请求体：

```json
{
  "enabled": true,
  "weekdays": [1, 2, 3, 4, 5],
  "start_time": "01:00",
  "end_time": "06:00",
  "timezone": "Asia/Shanghai",
  "max_duration": 240,
  "max_bandwidth": 50,
  "max_io": 100
}
```

**创建/更新任务时指定备份窗口**：

```json
{
  "name": "production-mysql-daily",
  "backup_window": {
    "enabled": true,
    "weekdays": [1, 2, 3, 4, 5],
    "start_time": "02:00",
    "end_time": "05:00",
    "max_duration": 180,
    "max_bandwidth": 20
  }
}
```

### 3.5 后端实现方案

**窗口检查逻辑**（调度器在触发任务前检查）：

```go
func (s *Scheduler) isInBackupWindow(task *model.BackupTask, globalWindow *model.BackupWindow) bool {
    // 1. 优先使用任务级别窗口
    window := task.BackupWindow
    if window == nil || !window.Enabled {
        window = globalWindow
    }
    if window == nil || !window.Enabled {
        return true // 未配置窗口，允许执行
    }
    
    now := time.Now()
    loc, _ := time.LoadLocation(window.Timezone)
    localNow := now.In(loc)
    
    // 2. 检查星期
    weekday := int(localNow.Weekday())
    if !slices.Contains(window.Weekdays, weekday) {
        return false
    }
    
    // 3. 检查时间范围
    start, _ := time.Parse("15:04", window.StartTime)
    end, _ := time.Parse("15:04", window.EndTime)
    current := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 
        localNow.Hour(), localNow.Minute(), 0, 0, loc)
    
    startFull := time.Date(localNow.Year(), localNow.Month(), localNow.Day(),
        start.Hour(), start.Minute(), 0, 0, loc)
    endFull := time.Date(localNow.Year(), localNow.Month(), localNow.Day(),
        end.Hour(), end.Minute(), 0, 0, loc)
    
    return !current.Before(startFull) && current.Before(endFull)
}
```

**超时控制**（后台 goroutine 定期检查）：

```go
func (s *Scheduler) startTimeoutChecker() {
    go func() {
        ticker := time.NewTicker(30 * time.Second)
        defer ticker.Stop()
        for range ticker.C {
            s.checkAndCancelTimeouts()
        }
    }()
}

func (s *Scheduler) checkAndCancelTimeouts() {
    // 查询所有 running 状态的记录
    // 检查是否超过 max_duration
    // 超时则取消执行并标记为 timeout
}
```

**限速控制**：

```go
// 使用 io.LimitReader 和 rate.Limiter 实现限速
type RateLimitedWriter struct {
    writer    io.Writer
    limiter   *rate.Limiter  // golang.org/x/time/rate
    bandwidth int64 // MB/s
}

func (w *RateLimitedWriter) Write(p []byte) (n int, err error) {
    if w.limiter != nil {
        // 等待令牌
        if err := w.limiter.WaitN(context.Background(), len(p)); err != nil {
            return 0, err
        }
    }
    return w.writer.Write(p)
}
```

### 3.6 前端交互

**可视化备份窗口选择器**：

```
┌──────────────────────────────────────────────────────────────┐
│  备份窗口配置                                                 │
│                                                              │
│  ☑ 启用备份窗口                                              │
│                                                              │
│  执行时间:  [01:00] — [06:00]    时区: [Asia/Shanghai ▼]     │
│                                                              │
│  执行日期:  ☑周一 ☑周二 ☑周三 ☑周四 ☑周五 ☐周六 ☐周日       │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐    │
│  │ 时间轴 (24小时)                                       │    │
│  │ 00  01  02  03  04  05  06  07  08  ...  22  23     │    │
│  │ ░░  ████████████████  ░░  ░░  ░░  ...  ░░  ░░     │    │
│  │     ↑允许执行窗口↑                                     │    │
│  └──────────────────────────────────────────────────────┘    │
│                                                              │
│  ── 资源限制 ─────────────────────────────                   │
│                                                              │
│  最大执行时长:  [240] 分钟  (超过则自动取消)                   │
│  带宽限制:      [20]  MB/s   (0 = 不限制)                    │
│  I/O 限制:      [100] MB/s   (0 = 不限制)                    │
│                                                              │
│  💡 建议将备份安排在业务低峰期（通常凌晨 1:00-6:00）           │
└──────────────────────────────────────────────────────────────┘
```

---

## 4. 备份文件浏览器

### 4.1 功能描述

在线浏览和管理所有存储中的备份文件，支持跨存储的文件操作。

### 4.2 文件列表展示

**文件信息字段**：

| 字段 | 说明 | 来源 |
|------|------|------|
| 文件名 | 备份文件名 | BackupRecord |
| 关联任务 | 所属备份任务 | BackupTask |
| 文件大小 | 压缩后大小 | BackupRecord |
| 原始大小 | 未压缩大小 | BackupRecord（需新增字段） |
| 创建时间 | 备份执行时间 | BackupRecord |
| 存储位置 | 所在存储及路径 | backup_storage_results |
| 校验状态 | checksum 是否已验证 | backup_storage_results |
| 压缩方式 | gzip/zstd/lz4 | BackupTask |
| 备份耗时 | 备份执行耗时 | BackupRecord |

### 4.3 文件操作

| 操作 | 说明 | 权限要求 |
|------|------|---------|
| 下载 | 从存储下载文件到本地 | backup:read |
| 删除 | 从存储删除备份文件 | backup:delete |
| 验证 | 重新计算校验和并比对 | backup:verify |
| 复制 | 复制到其他存储目标 | backup:write |
| 预览 | 查看备份文件元信息和摘要 | backup:read |

### 4.4 存储空间分析

**统计指标**：

| 指标 | 说明 |
|------|------|
| 已用空间 | 当前备份文件占用总量 |
| 可用空间 | 存储剩余容量 |
| 增长趋势 | 近 7/30 天空间增长曲线 |
| 备份数量 | 当前备份文件总数 |
| 清理建议 | 基于保留策略的可清理文件列表 |

### 4.5 API 设计

**文件列表**：

```
GET /api/v1/files?storage_id=1&task_id=xxx&start_time=2026-03-01&end_time=2026-03-27&page=1&page_size=20&sort=-created_at
```

响应：

```json
{
  "total": 156,
  "page": 1,
  "page_size": 20,
  "items": [
    {
      "id": 1024,
      "task_id": "task-uuid-001",
      "task_name": "production-mysql-daily",
      "file_name": "app_production_20260327.sql.gz",
      "file_size": 1073741824,
      "file_size_display": "1.0 GB",
      "created_at": "2026-03-27T02:00:00+08:00",
      "duration_ms": 152000,
      "duration_display": "2m 32s",
      "compression": "gzip",
      "checksum": "sha256:abc123...",
      "checksum_verified": true,
      "storages": [
        {
          "storage_id": 1,
          "storage_name": "本地存储",
          "storage_type": "local",
          "file_path": "/data/backups/mysql/app_production_20260327.sql.gz",
          "status": "success"
        },
        {
          "storage_id": 2,
          "storage_name": "S3 异地灾备",
          "storage_type": "s3",
          "file_path": "s3://my-db-backups/mysql/app_production_20260327.sql.gz",
          "status": "success"
        }
      ]
    }
  ]
}
```

**文件详情**：

```
GET /api/v1/files/:id
```

**删除文件**：

```
DELETE /api/v1/files/:id
```

请求体（可选，指定从哪些存储删除）：

```json
{
  "storage_ids": [1, 2],
  "reason": "手动清理过期备份"
}
```

**验证文件**：

```
POST /api/v1/files/:id/verify
```

请求体：

```json
{
  "storage_ids": [1, 2]
}
```

响应：

```json
{
  "file_id": 1024,
  "results": [
    {
      "storage_id": 1,
      "storage_name": "本地存储",
      "status": "pass",
      "expected_checksum": "sha256:abc123...",
      "actual_checksum": "sha256:abc123...",
      "file_size": 1073741824,
      "duration_ms": 2300
    }
  ],
  "overall": "pass"
}
```

**复制文件到其他存储**：

```
POST /api/v1/files/:id/copy
```

请求体：

```json
{
  "target_storage_ids": [3],
  "verify_after_copy": true
}
```

**存储空间统计**：

```
GET /api/v1/storage/stats?days=30
```

响应：

```json
{
  "storages": [
    {
      "storage_id": 1,
      "storage_name": "本地存储",
      "storage_type": "local",
      "used_bytes": 73303549952,
      "used_display": "68.3 GB",
      "total_bytes": 107374182400,
      "total_display": "100 GB",
      "usage_percent": 68.3,
      "file_count": 156,
      "trend": [
        { "date": "2026-03-21", "used_bytes": 65000000000 },
        { "date": "2026-03-22", "used_bytes": 66000000000 },
        { "date": "2026-03-27", "used_bytes": 73303549952 }
      ],
      "cleanup_suggestion": {
        "deletable_count": 23,
        "reclaimable_bytes": 10737418240,
        "reclaimable_display": "10 GB"
      }
    }
  ]
}
```

**批量操作**：

```
POST /api/v1/files/batch
```

```json
{
  "action": "delete",
  "file_ids": [1024, 1023, 1022],
  "storage_ids": [1]
}

{
  "action": "verify",
  "file_ids": [1024, 1023, 1022]
}

{
  "action": "copy",
  "file_ids": [1024, 1023],
  "target_storage_ids": [3]
}
```

### 4.6 前端交互

**文件浏览器页面布局**：

```
┌─────────────────────────────────────────────────────────────────────┐
│  🗄️ DB Backup    [仪表盘] [任务] [记录] [📄 文件] [恢复] [设置]     │
│                                                                     │
│  文件浏览器                                                         │
│                                                                     │
│  ┌─ 筛选 ──────────────────────────────────────────────────────┐    │
│  │ 存储: [全部 ▼]  任务: [全部 ▼]  日期: [2026-03-01] - [27]  │    │
│  │ 状态: [全部 ▼]  搜索: [文件名搜索...           ] [搜索]     │    │
│  └────────────────────────────────────────────────────────────┘    │
│                                                                     │
│  ┌─ 存储空间概览 ──────────────────────────────────────────────┐    │
│  │  📁 本地存储  ██████████░░░░  68.3/100 GB (156 文件)       │    │
│  │  ☁️ S3       ███████░░░░░░░  45.2/200 GB (156 文件)       │    │
│  │  ☁️ OSS      █████░░░░░░░░░  32.1/200 GB (150 文件)       │    │
│  └────────────────────────────────────────────────────────────┘    │
│                                                                     │
│  ☑ 全选  [验证] [复制] [删除]               共 156 条  < 1 2 3 >  │
│  ┌────────────────────────────────────────────────────────────┐    │
│  │ ☐ │ 📄 app_production_20260327.sql.gz                     │    │
│  │   │    production-mysql-daily  •  1.0 GB  •  2m 32s       │    │
│  │   │    2026-03-27 02:00  •  📁本地 ☁️S3  •  ✅ 已验证     │    │
│  ├────────────────────────────────────────────────────────────┤    │
│  │ ☐ │ 📄 app_production_20260326.sql.gz                     │    │
│  │   │    production-mysql-daily  •  0.98 GB  •  2m 28s      │    │
│  │   │    2026-03-26 02:00  •  📁本地 ☁️S3  •  ✅ 已验证     │    │
│  ├────────────────────────────────────────────────────────────┤    │
│  │ ☐ │ 📄 app_production_20260325.sql.gz                     │    │
│  │   │    production-mysql-daily  •  0.97 GB  •  2m 25s      │    │
│  │   │    2026-03-25 02:00  •  📁本地        •  ⚠️ 未验证    │    │
│  └────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
```

**文件详情侧边栏**：

```
┌──────────────────────────────────┐
│  📄 文件详情               [✕]   │
│                                  │
│  app_production_20260327.sql.gz  │
│                                  │
│  ── 基本信息 ──                  │
│  任务: production-mysql-daily    │
│  大小: 1.0 GB (压缩前 3.2 GB)   │
│  压缩: gzip (level 6)            │
│  压缩率: 68.8%                   │
│  创建时间: 2026-03-27 02:00      │
│  耗时: 2m 32s                    │
│                                  │
│  ── 存储位置 ──                  │
│  ✅ 本地存储                     │
│     /data/backups/mysql/...      │
│     校验: sha256:abc123...       │
│                                  │
│  ✅ S3 异地灾备                  │
│     s3://my-db-backups/...       │
│     校验: sha256:abc123...       │
│                                  │
│  ── 操作 ──                      │
│  [📥 下载]  [📋 复制到...]       │
│  [✅ 验证]  [🗑️ 删除]            │
└──────────────────────────────────┘
```

**存储空间分析页面**：

```
┌──────────────────────────────────────────────────────────────┐
│  存储空间分析                                                │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐    │
│  │  空间使用趋势 (近 30 天)                              │    │
│  │                                                       │    │
│  │  100G ┤                              ╱── 本地(68.3G) │    │
│  │   80G ┤                        ╱──╯                  │    │
│  │   60G ┤                  ╱──╯      ── S3(45.2G)     │    │
│  │   40G ┤            ╱──╯      ╱──╯                     │    │
│  │   20G ┤      ╱──╯      ╱──╯      ── OSS(32.1G)      │    │
│  │    0G ┤──╯──╯──╯──╯──╯                             │    │
│  │       W1      W2      W3      W4                    │    │
│  └──────────────────────────────────────────────────────┘    │
│                                                              │
│  ── 清理建议 ───────────────────────────────                │
│  根据保留策略，以下文件可安全清理:                             │
│  • 23 个过期文件，可释放 10 GB 空间                           │
│  • 本地存储 3 个重复文件，可释放 2.5 GB                       │
│  [查看详情]  [一键清理]                                       │
└──────────────────────────────────────────────────────────────┘
```

---

## 5. 实施计划

| 阶段 | 内容 | 预估工时 |
|------|------|---------|
| 5.1 | 数据库迁移 + 多存储模型 | 2 天 |
| 5.2 | 多存储写入协调器 | 3 天 |
| 5.3 | 存储同步 + 健康检查 | 2 天 |
| 5.4 | 备份窗口 + 限速控制 | 2 天 |
| 5.5 | 文件浏览器 API | 3 天 |
| 5.6 | 前端页面开发 | 5 天 |
| 5.7 | 联调测试 | 2 天 |
| **合计** | | **19 天** |

---

## 6. 风险与依赖

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| 多存储并发写入增加内存占用 | 大备份文件同时上传多个存储 | 流式传输 + 限速控制 |
| 存储同步耗时过长 | 新增存储时需同步大量历史数据 | 支持后台异步同步 + 断点续传 |
| 云存储 API 调用成本 | 频繁的健康检查和文件列表 | 合理设置检查频率，缓存结果 |
| 限速控制精度 | 粗粒度限速可能影响准确性 | 使用令牌桶算法，按块限速 |
