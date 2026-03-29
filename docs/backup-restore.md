# 备份恢复指南

本文档介绍如何使用 db-backup 系统进行数据库备份和恢复操作。

## 概述

db-backup 支持以下数据库的备份与恢复：

| 数据库 | 全量备份 | 增量备份 | 压缩 | 加密 |
|--------|----------|----------|------|------|
| MySQL | ✅ mysqldump | ✅ binlog | ✅ gzip | ✅ AES-256 |
| PostgreSQL | ✅ pg_dump | ✅ WAL | ✅ gzip | ✅ AES-256 |
| MongoDB | ✅ mongodump | ✅ oplog | ✅ gzip | ✅ AES-256 |
| SQL Server | ✅ BACKUP | ✅ 差异/日志 | ✅ native | ✅ AES-256 |
| Oracle | ✅ expdp/RMAN | ✅ RMAN | ✅ native | ✅ AES-256 |

---

## MySQL

### 全量备份

系统使用 `mysqldump` 进行全量备份：

```bash
mysqldump -h localhost -P 3306 -u root -p \
  --single-transaction \
  --routines \
  --triggers \
  --events \
  --set-gtid-purged=OFF \
  --compress \
  --result-file=backup.sql \
  mydb
```

### 增量备份

增量备份通过 binlog 实现：

1. **确保 binlog 已开启**

```sql
SHOW VARIABLES LIKE 'log_bin';
-- 值应为 ON
```

2. **配置 binlog 格式**

```sql
SHOW VARIABLES LIKE 'binlog_format';
-- 建议使用 ROW
```

3. **备份 binlog 文件**

```bash
mysqlbinlog --read-from-remote-server --raw \
  --host=localhost --port=3306 \
  --user=root --password \
  --stop-never binlog.000001 > binlog_backup.sql
```

### 恢复

**恢复全量备份**

```bash
mysql -h localhost -P 3306 -u root -p mydb < backup.sql
```

**恢复 binlog（增量）**

```bash
mysqlbinlog binlog_backup.sql | mysql -h localhost -P 3306 -u root -p
```

---

## PostgreSQL

### 全量备份

系统使用 `pg_dump` 进行全量备份：

```bash
# SQL 格式
pg_dump -h localhost -p 5432 -U postgres mydb > backup.sql

# 自定义格式（支持压缩）
pg_dump -h localhost -p 5432 -U postgres \
  -Fc -Z6 \
  -f backup.custom mydb
```

### 增量备份

增量备份通过 WAL 归档实现：

1. **配置 WAL 归档**

```ini
# postgresql.conf
wal_level = replica
archive_mode = on
archive_command = 'cp %p /archive/%f'
```

2. **手动切换 WAL**

```bash
psql -c "SELECT pg_switch_wal();"
```

3. **备份 WAL 文件**

```bash
cp /archive/* /backup/wal/
```

### 恢复

**恢复 SQL 格式**

```bash
psql -h localhost -p 5432 -U postgres mydb < backup.sql
```

**恢复自定义格式**

```bash
pg_restore -h localhost -p 5432 -U postgres \
  -d mydb_restored \
  -Fc backup.custom
```

---

## MongoDB

### 全量备份

系统使用 `mongodump` 进行全量备份：

```bash
mongodump --host localhost --port 27017 \
  --username admin --password \
  --authenticationDatabase admin \
  --db mydb \
  --out /backup/mongodb
```

### 增量备份

增量备份通过 oplog 实现（需要副本集）：

1. **确保是副本集**

```javascript
rs.status()
```

2. **备份 oplog**

```bash
mongodump --host localhost --port 27017 \
  --username admin --password \
  --authenticationDatabase admin \
  --db local \
  --collection oplog.rs \
  --out /backup/oplog
```

### 恢复

```bash
mongorestore --host localhost --port 27017 \
  --username admin --password \
  --authenticationDatabase admin \
  --db mydb \
  /backup/mongodb/mydb
```

---

## SQL Server

### 全量备份

使用 T-SQL 或 `sqlcmd`：

```sql
BACKUP DATABASE mydb
TO DISK = 'D:\backup\mydb_full.bak'
WITH FORMAT,
MEDIANAME = 'mydb_full',
DESCRIPTION = 'Full backup of mydb';
```

### 差异备份

```sql
BACKUP DATABASE mydb
TO DISK = 'D:\backup\mydb_diff.bak'
WITH DIFFERENTIAL,
MEDIANAME = 'mydb_diff',
DESCRIPTION = 'Differential backup of mydb';
```

### 事务日志备份

```sql
BACKUP LOG mydb
TO DISK = 'D:\backup\mydb_log.trn'
WITH MEDIANAME = 'mydb_log',
DESCRIPTION = 'Transaction log backup';
```

### 恢复

```sql
-- 步骤1: 恢复全量备份（NORECOVERY）
RESTORE DATABASE mydb
FROM DISK = 'D:\backup\mydb_full.bak'
WITH NORECOVERY;

-- 步骤2: 恢复差异备份（NORECOVERY）
RESTORE DATABASE mydb
FROM DISK = 'D:\backup\mydb_diff.bak'
WITH NORECOVERY;

-- 步骤3: 恢复事务日志（RECOVERY）
RESTORE LOG mydb
FROM DISK = 'D:\backup\mydb_log.trn'
WITH RECOVERY;
```

---

## Oracle

### 全量备份 (Data Pump)

```bash
expdp system/password \
  DIRECTORY=backup_dir \
  DUMPFILE=mydb_full_%U.dmp \
  FILESIZE=10G \
  SCHEMAS=mydb \
  COMPRESSION=ALL \
  LOGFILE=mydb_full.log
```

### 增量备份 (RMAN)

```bash
# 连接 RMAN
rman target /

# 配置增量备份
CONFIGURE RETENTION POLICY TO RECOVERY WINDOW OF 30 DAYS;
CONFIGURE BACKUP OPTIMIZATION ON;

# 执行增量备份
BACKUP INCREMENTAL LEVEL 1 DATABASE;
```

### 恢复

**Data Pump 恢复**

```bash
impdp system/password \
  DIRECTORY=backup_dir \
  DUMPFILE=mydb_full_%U.dmp \
  SCHEMAS=mydb \
  LOGFILE=mydb_restore.log
```

**RMAN 恢复**

```bash
rman target /

RESTORE DATABASE;
RECOVER DATABASE;
```

---

## 最佳实践

### 1. 备份策略

| 环境 | 策略 | 频率 |
|------|------|------|
| 生产 | 全量 + 增量 | 每日全量 + 每小时增量 |
| 测试 | 仅全量 | 每日 |
| 开发 | 仅全量 | 按需 |

### 2. 存储策略

| 方案 | 优点 | 缺点 |
|------|------|------|
| 本地存储 | 恢复快 | 无异地容灾 |
| 云存储 | 异地容灾 | 成本高 |
| 混合存储 | 兼顾速度和容灾 | 复杂度高 |

### 3. 保留策略

| 策略 | 配置 |
|------|------|
| 按时间 | 保留 30 天 |
| 按数量 | 保留最近 10 个 |
| 按周期 | 每周保留 1 个，每月保留 1 个 |

### 4. 安全建议

- ✅ 启用备份文件加密
- ✅ 定期验证备份可用性
- ✅ 异地备份存储
- ✅ 定期进行恢复演练
- ✅ 使用环境变量存储密码
- ✅ 限制备份文件访问权限

### 5. 监控告警

- 备份失败告警
- 备份超时告警
- 存储空间不足告警
- 备份文件损坏告警

---

## 常见问题

### Q: 备份时出现 "Too many connections" 错误？

**A:** 检查数据库最大连接数配置，适当减少并发备份数量。

### Q: 增量备份恢复顺序？

**A:** 先恢复全量备份，再按时间顺序恢复增量备份。

### Q: 备份文件太大怎么办？

**A:** 
1. 启用压缩
2. 分卷备份
3. 只备份必要数据

### Q: 如何验证备份是否可用？

**A:** 
1. 使用验证功能检查文件完整性
2. 定期进行恢复测试
3. 检查备份日志

---

*最后更新: 2026-03-20*
