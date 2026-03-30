package executor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// ScriptResult 脚本执行结果
type ScriptResult struct {
	Output    string
	Duration  time.Duration
	ExitCode  int
	Error     error
}

// ExecuteScript 执行脚本
func ExecuteScript(ctx context.Context, script string, timeout time.Duration) (*ScriptResult, error) {
	if script == "" {
		return nil, nil
	}

	result := &ScriptResult{}
	startTime := time.Now()

	// 使用 bash -c 执行脚本
	cmd := exec.CommandContext(ctx, "bash", "-c", script)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 设置超时
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	err := cmd.Run()
	result.Duration = time.Since(startTime)
	result.Output = stdout.String()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			result.Error = fmt.Errorf("脚本执行失败，退出码: %d, stderr: %s", result.ExitCode, stderr.String())
		} else {
			result.Error = err
		}
	}

	return result, result.Error
}

// ExecutePreScript 执行备份前脚本
func ExecutePreScript(ctx context.Context, script string) (*ScriptResult, error) {
	return ExecuteScript(ctx, script, 5*time.Minute)
}

// ExecutePostScript 执行备份后脚本
func ExecutePostScript(ctx context.Context, script string) (*ScriptResult, error) {
	return ExecuteScript(ctx, script, 5*time.Minute)
}
