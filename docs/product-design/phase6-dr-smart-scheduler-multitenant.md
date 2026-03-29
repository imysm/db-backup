# Phase 6: 一键灾难恢复 + 智能调度 + 多租户 — 产品设计文档

> 版本: v1.0 | 日期: 2026-03-27 | 状态: 设计中

## 1. 产品概述

Phase 6 从**灾难恢复能力**、**调度智能化**和**多租户隔离**三个维度完成系统的企业级能力闭环：

| 模块 | 解决的问题 | 核心价值 |
|------|-----------|---------|
| **一键灾难恢复** | 恢复操作复杂、耗时长、易出错 | 向导式恢复，从小时级降到分钟级 |
| **智能调度引擎** | 固定 cron 无法适应业务变化 | 自适应调度，确保 RPO 达标且不影响业务 |
| **多租户支持** | 多团队/多项目共用系统无法隔离 | 租户级资源隔离和配额管理 |

---

## 2. 一键灾难恢复

### 2.1 恢复场景分析

| 场景 | 恢复类型 | 数据丢失范围 | 恢复时效要求 | 恢复复杂度 |
|------|---------|-------------|-------------|-----------|
| **硬件故障** | 全量恢复 | 上次备份后的增量 | 高（< 1h） | 低 |
| **误删数据** | PITR（时间点恢复） | 精确到秒 | 高（< 30min） | 中 |
| **误删表/库** | 按表恢复 | 整表数据 | 高（< 30min） | 中 |
| **机房迁移** | 全量 + 增量 | 无 | 中（< 4h） | 高 |
| **合规审计** | 指定时间点快照 | 无 | 低（< 24h） | 低 |
| **勒索病毒** | 异地存储恢复 | 上次异地备份后 | 高（< 2h） | 中 |

### 2.2 一键恢复流程设计

```
┌─────────────────────────────────────────────────────────────────┐
│                      一键灾难恢复向导                             │
│                                                                 │
│  ┌─────┐   ┌─────┐   ┌─────┐   ┌─────┐   ┌─────┐   ┌─────┐  │
│  │ 1.  │──▶│ 2.  │──▶│ 3.  │──▶│ 4.  │──▶│ 5.  │──▶│ 6.  │  │
│  │选择 │   │环境 │   │预检 │   │执行 │   │验证 │   │报告 │  │
│  │恢复 │   │检测 │   │     │   │恢复 │   │     │   │     │  │
│  │源   │   │     │   │     │   │     │   │     │   │     │  │
│  └─────┘   └─────┘   └─────┘   └─────┘   └─────┘   └─────┘  │
│     ▲                                                    │     │
│     └────────────────────────────────────────────────────┘     │
│                      (失败可回退)                               │
└─────────────────────────────────────────────────────────────────┘
```

**Step 1: 选择恢复源**

```
┌──────────────────────────────────────────────────────┐
│  Step 1/6: 选择恢复源                                │
│                                                      │
│  恢复方式:                                            │
│  ┌─────────────────┐  ┌─────────────────┐           │
│  │  📦 按备份记录   │  │  ⏰ 按时间点     │           │
│  │     推荐        │  │     PITR        │           │
│  └─────────────────┘  └─────────────────┘           │
│                                                      │
│  选择备份记录:                                        │
│  ┌──────────────────────────────────────────────┐   │
│  │ ●  app_production_20260327.sql.gz            │   │
│  │    2026-03-27 02:00  |  1.0 GB  |  ✅ 已验证  │   │
│  ├──────────────────────────────────────────────┤   │
│  │ ○  app_production_20260326.sql.gz            │   │
│  │    2026-03-26 02:00  |  0.98 GB  |  ✅ 已验证  │   │
│  └──────────────────────────────────────────────┘   │
│                                                      │
│  存储来源: [本地存储 ▼] (自动选择可用的存储)          │
│                                                      │
│  恢复目标: [新连接配置] 或 [选择已有连接 ▼]           │
│                                                      │
│  [取消]                              [下一步 →]      │
└──────────────────────────────────────────────────────┘
```

**Step 2: 自动环境检测**

```
┌──────────────────────────────────────────────────────┐
│  Step 2/6: 环境检测                                   │
│                                                      │
│  目标数据库: mysql://10.0.0.5:3306/app_production     │
│                                                      │
│  ┌─ 检测结果 ─────────────────────────────────────┐  │
│  │ ✅ 连接测试         延迟: 2ms                   │  │
│  │ ✅ 磁盘空间         可用: 50 GB (需要: 3.2 GB)  │  │
│  │ ✅ 数据库版本       8.0.35 (源: 8.0.35)        │  │
│  │ ✅ 字符集           utf8mb4 (匹配)              │  │
│  │ ✅ 权限检查         CREATE, INSERT, ALTER ✅    │  │
│  │ ⚠️ 目标库非空      存在 12 张表                 │  │
│  │    ○ 覆盖 (DROP + 重建)                        │  │
│  │    ○ 仅恢复缺失的表                             │  │
│  │    ○ 重命名现有表为 _old 后缀                    │  │
│  └────────────────────────────────────────────────┘  │
│                                                      │
│  [← 上一步]                         [下一步 →]      │
└──────────────────────────────────────────────────────┘
```

**Step 3: 恢复预检**

```
┌──────────────────────────────────────────────────────┐
│  Step 3/6: 恢复预检                                   │
│                                                      │
│  ┌─ 预检报告 ─────────────────────────────────────┐  │
│  │                                                 │  │
│  │  📊 数据量预估                                  │  │
│  │  • 备份文件: 1.0 GB (gzip 压缩)                 │  │
│  │  • 预计解压后: 3.2 GB                           │  │
│  │  • 表数量: 48 张                                │  │
│  │  • 预计行数: ~2,560,000 行                      │  │
│  │                                                 │  │
│  │  ⏱️ 时间预估                                    │  │
│  │  • 下载备份: ~15s (本地存储)                     │  │
│  │  • 数据导入: ~3m 20s                            │  │
│  │  • 索引重建: ~1m 10s                            │  │
│  │  • 总计: ~4m 45s                                │  │
│  │                                                 │  │
│  │  🔍 一致性检查                                  │  │
│  │  • 校验和: ✅ 匹配                              │  │
│  │  • 文件完整性: ✅ 完整                           │  │
│  │  • 备份工具版本: ✅ 兼容                         │  │
│  │                                                 │  │
│  │  ⚠️ 风险提示                                    │  │
│  │  • 目标库存在数据，建议先备份目标库               │  │
│  │  • 恢复过程中目标库不可写入                       │  │
│  └────────────────────────────────────────────────┘  │
│                                                      │
│  ☑ 恢复前自动备份目标库                               │
│  ☑ 恢复完成后发送通知                                 │
│                                                      │
│  [← 上一步]                         [开始恢复 →]    │
└──────────────────────────────────────────────────────┘
```

**Step 4: 执行恢复**（WebSocket 实时推送进度）

```
┌──────────────────────────────────────────────────────┐
│  Step 4/6: 执行恢复                                   │
│                                                      │
│  ████████████████████░░░░░░░░  68%                   │
│                                                      │
│  当前步骤: 正在导入数据...                             │
│  已处理: 1,740,800 / 2,560,000 行                    │
│  已用时间: 3m 12s  |  预计剩余: 1m 33s               │
│                                                      │
│  ┌─ 步骤进度 ─────────────────────────────────────┐  │
│  │ ✅ 下载备份文件                    12s          │  │
│  │ ✅ 解压文件                         5s          │  │
│  │ ✅ 创建表结构                       8s          │  │
│  │ 🔄 导入数据                  3m 12s (进行中)    │  │
│  │ ⏳ 重建索引                        -           │  │
│  │ ⏳ 数据验证                        -           │  │
│  └────────────────────────────────────────────────┘  │
│                                                      │
│  [暂停]  [取消恢复]                                   │
└──────────────────────────────────────────────────────┘
```

**Step 5: 恢复后验证**

自动执行以下验证：

| 验证项 | 说明 | 判定标准 |
|--------|------|---------|
| 表数量对比 | 对比源备份与目标库的表数量 | 数量一致 |
| 行数抽样 | 抽取 10% 的表对比行数 | 偏差 < 1% |
| 关键表校验 | 对比关键表的 checksum | 完全一致 |
| 总行数对比 | 全部表的总行数 | 偏差 < 0.1% |
| 存储引擎/字符集 | 验证表属性一致 | 完全一致 |

**Step 6: 恢复报告**

```json
{
  "restore_id": "restore-uuid-001",
  "status": "success",
  "source": {
    "backup_record_id": 1024,
    "file_name": "app_production_20260327.sql.gz",
    "storage_name": "本地存储",
    "backup_time": "2026-03-27T02:00:00+08:00"
  },
  "target": {
    "host": "10.0.0.5",
    "port": 3306,
    "database": "app_production"
  },
  "duration_ms": 285000,
  "duration_display": "4m 45s",
  "steps": [
    { "name": "下载备份", "status": "success", "duration_ms": 12000 },
    { "name": "解压文件", "status": "success", "duration_ms": 5000 },
    { "name": "创建表结构", "status": "success", "duration_ms": 8000 },
    { "name": "导入数据", "status": "success", "duration_ms": 192000 },
    { "name": "重建索引", "status": "success", "duration_ms": 58000 },
    { "name": "数据验证", "status": "success", "duration_ms": 10000 }
  ],
  "verification": {
    "table_count": { "expected": 48, "actual": 48, "status": "pass" },
    "total_rows": { "expected": 2560000, "actual": 2559876, "status": "pass", "deviation": "0.005%" },
    "key_tables_checksum": { "checked": 5, "passed": 5, "status": "pass" }
  },
  "restored_at": "2026-03-27T15:30:00+08:00",
  "operator": "admin"
}
```

### 2.3 时间点恢复（PITR）设计

**支持数据库的 PITR 机制**：

| 数据库 | 日志类型 | 恢复命令 | 精度 |
|--------|---------|---------|------|
| MySQL | binlog | `mysqlbinlog --start-datetime --stop-datetime \| mysql` | 秒级 |
| PostgreSQL | WAL | `pg_restore` + `recovery_target_time` | 秒级 |
| MongoDB | oplog | `mongorestore` + oplog replay | 秒级 |

**PITR 流程**：

```
1. 选择最近的全量备份 (如 2026-03-27 02:00)
2. 指定目标时间点 (如 2026-03-27 14:30:00)
3. 系统自动:
   a. 恢复全量备份
   b. 应用 binlog/WAL/oplog 到目标时间点
   c. 停止在指定时间点
4. 验证数据一致性
```

**前置条件**：

- MySQL: 需开启 `log_bin`，且 binlog 保留时间覆盖 PITR 范围
- PostgreSQL: 需开启 WAL 归档 (`archive_mode=on`)
- MongoDB: 需为副本集，oplog 保留时间覆盖 PITR 范围

### 2.4 恢复沙箱

**功能描述**：在隔离环境中恢复备份，不影响生产数据库，用于验证备份可用性。

**沙箱架构**：

```
┌─────────────────────────────────────────────────────┐
│                  恢复沙箱                            │
│                                                     │
│  ┌──────────────┐     ┌──────────────────────┐     │
│  │ 备份文件      │────▶│ 临时数据库实例        │     │
│  │ (选定备份)    │     │ (Docker 容器)        │     │
│  └──────────────┘     │                      │     │
│                       │  • 自动创建          │     │
│  ┌──────────────┐     │  • 端口映射随机      │     │
│  │ 数据库镜像    │     │  • 限时自动销毁      │     │
│  │ (同版本)     │     │  • 资源限制          │     │
│  └──────────────┘     └──────────┬───────────┘     │
│                                  │                  │
│                                  ▼                  │
│                       ┌──────────────────┐         │
│                       │  临时连接信息      │         │
│                       │  host: 10.0.0.x   │         │
│                       │  port: 32768      │         │
│                       │  用户: sandbox    │         │
│                       │  密码: auto-gen   │         │
│                       └──────────────────┘         │
│                                                     │
│  ⏰ 沙箱将在 2 小时后自动销毁  [延期 1h] [立即销毁]  │
└─────────────────────────────────────────────────────┘
```

### 2.5 数据库设计

```sql
-- 恢复记录表
CREATE TABLE restore_records (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    restore_id      VARCHAR(36)    NOT NULL UNIQUE COMMENT '恢复唯一 ID',
    
    -- 恢复源
    backup_record_id BIGINT UNSIGNED NOT NULL COMMENT '关联 backup_records.id',
    storage_id      BIGINT UNSIGNED DEFAULT NULL COMMENT '关联 backup_storages.id',
    file_path       VARCHAR(512)   NOT NULL COMMENT '备份文件路径',
    
    -- 恢复方式
    restore_type    VARCHAR(20)    NOT NULL DEFAULT 'full' COMMENT 'full/pitr/sandbox',
    pitr_target_time DATETIME      DEFAULT NULL COMMENT 'PITR 目标时间',
    
    -- 恢复目标
    target_host     VARCHAR(100)   NOT NULL,
    target_port     INT            NOT NULL,
    target_database VARCHAR(100)   NOT NULL,
    target_username VARCHAR(50)    NOT NULL,
    
    -- 恢复选项
    overwrite       BOOLEAN        NOT NULL DEFAULT FALSE,
    backup_target   BOOLEAN        NOT NULL DEFAULT TRUE COMMENT '恢复前备份目标库',
    
    -- 执行状态
    status          VARCHAR(20)    NOT NULL DEFAULT 'pending' COMMENT 'pending/precheck/running/verifying/success/failed/cancelled',
    current_step    VARCHAR(50)    DEFAULT NULL COMMENT '当前执行步骤',
    progress        INT            NOT NULL DEFAULT 0 COMMENT '进度百分比 0-100',
    error_msg       TEXT           DEFAULT NULL,
    
    -- 预检结果 (JSON)
    precheck_result JSON           DEFAULT NULL COMMENT '环境检测结果',
    
    -- 验证结果 (JSON)
    verify_result   JSON           DEFAULT NULL COMMENT '恢复后验证结果',
    
    -- 恢复报告 (JSON)
    report          JSON           DEFAULT NULL COMMENT '完整恢复报告',
    
    -- 沙箱信息
    sandbox_info    JSON           DEFAULT NULL COMMENT '沙箱容器信息',
    sandbox_expires_at DATETIME    DEFAULT NULL COMMENT '沙箱过期时间',
    
    -- 耗时统计
    started_at      DATETIME       DEFAULT NULL,
    completed_at    DATETIME       DEFAULT NULL,
    duration_ms     BIGINT         DEFAULT NULL,
    
    -- 操作人
    operator_id     VARCHAR(36)    DEFAULT NULL,
    operator_name   VARCHAR(50)    DEFAULT NULL,
    
    created_at      DATETIME       DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME       DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_backup_record (backup_record_id),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='恢复记录';

-- 恢复步骤记录表
CREATE TABLE restore_steps (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    restore_id      VARCHAR(36)    NOT NULL,
    step_name       VARCHAR(50)    NOT NULL COMMENT '步骤名称',
    step_order      INT            NOT NULL,
    status          VARCHAR(20)    NOT NULL DEFAULT 'pending',
    started_at      DATETIME       DEFAULT NULL,
    completed_at    DATETIME       DEFAULT NULL,
    duration_ms     BIGINT         DEFAULT NULL,
    output          TEXT           DEFAULT NULL COMMENT '步骤输出',
    error_msg       TEXT           DEFAULT NULL,
    
    INDEX idx_restore_id (restore_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='恢复步骤记录';
```

### 2.6 API 设计

**创建恢复任务**：

```
POST /api/v1/restores
```

```json
{
  "backup_record_id": 1024,
  "storage_id": 1,
  "restore_type": "full",
  "target": {
    "host": "10.0.0.5",
    "port": 3306,
    "database": "app_production",
    "username": "root",
    "password": "***"
  },
  "options": {
    "overwrite": false,
    "backup_target": true,
    "notify_on_complete": true
  }
}
```

**PITR 恢复**：

```json
{
  "backup_record_id": 1024,
  "restore_type": "pitr",
  "pitr_target_time": "2026-03-27T14:30:00+08:00",
  "target": { "..." : "..." }
}
```

**沙箱恢复**：

```json
{
  "backup_record_id": 1024,
  "restore_type": "sandbox",
  "sandbox_options": {
    "ttl_hours": 2,
    "resource_limits": {
      "cpu": "1",
      "memory": "512m"
    }
  }
}
```

**其他恢复 API**：

```
GET    /api/v1/restores                      # 恢复记录列表
GET    /api/v1/restores/:id                   # 恢复记录详情
POST   /api/v1/restores/:id/cancel            # 取消恢复
POST   /api/v1/restores/:id/retry             # 重试恢复
DELETE /api/v1/restores/:id                   # 删除记录
POST   /api/v1/restores/:id/sandbox/extend    # 延长沙箱
DELETE /api/v1/restores/:id/sandbox           # 销毁沙箱
GET    /api/v1/restores/:id/logs              # 恢复日志 (SSE/WebSocket)
```

**环境预检**（创建恢复前先预检）：

```
POST /api/v1/restores/precheck
```

```json
{
  "backup_record_id": 1024,
  "target": {
    "host": "10.0.0.5",
    "port": 3306,
    "database": "app_production",
    "username": "root",
    "password": "***"
  }
}
```

响应：

```json
{
  "can_restore": true,
  "warnings": ["目标库非空，存在 12 张表"],
  "checks": {
    "connectivity": { "status": "ok", "latency_ms": 2 },
    "disk_space": { "status": "ok", "available_gb": 50, "required_gb": 3.2 },
    "version_compat": { "status": "ok", "source": "8.0.35", "target": "8.0.35" },
    "charset": { "status": "ok", "source": "utf8mb4", "target": "utf8mb4" },
    "permissions": { "status": "ok", "granted": ["CREATE", "INSERT", "ALTER", "DROP", "INDEX"] }
  },
  "estimates": {
    "file_size_gb": 3.2,
    "table_count": 48,
    "row_count": 2560000,
    "duration_seconds": 285
  }
}
```

---

## 3. 智能调度引擎

### 3.1 设计目标

| 目标 | 说明 |
|------|------|
| 避开业务高峰 | 自动识别业务高峰时段，避免备份影响业务 |
| 均衡负载 | 限制同时执行的备份数，错开大数据库的备份时间 |
| 确保 RPO 达标 | 如果备份时间需要调整，确保不超过 RPO 要求 |
| 自适应调整 | 数据库大小变化时自动调整调度时间 |

### 3.2 数据采集

**采集指标**：

| 指标 | 采集方式 | 用途 |
|------|---------|------|
| 历史备份耗时 | backup_records | 耗时预估 |
| 数据库大小变化 | 备份文件大小趋势 | 增长预测 |
| 系统负载 | （可选）系统指标 API | 避开高负载时段 |
| 备份失败时段 | backup_records 错误分析 | 避开问题时段 |

### 3.3 调度策略

**基于历史的耗时预估（线性回归）**：

```go
type DurationPredictor struct {
    // 基于历史备份文件大小和耗时的线性回归
    // predicted_duration = a * file_size + b
    a float64 // 斜率
    b float64 // 截距
}

// Predict 预测备份耗时
func (p *DurationPredictor) Predict(fileSize int64) time.Duration {
    predicted := p.a*float64(fileSize) + p.b
    // 加上 20% 安全余量
    return time.Duration(predicted * 1.2) * time.Second
}
```

**业务高峰识别**：

```
基于历史备份耗时异常检测：
1. 计算每个时间段的平均备份耗时
2. 如果某时段耗时显著高于其他时段 (> 1.5x)，标记为高峰时段
3. 高峰时段的备份自动延迟到低峰时段执行

示例：
  00:00-06:00  平均耗时: 2m  ← 低峰，优先调度
  06:00-12:00  平均耗时: 5m  ← 高峰，避免调度
  12:00-18:00  平均耗时: 4m  ← 高峰，避免调度
  18:00-24:00  平均耗时: 3m  ← 中峰，可调度
```

**资源均衡**：

```
调度规则：
1. 同时执行备份数上限: max_concurrent (可配置，默认 3)
2. 大数据库（> 1GB）错开调度，间隔至少 30 分钟
3. 同一服务器的备份任务串行执行
4. 总带宽限制: max_total_bandwidth (可配置)
```

**自适应调度**：

```
触发条件：
- 连续 3 次备份耗时超过预估 50% → 增加调度间隔
- 连续 3 次备份耗时低于预估 50% → 缩短调度间隔
- 数据库大小增长超过 20% → 重新计算预估

调整策略：
- 如果 cron 间隔为 4h，当前耗时 2h → 不调整
- 如果 cron 间隔为 1h，当前耗时 50min → 建议调整为 2h
```

### 3.4 配置模型设计

```sql
-- 智能调度配置表
CREATE TABLE smart_schedule_configs (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    task_id         VARCHAR(36)    NOT NULL COMMENT '关联任务，NULL 表示全局配置',
    
    -- 调度策略
    strategy        VARCHAR(20)    NOT NULL DEFAULT 'adaptive' COMMENT 'fixed/adaptive/smart',
    
    -- 资源限制
    max_concurrent  INT            NOT NULL DEFAULT 3 COMMENT '最大并发备份数',
    bandwidth_limit INT            NOT NULL DEFAULT 0 COMMENT '总带宽限制(MB/s), 0=不限',
    
    -- 大数据库阈值
    large_db_threshold BIGINT      NOT NULL DEFAULT 1073741824 COMMENT '大数据库阈值(字节), 默认 1GB',
    large_db_min_interval INT      NOT NULL DEFAULT 30 COMMENT '大数据库最小间隔(分钟)',
    
    -- RPO 约束
    max_rpo_minutes INT            NOT NULL DEFAULT 0 COMMENT '最大 RPO(分钟), 0=不限制',
    
    -- 自适应参数
    adapt_enabled  BOOLEAN        NOT NULL DEFAULT TRUE COMMENT '启用自适应',
    adapt_history_days INT        NOT NULL DEFAULT 7 COMMENT '分析历史天数',
    
    enabled         BOOLEAN        NOT NULL DEFAULT TRUE,
    created_at      DATETIME       DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME       DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    UNIQUE KEY uk_task (task_id),
    INDEX idx_enabled (enabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='智能调度配置';
```

### 3.5 API 设计

**获取调度配置**：

```
GET /api/v1/smart-schedule?task_id=xxx
```

响应：

```json
{
  "task_id": "task-uuid-001",
  "strategy": "adaptive",
  "max_concurrent": 3,
  "bandwidth_limit": 0,
  "large_db_threshold": 1073741824,
  "large_db_min_interval": 30,
  "max_rpo_minutes": 240,
  "adapt_enabled": true,
  "adapt_history_days": 7
}
```

**更新调度配置**：

```
PUT /api/v1/smart-schedule?task_id=xxx
```

**调度预览**（查看智能调度的建议时间）：

```
GET /api/v1/smart-schedule/preview?task_id=xxx&days=7
```

响应：

```json
{
  "task_id": "task-uuid-001",
  "current_cron": "0 */4 * * *",
  "suggested_cron": "0 2,8,14,20 * * *",
  "reason": "基于近 7 天数据分析，避开 06:00-10:00 业务高峰",
  "analysis": {
    "avg_duration": "2m 35s",
    "predicted_growth": "+5%/week",
    "peak_hours": [8, 9, 10],
    "recommended_slots": ["02:00", "08:00", "14:00", "20:00"]
  },
  "schedule_preview": [
    { "time": "2026-03-28 02:00", "estimated_duration": "2m 40s", "estimated_size": "1.05 GB" },
    { "time": "2026-03-28 08:00", "estimated_duration": "2m 42s", "estimated_size": "1.05 GB" },
    { "time": "2026-03-28 14:00", "estimated_duration": "2m 44s", "estimated_size": "1.06 GB" }
  ]
}
```

**手动覆盖**（临时跳过智能调度）：

```
POST /api/v1/smart-schedule/override
```

```json
{
  "task_id": "task-uuid-001",
  "override_type": "skip_next",
  "reason": "紧急维护，跳过下一次备份"
}
```

**调度历史**：

```
GET /api/v1/smart-schedule/history?task_id=xxx&days=7
```

### 3.6 前端交互

**调度时间线可视化**：

```
┌──────────────────────────────────────────────────────────────┐
│  智能调度 - production-mysql-daily                            │
│                                                              │
│  策略: 自适应调度  |  RPO: 4h  |  并发: 3                    │
│                                                              │
│  ┌─ 今日调度时间线 ─────────────────────────────────────┐    │
│  │ 00  02  04  06  08  10  12  14  16  18  20  22      │    │
│  │ ░░  ██  ░░  ░░  ░░  ░░  ██  ░░  ░░  ░░  ██  ░░     │    │
│  │     ↑         [高峰期]     ↑              ↑           │    │
│  │   02:00      06-10:00    12:00         20:00         │    │
│  │   ✅ 完成                  ⏳ 排队       ⏳ 排队       │    │
│  └────────────────────────────────────────────────────┘    │
│                                                              │
│  ── 调度分析 ───────────────────────────────                │
│  • 近 7 天平均耗时: 2m 35s (↓ 12s)                          │
│  • 数据库增长: +5%/week                                      │
│  • 高峰时段: 06:00-10:00 (耗时平均 1.8x)                    │
│  • RPO 达标率: 100%                                         │
│                                                              │
│  ── 调度建议 ───────────────────────────────                │
│  💡 当前调度合理，无需调整                                    │
│  ℹ️ 预计 2 周后数据库增长到 1.5 GB，建议提前调整 cron 间隔    │
│                                                              │
│  [应用建议]  [手动覆盖]  [编辑配置]                           │
└──────────────────────────────────────────────────────────────┘
```

---

## 4. 多租户支持

### 4.1 租户模型

```
┌──────────────────────────────────────────────────────────┐
│                    多租户架构                              │
│                                                          │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐               │
│  │ 租户 A   │  │ 租户 B   │  │ 租户 C   │               │
│  │ (默认)   │  │ (财务部) │  │ (测试)   │               │
│  │          │  │          │  │          │               │
│  │ 任务 x5  │  │ 任务 x3  │  │ 任务 x2  │               │
│  │ 50 GB    │  │ 30 GB    │  │ 10 GB    │               │
│  │ 成员 x3  │  │ 成员 x2  │  │ 成员 x1  │               │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘               │
│       │             │             │                      │
│       ▼             ▼             ▼                      │
│  ┌────────────────────────────────────────────────┐     │
│  │              数据隔离层                         │     │
│  │  • 所有查询自动添加 tenant_id 过滤              │     │
│  │  • 资源配额检查                                │     │
│  │  • 操作审计按租户记录                           │     │
│  └────────────────────────────────────────────────┘     │
└──────────────────────────────────────────────────────────┘
```

### 4.2 资源隔离

| 资源 | 隔离方式 | 说明 |
|------|---------|------|
| 备份任务 | `tenant_id` 字段 | 每个任务归属一个租户 |
| 备份记录 | `tenant_id` 字段 | 通过任务关联自动隔离 |
| 存储空间 | 存储路径前缀 | 每个租户独立的存储路径 (`/backups/{tenant_id}/`) |
| 通知策略 | 租户独立配置 | 每个租户可配置独立的通知渠道 |

### 4.3 配额管理

| 配额项 | 说明 | 默认值 |
|--------|------|--------|
| 任务数 | 最大备份任务数 | 100 |
| 存储空间 | 最大备份存储空间 | 500 GB |
| 备份保留天数 | 最大保留天数 | 90 天 |
| 并发备份数 | 最大同时执行数 | 5 |
| 成员数 | 最大租户成员数 | 20 |

### 4.4 数据库设计

```sql
-- 租户表
CREATE TABLE tenants (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    tenant_id       VARCHAR(36)    NOT NULL UNIQUE COMMENT '租户唯一标识',
    name            VARCHAR(100)   NOT NULL COMMENT '租户名称',
    slug            VARCHAR(50)    NOT NULL UNIQUE COMMENT '租户标识 (用于 URL)',
    description     VARCHAR(500)   DEFAULT NULL,
    
    -- 联系信息
    contact_name    VARCHAR(50)    DEFAULT NULL,
    contact_email   VARCHAR(100)   DEFAULT NULL,
    
    -- 状态
    status          VARCHAR(20)    NOT NULL DEFAULT 'active' COMMENT 'active/suspended/disabled',
    
    -- 配额 (JSON)
    quotas          JSON           NOT NULL COMMENT '配额配置',
    
    -- 通知配置 (JSON)
    notify_config   JSON           DEFAULT NULL COMMENT '租户级通知配置',
    
    -- 存储
    storage_prefix  VARCHAR(100)   NOT NULL DEFAULT '' COMMENT '存储路径前缀',
    
    created_at      DATETIME       DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME       DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='租户表';

-- 租户配额表
CREATE TABLE tenant_quotas (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    tenant_id       VARCHAR(36)    NOT NULL,
    
    -- 配额项
    quota_type      VARCHAR(30)    NOT NULL COMMENT 'tasks/storage_bytes/retention_days/concurrent/members',
    quota_limit     BIGINT         NOT NULL DEFAULT 0 COMMENT '配额上限, 0=无限制',
    quota_used      BIGINT         NOT NULL DEFAULT 0 COMMENT '当前使用量',
    
    -- 警告阈值
    warn_threshold  DECIMAL(5,2)   NOT NULL DEFAULT 80.00 COMMENT '警告阈值(%)',
    
    updated_at      DATETIME       DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    UNIQUE KEY uk_tenant_type (tenant_id, quota_type),
    INDEX idx_tenant (tenant_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='租户配额';

-- 租户成员表
CREATE TABLE tenant_members (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    tenant_id       VARCHAR(36)    NOT NULL,
    user_id         VARCHAR(36)    NOT NULL COMMENT '用户 ID',
    role            VARCHAR(20)    NOT NULL DEFAULT 'member' COMMENT 'owner/admin/member/viewer',
    
    joined_at       DATETIME       DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE KEY uk_tenant_user (tenant_id, user_id),
    INDEX idx_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='租户成员';
```

**已有表变更**（添加 `tenant_id` 字段）：

```sql
ALTER TABLE backup_tasks
    ADD COLUMN tenant_id VARCHAR(36) DEFAULT NULL AFTER id,
    ADD INDEX idx_tenant_id (tenant_id);

ALTER TABLE backup_records
    ADD COLUMN tenant_id VARCHAR(36) DEFAULT NULL AFTER id,
    ADD INDEX idx_tenant_id (tenant_id);

ALTER TABLE audit_logs
    ADD COLUMN tenant_id VARCHAR(36) DEFAULT NULL AFTER id,
    ADD INDEX idx_tenant_id (tenant_id);
```

### 4.5 API 设计

**租户管理（超级管理员）**：

```
GET    /api/v1/tenants                     # 租户列表
POST   /api/v1/tenants                     # 创建租户
GET    /api/v1/tenants/:id                 # 租户详情
PUT    /api/v1/tenants/:id                 # 更新租户
DELETE /api/v1/tenants/:id                 # 删除租户
PUT    /api/v1/tenants/:id/status          # 更新租户状态
```

**创建租户**：

```json
{
  "name": "财务部",
  "slug": "finance",
  "description": "财务系统数据库备份",
  "contact_name": "张三",
  "contact_email": "zhangsan@example.com",
  "quotas": {
    "tasks": { "limit": 10, "warn_threshold": 80 },
    "storage_bytes": { "limit": 53687091200, "warn_threshold": 80 },
    "retention_days": { "limit": 60 },
    "concurrent": { "limit": 3 },
    "members": { "limit": 10 }
  }
}
```

**配额管理**：

```
GET    /api/v1/tenants/:id/quotas         # 获取配额
PUT    /api/v1/tenants/:id/quotas         # 更新配额
GET    /api/v1/tenants/:id/quotas/usage   # 配额使用情况
```

**配额使用情况响应**：

```json
{
  "tenant_id": "tenant-001",
  "quotas": [
    {
      "quota_type": "tasks",
      "limit": 10,
      "used": 3,
      "usage_percent": 30.0,
      "warn_threshold": 80.0,
      "status": "ok"
    },
    {
      "quota_type": "storage_bytes",
      "limit": 53687091200,
      "used": 32212254720,
      "usage_percent": 60.0,
      "warn_threshold": 80.0,
      "status": "ok",
      "used_display": "30.0 GB",
      "limit_display": "50.0 GB"
    },
    {
      "quota_type": "retention_days",
      "limit": 60,
      "used": 30,
      "usage_percent": 50.0,
      "status": "ok"
    }
  ]
}
```

**租户成员管理**：

```
GET    /api/v1/tenants/:id/members        # 成员列表
POST   /api/v1/tenants/:id/members        # 添加成员
PUT    /api/v1/tenants/:id/members/:uid   # 更新成员角色
DELETE /api/v1/tenants/:id/members/:uid   # 移除成员
```

**租户切换**（普通用户在有权限的租户间切换）：

```
GET    /api/v1/user/tenants               # 用户所属租户列表
POST   /api/v1/user/switch-tenant         # 切换当前租户
```

```json
// POST /api/v1/user/switch-tenant
{ "tenant_id": "tenant-001" }

// 响应
{ "message": "已切换到租户: 财务部", "tenant_id": "tenant-001" }
```

### 4.6 前端交互

**租户管理页面（超级管理员）**：

```
┌─────────────────────────────────────────────────────────────────────┐
│  🗄️ DB Backup    [仪表盘] [任务] [记录] [恢复] [设置] [🏢 租户]    │
│                                                                     │
│  租户管理                                    [+ 新建租户]           │
│                                                                     │
│  ┌────────────────────────────────────────────────────────────┐    │
│  │  租户名称     │ 标识     │ 任务 │ 存储    │ 状态    │ 操作  │    │
│  ├────────────────────────────────────────────────────────────┤    │
│  │  🏢 默认租户   │ default  │ 5/100│ 30/500G │ 🟢 正常 │ [管理]│    │
│  │  💰 财务部     │ finance  │ 3/10 │ 30/50G  │ 🟢 正常 │ [管理]│    │
│  │  🧪 测试环境   │ testing  │ 2/5  │ 8/10G   │ 🟡 警告 │ [管理]│    │
│  │  🔒 已停用     │ old-dept │ 0/10 │ 0/20G   │ ⚫ 停用 │ [管理]│    │
│  └────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
```

**租户详情页面**：

```
┌──────────────────────────────────────────────────────────────┐
│  租户: 财务部                              [编辑] [停用]      │
│                                                              │
│  ── 配额概览 ───────────────────────────────                │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐       │
│  │ 任务数   │ │ 存储空间 │ │ 保留天数 │ │ 成员数   │       │
│  │ 3 / 10   │ │ 30 / 50G │ │ 30 / 60d │ │ 2 / 10   │       │
│  │  30% 🟢  │ │  60% 🟡  │ │  50% 🟢  │ │  20% 🟢  │       │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘       │
│                                                              │
│  ── 成员列表 ───────────────────────────────                │
│  👤 张三    admin    zhangsan@example.com    [编辑] [移除]   │
│  👤 李四    member   lisi@example.com        [编辑] [移除]   │
│                                    [+ 添加成员]              │
│                                                              │
│  ── 存储增长趋势 ───────────────────────────────            │
│  50G ┤                                  ╱──  30G (60%)     │
│  40G ┤                            ╱──╯                     │
│  30G ┤                      ╱──╯                          │
│  20G ┤                ╱──╯                                 │
│  10G ┤          ╱──╯                                       │
│   0G ┤──╯──────────────────                                │
│       M1      M2      M3      M4      M5      M6          │
└──────────────────────────────────────────────────────────────┘
```

**租户切换器（顶部导航栏）**：

```
┌──────────────────────────────────────────────────────────────┐
│  🗄️ DB Backup    [🏢 默认租户 ▼]  [仪表盘] [任务] ...      │
│                     ┌─────────────┐                         │
│                     │ ✅ 默认租户   │  ← 当前               │
│                     │    财务部     │                        │
│                     │    测试环境   │                        │
│                     ├─────────────┤                         │
│                     │ ⚙️ 租户管理  │  ← 超级管理员可见       │
│                     └─────────────┘                         │
└──────────────────────────────────────────────────────────────┘
```

### 4.7 向下兼容

**单租户模式**（默认）：

- 系统启动时自动创建 `default` 租户
- 所有已有任务的 `tenant_id` 默认为 `default`
- `tenant_id` 为空时等价于 `default`
- 查询自动填充当前用户的租户上下文
- 单租户模式下，租户切换器隐藏，无需任何额外配置

**迁移策略**：

```sql
-- 数据迁移：为已有数据设置默认租户
UPDATE backup_tasks SET tenant_id = 'default' WHERE tenant_id IS NULL;
UPDATE backup_records SET tenant_id = 'default' WHERE tenant_id IS NULL;

-- 创建默认租户
INSERT INTO tenants (tenant_id, name, slug, status, quotas) VALUES
('default', '默认租户', 'default', 'active', '{"tasks":{"limit":0},"storage_bytes":{"limit":0},"retention_days":{"limit":0},"concurrent":{"limit":0},"members":{"limit":0}}');
```

---

## 5. 实施计划

| 阶段 | 内容 | 预估工时 |
|------|------|---------|
| 6.1 | 恢复引擎核心（全量恢复 + 预检 + 验证） | 5 天 |
| 6.2 | PITR 恢复（MySQL binlog / PostgreSQL WAL） | 3 天 |
| 6.3 | 恢复沙箱（Docker 集成） | 3 天 |
| 6.4 | 智能调度引擎（数据采集 + 策略执行） | 4 天 |
| 6.5 | 多租户模型 + 数据隔离 | 3 天 |
| 6.6 | 配额管理 + 租户管理 API | 3 天 |
| 6.7 | 前端页面开发（恢复向导 + 调度 + 租户） | 8 天 |
| 6.8 | 联调测试 + 迁移脚本 | 3 天 |
| **合计** | | **32 天** |

---

## 6. 风险与依赖

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| PITR 依赖 binlog/WAL 配置 | 用户数据库未开启相关配置 | 预检时检测并明确提示 |
| 恢复沙箱依赖 Docker | 环境中需安装 Docker | 沙箱功能可选，不影响核心恢复 |
| 多租户数据迁移 | 已有数据需要补充 tenant_id | 提供自动迁移脚本，向下兼容 |
| 智能调度准确性 | 预估偏差可能导致调度不合理 | 保守预估 + 安全余量 + 手动覆盖 |
| 配额计算实时性 | 存储空间计算可能有延迟 | 定期刷新 + 异步计算 |
