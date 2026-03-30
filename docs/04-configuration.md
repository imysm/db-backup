# ⚙️ 配置详解

本文档详细介绍 db-backup 的所有配置项。

---

## 配置文件位置

默认按以下顺序查找配置文件：

1. `./configs/config.yaml`
2. `/etc/db-backup/config.yaml`
3. 环境变量 `DB_BACKUP_CONFIG`

---

## 完整配置示例

```yaml
# ============ 全局配置 ============
global:
  work_dir: /var/lib/db-backup          # 工作目录
  default_tz: Asia/Shanghai             # 默认时区
  max_concurrent: 5                      # 最大并发任务数
  timeout: 2h                           # 任务超时时间
  encryption_key: "your-32-byte-key"     # 加密密钥
  api_keys:                              # API 认证密钥
    - "your-api-key-1"
    - "your-api-key-2"

# ============ 数据库配置 ============
database:
  type: sqlite                          # sqlite / mysql / postgres
  dsn: /var/lib/db-backup/db-backup.db # 连接字符串
  max_conns: 10                        # 最大连接数

# ============ 日志配置 ============
log:
  level: info                           # debug / info / warn / error
  format: console                       # console / json
  file: /var/log/db-backup/app.log    # 日志文件路径

# ============ 任务配置 ============
tasks:
  - id: mysql-prod
    name: 生产环境MySQL备份
    enabled: true
    database:
      type: mysql
      host: localhost
      port: 3306
      username: backup_user
      password: "password"
      database: app_db
      params:
        max_allowed_packet: "1G"
        quick: "true"
    schedule:
      cron: "0 2 * * *"               # 每天凌晨2点
      timezone: Asia/Shanghai
    storage:
      type: local
      path: /data/backups/mysql
    retention:
      keep_last: 7
      keep_days: 30
      keep_weekly: 8
      keep_monthly: 6
    compression:
      enabled: true
      type: gzip
      level: 6
    encryption:
      enabled: false
      key_env: DB_BACKUP_ENCRYPTION_KEY
    notify:
      enabled: true
      type: dingtalk
      webhook: https://oapi.dingtalk.com/robot/send?access_token=xxx
```

---

## 配置项详解

### global 全局配置

| 配置项 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|--------|------|
| `work_dir` | string | 否 | `/tmp/db-backup` | 工作目录 |
| `default_tz` | string | 否 | `Asia/Shanghai` | 默认时区 |
| `max_concurrent` | int | 否 | `5` | 最大并发任务数 |
| `timeout` | duration | 否 | `2h` | 任务超时时间 |
| `encryption_key` | string | 否 | - | 加密密钥（32字节hex） |
| `api_keys` | []string | 否 | - | API 认证密钥列表 |
| `allowed_origins` | []string | 否 | 全部 | WebSocket Origin 白名单 |

### database 数据库配置

| 配置项 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|--------|------|
| `type` | string | 是 | `sqlite` | 数据库类型：sqlite / mysql / postgres |
| `dsn` | string | 是 | - | 连接字符串 |
| `max_conns` | int | 否 | `10` | 最大连接数 |

**DSN 示例**：

```yaml
# SQLite
dsn: /var/lib/db-backup/db-backup.db

# MySQL
dsn: "user:password@tcp(localhost:3306)/db_backup?charset=utf8mb4&parseTime=True"

# PostgreSQL
dsn: "host=localhost user=postgres password=xxx dbname=db_backup sslmode=disable"
```

### log 日志配置

| 配置项 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|--------|------|
| `level` | string | 否 | `info` | 日志级别 |
| `format` | string | 否 | `console` | 输出格式 |
| `file` | string | 否 | - | 日志文件路径 |

### tasks 任务配置

| 配置项 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `id` | string | 是 | 任务唯一标识 |
| `name` | string | 否 | 任务显示名称 |
| `enabled` | bool | 否 | 是否启用 |
| `database` | object | 是 | 数据库配置 |
| `schedule` | object | 是 | 调度配置 |
| `storage` | object | 是 | 存储配置 |
| `retention` | object | 否 | 保留策略 |
| `compression` | object | 否 | 压缩配置 |
| `encryption` | object | 否 | 加密配置 |
| `notify` | object | 否 | 通知配置 |

---

## 数据库配置 (database)

### MySQL

```yaml
database:
  type: mysql
  host: localhost
  port: 3306
  username: root
  password: password
  database: mydb
  params:
    max_allowed_packet: "1G"
    quick: "true"
```

### PostgreSQL

```yaml
database:
  type: postgresql
  host: localhost
  port: 5432
  username: postgres
  password: password
  database: mydb
```

### MongoDB

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

### SQL Server

```yaml
database:
  type: sqlserver
  host: localhost
  port: 1433
  username: sa
  password: password
  database: mydb
```

### Oracle

```yaml
database:
  type: oracle
  host: localhost
  port: 1521
  username: system
  password: password
  service_name: ORCL
```

---

## 调度配置 (schedule)

```yaml
schedule:
  cron: "0 2 * * *"          # Cron 表达式
  timezone: Asia/Shanghai     # 时区
```

**Cron 表达式格式**：`秒 分 时 日 月 周`

| 示例 | 说明 |
|------|------|
| `0 2 * * *` | 每天凌晨 2:00 |
| `0 */6 * * *` | 每 6 小时 |
| `0 0 * * 0` | 每周日 0:00 |
| `0 0 1 * *` | 每月 1 日 0:00 |
| `0 30 4 * * *` | 每天 4:30 |

---

## 存储配置 (storage)

### local 本地存储

```yaml
storage:
  type: local
  path: /data/backups/mysql
```

### S3/MinIO

```yaml
storage:
  type: s3
  endpoint: https://s3.amazonaws.com
  bucket: my-backups
  region: us-east-1
  access_key: AKIAIOSFODNN7EXAMPLE
  secret_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
  path: backups/mysql
```

### 阿里云 OSS

```yaml
storage:
  type: oss
  endpoint: oss-cn-shanghai.aliyuncs.com
  bucket: my-backups
  access_key: LTAI5txxxxxx
  secret_key: iM8xxxxxx
  path: backups/mysql
```

### 腾讯云 COS

```yaml
storage:
  type: cos
  region: ap-shanghai
  bucket: my-backups-1251234567
  secret_id: AKIDxxxxxx
  secret_key: xxxxxx
  path: backups/mysql
```

---

## 加密配置 (encryption)

```yaml
encryption:
  enabled: true
  key_env: DB_BACKUP_ENCRYPTION_KEY  # 从环境变量读取
```

或直接配置密钥（不推荐）：

```yaml
encryption:
  enabled: true
  key: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
```

**密钥生成**：

```bash
openssl rand -hex 32
```

---

## 压缩配置 (compression)

```yaml
compression:
  enabled: true
  type: gzip          # gzip / zstd / lz4
  level: 6            # 压缩级别 1-9
```

---

## 保留策略 (retention)

```yaml
retention:
  keep_last: 7        # 保留最近 7 个
  keep_days: 30       # 保留最近 30 天
  keep_weekly: 8      # 保留最近 8 周
  keep_monthly: 6     # 保留最近 6 个月
```

---

## 通知配置 (notify)

### 钉钉

```yaml
notify:
  enabled: true
  type: dingtalk
  webhook: https://oapi.dingtalk.com/robot/send?access_token=xxx
```

### 飞书

```yaml
notify:
  enabled: true
  type: feishu
  webhook: https://open.feishu.cn/open-apis/bot/v2/hook/xxx
```

### 企业微信

```yaml
notify:
  enabled: true
  type: wecom
  webhook: https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=xxx
```

### 邮件

```yaml
notify:
  enabled: true
  type: email
  smtp_host: smtp.example.com
  smtp_port: 465
  smtp_user: notification@example.com
  smtp_password: password
  to:
    - admin@example.com
    - ops@example.com
```

---

## 环境变量

| 变量 | 说明 |
|------|------|
| `DB_BACKUP_CONFIG` | 配置文件路径 |
| `DB_BACKUP_WORK_DIR` | 工作目录 |
| `DB_BACKUP_LOG_LEVEL` | 日志级别 |
| `DB_BACKUP_ENCRYPTION_KEY` | 加密密钥 |
| `MYSQL_PASSWORD` | MySQL 密码 |
| `PGPASSWORD` | PostgreSQL 密码 |
| `MONGOPASSWORD` | MongoDB 密码 |

---

## 下一步

- 📖 [快速入门](01-quick-start.md) - 快速启动
- 🚢 [部署指南](03-deployment.md) - 生产环境部署
- 🔒 [安全配置](06-security.md) - 安全加固

---

*有问题？查看 [常见问题](08-faq.md)。*
