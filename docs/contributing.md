# 开发规范 + 测试 + 提交

本文档介绍如何参与 db-backup 项目的开发。

## 开发环境设置

### 前置要求

- Go 1.21+
- Git
- Make (可选)

### 克隆项目

```bash
git clone https://github.com/imysm/db-backup.git
cd db-backup
go mod download
```

---

## 代码规范

### Go 代码规范

遵循 [Effective Go](https://golang.org/doc/effective_go) 和 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)。

### 命名规范

| 类型 | 规范 | 示例 |
|------|------|------|
| 包名 | 小写单词 | `backup`, `scheduler` |
| 导出函数 | 大驼峰 | `NewScheduler`, `GetJobByID` |
| 私有函数 | 小驼峰 | `buildArgs`, `validateConfig` |
| 常量 | 大驼峰或全大写 | `DefaultTimeout`, `MAX_RETRIES` |
| 接口 | 名词+er | `Executor`, `Storage`, `Notifier` |

### 注释规范

```go
// NewScheduler 创建一个新的调度器。
// cfg 是配置对象，不能为 nil。
// 返回调度器实例和可能的错误。
func NewScheduler(cfg *config.Config) (*Scheduler, error) {
    // ...
}
```

### 错误处理

```go
// 推荐：包装错误
if err != nil {
    return fmt.Errorf("执行备份失败: %w", err)
}

// 不推荐：忽略错误
if err != nil {
    log.Println(err)  // 错误！
}
```

### Context 传播

```go
// 推荐：始终传递 context
func (e *Executor) Backup(ctx context.Context, task *Task) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        // 执行备份
    }
}
```

---

## 测试规范

### 单元测试

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

### 表驱动测试

```go
func TestParseCronExpression(t *testing.T) {
    tests := []struct {
        name    string
        expr    string
        wantErr bool
    }{
        {"每天凌晨2点", "0 2 * * *", false},
        {"每小时", "0 * * * *", false},
        {"无效表达式", "invalid", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := ParseCron(tt.expr)
            if (err != nil) != tt.wantErr {
                t.Errorf("ParseCron() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定包
go test ./internal/backup/...

# 查看覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 运行基准测试
go test -bench=. ./...
```

### 测试覆盖率要求

| 模块 | 最低覆盖率 |
|------|-----------|
| 核心业务逻辑 | 70% |
| 工具函数 | 80% |
| API Handler | 60% |

---

## Git 提交规范

### 分支命名

| 类型 | 格式 | 示例 |
|------|------|------|
| 功能 | `feat/<描述>` | `feat/mongodb-support` |
| 修复 | `fix/<描述>` | `fix/connection-timeout` |
| 重构 | `refactor/<描述>` | `refactor/executor-interface` |
| 文档 | `docs/<描述>` | `docs/api-reference` |
| 测试 | `test/<描述>` | `test/backup-coverage` |

### Commit Message

格式：`<type>(<scope>): <subject>`

| Type | 说明 | 示例 |
|------|------|------|
| `feat` | 新功能 | `feat(backup): 添加 MongoDB 支持` |
| `fix` | Bug 修复 | `fix(scheduler): 修复时区问题` |
| `docs` | 文档 | `docs: 更新 API 文档` |
| `style` | 格式 | `style: 格式化代码` |
| `refactor` | 重构 | `refactor(executor): 重构接口` |
| `test` | 测试 | `test(backup): 添加单元测试` |
| `chore` | 杂项 | `chore: 更新依赖版本` |

### 提交示例

```bash
# 好的提交
git commit -m "feat(backup): 添加 MongoDB 增量备份支持"
git commit -m "fix(scheduler): 修复 Cron 表达式解析错误"
git commit -m "docs(api): 更新 REST API 文档"

# 不好的提交
git commit -m "fix bug"
git commit -m "update"
git commit -m "changes"
```

---

## Pull Request 流程

### 1. 创建分支

```bash
git checkout -b feat/your-feature
```

### 2. 开发并测试

```bash
# 编写代码
# 编写测试

# 运行测试
go test ./...

# 代码检查
go vet ./...
gofmt -w .
```

### 3. 提交

```bash
git add .
git commit -m "feat(xxx): 添加新功能"
git push origin feat/your-feature
```

### 4. 创建 PR

- 填写 PR 模板
- 关联 Issue
- 等待 Code Review

### PR 检查清单

- [ ] 代码通过 `go vet`
- [ ] 测试覆盖率达标
- [ ] Commit Message 规范
- [ ] 更新相关文档

---

## 代码审查

### 审查要点

- 逻辑正确性
- 代码可读性
- 错误处理
- 测试覆盖
- 性能影响
- 安全隐患

### 审查流程

1. 检查代码风格
2. 运行测试
3. 审查逻辑
4. 提出建议
5. 批准或请求修改

---

## 发布流程

### 版本号

遵循 [Semantic Versioning](https://semver.org/)

```
vMAJOR.MINOR.PATCH

v1.0.0 → v1.0.1 (Bug 修复)
v1.0.1 → v1.1.0 (新功能)
v1.1.0 → v2.0.0 (重大变更)
```

### 发布步骤

1. 更新版本号
2. 更新 CHANGELOG.md
3. 创建 tag
4. 构建 release

```bash
# 创建 tag
git tag v0.2.0
git push origin v0.2.0

# 构建
make release
```

---

## 目录结构

```
db-backup/
├── cmd/                    # 入口程序
│   └── server/
├── internal/               # 内部包
│   ├── api/               # HTTP API
│   ├── backup/            # 备份引擎
│   ├── config/            # 配置管理
│   ├── executor/          # 执行器
│   ├── storage/           # 存储
│   └── ...
├── configs/               # 配置文件
├── tests/                 # 集成测试
├── docs/                  # 文档
├── go.mod
├── go.sum
└── Makefile
```

---

## 常见问题

### Q: 如何跳过需要数据库的测试？

```bash
go test -short ./...
```

### Q: 如何调试测试？

```bash
go test -v -run TestName ./...
```

### Q: 如何查看测试覆盖率？

```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

---

*最后更新: 2026-03-20*
