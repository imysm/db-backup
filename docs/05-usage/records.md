# 备份记录查询和管理

本文档介绍如何查看和管理备份记录。

## 概述

每次备份执行都会生成一条备份记录，包含：

| 字段 | 说明 |
|------|------|
| ID | 记录唯一标识 |
| 任务 ID | 关联的备份任务 |
| 状态 | success / failed / running |
| 文件路径 | 备份文件存储位置 |
| 文件大小 | 备份文件大小 |
| 执行时长 | 备份耗时（秒） |
| 开始时间 | 备份开始时间 |
| 结束时间 | 备份结束时间 |

## 查看备份记录

### CLI

```bash
# 查看所有备份记录（需要实现 CLI 查询功能）
# 当前版本通过查看备份目录了解记录
ls -lh /data/backups/mysql/
```

### API

```bash
# 获取备份记录列表
curl http://localhost:8080/api/records

# 按任务过滤
curl "http://localhost:8080/api/records?job_id=mysql-prod-backup"

# 按状态过滤
curl "http://localhost:8080/api/records?status=success"

# 按时间范围过滤
curl "http://localhost:8080/api/records?start_time=2026-03-01T00:00:00Z&end_time=2026-03-20T23:59:59Z"

# 分页
curl "http://localhost:8080/api/records?page=1&size=20"
```

### 响应示例

```json
{
  "code": 0,
  "data": {
    "total": 100,
    "page": 1,
    "size": 20,
    "items": [
      {
        "id": "rec_20260320020000",
        "job_id": "mysql-prod-backup",
        "status": "success",
        "file_path": "/data/backups/mysql/mydb_20260320_020000.sql.gz",
        "file_size": 1073741824,
        "duration": 120,
        "started_at": "2026-03-20T02:00:00Z",
        "finished_at": "2026-03-20T02:02:00Z"
      }
    ]
  }
}
```

## 查看记录详情

```bash
# 获取单个备份记录详情
curl http://localhost:8080/api/records/rec_20260320020000
```

### 响应示例

```json
{
  "code": 0,
  "data": {
    "id": "rec_20260320020000",
    "job_id": "mysql-prod-backup",
    "status": "success",
    "mode": "full",
    "file_path": "/data/backups/mysql/mydb_20260320_020000.sql.gz",
    "file_size": 1073741824,
    "checksum": "sha256:abc123...",
    "duration": 120,
    "started_at": "2026-03-20T02:00:00Z",
    "finished_at": "2026-03-20T02:02:00Z",
    "log": "[02:00:00] 开始备份...\n[02:00:01] 连接数据库...\n[02:02:00] 备份完成"
  }
}
```

## 备份状态

| 状态 | 说明 |
|------|------|
| `running` | 备份正在执行中 |
| `success` | 备份成功完成 |
| `failed` | 备份失败 |

## 实时日志

### WebSocket

连接 WebSocket 获取实时备份日志：

```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

// 订阅特定任务的日志
ws.send(JSON.stringify({
  action: 'subscribe',
  job_id: 'mysql-prod-backup'
}));

// 接收日志消息
ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  console.log(`[${msg.timestamp}] ${msg.message}`);
};
```

### 消息格式

```json
{
  "type": "log",
  "job_id": "mysql-prod-backup",
  "timestamp": "2026-03-20T02:00:00Z",
  "data": {
    "message": "[02:00:00] 开始备份...",
    "level": "info"
  }
}
```

## 下载备份文件

```bash
# 直接从存储路径复制
cp /data/backups/mysql/mydb_20260320_020000.sql.gz ./

# 或通过 API 下载（如果启用）
curl -O http://localhost:8080/api/records/rec_20260320020000/download
```

## 删除备份记录

```bash
# 删除备份记录和文件
curl -X DELETE http://localhost:8080/api/records/rec_20260320020000
```

> ⚠️ **注意**: 删除记录会同时删除备份文件，请谨慎操作。

## 备份统计

### 按任务统计

```bash
# 获取任务备份统计
curl http://localhost:8080/api/jobs/mysql-prod-backup/stats
```

### 响应示例

```json
{
  "code": 0,
  "data": {
    "total_backups": 100,
    "success_count": 95,
    "failed_count": 5,
    "total_size": 107374182400,
    "avg_duration": 120,
    "last_backup": "2026-03-20T02:00:00Z",
    "next_backup": "2026-03-21T02:00:00Z"
  }
}
```

## 最佳实践

### 1. 定期检查备份记录

```bash
# 每日检查失败备份
curl "http://localhost:8080/api/records?status=failed&start_time=$(date -d '1 day ago' -Iseconds)"
```

### 2. 监控存储空间

```bash
# 查看备份目录大小
du -sh /data/backups/*
```

### 3. 定期验证备份

```bash
# 验证备份文件完整性
curl -X POST http://localhost:8080/api/verify/rec_20260320020000
```

---

*最后更新: 2026-03-20*
