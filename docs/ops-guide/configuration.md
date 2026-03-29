# 配置详解

本文档详细介绍 config.yaml 的所有配置选项。

## 完整配置示例

```yaml
global:
  work_dir: /data/backups
  default_tz: Asia/Shanghai
  max_concurrent: 5
  timeout: 2h

log:
  level: info
  format: console
  file: /var/log/db-backup/app.log

encryption:
  enabled: false
  key: ""

storage:
  type: local
  path: /data/backups

retention:
  default_days: 30
  default_count: 10

notify:
  dingtalk:
    enabled: false
    webhook: ""
  feishu:
    enabled: false
    webhook: ""
  wechat:
    enabled: false
    webhook: ""
  email:
    enabled: false
    smtp_host: ""
    smtp_port: 465

tasks:
  - id: mysql-backup
    enabled: true
    database:
      type: mysql
      host: localhost
      port: 3306
      username: root
      password: ""
      database: mydb
    schedule: "0 2 * * *"
    mode: full
    compression:
      enabled: true
      level: 6
    encryption:
      enabled: false
    storage: local
    retention:
      days: 30
      count: 10
    notify:
      on_success:
        - feishu
      on_failure:
        - dingtalk
        - email
```

---

## global 全局配置

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `work_dir` | string | /tmp/db-backup | 工作目录，临时文件存放位置 |
| `default_tz` | string | Asia/Shanghai | 默认时区 |
| `max_concurrent` | int | 5 | 最大并发任务数 |
| `timeout` | duration | 2h | 全局超时时间 |

```yaml
global:
  work_dir: /data/backups
  default_tz: Asia/Shanghai
  max_concurrent: 5
  timeout: 2h
```

---

## log 日志配置

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `level` | string | info | 日志级别: debug/info/warn/error |
| `format` | string | console | 日志格式: console/json |
| `file` | string | 空 | 日志文件路径 |

```yaml
log:
  level: info
  format: console
  file: /var/log/db-backup/app.log
```

---

## encryption 加密配置

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `enabled` | bool | false | 是否启用加密 |
| `key` | string | - | 加密密钥（任意长度） |

```yaml
encryption:
  enabled: true
  key: ${DB_BACKUP_ENCRYPTION_KEY}
```

> 加密算法：AES-256-GCM

---

## storage 存储配置

### local 本地存储

```yaml
storage:
  type: local
  path: /data/backups
```

| 配置项 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `type` | string | 是 | 固定值: local |
| `path` | string | 是 | 存储路径 |

### s3 S3/MinIO 存储

```yaml
storage:
  type: s3
  endpoint: https://s3.amazonaws.com
  bucket: my-backups
  access_key: ${AWS_ACCESS_KEY}
  secret_key: ${AWS_SECRET_KEY}
  region: us-east-1
```

| 配置项 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `type` | string | 是 | 固定值: s3 |
| `endpoint` | string | 是 | S3 端点地址 |
| `bucket` | string | 是 | 存储桶名称 |
| `access_key` | string | 是 | Access Key ID |
| `secret_key` | string | 是 | Access Key Secret |
| `region` | string | 否 | 区域 |

### oss 阿里云 OSS

```yaml
storage:
  type: oss
  endpoint: https://oss-cn-hangzhou.aliyuncs.com
  bucket: my-backups
  access_key: ${ALIYUN_ACCESS_KEY}
  secret_key: ${ALIYUN_SECRET_KEY}
```

### cos 腾讯云 COS

```yaml
storage:
  type: cos
  endpoint: https://cos.ap-guangzhou.myqcloud.com
  bucket: my-backups-1234567890
  secret_id: ${TENCENT_SECRET_ID}
  secret_key: ${TENCENT_SECRET_KEY}
```

---

## retention 保留策略

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `default_days` | int | 30 | 默认保留天数 |
| `default_count` | int | 10 | 默认保留数量 |

```yaml
retention:
  default_days: 30
  default_count: 10
```

---

## tasks 任务配置

### 基础字段

| 配置项 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `id` | string | 是 | 任务唯一标识 |
| `enabled` | bool | 否 | 是否启用，默认 true |
| `database` | object | 是 | 数据库配置 |
| `schedule` | string | 是 | Cron 表达式 |
| `mode` | string | 否 | 备份模式: full/incremental |

### database 数据库配置

#### MySQL

```yaml
database:
  type: mysql
  host: localhost
  port: 3306
  username: root
  password: ${MYSQL_PASSWORD}
  database: mydb
```

#### PostgreSQL

```yaml
database:
  type: postgresql
  host: localhost
  port: 5432
  username: postgres
  password: ${PGPASSWORD}
  database: mydb
```

#### MongoDB

```yaml
database:
  type: mongodb
  host: localhost
  port: 27017
  username: admin
  password: ${MONGO_PASSWORD}
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
  password: ${SQLSERVER_PASSWORD}
  database: mydb
```

#### Oracle

```yaml
database:
  type: oracle
  host: localhost
  port: 1521
  username: system
  password: ${ORACLE_PASSWORD}
  service_name: ORCL
```

### compression 压缩配置

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `enabled` | bool | false | 是否启用压缩 |
| `level` | int | 6 | 压缩级别 (1-9) |

```yaml
compression:
  enabled: true
  level: 6
```

### notify 通知配置

```yaml
notify:
  on_success:
    - feishu
  on_failure:
    - dingtalk
    - email
```

---

## 环境变量

配置文件支持环境变量替换：

```yaml
password: ${MYSQL_PASSWORD}
```

### 支持的环境变量

| 变量 | 说明 |
|------|------|
| `DB_BACKUP_TEMP_DIR` | 临时目录 |
| `DB_BACKUP_ENCRYPTION_KEY` | 加密密钥 |
| `MYSQL_PASSWORD` | MySQL 密码 |
| `PGPASSWORD` | PostgreSQL 密码 |
| `MONGO_PASSWORD` | MongoDB 密码 |

---

## 配置验证

```bash
./db-backup -config configs/config.yaml -validate
```

---

*最后更新: 2026-03-20*
