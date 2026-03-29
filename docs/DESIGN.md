# 数据库备份系统设计方案

## 一、项目概述

### 1.1 系统定位
企业级多源数据库备份系统，支持 MySQL、PostgreSQL、SQL Server、Oracle 四种主流数据库的逻辑/物理备份。

### 1.2 核心特性
- **多数据库支持**: MySQL、PostgreSQL、SQL Server、Oracle
- **Web 管理控制台**: 任务配置、实时日志、历史记录
- **灵活调度**: Cron 表达式、热重载
- **多种存储**: 本地存储、S3/OSS
- **保留策略**: 按天数/个数/周期自动清理
- **通知告警**: 钉钉、企业微信、飞书、Webhook
- **安全加密**: 密码加密存储、文件加密备份

---

## 二、系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                      Browser (Vue3 SPA)                      │
└─────────────────────────┬───────────────────────────────────┘
                          │ HTTP/WebSocket
┌─────────────────────────┴───────────────────────────────────┐
│                    Go Backend Server                         │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐    │
│  │ API Layer│ │WebSocket │ │Scheduler │ │   Service    │    │
│  │  (Gin)   │ │   Hub    │ │  (Cron)  │ │    Layer     │    │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └──────┬───────┘    │
│       │            │            │              │             │
│  ┌────┴────────────┴────────────┴──────────────┴───────┐    │
│  │                   Executor Layer                      │    │
│  │  ┌────────┐ ┌──────────┐ ┌───────────┐ ┌─────────┐  │    │
│  │  │ MySQL  │ │PostgreSQL│ │SQL Server │ │ Oracle  │  │    │
│  │  └────────┘ └──────────┘ └───────────┘ └─────────┘  │    │
│  └───────────────────────────────────────────────────────┘    │
└───────────────────────────┬───────────────────────────────────┘
                            │
          ┌─────────────────┼─────────────────┐
          │                 │                 │
    ┌─────┴─────┐     ┌─────┴─────┐     ┌─────┴─────┐
    │  SQLite   │     │  Storage  │     │   Redis   │
    │ /PostgreSQL│     │ Local/S3  │     │  (Lock)   │
    └───────────┘     └───────────┘     └───────────┘
```

---

## 三、技术栈

| 层级 | 技术选型 | 说明 |
|------|----------|------|
| 后端框架 | Go 1.21+ / Gin | 高性能、简洁 |
| ORM | GORM | 成熟的 Go ORM |
| 调度 | robfig/cron | Cron 表达式调度 |
| WebSocket | gorilla/websocket | 实时日志推送 |
| 前端 | Vue3 + Element Plus | 企业级 UI |
| 数据库 | SQLite / PostgreSQL | 元数据存储 |
| 缓存 | Redis | 分布式锁、会话 |

---

## 四、开发计划

### Phase 1: MVP 核心闭环 ✅
**目标**: 能通过代码配置 MySQL 备份并成功执行

- [x] 项目结构搭建
- [x] 数据模型定义
- [x] 配置管理
- [x] MySQL 执行器
- [x] PostgreSQL 执行器
- [x] MongoDB 执行器
- [x] SQL Server 执行器
- [x] Oracle 执行器
- [x] 本地存储
- [x] S3/OSS/COS 存储
- [x] 简单调度器
- [x] 单元测试 (backup: 61.7%)

### Phase 2: 生产可用 ✅
- [x] PostgreSQL/SQL Server/Oracle 执行器
- [x] WebSocket 实时日志
- [x] S3/OSS/COS 存储
- [x] 保留策略
- [x] 密码加密 (AES-256-GCM)
- [x] 压缩支持 (gzip)

### Phase 3: 企业级体验 ✅
- [x] Web API (Gin)
- [x] Vue3 前端
- [x] Prometheus 监控
- [x] Context 传播与取消

### Phase 4: 高阶功能 (待开发)
- [ ] 多线程备份 (mydumper)
- [ ] 自动恢复校验
- [ ] Agent 模式
- [ ] Redis 分布式锁

---

## 五、项目结构

```
db-backup/
├── cmd/
│   └── server/
│       └── main.go              # 入口
├── internal/
│   ├── config/                  # 配置管理
│   ├── model/                   # 数据模型
│   ├── executor/                # 备份执行器
│   │   ├── executor.go          # 接口定义
│   │   ├── mysql.go             # MySQL
│   │   ├── postgres.go          # PostgreSQL
│   │   ├── sqlserver.go         # SQL Server
│   │   └── oracle.go            # Oracle
│   ├── storage/                 # 存储管理
│   ├── retention/               # 保留策略
│   ├── scheduler/               # 任务调度
│   ├── notify/                  # 通知模块
│   ├── api/                     # HTTP API
│   ├── ws/                      # WebSocket
│   ├── middleware/              # 中间件
│   ├── repository/              # 数据访问
│   └── service/                 # 业务逻辑
├── configs/
│   └── config.yaml              # 配置文件
├── tests/                       # 测试
├── docs/                        # 文档
└── go.mod
```

---

## 六、核心接口设计

### Executor 接口
```go
type Executor interface {
    Backup(ctx context.Context, task *TaskConfig, writer LogWriter) (*BackupResult, error)
    Validate(ctx context.Context, cfg *DatabaseConfig) error
    Type() string
}
```

### Storage 接口
```go
type Storage interface {
    Save(ctx context.Context, localPath, remotePath string) error
    List(ctx context.Context, prefix string) ([]BackupRecord, error)
    Delete(ctx context.Context, filePath string) error
    Type() string
}
```

---

*文档创建时间: 2026-03-18*
*最后更新: 2026-03-20*
