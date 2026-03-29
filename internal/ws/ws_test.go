package ws

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestNewHub(t *testing.T) {
	h := NewHub()
	if h == nil {
		t.Fatal("NewHub() returned nil")
	}
	if h.GetClientCount() != 0 {
		t.Errorf("new hub should have 0 clients, got %d", h.GetClientCount())
	}
	if h.GetTaskClientCount("task1") != 0 {
		t.Errorf("new hub should have 0 task clients, got %d", h.GetTaskClientCount("task1"))
	}
}

func TestHub_RegisterUnregister(t *testing.T) {
	h := NewHub()
	go h.Run()
	defer h.Stop()

	var serverConn *websocket.Conn
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)
		if err != nil {
			return
		}
		serverConn = conn
	}))
	defer s.Close()

	SetAllowedOrigins([]string{})
	wsURL := "ws" + strings.TrimPrefix(s.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer clientConn.Close()

	client := NewClient(serverConn, h, "task1")
	h.Register(client)
	time.Sleep(100 * time.Millisecond)

	if count := h.GetClientCount(); count != 1 {
		t.Errorf("expected 1 client, got %d", count)
	}
	if count := h.GetTaskClientCount("task1"); count != 1 {
		t.Errorf("expected 1 task client, got %d", count)
	}

	h.Unregister(client)
	time.Sleep(100 * time.Millisecond)

	if count := h.GetClientCount(); count != 0 {
		t.Errorf("expected 0 clients after unregister, got %d", count)
	}
}

func TestHub_Broadcast_ToTaskClient(t *testing.T) {
	h := NewHub()
	go h.Run()

	var serverConn *websocket.Conn
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)
		if err != nil {
			return
		}
		serverConn = conn
	}))
	defer s.Close()

	SetAllowedOrigins([]string{})
	wsURL := "ws" + strings.TrimPrefix(s.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer clientConn.Close()

	client := NewClient(serverConn, h, "task1")
	go client.WritePump()
	h.Register(client)
	time.Sleep(100 * time.Millisecond)

	clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	h.Broadcast("task1", []byte("hello world"))
	time.Sleep(100 * time.Millisecond)

	_, msg, err := clientConn.ReadMessage()
	if err != nil {
		h.Stop()
		t.Fatalf("ReadMessage() error: %v", err)
	}
	if string(msg) != "hello world" {
		h.Stop()
		t.Errorf("got %q, want %q", string(msg), "hello world")
	}
	h.Stop()
}

func TestHub_Broadcast_Global(t *testing.T) {
	h := NewHub()
	go h.Run()

	var serverConn *websocket.Conn
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)
		if err != nil {
			return
		}
		serverConn = conn
	}))
	defer s.Close()

	SetAllowedOrigins([]string{})
	wsURL := "ws" + strings.TrimPrefix(s.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer clientConn.Close()

	client := NewClient(serverConn, h, "") // no task ID
	go client.WritePump()
	h.Register(client)
	time.Sleep(100 * time.Millisecond)

	clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	h.Broadcast("", []byte("global message"))
	time.Sleep(100 * time.Millisecond)

	_, msg, err := clientConn.ReadMessage()
	if err != nil {
		h.Stop()
		t.Fatalf("ReadMessage() error: %v", err)
	}
	if string(msg) != "global message" {
		h.Stop()
		t.Errorf("got %q, want %q", string(msg), "global message")
	}
	h.Stop()
}

func TestHub_BroadcastLog(t *testing.T) {
	h := NewHub()
	h.BroadcastLog("task1", "test log message") // should not panic
}

func TestHub_Stop(t *testing.T) {
	h := NewHub()
	go h.Run()
	h.Stop() // should not panic
	h.Stop() // double stop should not panic
}

func TestHub_RegisterClient_NoTaskID(t *testing.T) {
	h := NewHub()
	h.registerClient(&Client{taskID: ""})
	if h.GetClientCount() != 1 {
		t.Errorf("expected 1 client, got %d", h.GetClientCount())
	}
}

func TestHub_UnregisterClient_NotRegistered(t *testing.T) {
	h := NewHub()
	client := &Client{taskID: "task1", send: make(chan []byte, 256)}
	h.unregisterClient(client) // should not panic
}

func TestHub_Broadcast_NonExistentTask(t *testing.T) {
	h := NewHub()
	go h.Run()
	defer h.Stop()
	// Should not panic when broadcasting to non-existent task
	h.Broadcast("nonexistent", []byte("test"))
	time.Sleep(50 * time.Millisecond)
}

func TestSetAllowedOrigins(t *testing.T) {
	tests := []struct {
		name      string
		origins   []string
		reqOrigin string
		want      bool
	}{
		{"empty list - no origin", []string{}, "", true},
		{"empty list - with origin", []string{}, "http://evil.com", false},
		{"matching origin", []string{"http://localhost:3000"}, "http://localhost:3000", true},
		{"non-matching origin", []string{"http://localhost:3000"}, "http://evil.com", false},
		{"multiple origins - match", []string{"http://a.com", "http://b.com"}, "http://b.com", true},
		{"no origin header (tool)", []string{"http://a.com"}, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetAllowedOrigins(tt.origins)
			req := httptest.NewRequest("GET", "/", nil)
			if tt.reqOrigin != "" {
				req.Header.Set("Origin", tt.reqOrigin)
			}
			got := upgrader.CheckOrigin(req)
			if got != tt.want {
				t.Errorf("CheckOrigin() = %v, want %v", got, tt.want)
			}
		})
	}
	SetAllowedOrigins([]string{})
}

func TestMultiLogWriter(t *testing.T) {
	var buf1, buf2 strings.Builder
	writer1 := &testLogWriter{&buf1}
	writer2 := &testLogWriter{&buf2}

	mw := NewMultiLogWriter(writer1, writer2)

	n, err := mw.Write([]byte("hello"))
	if err != nil {
		t.Errorf("Write() error = %v", err)
	}
	if n != 5 {
		t.Errorf("Write() = %d, want 5", n)
	}
	if buf1.String() != "hello" || buf2.String() != "hello" {
		t.Error("Write() should write to all writers")
	}

	n, err = mw.WriteString("world")
	if err != nil {
		t.Errorf("WriteString() error = %v", err)
	}
	if n != 5 {
		t.Errorf("WriteString() = %d, want 5", n)
	}

	if err := mw.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestMultiLogWriter_Empty(t *testing.T) {
	mw := NewMultiLogWriter()
	n, err := mw.Write([]byte("hello"))
	if err != nil {
		t.Errorf("Write() error = %v", err)
	}
	if n != 5 {
		t.Errorf("Write() = %d, want 5", n)
	}
}

type testLogWriter struct{ buf *strings.Builder }

func (w *testLogWriter) Write(p []byte) (n int, err error)      { return w.buf.Write(p) }
func (w *testLogWriter) WriteString(s string) (n int, err error) { return w.buf.WriteString(s) }
func (w *testLogWriter) Close() error                            { return nil }

func TestWebSocketLogWriter(t *testing.T) {
	h := NewHub()
	go h.Run()
	defer h.Stop()

	lw := NewWebSocketLogWriter(h, "task1")
	n, err := lw.Write([]byte("log line"))
	if err != nil || n != 8 {
		t.Errorf("Write() = (%d, %v), want (8, nil)", n, err)
	}
	n, err = lw.WriteString("another line")
	if err != nil || n != 12 {
		t.Errorf("WriteString() = (%d, %v), want (12, nil)", n, err)
	}
	if err := lw.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestNewClient(t *testing.T) {
	h := NewHub()
	var serverConn *websocket.Conn
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _ := websocket.Upgrade(w, r, nil, 1024, 1024)
		serverConn = conn
	}))
	defer s.Close()

	SetAllowedOrigins([]string{})
	wsURL := "ws" + strings.TrimPrefix(s.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer conn.Close()

	time.Sleep(50 * time.Millisecond)
	client := NewClient(serverConn, h, "task-abc")
	if client.taskID != "task-abc" {
		t.Errorf("taskID = %q, want %q", client.taskID, "task-abc")
	}
	if client.hub != h {
		t.Error("hub not set")
	}
	client.Close()
}

func TestClient_Close_Idempotent(t *testing.T) {
	h := NewHub()
	var serverConn *websocket.Conn
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _ := websocket.Upgrade(w, r, nil, 1024, 1024)
		serverConn = conn
	}))
	defer s.Close()

	SetAllowedOrigins([]string{})
	wsURL := "ws" + strings.TrimPrefix(s.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer conn.Close()

	time.Sleep(50 * time.Millisecond)
	client := NewClient(serverConn, h, "")
	client.Close()
	client.Close() // should not panic
}
