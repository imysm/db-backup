// Package executor 提供数据库备份执行器接口和实现
package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/imysm/db-backup/internal/model"
)

// Executor 备份执行器接口
type Executor interface {
	// Backup 执行备份，返回备份结果
	Backup(ctx context.Context, task *model.BackupTask, writer model.LogWriter) (*model.BackupResult, error)

	// Validate 验证数据库连接
	Validate(ctx context.Context, cfg *model.DatabaseConfig) error

	// Type 返回执行器类型
	Type() string
}

// NewExecutor 根据数据库类型创建执行器
func NewExecutor(dbType model.DatabaseType) (Executor, error) {
	switch dbType {
	case model.MySQL:
		return NewMySQLExecutor(), nil
	case model.PostgreSQL:
		return NewPostgresExecutor(), nil
	case model.MongoDB:
		return NewMongoDBExecutor(), nil
	case model.SQLServer:
		return NewSQLServerExecutor(), nil
	case model.Oracle:
		return NewOracleExecutor(), nil
	default:
		return nil, fmt.Errorf("不支持的数据库类型: %s", dbType)
	}
}

// NewExecutorWithDeps 创建带依赖的执行器（用于测试）
func NewExecutorWithDeps(dbType model.DatabaseType, cmdRunner CommandRunner) (Executor, error) {
	switch dbType {
	case model.MySQL:
		return &MySQLExecutor{cmdRunner: cmdRunner}, nil
	case model.PostgreSQL:
		return &PostgresExecutor{cmdRunner: cmdRunner}, nil
	case model.MongoDB:
		return &MongoDBExecutor{cmdRunner: cmdRunner}, nil
	case model.SQLServer:
		return &SQLServerExecutor{cmdRunner: cmdRunner}, nil
	case model.Oracle:
		return &OracleExecutor{cmdRunner: cmdRunner}, nil
	default:
		return nil, fmt.Errorf("不支持的数据库类型: %s", dbType)
	}
}

// CommandRunner 命令执行接口（用于测试 Mock）
type CommandRunner interface {
	Run(ctx context.Context, name string, args []string, env []string) ([]byte, error)
	RunWithOutput(ctx context.Context, name string, args []string, env []string, writer model.LogWriter) error
}

// DefaultCommandRunner 默认命令执行器
type DefaultCommandRunner struct{}

// Run 执行命令并返回输出
func (r *DefaultCommandRunner) Run(ctx context.Context, name string, args []string, env []string) ([]byte, error) {
	return runCommand(ctx, name, args, env)
}

// RunWithOutput 执行命令并实时输出
func (r *DefaultCommandRunner) RunWithOutput(ctx context.Context, name string, args []string, env []string, writer model.LogWriter) error {
	return runCommandWithOutput(ctx, name, args, env, writer)
}

// failResult 创建失败结果
func failResult(taskID, traceID string, startTime time.Time, err error) *model.BackupResult {
	endTime := time.Now()
	return &model.BackupResult{
		TaskID:    taskID,
		TraceID:   traceID,
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  endTime.Sub(startTime),
		Status:    model.TaskStatusFailed,
		Error:     err.Error(),
	}
}
