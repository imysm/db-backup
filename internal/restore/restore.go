package restore

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/imysm/db-backup/internal/executor"
	"github.com/imysm/db-backup/internal/util"
)

// buildMinimalEnv 构建最小化环境变量，只继承 PATH 和 LANG，加上额外的环境变量
func buildMinimalEnv(extraEnv []string) []string {
	env := []string{"PATH=" + os.Getenv("PATH")}
	if lang := os.Getenv("LANG"); lang != "" {
		env = append(env, "LANG="+lang)
	}
	env = append(env, extraEnv...)
	return env
}

// Type 恢复类型
type Type string

const (
	TypeLocal  Type = "local"  // 本机恢复
	TypeRemote Type = "remote" // 异机恢复
)

// RestoreRequest 恢复请求
type RestoreRequest struct {
	BackupFile   string // 备份文件路径或存储key
	TargetHost   string // 目标主机 (异机恢复时使用)
	TargetPort   int    // 目标端口
	TargetDB     string // 目标数据库
	TargetUser   string // 目标用户
	TargetPass   string // 目标密码
	DBType       string // 数据库类型: postgres, mysql, mongodb, sqlserver
	StorageType  string // 存储类型: local, s3, oss, cos
	NoRestore    bool   // 只校验不恢复
	EncryptKey   string // 加密密钥（hex格式），用于解密加密的备份文件
	RecoveryMode string // 恢复模式: recovery, norecovery, standby (SQL Server 链式恢复用)
	Replace      bool   // 是否使用 REPLACE 选项覆盖已有数据库 (SQL Server)
}

// RestoreResult 恢复结果
type RestoreResult struct {
	Success      bool
	StartTime    string
	EndTime      string
	Duration     int // 秒
	RowsAffected int64
	Error        string
}

// Restorer 恢复器接口
type Restorer interface {
	Restore(ctx context.Context, req *RestoreRequest) (*RestoreResult, error)
	Validate(ctx context.Context, backupFile string) error
}

// PostgresRestorer PostgreSQL 恢复器
type PostgresRestorer struct {
	AllowedDir string // 允许的备份文件目录
}

// NewPostgresRestorer 创建 PostgreSQL 恢复器
func NewPostgresRestorer() *PostgresRestorer {
	return &PostgresRestorer{}
}

// NewPostgresRestorerWithDir 创建带路径限制的 PostgreSQL 恢复器
func NewPostgresRestorerWithDir(allowedDir string) *PostgresRestorer {
	return &PostgresRestorer{AllowedDir: allowedDir}
}

// Validate 验证备份文件
func (r *PostgresRestorer) Validate(ctx context.Context, backupFile string) error {
	// 检查文件是否存在
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		return fmt.Errorf("备份文件不存在: %s", backupFile)
	}

	// 检查文件大小
	info, err := os.Stat(backupFile)
	if err != nil {
		return err
	}
	if info.Size() == 0 {
		return fmt.Errorf("备份文件为空")
	}

	// 检查文件扩展名
	ext := filepath.Ext(backupFile)
	validExts := []string{".sql", ".dump", ".tar.gz", ".gz", ".backup"}
	isValid := false
	for _, validExt := range validExts {
		if ext == validExt {
			isValid = true
			break
		}
	}
	if !isValid {
		return fmt.Errorf("未知的备份文件格式: %s", ext)
	}

	return nil
}

// Restore 执行恢复
func (r *PostgresRestorer) Restore(ctx context.Context, req *RestoreRequest) (*RestoreResult, error) {
	result := &RestoreResult{}

	// 防御性路径验证（纵深防御）
	if r.AllowedDir != "" {
		if err := util.ValidateFilePath(req.BackupFile, r.AllowedDir); err != nil {
			result.Error = err.Error()
			return result, err
		}
	}

	// 验证备份文件
	if err := r.Validate(ctx, req.BackupFile); err != nil {
		result.Error = err.Error()
		return result, err
	}

	// 如果是只校验不恢复
	if req.NoRestore {
		result.Success = true
		return result, nil
	}

	// 构建 psql 命令
	cmd := exec.CommandContext(ctx, "psql",
		"-h", req.TargetHost,
		"-p", fmt.Sprintf("%d", req.TargetPort),
		"-U", req.TargetUser,
		"-d", req.TargetDB,
		"-f", req.BackupFile,
	)
	cmd.Env = buildMinimalEnv([]string{fmt.Sprintf("PGPASSWORD=%s", req.TargetPass)})

	// 执行恢复
	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Error = fmt.Sprintf("恢复失败: %v, output: %s", err, string(output))
		return result, err
	}

	result.Success = true
	return result, nil
}

// MySQLRestorer MySQL 恢复器
type MySQLRestorer struct {
	AllowedDir string
}

// NewMySQLRestorer 创建 MySQL 恢复器
func NewMySQLRestorer() *MySQLRestorer {
	return &MySQLRestorer{}
}

// NewMySQLRestorerWithDir 创建带路径限制的 MySQL 恢复器
func NewMySQLRestorerWithDir(allowedDir string) *MySQLRestorer {
	return &MySQLRestorer{AllowedDir: allowedDir}
}

// Validate 验证备份文件
func (r *MySQLRestorer) Validate(ctx context.Context, backupFile string) error {
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		return fmt.Errorf("备份文件不存在: %s", backupFile)
	}

	info, err := os.Stat(backupFile)
	if err != nil {
		return err
	}
	if info.Size() == 0 {
		return fmt.Errorf("备份文件为空")
	}

	return nil
}

// Restore 执行恢复
func (r *MySQLRestorer) Restore(ctx context.Context, req *RestoreRequest) (*RestoreResult, error) {
	result := &RestoreResult{}

	// 防御性路径验证（纵深防御）
	if r.AllowedDir != "" {
		if err := util.ValidateFilePath(req.BackupFile, r.AllowedDir); err != nil {
			result.Error = err.Error()
			return result, err
		}
	}

	if err := r.Validate(ctx, req.BackupFile); err != nil {
		result.Error = err.Error()
		return result, err
	}

	if req.NoRestore {
		result.Success = true
		return result, nil
	}

	// 处理加密备份文件
	backupFile := req.BackupFile
	cleanupDecrypted := func() {}
	if req.EncryptKey != "" {
		// 验证密钥
		if err := executor.ValidateEncryptionKey(req.EncryptKey); err != nil {
			result.Error = err.Error()
			return result, err
		}

		// 解密文件
		decryptedPath := backupFile + ".decrypted"
		if err := executor.DecryptFile(backupFile, decryptedPath, req.EncryptKey); err != nil {
			result.Error = fmt.Sprintf("解密备份文件失败: %v", err)
			return result, err
		}
		backupFile = decryptedPath
		cleanupDecrypted = func() { os.Remove(decryptedPath) }
		defer cleanupDecrypted()
	}

	// 构建 mysql 命令（使用 --defaults-extra-file 传递密码）
	defaultsFile, err := os.CreateTemp("", "mysql_defaults_*.cnf")
	if err != nil {
		result.Error = fmt.Sprintf("创建临时配置文件失败: %v", err)
		return result, err
	}
	defer os.Remove(defaultsFile.Name())

	if err := defaultsFile.Chmod(0600); err != nil {
		defaultsFile.Close()
		result.Error = fmt.Sprintf("设置配置文件权限失败: %v", err)
		return result, err
	}
	fmt.Fprintf(defaultsFile, "[client]\npassword=%s\n", req.TargetPass)
	defaultsFile.Close()

	cmd := exec.CommandContext(ctx, "mysql",
		fmt.Sprintf("--defaults-extra-file=%s", defaultsFile.Name()),
		"-h", req.TargetHost,
		"-P", fmt.Sprintf("%d", req.TargetPort),
		"-u", req.TargetUser,
		req.TargetDB,
	)
	cmd.Env = buildMinimalEnv(nil)

	// 从文件输入
	f, err := os.Open(backupFile)
	if err != nil {
		result.Error = err.Error()
		return result, err
	}
	defer f.Close()

	cmd.Stdin = f

	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Error = fmt.Sprintf("恢复失败: %v, output: %s", err, string(output))
		return result, err
	}

	result.Success = true
	return result, nil
}

// CommandRunner 命令运行器接口（用于测试注入）
type CommandRunner interface {
	RunCommand(ctx context.Context, name string, args []string, env []string) ([]byte, error)
}

// DefaultCommandRunner 默认命令运行器
type DefaultCommandRunner struct{}

func (r *DefaultCommandRunner) RunCommand(ctx context.Context, name string, args []string, env []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if len(env) > 0 {
		cmd.Env = env
	}
	return cmd.CombinedOutput()
}

// MongoRestorer MongoDB 恢复器
type MongoRestorer struct {
	AllowedDir string
	cmdRunner  CommandRunner
}

// NewMongoRestorer 创建 MongoDB 恢复器
func NewMongoRestorer() *MongoRestorer {
	return &MongoRestorer{cmdRunner: &DefaultCommandRunner{}}
}

// NewMongoRestorerWithDir 创建带路径限制的 MongoDB 恢复器
func NewMongoRestorerWithDir(allowedDir string) *MongoRestorer {
	return &MongoRestorer{AllowedDir: allowedDir, cmdRunner: &DefaultCommandRunner{}}
}

// Validate 验证备份文件
func (r *MongoRestorer) Validate(ctx context.Context, backupFile string) error {
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		return fmt.Errorf("备份文件不存在: %s", backupFile)
	}
	return nil
}

// buildRestoreArgs 构建 mongorestore 命令参数和环境变量
// 密码通过环境变量 MONGOPASSWORD 传递，不出现在命令行参数中
func (r *MongoRestorer) buildRestoreArgs(req *RestoreRequest) (args []string, env []string) {
	args = []string{
		"--host", req.TargetHost,
		"--port", fmt.Sprintf("%d", req.TargetPort),
		"--db", req.TargetDB,
		"--drop",
		req.BackupFile,
	}

	if req.TargetUser != "" {
		args = append(args,
			"--username", req.TargetUser,
			"--authenticationDatabase", "admin",
		)
	}

	if req.TargetPass != "" {
		env = append(env, fmt.Sprintf("MONGOPASSWORD=%s", req.TargetPass))
	}

	return args, env
}

// Restore 执行恢复
func (r *MongoRestorer) Restore(ctx context.Context, req *RestoreRequest) (*RestoreResult, error) {
	result := &RestoreResult{}

	// 防御性路径验证（纵深防御）
	if r.AllowedDir != "" {
		if err := util.ValidateFilePath(req.BackupFile, r.AllowedDir); err != nil {
			result.Error = err.Error()
			return result, err
		}
	}

	if err := r.Validate(ctx, req.BackupFile); err != nil {
		result.Error = err.Error()
		return result, err
	}

	if req.NoRestore {
		result.Success = true
		return result, nil
	}

	args, env := r.buildRestoreArgs(req)

	output, err := r.cmdRunner.RunCommand(ctx, "mongorestore", args, buildMinimalEnv(env))
	if err != nil {
		result.Error = fmt.Sprintf("恢复失败: %v, output: %s", err, string(output))
		return result, err
	}

	result.Success = true
	return result, nil
}

// GetRestorer 获取恢复器
func GetRestorer(dbType string) (Restorer, error) {
	switch dbType {
	case "postgres", "postgresql":
		return NewPostgresRestorer(), nil
	case "mysql":
		return NewMySQLRestorer(), nil
	case "mongodb", "mongo":
		return NewMongoRestorer(), nil
	default:
		return nil, fmt.Errorf("不支持的数据库类型: %s", dbType)
	}
}

// CopyFromStorage 从存储复制到本地
func CopyFromStorage(ctx context.Context, storageType string, storageKey string, localPath string, reader io.Reader) error {
	// 创建本地目录
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 写入本地文件
	f, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer f.Close()

	_, err = io.Copy(f, reader)
	return err
}
