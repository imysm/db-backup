# db-backup

企业级多源数据库备份系统，支持 MySQL、PostgreSQL、MongoDB、SQL Server、Oracle 五种主流数据库的备份与恢复。

## 特性

- **多数据库支持**: MySQL、PostgreSQL、MongoDB、SQL Server、Oracle
- **全量/增量备份**: 支持 binlog、WAL、oplog 等增量备份方式
- **多种存储后端**: 本地存储、S3/MinIO、阿里云 OSS、腾讯云 COS
- **Web 管理控制台**: 任务配置、实时日志、历史记录
- **加密备份**: AES-256-GCM 加密
- **压缩支持**: gzip 压缩，可配置压缩级别
- **WebSocket 实时日志**: 备份进度实时推送
- **Prometheus 监控**: 内置监控指标

## 快速开始

```bash
# 1. 克隆项目
git clone https://github.com/imysm/db-backup.git
cd db-backup

# 2. 构建
go build -o db-backup ./cmd/server

# 3. 创建配置文件
cat > configs/config.yaml << EOF
global:
  work_dir: /tmp/db-backup
tasks:
  - id: mysql-backup
    database:
      type: mysql
      host: localhost
      port: 3306
      username: root
      password: password
      database: mydb
    schedule: "0 2 * * *"
EOF

# 4. 验证配置
./db-backup -config configs/config.yaml -validate

# 5. 执行备份
./db-backup -config configs/config.yaml -run mysql-backup
```

详细步骤请参阅 [5 分钟快速上手](docs/quick-start.md)。

## 文档

### 快速入门
- [5 分钟快速上手](docs/quick-start.md) - 快速启动第一次备份
- [完整安装部署](docs/getting-started.md) - 生产环境部署指南

### 用户手册
- [概述和核心概念](docs/user-guide/overview.md) - 系统核心概念
- [任务管理](docs/user-guide/jobs.md) - 任务 CRUD + Cron 表达式
- [备份记录](docs/user-guide/records.md) - 备份记录查询和管理
- [数据恢复](docs/user-guide/restore.md) - 本机/异机/测试恢复
- [通知配置](docs/user-guide/notifications.md) - 钉钉/飞书/邮件/企微配置

### 运维指南
- [部署指南](docs/ops-guide/deployment.md) - 单机/高可用/Docker/K8s
- [配置详解](docs/ops-guide/configuration.md) - config.yaml 完整详解
- [监控告警](docs/ops-guide/monitoring.md) - Prometheus + Grafana + 告警
- [备份策略](docs/ops-guide/backup.md) - 各数据库备份策略

### 开发文档
- [架构设计](docs/dev-guide/architecture.md) - 架构设计 + 数据流
- [API 参考](docs/dev-guide/api.md) - 完整 RESTful API 参考
- [开发规范](docs/dev-guide/contributing.md) - 开发规范 + 测试 + 提交

### 设计文档
- [系统设计](docs/DESIGN.md) - 系统架构设计

## 支持的数据库

| 数据库 | 全量备份 | 增量备份 | 压缩 | 恢复 |
|--------|----------|----------|------|------|
| MySQL | ✅ mysqldump | ✅ binlog | ✅ | ✅ |
| PostgreSQL | ✅ pg_dump | ✅ WAL | ✅ | ✅ |
| MongoDB | ✅ mongodump | ✅ oplog | ✅ | ✅ |
| SQL Server | ✅ BACKUP | ✅ 差异/日志 | ✅ | ✅ |
| Oracle | ✅ expdp/RMAN | ✅ RMAN | ✅ | ✅ |

## 支持的存储

| 存储 | 状态 | 说明 |
|------|------|------|
| 本地存储 | ✅ | 存储到本地磁盘 |
| S3/MinIO | ✅ | 兼容 S3 协议 |
| 阿里云 OSS | ✅ | 阿里云对象存储 |
| 腾讯云 COS | ✅ | 腾讯云对象存储 |

## API 接口

```
GET  /health              # 健康检查
GET  /api/jobs            # 任务列表
POST /api/jobs            # 创建任务
GET  /api/jobs/:id        # 任务详情
PUT  /api/jobs/:id        # 更新任务
DELETE /api/jobs/:id      # 删除任务
POST /api/jobs/:id/run    # 立即执行
GET  /api/records         # 备份记录
POST /api/restore         # 执行恢复
POST /api/verify/:id      # 验证备份
GET  /ws                  # WebSocket 实时日志
GET  /metrics             # Prometheus 指标
```

详细 API 文档请参阅 [API 参考](docs/dev-guide/api.md)。

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `DB_BACKUP_TEMP_DIR` | 临时目录 | `/tmp/db-backup` |
| `DB_BACKUP_KEEP_TEMP` | 保留临时文件 | `false` |

## 开发

```bash
# 运行测试
go test ./...

# 查看覆盖率
go test -cover ./...

# 构建
go build -o db-backup ./cmd/server
```

详细开发指南请参阅 [开发规范](docs/dev-guide/contributing.md)。

## License

MIT

---

*最后更新: 2026-03-20*
