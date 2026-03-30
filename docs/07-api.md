# API 文档

本文档介绍 db-backup 系统的 REST API 接口。

## 基础信息

- **Base URL**: `http://localhost:8080/api`
- **Content-Type**: `application/json`
- **认证**: Bearer Token (可选)

## 健康检查

### GET /health

健康检查接口。

**响应**

```json
{
  "status": "ok",
  "version": "0.1.0",
  "uptime": 3600
}
```

---

## 任务管理

### GET /api/jobs

获取任务列表。

**响应**

```json
{
  "code": 0,
  "data": [
    {
      "id": "mysql-prod-backup",
      "enabled": true,
      "database": {
        "type": "mysql",
        "host": "localhost",
        "port": 3306,
        "database": "mydb"
      },
      "schedule": "0 2 * * *",
      "last_run": "2026-03-20T02:00:00Z",
      "next_run": "2026-03-21T02:00:00Z"
    }
  ]
}
```

### POST /api/jobs

创建新任务。

**请求体**

```json
{
  "id": "mysql-prod-backup",
  "database": {
    "type": "mysql",
    "host": "localhost",
    "port": 3306,
    "username": "root",
    "password": "password",
    "database": "mydb"
  },
  "schedule": "0 2 * * *",
  "mode": "full",
  "compression": {
    "enabled": true,
    "level": 6
  }
}
```

**响应**

```json
{
  "code": 0,
  "data": {
    "id": "mysql-prod-backup",
    "message": "任务创建成功"
  }
}
```

### GET /api/jobs/:id

获取任务详情。

**响应**

```json
{
  "code": 0,
  "data": {
    "id": "mysql-prod-backup",
    "enabled": true,
    "database": {
      "type": "mysql",
      "host": "localhost",
      "port": 3306,
      "database": "mydb"
    },
    "schedule": "0 2 * * *",
    "last_run": "2026-03-20T02:00:00Z",
    "next_run": "2026-03-21T02:00:00Z"
  }
}
```

### PUT /api/jobs/:id

更新任务。

### DELETE /api/jobs/:id

删除任务。

### POST /api/jobs/:id/run

立即执行任务。

**响应**

```json
{
  "code": 0,
  "data": {
    "message": "任务已触发执行"
  }
}
```

---

## 备份记录

### GET /api/records

获取备份记录列表。

**查询参数**

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `job_id` | string | - | 按任务 ID 过滤 |
| `status` | string | - | 按状态过滤: success/failed |
| `page` | int | 1 | 页码 |
| `size` | int | 20 | 每页数量 |

**响应**

```json
{
  "code": 0,
  "data": {
    "total": 100,
    "page": 1,
    "size": 20,
    "items": [
      {
        "id": "rec_123",
        "job_id": "mysql-prod-backup",
        "status": "success",
        "file_path": "/data/backups/mysql/mydb_20260320_020000.sql.gz",
        "file_size": 1073741824,
        "started_at": "2026-03-20T02:00:00Z",
        "finished_at": "2026-03-20T02:02:00Z"
      }
    ]
  }
}
```

### GET /api/records/:id

获取备份记录详情。

**响应**

```json
{
  "code": 0,
  "data": {
    "id": "rec_123",
    "job_id": "mysql-prod-backup",
    "status": "success",
    "file_path": "/data/backups/mysql/mydb_20260320_020000.sql.gz",
    "file_size": 1073741824,
    "checksum": "sha256:abc123...",
    "duration": 120,
    "started_at": "2026-03-20T02:00:00Z",
    "finished_at": "2026-03-20T02:02:00Z",
    "log": "[02:00:00] 开始备份...\n[02:02:00] 备份完成"
  }
}
```

---

## 恢复操作

### POST /api/restore

执行恢复操作。

**请求体**

```json
{
  "record_id": "rec_123",
  "target_host": "localhost",
  "target_port": 3306,
  "target_database": "mydb_restored"
}
```

**响应**

```json
{
  "code": 0,
  "data": {
    "restore_id": "restore_456",
    "status": "running",
    "message": "恢复任务已启动"
  }
}
```

---

## 验证操作

### POST /api/verify/:id

验证备份文件。

**响应**

```json
{
  "code": 0,
  "data": {
    "valid": true,
    "message": "备份文件验证通过"
  }
}
```

---

## WebSocket API

### 连接

```
ws://localhost:8080/ws
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

### 消息类型

| 类型 | 说明 |
|------|------|
| `log` | 日志消息 |
| `status` | 状态变更 |
| `error` | 错误消息 |
| `progress` | 进度更新 |

---

## Prometheus 指标

### 访问

```
GET /metrics
```

### 可用指标

| 指标 | 类型 | 说明 |
|------|------|------|
| `db_backup_tasks_total` | Gauge | 任务总数 |
| `db_backup_runs_total` | Counter | 备份执行次数 |
| `db_backup_duration_seconds` | Histogram | 备份耗时 |
| `db_backup_size_bytes` | Histogram | 备份文件大小 |
| `db_backup_errors_total` | Counter | 备份错误次数 |

---

## 错误码

| 错误码 | HTTP 状态 | 说明 |
|--------|----------|------|
| `0` | 200 | 成功 |
| `400` | 400 | 请求参数错误 |
| `401` | 401 | 未授权 |
| `404` | 404 | 资源不存在 |
| `500` | 500 | 服务器内部错误 |

---

## 速率限制

| 端点 | 限制 |
|------|------|
| POST /api/jobs | 10次/分钟 |
| POST /api/jobs/:id/run | 5次/分钟 |
| GET /api/* | 100次/分钟 |
| WebSocket | 10个连接 |

---

## 代码示例

### cURL

```bash
# 获取任务列表
curl http://localhost:8080/api/jobs

# 创建任务
curl -X POST http://localhost:8080/api/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "id": "mysql-backup",
    "database": {
      "type": "mysql",
      "host": "localhost",
      "port": 3306,
      "username": "root",
      "password": "password",
      "database": "mydb"
    },
    "schedule": "0 2 * * *"
  }'

# 立即执行任务
curl -X POST http://localhost:8080/api/jobs/mysql-backup/run

# 获取备份记录
curl http://localhost:8080/api/records?job_id=mysql-backup
```

### Python

```python
import requests

API_BASE = "http://localhost:8080/api"

# 获取任务列表
response = requests.get(f"{API_BASE}/jobs")
jobs = response.json()
print(f"共有 {len(jobs['data'])} 个任务")

# 创建任务
task = {
    "id": "mysql-backup",
    "database": {
        "type": "mysql",
        "host": "localhost",
        "port": 3306,
        "username": "root",
        "password": "password",
        "database": "mydb"
    },
    "schedule": "0 2 * * *"
}

response = requests.post(f"{API_BASE}/jobs", json=task)
result = response.json()
print(f"创建任务: {result}")
```

### JavaScript

```javascript
// 使用 Fetch API
const response = await fetch('/api/jobs');
const jobs = await response.json();

// 使用 Axios
import axios from 'axios';

const api = axios.create({
  baseURL: 'http://localhost:8080/api',
});

// 获取任务列表
const { data } = await api.get('/jobs');

// 创建任务
const response = await api.post('/jobs', {
  id: 'mysql-backup',
  database: {
    type: 'mysql',
    host: 'localhost',
    port: 3306,
    username: 'root',
    password: 'password',
    database: 'mydb'
  },
  schedule: '0 2 * * *'
});
```

---

*最后更新: 2026-03-20*
