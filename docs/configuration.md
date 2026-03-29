# 配置文档

本文档详细介绍数据库备份系统的配置选项。

## 配置文件

默认配置文件路径：`configs/config.yaml`

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
    webhook: ""
  wechat:
    webhook: ""
  feishu:
    webhook: ""

tasks:
  - id: mysql-prod-backup
    enabled: true
    database:
      type: mysql
      host: localhost
      port: 3306
      username: root
      password: ${MYSQL_PASSWORD}
      database: mydb
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

## 配置项详解

### global 全局配置

| 配置项 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|--------|------|
| `work_dir` | string | 否 | `/tmp/db-backup` | 工作目录 |
| `default_tz` | string | 否 | `Asia/Shanghai` | 默认时区 |
| `max_concurrent` | int | 否 | `5` | 最大并发任务数 |
| `timeout` | duration | 否 | `2h` | 全局超时时间 |

### log 日志配置

| 配置项 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|--------|------|
| `level` | string | 否 | `info` | 日志级别: debug/info/warn/error |
| `format` | string | 否 | `console` | 日志格式: console/json |
| `file` | string | 否 | 空 | 日志文件路径 |

### encryption 加密配置

| 配置项 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|--------|------|
| `enabled` | bool | 否 | `false` | 是否启用加密 |
| `key` | string | 是* | - | 加密密钥 |

*当 `enabled: true` 时必填

### storage 存储配置

#### local 本地存储

| 配置项 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `type` | string | 是 | 固定值: `local` |
| `path` | string | 是 | 存储路径 |

#### s3 S3/MinIO 存储

| 配置项 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `type` | string | 是 | 固定值: `s3` |
| `endpoint` | string | 是 | S3 端点地址 |
| `bucket` | string | 是 | 存储桶名称 |
| `access_key` | string | 是 | Access Key ID |
| `secret_key` | string | 是 | Access Key Secret |

#### oss 阿里云 OSS 存储

| 配置项 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `type` | string | 是 | 固定值: `oss` |
| `endpoint` | string | 是 | OSS 端点地址 |
| `bucket` | string | 是 | 存储桶名称 |
| `access_key` | string | 是 | Access Key ID |
| `secret_key` | string | 是 | Access Key Secret |

#### cos 腾讯云 COS 存储

| 配置项 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `type` | string | 是 | 固定值: `cos` |
| `endpoint` | string | 是 | COS 端点地址 |
| `bucket` | string | 是 | 存储桶名称（含 APPID） |
| `secret_id` | string | 是 | Secret ID |
| `secret_key` | string | 是 | Secret Key |

### retention 保留策略

| 配置项 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|--------|------|
| `default_days` | int | 否 | `30` | 默认保留天数 |
| `default_count` | int | 否 | `10` | 默认保留数量 |

### tasks 任务配置

| 配置项 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `id` | string | 是 | 任务唯一标识 |
| `enabled` | bool | 否 | 是否启用 |
| `database` | object | 是 | 数据库配置 |
| `schedule` | string | 是 | Cron 表达式 |
| `mode` | string | 否 | 备份模式: full/incremental |
| `compression` | object | 否 | 压缩配置 |
| `encryption` | object | 否 | 加密配置 |
| `storage` | string | 否 | 存储类型 |
| `retention` | object | 否 | 保留策略 |

#### database 数据库配置

| 配置项 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `type` | string | 是 | 数据库类型: mysql/postgresql/mongodb/sqlserver/oracle |
| `host` | string | 是 | 主机地址 |
| `port` | int | 是 | 端口 |
| `username` | string | 是 | 用户名 |
| `password` | string | 是 | 密码（建议使用环境变量） |

**MySQL 特有字段**

| 配置项 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `database` | string | 是 | 数据库名 |

**PostgreSQL 特有字段**

| 配置项 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `database` | string | 是 | 数据库名 |

**MongoDB 特有字段**

| 配置项 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `database` | string | 是 | 数据库名 |

**SQL Server 特有字段**

| 配置项 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `database` | string | 是 | 数据库名 |

**Oracle 特有字段**

| 配置项 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `service_name` | string | 是 | 服务名 |

#### compression 压缩配置

| 配置项 | 类型 | 必填 | 默认值 | 说明 |
|--------|------|------|--------|------|
| `enabled` | bool | 否 | `false` | 是否启用压缩 |
| `level` | int | 否 | `6` | 压缩级别 (1-9) |

## 环境变量

配置文件支持使用环境变量

```yaml
password: ${MYSQL_PASSWORD}
```

### 支持的环境变量

| 变量 | 说明 |
|------|------|
| `DB_BACKUP_TEMP_DIR` | 临时目录 |
| `DB_BACKUP_KEEP_TEMP` | 保留临时文件 |
| `DB_BACKUP_ENCRYPTION_KEY` | 加密密钥 |
| `MYSQL_PASSWORD` | MySQL 密码 |
| `PGPASSWORD` | PostgreSQL 密码 |
| `MONGO_PASSWORD` | MongoDB 密码 |
| `AWS_ACCESS_KEY` | AWS Access Key |
| `AWS_SECRET_KEY` | AWS Secret Key |
| `ALIYUN_ACCESS_KEY` | 阿里云 Access Key |
| `ALIYUN_SECRET_KEY` | 阿里云 Secret Key |
| `TENCENT_SECRET_ID` | 腾讯云 Secret ID |
| `TENCENT_SECRET_KEY` | 腾讯云 Secret Key |

## Cron 表达式

格式：`秒 分 时 日 月 周`

| 字段 | 允许值 | 特殊字符 |
|------|--------|----------|
| 秒 | 0-59 | * / , - |
| 分 | 0-59 | * / , - |
| 时 | 0-23 | * / , - |
| 日 | 1-31 | * / , - |
| 月 | 1-12 | * / , - |
| 周 | 0-6 | * / , - |

**示例**

| 表达式 | 说明 |
|--------|------|
| `0 2 * * *` | 每天凌晨 2:00 |
| `0 */6 * * *` | 每 6 小时 |
| `0 0 * * 0` | 每周日 0:00 |
| `0 0 1 * *` | 每月 1 日 0:00 |
