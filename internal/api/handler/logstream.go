package handler

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// LogStreamHandler SSE 日志流处理器
type LogStreamHandler struct {
	mu          sync.RWMutex
	subscribers map[string]chan string // taskID -> log channel
}

// NewLogStreamHandler 创建日志流处理器
func NewLogStreamHandler() *LogStreamHandler {
	return &LogStreamHandler{
		subscribers: make(map[string]chan string),
	}
}

// SubscribeLog 订阅任务日志 (SSE)
func (h *LogStreamHandler) SubscribeLog(c *gin.Context) {
	taskID := c.Query("task_id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task_id is required"})
		return
	}

	// 创建日志通道
	logCh := make(chan string, 100)
	h.register(taskID, logCh)
	defer h.unregister(taskID)

	// 设置 SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")
	c.Header("X-Accel-Buffering", "no") // 禁用 Nginx 缓冲

	// 发送连接成功事件
	c.SSEvent("connected", fmt.Sprintf(`{"task_id":"%s","status":"connected"}`, taskID))
	c.Writer.Flush()

	// 心跳 ticker
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	// 监听日志和心跳
	clientGone := c.Request.Context().Done()

	for {
		select {
		case <-clientGone:
			// 客户端断开
			c.SSEvent("disconnected", "client disconnected")
			c.Writer.Flush()
			return

		case log, ok := <-logCh:
			if !ok {
				// 通道关闭
				return
			}
			c.SSEvent("log", log)
			c.Writer.Flush()

		case <-heartbeat.C:
			// 发送心跳
			c.SSEvent("ping", fmt.Sprintf(`{"time":"%s"}`, time.Now().Format(time.RFC3339)))
			c.Writer.Flush()
		}
	}
}

// register 注册订阅者
func (h *LogStreamHandler) register(taskID string, ch chan string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.subscribers[taskID] = ch
}

// unregister 取消注册
func (h *LogStreamHandler) unregister(taskID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if ch, ok := h.subscribers[taskID]; ok {
		close(ch)
		delete(h.subscribers, taskID)
	}
}

// PublishLog 发布日志到指定任务
func (h *LogStreamHandler) PublishLog(taskID string, log string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if ch, ok := h.subscribers[taskID]; ok {
		select {
		case ch <- log:
		default:
			// 通道已满，跳过
		}
	}
}

// GetSubscriberCount 获取订阅者数量
func (h *LogStreamHandler) GetSubscriberCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subscribers)
}

// MockLogGenerator 模拟日志生成器（用于测试）
type MockLogGenerator struct {
	handler *LogStreamHandler
	taskID  string
	stopCh  chan struct{}
}

// NewMockLogGenerator 创建模拟日志生成器
func NewMockLogGenerator(handler *LogStreamHandler, taskID string) *MockLogGenerator {
	return &MockLogGenerator{
		handler: handler,
		taskID:  taskID,
		stopCh:  make(chan struct{}),
	}
}

// Start 开始生成模拟日志
func (g *MockLogGenerator) Start() {
	go func() {
		logTypes := []string{"INFO", "WARN", "ERROR"}
		messages := []string{
			"连接到数据库服务器...",
			"开始备份任务",
			"读取表结构...",
			"导出数据...",
			"压缩备份文件...",
			"计算校验和...",
			"备份完成",
		}

		for {
			select {
			case <-g.stopCh:
				return
			case <-time.After(time.Duration(500+rand.Intn(1000)) * time.Millisecond):
				logType := logTypes[rand.Intn(len(logTypes))]
				msg := messages[rand.Intn(len(messages))]
				log := fmt.Sprintf("[%s] %s - %s", time.Now().Format("15:04:05"), logType, msg)
				g.handler.PublishLog(g.taskID, log)
			}
		}
	}()
}

// Stop 停止生成日志
func (g *MockLogGenerator) Stop() {
	close(g.stopCh)
}

// StreamLog 流式日志（用于测试连接）
func (h *LogStreamHandler) StreamLog(c *gin.Context) {
	taskID := c.Query("task_id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task_id is required"})
		return
	}

	// 设置 SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	// 发送初始消息
	c.SSEvent("message", fmt.Sprintf("已连接到任务 %s 的日志流", taskID))
	c.Writer.Flush()

	// 保持连接
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return
	}

	// 发送一些测试日志
	testLogs := []string{
		"[23:19:00] INFO - 日志流已建立",
		"[23:19:01] INFO - 等待任务输出...",
	}
	for _, log := range testLogs {
		c.SSEvent("log", log)
		flusher.Flush()
		time.Sleep(100 * time.Millisecond)
	}
}


