# 开发指南

本文档介绍如何参与 db-backup 项目的开发。

## 禂述

db-backup 是一个企业级多源数据库备份系统，使用 Go 语言开发。

## 技术栈

| 组件 | 技术 | 版本 |
|------|------|------|
| 语言 | Go | 1.21+ |
| Web 框架 | Gin | 1.9+ |
| ORM | GORM | 1.25+ |
| 调度 | robfig/cron | 3.0+ |
| WebSocket | gorilla/websocket | 1.5+ |
| 配置 | YAML | - |

## 项目结构

```
db-backup/
├── cmd/
│   └── server/           # 入口程序
├── internal/
│   ├── api/              # HTTP API
│   ├── backup/           # 备份引擎
│   ├── config/           # 配置管理
│   ├── crypto/           # 加密模块
│   ├── executor/         # 数据库执行器
│   ├── handler/          # API Handler
│   ├── logger/           # 日志模块
│   ├── metrics/          # Prometheus 挌标
│   ├── model/            # 数据模型
│   ├── notify/           # 通知模块
│   ├── restore/          # 恢复模块
│   ├── retention/        # 保留策略
│   ├── router/           # 路由配置
│   ├── scheduler/        # 任务调度
│   ├── storage/          # 存储后端
│   ├── verify/           # 备份验证
│   └── ws/               # WebSocket
├── configs/              # 配置文件
├── tests/                # 集成测试
└── docs/                 # 文档
```

## 开发环境设置

### 1. 安装 Go

```bash
# macOS
brew install go

# Ubuntu/Debian
sudo apt install golang-go

# 从源码安装
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

### 2. 克隆项目

```bash
git clone https://github.com/imysm/db-backup.git
cd db-backup
```

### 3. 安装依赖

```bash
go mod download
```

### 4. 运行开发服务器

```bash
go run ./cmd/server -config configs/config.yaml
```

## 开发工作流

### 创建新功能

1. 创建分支
```bash
git checkout -b feat/your-feature
```

2. 编写代码
3. 编写测试
4. 运行测试
```bash
go test ./...
```

5. 提交代码
```bash
git add .
git commit -m "feat: 添加新功能"
```

### 代码规范

#### 命名规范

| 类型 | 规范 | 示例 |
|------|------|------|
| 包名 | 小写单词 | `backup` |
| 导出函数 | 大驼峰 | `NewMySQLBackup` |
| 私有函数 | 小驼峰 | `buildArgs` |
| 常量 | 大驼峰 | `DefaultTimeout` |
| 接口 | 名词+er | `Executor` |

#### 错误处理

```go
// 推荐
if err != nil {
    return fmt.Errorf("操作失败: %w", err)
}

// 不推荐
if err != nil {
    panic(err)
}
```

#### Context 传播

```go
// 推荐：始终传递 context
func (e *Executor) Backup(ctx context.Context, task *Task) error {
    // 使用 ctx 进行超时控制
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        // 执行备份
    }
}
```

## 测试

### 单元测试

```bash
# 运行所有测试
go test ./...

# 运行特定包测试
go test ./internal/backup/...

# 查看覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 测试规范

```go
func TestMySQLBackup_FullBackup(t *testing.T) {
    // Arrange
    job := &model.BackupJob{
        Host:     "localhost",
        Port:     3306,
        Database: "testdb",
    }
    engine, err := backup.NewMySQLBackup(job)
    if err != nil {
        t.Fatalf("创建引擎失败: %v", err)
    }

    // Act
    ctx := context.Background()
    result, err := engine.FullBackup(ctx, job)

    // Assert
    if err != nil {
        t.Errorf("备份失败: %v", err)
    }
    if result == nil {
        t.Error("结果不应为 nil")
    }
}
```

## 构建与部署

### 构建

```bash
# 本地构建
go build -o db-backup ./cmd/server

# 跨平台构建
GOOS=linux GOARCH=amd64 go build -o db-backup-linux ./cmd/server
GOOS=darwin GOARCH=amd64 go build -o db-backup-darwin ./cmd/server
```

### Docker 部署

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o db-backup ./cmd/server

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/db-backup .
COPY configs/config.yaml /app/configs/
ENTRYPOINT ["./db-backup"]
CMD ["-config", "configs/config.yaml"]
```

## 调试

### 日志级别

```yaml
log:
  level: debug  # 设置为 debug 查看详细日志
```

### pprof 性能分析

```bash
# 启用 pprof
go run ./cmd/server -config configs/config.yaml &

# 查看 pprof
go tool pprof http://localhost:6060/debug/pprof/profile
```

## 常见问题

### Q: 测试时数据库不可用怎么办？

使用 mock 或跳过测试：

```go
func TestMySQLBackup(t *testing.T) {
    if testing.Short() {
        t.Skip("跳过需要数据库的测试")
    }
    // 测试代码
}
```

运行：`go test -short ./...`

### Q: 如何添加新的数据库支持？

1. 在 `internal/executor/` 创建新文件
2. 实现 `Executor` 接口
3. 在 `internal/backup/` 注册新执行器
4. 添加测试

### Q: 如何添加新的存储后端？

1. 在 `internal/storage/` 创建新文件
2. 实现 `Storage` 接口
3. 在配置中添加新存储类型
