package auth

import (
	"sync"
	"time"
)

// LoginAttempt 登录尝试记录
type LoginAttempt struct {
	Count       int
	FirstAt     time.Time
	LockedUntil time.Time // 锁定截止时间
}

// LoginRateLimiter 登录频率限制器
type LoginRateLimiter struct {
	mu               sync.Mutex
	attempts         map[string]*LoginAttempt
	maxAttempts      int
	lockoutWindow    time.Duration // 锁定时间窗口
	lockoutDuration  time.Duration // 锁定持续时间
	cleanupInterval  time.Duration
	stopCleanup      chan struct{}
	stopWg           sync.WaitGroup // 等待 cleanup goroutine 退出
}

// DefaultLoginRateLimiter 默认的登录限制器实例
var DefaultLoginRateLimiter = NewLoginRateLimiter(5, 15*time.Minute, 15*time.Minute)

// NewLoginRateLimiter 创建登录频率限制器
// maxAttempts: 窗口期内的最大失败尝试次数
// lockoutWindow: 统计失败尝试的时间窗口
// lockoutDuration: 锁定持续时间
func NewLoginRateLimiter(maxAttempts int, lockoutWindow, lockoutDuration time.Duration) *LoginRateLimiter {
	l := &LoginRateLimiter{
		attempts:        make(map[string]*LoginAttempt),
		maxAttempts:     maxAttempts,
		lockoutWindow:   lockoutWindow,
		lockoutDuration: lockoutDuration,
		cleanupInterval: lockoutWindow,
		stopCleanup:     make(chan struct{}),
	}

	// 启动定期清理过期记录
	go l.cleanupLoop()

	return l
}

// IsLocked 检查指定用户是否被锁定
func (l *LoginRateLimiter) IsLocked(username string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	attempt, ok := l.attempts[username]
	if !ok {
		return false
	}

	now := time.Now()

	// 检查是否在锁定期间
	if now.Before(attempt.LockedUntil) {
		return true
	}

	// 锁定已过期，清除记录
	if now.After(attempt.LockedUntil.Add(l.lockoutWindow)) {
		delete(l.attempts, username)
	}

	return false
}

// RecordFailure 记录一次失败的登录尝试
// 返回是否已被锁定
func (l *LoginRateLimiter) RecordFailure(username string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	attempt, ok := l.attempts[username]

	if !ok {
		l.attempts[username] = &LoginAttempt{
			Count:   1,
			FirstAt: now,
		}
		return false
	}

	// 如果之前已锁定但已过期（LockedUntil 非零且已过），重置
	if !attempt.LockedUntil.IsZero() && now.After(attempt.LockedUntil) {
		attempt.Count = 1
		attempt.FirstAt = now
		attempt.LockedUntil = time.Time{}
		return false
	}

	// 在锁定期间
	if now.Before(attempt.LockedUntil) {
		return true
	}

	// 重置计数（窗口期已过）
	if now.After(attempt.FirstAt.Add(l.lockoutWindow)) {
		attempt.Count = 1
		attempt.FirstAt = now
		attempt.LockedUntil = time.Time{}
		return false
	}

	// 在窗口期内增加计数
	attempt.Count++

	// 达到最大尝试次数，实施锁定
	if attempt.Count >= l.maxAttempts {
		attempt.LockedUntil = now.Add(l.lockoutDuration)
		return true
	}

	return false
}

// RecordSuccess 记录一次成功的登录，清除失败记录
func (l *LoginRateLimiter) RecordSuccess(username string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, username)
}

// GetRemainingAttempts 获取剩余尝试次数
func (l *LoginRateLimiter) GetRemainingAttempts(username string) int {
	l.mu.Lock()
	defer l.mu.Unlock()

	attempt, ok := l.attempts[username]
	if !ok {
		return l.maxAttempts
	}

	now := time.Now()
	if now.Before(attempt.LockedUntil) {
		return 0
	}
	if now.After(attempt.FirstAt.Add(l.lockoutWindow)) {
		return l.maxAttempts
	}

	remaining := l.maxAttempts - attempt.Count
	if remaining < 0 {
		remaining = 0
	}
	return remaining
}

// cleanupLoop 定期清理过期记录
func (l *LoginRateLimiter) cleanupLoop() {
	l.stopWg.Add(1)
	defer l.stopWg.Done()

	ticker := time.NewTicker(l.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.cleanup()
		case <-l.stopCleanup:
			return
		}
	}
}

// cleanup 清理过期记录
func (l *LoginRateLimiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	for username, attempt := range l.attempts {
		// 清理超过窗口期的记录
		if now.After(attempt.FirstAt.Add(l.lockoutWindow)) {
			delete(l.attempts, username)
		}
	}
}

// Stop 停止清理goroutine
func (l *LoginRateLimiter) Stop() {
	close(l.stopCleanup)
	l.stopWg.Wait() // 等待 cleanup goroutine 退出
}
