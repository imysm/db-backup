// Package ws 提供 WebSocket 实时日志推送功能
package ws

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Client WebSocket 客户端
type Client struct {
	conn    *websocket.Conn
	taskID  string
	send    chan []byte
	hub     *Hub
	closeMu sync.Mutex
	closed  bool
}

// Hub WebSocket 连接管理中心
type Hub struct {
	clients       map[*Client]bool
	clientsMu     sync.RWMutex
	broadcast     chan *Message
	register      chan *Client
	unregister    chan *Client
	taskClients   map[string]map[*Client]bool // taskID -> clients
	taskClientsMu sync.RWMutex
	done          chan struct{}
	stopOnce      sync.Once
}

// Message 日志消息
type Message struct {
	TaskID string
	Data   []byte
}

// NewHub 创建 WebSocket Hub
func NewHub() *Hub {
	return &Hub{
		clients:     make(map[*Client]bool),
		broadcast:   make(chan *Message, 256),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		taskClients: make(map[string]map[*Client]bool),
		done:        make(chan struct{}),
	}
}

// upgrader WebSocket 升级器（占位，实际 upgrader 在 ServeWS 中通过 SetCheckOrigin 动态设置）
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return false }, // 默认拒绝，由 SetAllowedOrigins 覆盖
}

// SetAllowedOrigins 设置允许的 Origin 白名单
func SetAllowedOrigins(origins []string) {
	if len(origins) == 0 {
		// 空白名单：拒绝所有跨域请求
		upgrader.CheckOrigin = func(r *http.Request) bool {
			return r.Header.Get("Origin") == ""
		}
		return
	}
	allowed := make(map[string]bool, len(origins))
	for _, o := range origins {
		allowed[o] = true
	}
	upgrader.CheckOrigin = func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // 无 Origin header（非浏览器工具直接连接）允许
		}
		return allowed[origin]
	}
}

// ServeWS 处理 WebSocket 连接请求
func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	taskID := r.URL.Query().Get("task_id")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := NewClient(conn, hub, taskID)
	hub.Register(client)
	go client.WritePump()
	go client.ReadPump()
}

// Run 启动 Hub
func (h *Hub) Run() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-h.done:
			// 关闭所有客户端
			h.closeAllClients()
			return
		case client := <-h.register:
			h.registerClient(client)
		case client := <-h.unregister:
			h.unregisterClient(client)
		case message := <-h.broadcast:
			h.broadcastMessage(message)
		case <-ticker.C:
			h.pingAll()
		}
	}
}

// Stop 优雅停止 Hub，关闭所有客户端连接
func (h *Hub) Stop() {
	h.stopOnce.Do(func() {
		close(h.done)
	})
}

// closeAllClients 关闭所有已注册的客户端连接
func (h *Hub) closeAllClients() {
	h.clientsMu.Lock()
	for client := range h.clients {
		client.Close()
		close(client.send)
		delete(h.clients, client)
	}
	h.clientsMu.Unlock()
}

// registerClient 注册客户端
func (h *Hub) registerClient(client *Client) {
	h.clientsMu.Lock()
	h.clients[client] = true
	h.clientsMu.Unlock()

	if client.taskID != "" {
		h.taskClientsMu.Lock()
		if h.taskClients[client.taskID] == nil {
			h.taskClients[client.taskID] = make(map[*Client]bool)
		}
		h.taskClients[client.taskID][client] = true
		h.taskClientsMu.Unlock()
	}
}

// unregisterClient 注销客户端
func (h *Hub) unregisterClient(client *Client) {
	h.clientsMu.Lock()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
	}
	h.clientsMu.Unlock()

	if client.taskID != "" {
		h.taskClientsMu.Lock()
		if clients, ok := h.taskClients[client.taskID]; ok {
			delete(clients, client)
			if len(clients) == 0 {
				delete(h.taskClients, client.taskID)
			}
		}
		h.taskClientsMu.Unlock()
	}
}

// broadcastMessage 广播消息
func (h *Hub) broadcastMessage(message *Message) {
	if message.TaskID != "" {
		// 发送给特定任务的客户端
		h.taskClientsMu.RLock()
		clients, ok := h.taskClients[message.TaskID]
		if !ok {
			h.taskClientsMu.RUnlock()
			return
		}

		for client := range clients {
			select {
			case client.send <- message.Data:
			default:
				// 发送失败，关闭连接
				go client.Close()
			}
		}
		h.taskClientsMu.RUnlock()
	} else {
		// 广播给所有客户端
		h.clientsMu.RLock()
		for client := range h.clients {
			select {
			case client.send <- message.Data:
			default:
				go client.Close()
			}
		}
		h.clientsMu.RUnlock()
	}
}

// pingAll 发送心跳
func (h *Hub) pingAll() {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()

	for client := range h.clients {
		client.closeMu.Lock()
		if !client.closed {
			client.conn.WriteMessage(websocket.PingMessage, nil)
		}
		client.closeMu.Unlock()
	}
}

// Register 注册客户端
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister 注销客户端
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Broadcast 广播消息
func (h *Hub) Broadcast(taskID string, data []byte) {
	h.broadcast <- &Message{
		TaskID: taskID,
		Data:   data,
	}
}

// BroadcastLog 广播日志消息（带时间戳）
func (h *Hub) BroadcastLog(taskID string, log string) {
	data := []byte(log)
	h.Broadcast(taskID, data)
}

// GetClientCount 获取客户端数量
func (h *Hub) GetClientCount() int {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()
	return len(h.clients)
}

// GetTaskClientCount 获取特定任务的客户端数量
func (h *Hub) GetTaskClientCount(taskID string) int {
	h.taskClientsMu.RLock()
	defer h.taskClientsMu.RUnlock()
	if clients, ok := h.taskClients[taskID]; ok {
		return len(clients)
	}
	return 0
}

// NewClient 创建 WebSocket 客户端
func NewClient(conn *websocket.Conn, hub *Hub, taskID string) *Client {
	return &Client{
		conn:   conn,
		taskID: taskID,
		send:   make(chan []byte, 256),
		hub:    hub,
	}
}

// ReadPump 读取消息循环
func (c *Client) ReadPump() {
	defer func() {
		c.Close()
		c.hub.Unregister(c)
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// WritePump 写入消息循环
func (c *Client) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 批量发送队列中的消息
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Close 关闭客户端连接
func (c *Client) Close() {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()

	if c.closed {
		return
	}
	c.closed = true
	c.conn.Close()
}

// WebSocketLogWriter 实现 model.LogWriter 接口
type WebSocketLogWriter struct {
	hub    *Hub
	taskID string
}

// NewWebSocketLogWriter 创建 WebSocket 日志写入器
func NewWebSocketLogWriter(hub *Hub, taskID string) *WebSocketLogWriter {
	return &WebSocketLogWriter{
		hub:    hub,
		taskID: taskID,
	}
}

// Write 写入日志
func (w *WebSocketLogWriter) Write(p []byte) (n int, err error) {
	w.hub.BroadcastLog(w.taskID, string(p))
	return len(p), nil
}

// WriteString 写入字符串日志
func (w *WebSocketLogWriter) WriteString(s string) (n int, err error) {
	w.hub.BroadcastLog(w.taskID, s)
	return len(s), nil
}

// Close 关闭写入器
func (w *WebSocketLogWriter) Close() error {
	return nil
}

// MultiLogWriter 多路日志写入器（同时写入多个目标）
type MultiLogWriter struct {
	writers []interface {
		Write(p []byte) (n int, err error)
		WriteString(s string) (n int, err error)
		Close() error
	}
}

// NewMultiLogWriter 创建多路日志写入器
func NewMultiLogWriter(writers ...interface {
	Write(p []byte) (n int, err error)
	WriteString(s string) (n int, err error)
	Close() error
}) *MultiLogWriter {
	return &MultiLogWriter{writers: writers}
}

// Write 写入日志到所有写入器，返回第一个错误
func (w *MultiLogWriter) Write(p []byte) (n int, err error) {
	min := len(p)
	for _, writer := range w.writers {
		written, e := writer.Write(p)
		if e != nil && err == nil {
			err = e
		}
		if written < min {
			min = written
		}
	}
	return min, err
}

// WriteString 写入字符串到所有写入器，返回第一个错误
func (w *MultiLogWriter) WriteString(s string) (n int, err error) {
	min := len(s)
	for _, writer := range w.writers {
		written, e := writer.WriteString(s)
		if e != nil && err == nil {
			err = e
		}
		if written < min {
			min = written
		}
	}
	return min, err
}

// Close 关闭所有写入器，返回第一个错误
func (w *MultiLogWriter) Close() error {
	var firstErr error
	for _, writer := range w.writers {
		if err := writer.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
