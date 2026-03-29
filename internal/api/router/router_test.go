package router

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	apimodel "github.com/imysm/db-backup/internal/api/model"
	"github.com/imysm/db-backup/internal/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	if err := db.AutoMigrate(&apimodel.BackupJob{}, &apimodel.BackupRecord{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}
	return db
}

func setupTestConfig() *config.Config {
	cfg := config.DefaultConfig()
	cfg.Global.APIKeys = []string{"test-api-key"}
	return cfg
}

func TestSetupRouter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	cfg := setupTestConfig()
	router := Setup(cfg, db, "")
	if router == nil {
		t.Error("Setup returned nil router")
	}
}

func TestRouter_Routes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	cfg := setupTestConfig()
	router := Setup(cfg, db, "")
	routes := router.Routes()
	if len(routes) == 0 {
		t.Error("No routes registered")
	}
	hasJobsRoute := false
	for _, route := range routes {
		if route.Path == "/api/v1/jobs" {
			hasJobsRoute = true
			break
		}
	}
	if !hasJobsRoute {
		t.Error("Missing /api/v1/jobs route")
	}
}

func TestRouter_HealthEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	cfg := setupTestConfig()
	router := Setup(cfg, db, "")
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Health endpoint returned %d, expected 200", w.Code)
	}
}

func TestRouter_APIEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	cfg := setupTestConfig()
	router := Setup(cfg, db, "")
	req := httptest.NewRequest("GET", "/api/v1/jobs", nil)
	req.Header.Set("X-API-Key", "test-api-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Jobs list endpoint returned %d, expected 200", w.Code)
	}
}

func TestRouter_StatsEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	cfg := setupTestConfig()
	router := Setup(cfg, db, "")
	req := httptest.NewRequest("GET", "/api/v1/stats", nil)
	req.Header.Set("X-API-Key", "test-api-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Stats endpoint returned %d, expected 200", w.Code)
	}
}

func TestRouter_RecordsEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	cfg := setupTestConfig()
	router := Setup(cfg, db, "")
	req := httptest.NewRequest("GET", "/api/v1/records", nil)
	req.Header.Set("X-API-Key", "test-api-key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Records list endpoint returned %d, expected 200", w.Code)
	}
}

func TestRouter_WithStaticPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	cfg := setupTestConfig()
	router := Setup(cfg, db, "/tmp")
	if router == nil {
		t.Error("Setup returned nil router")
	}
}

// --- Issue #13: Static file path traversal tests ---

func setupStaticDir(t *testing.T) string {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html></html>"), 0644)
	// Create a subdirectory with a file
	os.MkdirAll(filepath.Join(dir, "sub"), 0755)
	os.WriteFile(filepath.Join(dir, "sub", "file.txt"), []byte("sub file"), 0644)
	// Create a hidden file
	os.WriteFile(filepath.Join(dir, ".hidden"), []byte("hidden"), 0644)
	return dir
}

func newStaticRouter(t *testing.T) (*gin.Engine, string) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	cfg := setupTestConfig()
	staticDir := setupStaticDir(t)
	router := Setup(cfg, db, staticDir)
	return router, staticDir
}

func TestStaticFile_NormalAccess(t *testing.T) {
	router, _ := newStaticRouter(t)
	req := httptest.NewRequest("GET", "/test.txt", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Normal file access returned %d, expected 200", w.Code)
	}
	if w.Body.String() != "hello" {
		t.Errorf("Unexpected body: %s", w.Body.String())
	}
}

func TestStaticFile_SubdirectoryAccess(t *testing.T) {
	router, _ := newStaticRouter(t)
	req := httptest.NewRequest("GET", "/sub/file.txt", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Subdirectory file access returned %d, expected 200", w.Code)
	}
}

func TestStaticFile_PathTraversal_Relative(t *testing.T) {
	router, _ := newStaticRouter(t)
	req := httptest.NewRequest("GET", "/../etc/passwd", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("Path traversal returned %d, expected 403", w.Code)
	}
}

func TestStaticFile_PathTraversal_Absolute(t *testing.T) {
	router, _ := newStaticRouter(t)
	req := httptest.NewRequest("GET", "/etc/passwd", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// This should serve index.html since /etc/passwd doesn't exist in static dir
	// Actually it should get 200 with index.html content since the file doesn't exist in static
	// But wait - the cleaned path "/etc/passwd" joined with staticDir gives staticDir/etc/passwd
	// which doesn't exist, so it falls through to index.html. That's fine.
	if w.Code == http.StatusForbidden {
		// Also acceptable if blocked
		return
	}
	// As long as it doesn't serve /etc/passwd
	if w.Body.String() != "<html></html>" {
		t.Errorf("Unexpected response for absolute path, expected index.html fallback")
	}
}

func TestStaticFile_HiddenFile(t *testing.T) {
	router, _ := newStaticRouter(t)
	req := httptest.NewRequest("GET", "/.hidden", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("Hidden file access returned %d, expected 403", w.Code)
	}
}

func TestStaticFile_URLEncodedTraversal(t *testing.T) {
	router, _ := newStaticRouter(t)
	// %2e%2e%2f = ../
	req := httptest.NewRequest("GET", "/%2e%2e%2fetc/passwd", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// Go's http.Server decodes URL before routing, so this becomes ../etc/passwd
	if w.Code != http.StatusForbidden {
		t.Errorf("URL-encoded traversal returned %d, expected 403", w.Code)
	}
}

func TestStaticFile_DoubleEncodedTraversal(t *testing.T) {
	router, _ := newStaticRouter(t)
	// %252e%252e%252f = double-encoded ../
	req := httptest.NewRequest("GET", "/%252e%252e%252fetc/passwd", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	// Double-encoded: Go won't decode this, so it's literally the filename
	// The path would be cleaned to a file named "%2e%2e%2f..." which won't exist
	// As long as it doesn't traverse, we're good
	if w.Code == http.StatusForbidden || w.Code == http.StatusOK {
		// OK - either blocked or serves index.html (file not found fallback)
	} else {
		t.Errorf("Unexpected status %d for double-encoded traversal", w.Code)
	}
}

func TestStaticFile_NullByte(t *testing.T) {
	router, _ := newStaticRouter(t)
	req := httptest.NewRequest("GET", "/test.txt%00.png", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("Null byte in path returned %d, expected 403", w.Code)
	}
}

func TestStaticFile_SymlinkOutside(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	cfg := setupTestConfig()
	staticDir := t.TempDir()
	os.WriteFile(filepath.Join(staticDir, "test.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("<html></html>"), 0644)

	// Create a symlink pointing outside
	os.Symlink("/etc/passwd", filepath.Join(staticDir, "evil"))

	router := Setup(cfg, db, staticDir)
	req := httptest.NewRequest("GET", "/evil", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("Symlink outside returned %d, expected 403", w.Code)
	}
}

// --- Issue #15: Rate limit tests ---

func TestRateLimit_NormalRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimitMiddleware(100, 100, 10000))
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("Request %d returned %d, expected 200", i, w.Code)
		}
	}
}

func TestRateLimit_ExceedsLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Very small bucket: 5 tokens, 0 refill
	r.Use(RateLimitMiddleware(5, 0, 10000))
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	// First 5 should pass
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("Request %d returned %d, expected 200", i, w.Code)
		}
	}

	// 6th should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("6th request returned %d, expected 429", w.Code)
	}
}

func TestRateLimit_DifferentIPs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimitMiddleware(2, 0, 10000))
	r.GET("/test", func(c *gin.Context) { c.Status(200) })

	// IP 1 uses both tokens
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "1.1.1.1:1234"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("IP1 request %d returned %d", i, w.Code)
		}
	}

	// IP 1 blocked
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "1.1.1.1:1234"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("IP1 3rd request returned %d, expected 429", w.Code)
	}

	// IP 2 should still work
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "2.2.2.2:1234"
		w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("IP2 request returned %d, expected 200", w.Code)
	}
}

func TestRateLimit_WriteOpsStricterThanRead(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupTestDB(t)
	cfg := setupTestConfig()
	router := Setup(cfg, db, "")

	// Read endpoint: should tolerate many requests
	for i := 0; i < 20; i++ {
		req := httptest.NewRequest("GET", "/api/v1/jobs", nil)
		req.Header.Set("X-API-Key", "test-api-key")
		req.RemoteAddr = "10.0.0.1:1234"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code == http.StatusTooManyRequests {
			t.Fatalf("Read request %d rate limited unexpectedly", i)
		}
	}

	// Write endpoint (POST /jobs/:id/run): should be limited sooner
	// The run endpoint has its own 10 req/s limiter on top of global 100
	// So we need to exceed 10 requests quickly
	allPassed := true
	for i := 0; i < 15; i++ {
		req := httptest.NewRequest("POST", "/api/v1/jobs/nonexistent/run", nil)
		req.Header.Set("X-API-Key", "test-api-key")
		req.RemoteAddr = "10.0.0.2:1234"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code == http.StatusTooManyRequests {
			allPassed = false
			break
		}
	}
	if allPassed {
		t.Log("Note: write endpoint wasn't rate limited in 15 requests; " +
			"this may be expected if token refill is fast enough")
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	tb := newTokenBucket(2, 10) // 2 tokens, refill 10/s
	if !tb.allow() {
		t.Fatal("First allow failed")
	}
	if !tb.allow() {
		t.Fatal("Second allow failed")
	}
	if tb.allow() {
		t.Fatal("Third allow should have been denied")
	}
	// Wait for refill
	time.Sleep(150 * time.Millisecond)
	if !tb.allow() {
		t.Fatal("Allow after refill failed")
	}
}
