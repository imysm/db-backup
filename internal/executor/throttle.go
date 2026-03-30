package executor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
)

// ThrottleController 限速控制器
type ThrottleController struct {
	semaphore   chan struct{} // 信号量用于并发控制
	cpuLimit    int           // CPU 限制百分比
	ioLimitMBPS int           // IO 限制 Mbps
	mu          sync.Mutex    // 保护 activeCount
	activeCount int           // 当前活跃的备份任务数
}

// NewThrottleController 创建限速控制器
func NewThrottleController(maxConcurrent, cpuLimit, ioLimitMBPS int) *ThrottleController {
	tc := &ThrottleController{
		semaphore:   make(chan struct{}, maxConcurrent),
		cpuLimit:    cpuLimit,
		ioLimitMBPS: ioLimitMBPS,
	}
	return tc
}

// Acquire 获取执行许可（阻塞直到可用）
func (tc *ThrottleController) Acquire() {
	tc.semaphore <- struct{}{}
	tc.mu.Lock()
	tc.activeCount++
	tc.mu.Unlock()
}

// Release 释放执行许可
func (tc *ThrottleController) Release() {
	tc.mu.Lock()
	tc.activeCount--
	tc.mu.Unlock()
	<-tc.semaphore
}

// ActiveCount 返回当前活跃的备份任务数
func (tc *ThrottleController) ActiveCount() int {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	return tc.activeCount
}

// BuildThrottleCmd 构建带有限速前缀的命令
// 例如: ionice -c 2 -n 7 mysqldump ...
func (tc *ThrottleController) BuildThrottleCmd(ctx context.Context, cmd string, args ...string) *exec.Cmd {
	if tc.ioLimitMBPS <= 0 {
		return exec.CommandContext(ctx, cmd, args...)
	}

	// 使用 ionice 包装命令
	return exec.CommandContext(ctx, "ionice", append([]string{"-c", "2", "-n", "7", cmd}, args...)...)
}

// ThrottleContext 返回一个带有进程级限速的上下文
func ThrottleContext(ctx context.Context, cpuLimitPercent, ioLimitMBPS int) (context.Context, error) {
	if cpuLimitPercent == 0 && ioLimitMBPS == 0 {
		return ctx, nil
	}

	pid := os.Getpid()

	// 设置 IO 调度类为最佳努力，优先级最低
	if ioLimitMBPS > 0 {
		if err := SetIOLimitClass(pid, 2, 7); err != nil {
			fmt.Printf("警告: 设置 IO 限速失败: %v\n", err)
		}
	}

	// 设置 nice 值
	if cpuLimitPercent > 0 && cpuLimitPercent < 100 {
		niceValue := 20 - (cpuLimitPercent * 20 / 100)
		if niceValue < -20 {
			niceValue = -20
		}
		if niceValue > 19 {
			niceValue = 19
		}

		if err := syscall.Setpriority(syscall.PRIO_PROCESS, pid, niceValue); err != nil {
			fmt.Printf("警告: 设置 CPU 限速失败: %v\n", err)
		}
	}

	return ctx, nil
}

// SetIOLimitClass 设置进程 IO 调度类
func SetIOLimitClass(pid int, class, priority int) error {
	cmd := exec.Command("ionice", "-p", strconv.Itoa(pid), "-c", strconv.Itoa(class), "-n", strconv.Itoa(priority))
	return cmd.Run()
}
