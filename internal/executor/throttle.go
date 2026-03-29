package executor

import (
	"context"
	"io"
	"sync"
	"time"
)

// ThrottleConfig 限流配置
type ThrottleConfig struct {
	MaxBytesPerSecond int           // 最大字节/秒 (0 = 不限流)
	WindowSize        time.Duration // 统计窗口大小
}

// ThrottleManager 备份限流管理器
type ThrottleManager struct {
	config     *ThrottleConfig
	mu         sync.Mutex
	currentQ   int64
	windowStart time.Time
}

// NewThrottleManager 创建限流管理器
func NewThrottleManager(config *ThrottleConfig) *ThrottleManager {
	if config == nil {
		config = &ThrottleConfig{
			MaxBytesPerSecond: 0, // 默认不限流
			WindowSize:        1 * time.Second,
		}
	}

	return &ThrottleManager{
		config:     config,
		windowStart: time.Now(),
	}
}

// Wait 限速等待
func (m *ThrottleManager) Wait(ctx context.Context, bytes int64) error {
	if m.config.MaxBytesPerSecond <= 0 {
		return nil // 不限流
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	elapsed := time.Since(m.windowStart)
	if elapsed >= m.config.WindowSize {
		// 重置窗口
		m.currentQ = 0
		m.windowStart = time.Now()
	}

	// 检查是否超限
	targetQ := m.currentQ + bytes
	if targetQ <= int64(m.config.MaxBytesPerSecond) {
		m.currentQ = targetQ
		return nil
	}

	// 需要等待
	waitTime := m.config.WindowSize - elapsed
	if waitTime > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			m.currentQ = bytes
			m.windowStart = time.Now()
		}
	}

	return nil
}

// SetLimit 动态调整限流
func (m *ThrottleManager) SetLimit(bytesPerSecond int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.MaxBytesPerSecond = bytesPerSecond
}

// GetLimit 获取当前限流值
func (m *ThrottleManager) GetLimit() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.config.MaxBytesPerSecond
}

// WriteCounter 计数写入器（用于限流）
type WriteCounter struct {
	w        io.Writer
	throttle *ThrottleManager
	ctx      context.Context
	total    int64
}

// NewWriteCounter 创建计数写入器
func NewWriteCounter(w io.Writer, throttle *ThrottleManager, ctx context.Context) *WriteCounter {
	return &WriteCounter{
		w:        w,
		throttle: throttle,
		ctx:      ctx,
	}
}

// Write 限流写入
func (wc *WriteCounter) Write(p []byte) (n int, err error) {
	if err := wc.throttle.Wait(wc.ctx, int64(len(p))); err != nil {
		return 0, err
	}
	n, err = wc.w.Write(p)
	wc.total += int64(n)
	return n, err
}

// Total 返回总写入字节数
func (wc *WriteCounter) Total() int64 {
	return wc.total
}

// GlobalThrottle 全局限流管理器
var globalThrottle = NewThrottleManager(nil)

// SetGlobalThrottle 设置全局限流
func SetGlobalThrottle(bytesPerSecond int) {
	globalThrottle.SetLimit(bytesPerSecond)
}

// GetGlobalThrottle 获取全局限流
func GetGlobalThrottle() int {
	return globalThrottle.GetLimit()
}
