package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestAPIKeyAuth(t *testing.T) {
	tests := []struct {
		name       string
		apiKeys    []string
		setupReq   func() *http.Request
		wantStatus int
	}{
		{
			name:    "valid key via Authorization Bearer",
			apiKeys: []string{"secret-key"},
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Authorization", "Bearer secret-key")
				return req
			},
			wantStatus: http.StatusOK,
		},
		{
			name:    "valid key via X-API-Key",
			apiKeys: []string{"my-api-key"},
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("X-API-Key", "my-api-key")
				return req
			},
			wantStatus: http.StatusOK,
		},
		{
			name:    "valid key via query param",
			apiKeys: []string{"ws-token"},
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/test?token=ws-token", nil)
				return req
			},
			wantStatus: http.StatusOK,
		},
		{
			name:    "invalid key",
			apiKeys: []string{"correct-key"},
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("X-API-Key", "wrong-key")
				return req
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:    "missing key",
			apiKeys: []string{"secret-key"},
			setupReq: func() *http.Request {
				return httptest.NewRequest("GET", "/test", nil)
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:    "empty Bearer token",
			apiKeys: []string{"secret-key"},
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Authorization", "Bearer ")
				return req
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:    "Bearer without space",
			apiKeys: []string{"secret-key"},
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Authorization", "Bearer")
				return req
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:    "multiple keys - match second",
			apiKeys: []string{"key1", "key2", "key3"},
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("X-API-Key", "key2")
				return req
			},
			wantStatus: http.StatusOK,
		},
		{
			name:    "empty apiKeys list",
			apiKeys: []string{},
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("X-API-Key", "any-key")
				return req
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:    "Authorization priority over X-API-Key",
			apiKeys: []string{"bearer-key"},
			setupReq: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Authorization", "Bearer bearer-key")
				req.Header.Set("X-API-Key", "wrong-key")
				return req
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(APIKeyAuth(tt.apiKeys))
			r.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"ok": true})
			})

			w := httptest.NewRecorder()
			r.ServeHTTP(w, tt.setupReq())

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", w.Code, tt.wantStatus, w.Body.String())
			}
		})
	}
}
