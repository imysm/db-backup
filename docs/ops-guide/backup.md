# 备份策略

本文档介绍各数据库的备份策略和最佳实践。

## 概述

### 备份类型

| 类型 | 说明 | 优点 | 缺点 |
|------|------|------|------|
| **全量备份** | 备份整个数据库 | 恢复简单 | 耗时、占用空间大 |
| **增量备份** | 仅备份变更数据 | 快速、占用空间小 | 恢复复杂 |

### 备份频率

| 场景 | 全量备份 | 增量备份 |
|------|----------|----------|
| 高频交易 | 每日 | 每小时 |
| 普通业务 | 每日 | 每 4 小时 |
| 低频业务 | 每周 | 每日 |

---

## MySQL 备份策略

### 推荐策略

- **全量备份**: 每天凌晨 2:00
- **增量备份**: 每小时 binlog 备份
- **保留策略**: 全量 30 天，binlog 7 天

### 配置示例

```yaml
tasks:
  - id: mysql-full-backup
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
    retention:
      days: 30
      count: 10
```

### mysqldump 参数

| 参数 | 说明 |
|------|------|
| `--single-transaction` | InnoDB 一致性快照 |
| `--routines` | 备份存储过程 |
| `--triggers` | 备份触发器 |
| `--events` | 备份事件 |
| `--set-gtid-purged=OFF` | 避免 GTID 问题 |

---

## PostgreSQL 备份策略

### 推荐策略

- **全量备份**: 每天凌晨 2:00
- **增量备份**: WAL 归档
- **保留策略**: 全量 30 天，WAL 7 天

### 配置示例

```yaml
tasks:
  - id: postgres-full-backup
    database:
      type: postgresql
      host: localhost
      port: 5432
      username: postgres
      password: ${PGPASSWORD}
      database: mydb
    schedule: "0 2 * * *"
    mode: full
    compression:
      enabled: true
      level: 6
    retention:
      days: 30
```

### WAL 归档配置

```ini
# postgresql.conf
wal_level = replica
archive_mode = on
archive_command = 'cp %p /archive/%f'
```

---

## MongoDB 备份策略

### 推荐策略

- **全量备份**: 每天凌晨 2:00
- **增量备份**: oplog 备份（需副本集）
- **保留策略**: 全量 30 天，oplog 7 天

### 配置示例

```yaml
tasks:
  - id: mongodb-full-backup
    database:
      type: mongodb
      host: localhost
      port: 27017
      username: admin
      password: ${MONGO_PASSWORD}
      database: mydb
      auth_database: admin
    schedule: "0 2 * * *"
    mode: full
    retention:
      days: 30
```

---

## SQL Server 备份策略

### 推荐策略

- **全量备份**: 每天凌晨 2:00
- **差异备份**: 每 6 小时
- **事务日志**: 每 15 分钟
- **保留策略**: 全量 30 天，差异 7 天，日志 24 小时

### 配置示例

```yaml
tasks:
  - id: sqlserver-full-backup
    database:
      type: sqlserver
      host: localhost
      port: 1433
      username: sa
      password: ${SQLSERVER_PASSWORD}
      database: mydb
    schedule: "0 2 * * *"
    mode: full
    retention:
      days: 30
```

---

## Oracle 备份策略

### 推荐策略

- **全量备份**: 每天凌晨 2:00 (Data Pump)
- **增量备份**: RMAN 增量
- **归档日志**: 实时备份
- **保留策略**: 全量 30 天，增量 7 天

### 配置示例

```yaml
tasks:
  - id: oracle-full-backup
    database:
      type: oracle
      host: localhost
      port: 1521
      username: system
      password: ${ORACLE_PASSWORD}
      service_name: ORCL
    schedule: "0 2 * * *"
    mode: full
    retention:
      days: 30
```

---

## 保留策略

### 按时间

```yaml
retention:
  days: 30  # 保留 30 天
```

### 按数量

```yaml
retention:
  count: 10  # 保留最近 10 个
```

### 组合策略

```yaml
retention:
  days: 30
  count: 10
```

---

## 最佳实践

### 1. 3-2-1 原则

- 3 份备份
- 2 种存储介质
- 1 份异地

### 2. 定期验证

- 每周自动验证备份完整性
- 每月执行恢复演练

### 3. 监控告警

- 备份失败立即告警
- 存储空间不足告警
- 长时间无成功备份告警

---

*最后更新: 2026-03-20*
