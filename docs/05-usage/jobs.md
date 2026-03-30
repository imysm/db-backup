# 任务管理

本文档介绍如何创建、管理和执行备份任务。

## 任务概述

每个备份任务定义了：
- 要备份的数据库
- 备份执行时间
- 备份存储位置
- 文件保留策略

## 创建任务

### 通过配置文件

编辑 `configs/config.yaml`：

```yaml
tasks:
  - id: mysql-prod-backup
    enabled: true
    database:
      type: mysql
      host: 192.168.1.100
      port: 3306
      username: backup_user
      password: ${MYSQL_PASSWORD}
      database: production
    schedule: "0 2 * * *"
    mode: full
    compression:
      enabled: true
      level: 6
    encryption:
      enabled: true
    storage: local
    retention:
      days: 30
      count: 10
```

### 通过 API

```bash
curl -X POST http://localhost:8080/api/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "id": "mysql-prod-backup",
    "database": {
      "type": "mysql",
      "host": "192.168.1.100",
      "port": 3306,
      "username": "backup_user",
      "password": "password",
      "database": "production"
    },
    "schedule": "0 2 * * *"
  }'
```

## 任务配置详解

### 基础配置

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `id` | string | 是 | 任务唯一标识 |
| `enabled` | bool | 否 | 是否启用，默认 true |

### 数据库配置

#### MySQL

```yaml
database:
  type: mysql
  host: localhost
  port: 3306
  username: root
  password: password
  database: mydb
```

#### PostgreSQL

```yaml
database:
  type: postgresql
  host: localhost
  port: 5432
  username: postgres
  password: password
  database: mydb
```

#### MongoDB

```yaml
database:
  type: mongodb
  host: localhost
  port: 27017
  username: admin
  password: password
  database: mydb
  auth_database: admin
```

#### SQL Server

```yaml
database:
  type: sqlserver
  host: localhost
  port: 1433
  username: sa
  password: password
  database: mydb
```

#### Oracle

```yaml
database:
  type: oracle
  host: localhost
  port: 1521
  username: system
  password: password
  service_name: ORCL
```

### 调度配置

使用 Cron 表达式定义执行时间：

```yaml
schedule: "0 2 * * *"
```

## Cron 表达式

### 格式

```
┌───────────── 分钟 (0 - 59)
│ ┌───────────── 小时 (0 - 23)
│ │ ┌───────────── 日 (1 - 31)
│ │ │ ┌───────────── 月 (1 - 12)
│ │ │ │ ┌───────────── 星期 (0 - 6)
│ │ │ │ │
* * * * *
```

### 特殊字符

| 字符 | 说明 | 示例 |
|------|------|------|
| `*` | 任意值 | `* * * * *` = 每分钟 |
| `/` | 间隔 | `*/5 * * * *` = 每 5 分钟 |
| `,` | 列表 | `0,30 * * * *` = 每小时 0,30 分 |
| `-` | 范围 | `0 9-17 * * *` = 9-17 点整点 |

### 常用示例

| 表达式 | 说明 |
|--------|------|
| `0 2 * * *` | 每天凌晨 2:00 |
| `0 */6 * * *` | 每 6 小时 |
| `30 1 * * *` | 每天凌晨 1:30 |
| `0 0 * * 0` | 每周日 0:00 |
| `0 0 1 * *` | 每月 1 日 0:00 |
| `0 9 * * 1-5` | 周一到周五 9:00 |

### 时区

默认使用配置文件中的时区：

```yaml
global:
  default_tz: Asia/Shanghai
```

## 压缩配置

```yaml
compression:
  enabled: true
  level: 6    # 1-9, 数字越大压缩率越高，速度越慢
```

| 级别 | 压缩率 | 速度 | 推荐场景 |
|------|--------|------|----------|
| 1 | 低 | 最快 | 大文件，快速备份 |
| 6 | 中 | 中等 | 默认，平衡选择 |
| 9 | 高 | 最慢 | 小文件，存储空间紧张 |

## 加密配置

```yaml
encryption:
  enabled: true
  key: ${DB_BACKUP_ENCRYPTION_KEY}
```

> ⚠️ **安全提示**: 使用环境变量存储加密密钥，不要硬编码在配置文件中。

## 保留策略

```yaml
retention:
  days: 30      # 保留 30 天
  count: 10     # 保留最近 10 个备份
```

策略组合使用时，同时满足两个条件。

## 管理任务

### 查看任务列表

**CLI**

```bash
# 通过 API 查看（需要服务运行）
curl http://localhost:8080/api/jobs
```

**配置文件**

```bash
# 直接查看配置文件
cat configs/config.yaml | grep -A 20 "tasks:"
```

### 启用/禁用任务

**配置文件**

```yaml
tasks:
  - id: mysql-backup
    enabled: false  # 禁用任务
```

**API**

```bash
curl -X PUT http://localhost:8080/api/jobs/mysql-backup \
  -H "Content-Type: application/json" \
  -d '{"enabled": false}'
```

### 立即执行任务

```bash
# CLI
./db-backup -config configs/config.yaml -run mysql-backup

# API
curl -X POST http://localhost:8080/api/jobs/mysql-backup/run
```

### 删除任务

**配置文件**

从 `configs/config.yaml` 中删除对应的任务配置。

**API**

```bash
curl -X DELETE http://localhost:8080/api/jobs/mysql-backup
```

## 任务状态

| 状态 | 说明 |
|------|------|
| `enabled` | 任务已启用，等待调度 |
| `disabled` | 任务已禁用 |
| `running` | 任务正在执行 |

## 最佳实践

### 1. 命名规范

使用有意义的任务 ID：

```yaml
# 好的命名
id: mysql-prod-orders-backup
id: postgres-staging-users-backup

# 不好的命名
id: backup1
id: test
```

### 2. 环境隔离

为不同环境创建独立任务：

```yaml
tasks:
  - id: mysql-prod-backup
    database:
      host: prod-db.example.com
    schedule: "0 2 * * *"

  - id: mysql-staging-backup
    database:
      host: staging-db.example.com
    schedule: "0 3 * * *"
```

### 3. 错峰备份

避免同时执行大量备份任务：

```yaml
# 分散到不同时间
tasks:
  - id: db1-backup
    schedule: "0 1 * * *"   # 1:00
  - id: db2-backup
    schedule: "0 2 * * *"   # 2:00
  - id: db3-backup
    schedule: "0 3 * * *"   # 3:00
```

### 4. 使用环境变量

敏感信息使用环境变量：

```yaml
database:
  password: ${MYSQL_PASSWORD}
```

### 5. 合理设置保留策略

```yaml
# 高频备份 - 保留少
retention:
  days: 7
  count: 20

# 低频备份 - 保留多
retention:
  days: 90
  count: 12
```

---

## 下一步

- 📁 [备份记录](records.md) - 查看备份历史
- 🔄 [数据恢复](restore.md) - 从备份恢复数据
- ⚙️ [配置详解](../ops-guide/configuration.md) - 完整配置选项

---

*最后更新: 2026-03-20*
