# 数据恢复指南

本文档介绍如何从备份恢复数据，包括本机恢复、异机恢复和测试恢复。

## 恢复类型

| 类型 | 说明 | 适用场景 |
|------|------|----------|
| **本机恢复** | 恢复到原数据库服务器 | 数据损坏、误删除 |
| **异机恢复** | 恢复到不同服务器 | 灾难恢复、数据迁移 |
| **测试恢复** | 恢复到测试环境 | 验证备份可用性 |

---

## 快速恢复

### CLI 方式

```bash
# 立即恢复（交互式）
./db-backup -config configs/config.yaml -restore rec_20260320020000
```

### API 方式

```bash
curl -X POST http://localhost:8080/api/restore \
  -H "Content-Type: application/json" \
  -d '{
    "record_id": "rec_20260320020000",
    "target_host": "localhost",
    "target_port": 3306,
    "target_database": "mydb_restored"
  }'
```

---

## MySQL 恢复

### 本机恢复

```bash
# 1. 解压备份文件
gunzip mydb_20260320_020000.sql.gz

# 2. 恢复到原数据库
mysql -h localhost -P 3306 -u root -p mydb < mydb_20260320_020000.sql
```

### 异机恢复

```bash
# 1. 复制备份文件到目标服务器
scp mydb_20260320_020000.sql.gz target-server:/tmp/

# 2. 在目标服务器执行恢复
ssh target-server
gunzip /tmp/mydb_20260320_020000.sql.gz
mysql -h localhost -P 3306 -u root -p mydb_restored < /tmp/mydb_20260320_020000.sql
```

### 增量恢复（binlog）

```bash
# 1. 先恢复全量备份
mysql -h localhost -P 3306 -u root -p mydb < full_backup.sql

# 2. 再应用 binlog
mysqlbinlog binlog.000001 binlog.000002 | mysql -h localhost -P 3306 -u root -p
```

---

## PostgreSQL 恢复

### SQL 格式恢复

```bash
# 恢复 SQL 格式备份
psql -h localhost -p 5432 -U postgres mydb_restored < backup.sql
```

### 自定义格式恢复

```bash
# 恢复自定义格式备份（支持并行）
pg_restore -h localhost -p 5432 -U postgres \
  -d mydb_restored \
  -j 4 \
  backup.custom
```

### PITR 恢复

```bash
# 1. 恢复基础备份
pg_basebackup -h localhost -D /var/lib/postgresql/data_restore

# 2. 配置恢复
cat > /var/lib/postgresql/data_restore/postgresql.auto.conf << EOF
restore_command = 'cp /archive/%f %p'
recovery_target_time = '2026-03-20 02:00:00'
EOF

# 3. 启动恢复
touch /var/lib/postgresql/data_restore/recovery.signal
pg_ctl -D /var/lib/postgresql/data_restore start
```

---

## MongoDB 恢复

### 全量恢复

```bash
mongorestore --host localhost --port 27017 \
  --username admin --password \
  --authenticationDatabase admin \
  --db mydb_restored \
  /backup/mongodb/mydb
```

### Oplog 恢复

```bash
# 1. 恢复全量备份
mongorestore /backup/mongodb

# 2. 应用 oplog
mongorestore --oplogReplay /backup/oplog
```

---

## SQL Server 恢复

### 完整恢复

```sql
-- 恢复全量备份
RESTORE DATABASE mydb_restored
FROM DISK = 'D:\backup\mydb_full.bak'
WITH MOVE 'mydb' TO 'D:\data\mydb_restored.mdf',
     MOVE 'mydb_log' TO 'D:\data\mydb_restored_log.ldf',
RECOVERY;
```

### 时间点恢复

```sql
-- 1. 恢复全量备份（NORECOVERY）
RESTORE DATABASE mydb_restored
FROM DISK = 'D:\backup\mydb_full.bak'
WITH NORECOVERY;

-- 2. 恢复事务日志到指定时间点
RESTORE LOG mydb_restored
FROM DISK = 'D:\backup\mydb_log.trn'
WITH STOPAT = '2026-03-20 02:00:00',
RECOVERY;
```

---

## Oracle 恢复

### Data Pump 恢复

```bash
impdp system/password \
  DIRECTORY=backup_dir \
  DUMPFILE=mydb_full_%U.dmp \
  SCHEMAS=mydb \
  REMAP_SCHEMA=mydb:mydb_restored \
  LOGFILE=restore.log
```

### RMAN 恢复

```bash
rman target /

# 恢复数据库
RUN {
  SET UNTIL TIME '2026-03-20 02:00:00';
  RESTORE DATABASE;
  RECOVER DATABASE;
}

# 打开数据库
ALTER DATABASE OPEN RESETLOGS;
```

---

## 恢复验证

### 数据完整性检查

```bash
# MySQL
mysql -e "SELECT COUNT(*) FROM mydb.users"

# PostgreSQL
psql -c "SELECT COUNT(*) FROM users" mydb

# MongoDB
mongosh --eval "db.users.count()"
```

### 应用连接测试

```bash
# 测试数据库连接
mysql -h localhost -u app_user -p mydb_restored -e "SELECT 1"
```

---

## 最佳实践

### 1. 恢复前检查

- ✅ 确认备份文件完整
- ✅ 验证目标数据库状态
- ✅ 准备足够的磁盘空间

### 2. 恢复步骤

1. 停止应用服务
2. 备份当前数据（以防万一）
3. 执行恢复操作
4. 验证数据完整性
5. 重启应用服务

### 3. 定期演练

- 每月执行一次测试恢复
- 记录恢复时间（RTO）
- 验证数据完整性

---

## 常见问题

### Q: 恢复时提示数据库已存在？

```bash
# MySQL: 删除或重命名现有数据库
mysql -e "DROP DATABASE IF EXISTS mydb_restored"

# PostgreSQL: 删除现有数据库
psql -c "DROP DATABASE IF EXISTS mydb_restored"
```

### Q: 恢复速度太慢？

```bash
# MySQL: 禁用外键检查
mysql -e "SET FOREIGN_KEY_CHECKS=0; SOURCE backup.sql; SET FOREIGN_KEY_CHECKS=1;"

# PostgreSQL: 并行恢复
pg_restore -j 4 backup.custom
```

### Q: 恢复后数据不完整？

检查：
1. 备份文件是否完整
2. 是否需要增量备份
3. 恢复过程是否有错误

---

*最后更新: 2026-03-20*
