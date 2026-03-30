package restore

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/imysm/db-backup/internal/executor"
)

// SQLServerRestorer SQL Server 恢复器
type SQLServerRestorer struct{}

// NewSQLServerRestorer 创建 SQL Server 恢复器
func NewSQLServerRestorer() *SQLServerRestorer {
	return &SQLServerRestorer{}
}

// Validate 验证备份文件
func (r *SQLServerRestorer) Validate(ctx context.Context, backupFile string) error {
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

	// 检查文件扩展名
	ext := strings.ToLower(filepath.Ext(backupFile))
	validExts := []string{".bak", ".trn", ".full", ".diff"}
	isValid := false
	for _, e := range validExts {
		if ext == e {
			isValid = true
			break
		}
	}
	if !isValid {
		fmt.Printf("警告: 未知的备份文件格式: %s\n", ext)
	}

	return nil
}

// Restore 执行恢复
func (r *SQLServerRestorer) Restore(ctx context.Context, req *RestoreRequest) (*RestoreResult, error) {
	result := &RestoreResult{}

	if err := r.Validate(ctx, req.BackupFile); err != nil {
		result.Error = err.Error()
		return result, err
	}

	if req.NoRestore {
		result.Success = true
		return result, nil
	}

	// 确定备份文件路径
	backupFile := req.BackupFile

	// 如果是加密文件，需要解密
	if req.EncryptKey != "" {
		if err := executor.ValidateEncryptionKey(req.EncryptKey); err != nil {
			result.Error = err.Error()
			return result, err
		}
		decryptedFile := backupFile + ".decrypted"
		if err := executor.DecryptFile(backupFile, decryptedFile, req.EncryptKey); err != nil {
			result.Error = fmt.Sprintf("解密失败: %v", err)
			return result, err
		}
		backupFile = decryptedFile
		defer os.Remove(backupFile)
	}

	// 构建 sqlcmd RESTORE 命令
	restoreSQL := r.buildRestoreSQL(req, backupFile)

	// 使用 sqlcmd 执行恢复
	cmd := exec.CommandContext(ctx, "sqlcmd",
		"-S", fmt.Sprintf("%s,%d", req.TargetHost, req.TargetPort),
		"-U", req.TargetUser,
		"-Q", restoreSQL,
	)
	cmd.Env = append(os.Environ(), "SQLCMDPASSWORD="+req.TargetPass)

	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Error = fmt.Sprintf("恢复失败: %v, output: %s", err, string(output))
		return result, err
	}

	result.Success = true
	return result, nil
}

// buildRestoreSQL 构建 RESTORE SQL 语句
func (r *SQLServerRestorer) buildRestoreSQL(req *RestoreRequest, backupFile string) string {
	// 转义单引号
	targetDB := strings.Replace(req.TargetDB, "'", "''", -1)
	backupFile = strings.Replace(backupFile, "'", "''", -1)
	// 处理文件路径中的反斜杠（Windows 风格）
	backupFile = strings.ReplaceAll(backupFile, "\\", "\\\\")

	var sb strings.Builder
	sb.WriteString("RESTORE DATABASE [")
	sb.WriteString(targetDB)
	sb.WriteString("] FROM DISK = N'")
	sb.WriteString(backupFile)
	sb.WriteString("' WITH ")

	// 根据 RecoveryMode 设置恢复选项
	switch strings.ToLower(req.RecoveryMode) {
	case "norecovery":
		sb.WriteString("NORECOVERY")
	case "standby":
		sb.WriteString("STANDBY = N'")
		sb.WriteString(targetDB)
		sb.WriteString("_undo.dat'")
	case "recovery":
		fallthrough
	default:
		sb.WriteString("RECOVERY")
	}

	// 根据 Replace 字段控制是否使用 REPLACE 选项
	if req.Replace {
		sb.WriteString(", REPLACE")
	}
	sb.WriteString("; SELECT 'Restore completed successfully' AS Result;")

	return sb.String()
}

// RestoreLog 恢复事务日志（链式恢复使用）
func (r *SQLServerRestorer) RestoreLog(ctx context.Context, req *RestoreRequest, logBackupFile string) (*RestoreResult, error) {
	result := &RestoreResult{}

	if err := r.Validate(ctx, logBackupFile); err != nil {
		result.Error = err.Error()
		return result, err
	}

	if req.NoRestore {
		result.Success = true
		return result, nil
	}

	var sb strings.Builder
	// 转义单引号
	targetDB := strings.Replace(req.TargetDB, "'", "''", -1)
	logBackupFileEscaped := strings.Replace(logBackupFile, "'", "''", -1)
	sb.WriteString("RESTORE LOG [")
	sb.WriteString(targetDB)
	sb.WriteString("] FROM DISK = N'")
	sb.WriteString(logBackupFileEscaped)
	sb.WriteString("' WITH ")

	switch strings.ToLower(req.RecoveryMode) {
	case "norecovery":
		sb.WriteString("NORECOVERY")
	case "standby":
		sb.WriteString("STANDBY = N'")
		sb.WriteString(targetDB)
		sb.WriteString("_undo.dat'")
	case "recovery":
		fallthrough
	default:
		sb.WriteString("RECOVERY")
	}

	sb.WriteString(";")

	cmd := exec.CommandContext(ctx, "sqlcmd",
		"-S", fmt.Sprintf("%s,%d", req.TargetHost, req.TargetPort),
		"-U", req.TargetUser,
		"-Q", sb.String(),
	)
	cmd.Env = append(os.Environ(), "SQLCMDPASSWORD="+req.TargetPass)

	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Error = fmt.Sprintf("恢复事务日志失败: %v, output: %s", err, string(output))
		return result, err
	}

	result.Success = true
	return result, nil
}
