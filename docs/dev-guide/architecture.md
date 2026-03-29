# 架构设计

本文档介绍 db-backup 系统的架构设计和数据流。

## 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                      用户接口层                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │  Web UI  │  │   CLI    │  │ REST API │  │WebSocket │   │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘   │
└───────┼─────────────┼─────────────┼─────────────┼─────────┘
        │             │             │             │
┌───────┴─────────────┴─────────────┴─────────────┴─────────┐
│                      服务层                                 │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  │
│  │Scheduler │  │ Executor │  │ Storage  │  │  Notify  │  │
│  │ (调度)   │  │ (执行)   │  │ (存储)   │  │ (通知)   │  │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘  │
└───────┼─────────────┼─────────────┼─────────────┼─────────┘
        │             │             │             │
┌───────┴─────────────┴─────────────┴─────────────┴─────────┐
│                      基础设施层                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  │
│  │ Database │  │   Log    │  │ Metrics  │  │  Config  │  │
│  │ (元数据) │  │  (日志)  │  │  (监控)  │  │  (配置)  │  │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘  │
└───────────────────────────────────────────────────────────┘
```

---

## 核心组件

### 1. Scheduler (调度器)

负责定时触发备份任务。

```go
type Scheduler struct {
    cron    *cron.Cron
    tasks   map[string]*Task
    running map[string]bool
}
```

**功能**:
- Cron 表达式解析
- 任务队列管理
- 并发控制

### 2. Executor (执行器)

负责执行实际的备份操作。

```go
type Executor interface {
    Backup(ctx context.Context, task *Task, writer LogWriter) (*Result, error)
    Restore(ctx context.Context, task *RestoreTask, writer LogWriter) error
    Validate(ctx context.Context) error
    Type() string
}
```

**实现**:
- MySQLExecutor
- PostgreSQLExecutor
- MongoDBExecutor
- SQLServerExecutor
- OracleExecutor

### 3. Storage (存储)

负责备份文件的存储和管理。

```go
type Storage interface {
    Save(ctx context.Context, localPath, remotePath string) error
    List(ctx context.Context, prefix string) ([]BackupRecord, error)
    Delete(ctx context.Context, path string) error
    Type() string
}
```

**实现**:
- LocalStorage
- S3Storage
- OSSStorage
- COSStorage

### 4. Notify (通知)

负责发送备份通知。

```go
type Notifier interface {
    Send(ctx context.Context, notification *Notification) error
    Type() string
}
```

**实现**:
- DingTalkNotifier
- FeishuNotifier
- WechatNotifier
- EmailNotifier
- WebhookNotifier

---

## 数据流

### 备份流程

```
┌─────────┐     ┌──────────┐     ┌──────────┐
│Scheduler│────▶│ Executor │────▶│ Database │
└─────────┘     └────┬─────┘     └──────────┘
                     │
                     ▼
               ┌──────────┐
               │   Dump   │
               └────┬─────┘
                    │
     ┌──────────────┼──────────────┐
     │              │              │
     ▼              ▼              ▼
┌─────────┐  ┌──────────┐  ┌──────────┐
│Compress │  │  Encrypt │  │  Verify  │
└────┬────┘  └────┬─────┘  └────┬─────┘
     │            │              │
     └────────────┼──────────────┘
                  │
                  ▼
            ┌──────────┐
            │ Storage  │
            └────┬─────┘
                 │
     ┌───────────┼───────────┐
     │           │           │
     ▼           ▼           ▼
┌────────┐ ┌────────┐ ┌────────┐
│ Local  │ │   S3   │ │  OSS   │
└────────┘ └────────┘ └────────┘
```

### 恢复流程

```
┌─────────┐     ┌──────────┐     ┌──────────┐
│   API   │────▶│ Storage  │────▶│  Backup  │
└─────────┘     └────┬─────┘     │  File    │
                     │           └──────────┘
                     ▼
               ┌──────────┐
               │ Download │
               └────┬─────┘
                    │
     ┌──────────────┼──────────────┐
     │              │              │
     ▼              ▼              ▼
┌─────────┐  ┌──────────┐  ┌──────────┐
│Decrypt  │  │ Decompress│  │  Verify  │
└────┬────┘  └────┬─────┘  └────┬─────┘
     │            │              │
     └────────────┼──────────────┘
                  │
                  ▼
            ┌──────────┐
            │ Restore  │
            └────┬─────┘
                 │
                 ▼
            ┌──────────┐
            │ Database │
            └──────────┘
```

---

## 目录结构

```
internal/
├── api/                    # HTTP API 层
│   ├── handler/            # 请求处理器
│   │   ├── job.go          # 任务管理
│   │   ├── record.go       # 备份记录
│   │   ├── restore.go      # 恢复操作
│   │   └── websocket.go    # WebSocket
│   ├── model/              # API 模型
│   ├── router/             # 路由配置
│   └── service/            # 业务服务
├── backup/                 # 备份引擎
│   ├── mysql.go            # MySQL 备份
│   ├── postgres.go         # PostgreSQL 备份
│   └── mongodb.go          # MongoDB 备份
├── config/                 # 配置管理
├── crypto/                 # 加密模块
├── executor/               # 数据库执行器
├── logger/                 # 日志模块
├── metrics/                # Prometheus 指标
├── model/                  # 数据模型
├── notify/                 # 通知模块
├── restore/                # 恢复模块
├── retention/              # 保留策略
├── scheduler/              # 任务调度
├── storage/                # 存储后端
├── verify/                 # 备份验证
└── ws/                     # WebSocket
```

---

## 接口设计

### Executor 接口

```go
type Executor interface {
    // Backup 执行备份
    Backup(ctx context.Context, task *BackupTask, writer LogWriter) (*BackupResult, error)
    
    // Restore 执行恢复
    Restore(ctx context.Context, task *RestoreTask, writer LogWriter) error
    
    // Validate 验证连接
    Validate(ctx context.Context) error
    
    // Type 返回执行器类型
    Type() string
}
```

### Storage 接口

```go
type Storage interface {
    // Save 保存文件
    Save(ctx context.Context, localPath, remotePath string) error
    
    // List 列出文件
    List(ctx context.Context, prefix string) ([]BackupRecord, error)
    
    // Delete 删除文件
    Delete(ctx context.Context, path string) error
    
    // Type 返回存储类型
    Type() string
}
```

---

## 扩展点

### 添加新数据库支持

1. 实现 `Executor` 接口
2. 在 `executor/` 目录创建新文件
3. 注册到执行器工厂

### 添加新存储后端

1. 实现 `Storage` 接口
2. 在 `storage/` 目录创建新文件
3. 注册到存储工厂

### 添加新通知渠道

1. 实现 `Notifier` 接口
2. 在 `notify/` 目录创建新文件
3. 注册到通知工厂

---

*最后更新: 2026-03-20*
