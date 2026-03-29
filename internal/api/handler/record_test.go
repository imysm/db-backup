package handler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	apiModel "github.com/imysm/db-backup/internal/api/model"
	"github.com/imysm/db-backup/internal/model"
	"github.com/imysm/db-backup/internal/storage"
)

// mockStorage 用于测试的存储 mock
type mockStorage struct {
	deleteErr error
	deleted   []string
}

func (m *mockStorage) Save(_ context.Context, _, _ string) error             { return nil }
func (m *mockStorage) Download(_ context.Context, _, _ string) error          { return nil }
func (m *mockStorage) List(_ context.Context, _ string) ([]model.BackupRecord, error) {
	return nil, nil
}
func (m *mockStorage) Delete(_ context.Context, filePath string) error {
	m.deleted = append(m.deleted, filePath)
	return m.deleteErr
}
func (m *mockStorage) Exists(_ context.Context, _ string) (bool, error) { return true, nil }
func (m *mockStorage) Size(_ context.Context, _ string) (int64, error)  { return 0, nil }
func (m *mockStorage) GetUsage(_ context.Context, _ string) (int64, error) { return 0, nil }
func (m *mockStorage) Type() string { return "mock" }

// Compile-time check
var _ storage.Storage = (*mockStorage)(nil)

func TestRecordHandler_Delete_WithStorage(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("正常删除同时清理存储文件", func(t *testing.T) {
		db := setupTestDB(t)
		mock := &mockStorage{}
		h := NewRecordHandlerWithStorage(db, mock)

		db.Create(&apiModel.BackupRecord{JobID: 1, FilePath: "backup/test.sql", Status: "success"})

		router := gin.New()
		router.DELETE("/records/:id", h.Delete)

		req := httptest.NewRequest(http.MethodDelete, "/records/1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("期望 200，got %d", w.Code)
		}
		if len(mock.deleted) != 1 || mock.deleted[0] != "backup/test.sql" {
			t.Errorf("期望删除 backup/test.sql，got %v", mock.deleted)
		}
	})

	t.Run("存储删除失败仍删除记录", func(t *testing.T) {
		db := setupTestDB(t)
		mock := &mockStorage{deleteErr: errors.New("存储不可达")}
		h := NewRecordHandlerWithStorage(db, mock)
		h.logger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelWarn}))

		db.Create(&apiModel.BackupRecord{JobID: 1, FilePath: "backup/test.sql", Status: "success"})

		router := gin.New()
		router.DELETE("/records/:id", h.Delete)

		req := httptest.NewRequest(http.MethodDelete, "/records/1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("记录应被删除，got %d", w.Code)
		}
		if len(mock.deleted) != 1 {
			t.Error("应尝试删除存储文件")
		}
		var count int64
		db.Model(&apiModel.BackupRecord{}).Count(&count)
		if count != 0 {
			t.Error("记录应已从数据库删除")
		}
	})

	t.Run("无存储时不崩溃", func(t *testing.T) {
		db := setupTestDB(t)
		h := NewRecordHandler(db)

		db.Create(&apiModel.BackupRecord{JobID: 1, FilePath: "backup/test.sql", Status: "success"})

		router := gin.New()
		router.DELETE("/records/:id", h.Delete)

		req := httptest.NewRequest(http.MethodDelete, "/records/1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("期望 200，got %d", w.Code)
		}
	})

	t.Run("记录不存在返回404", func(t *testing.T) {
		db := setupTestDB(t)
		h := NewRecordHandler(db)

		router := gin.New()
		router.DELETE("/records/:id", h.Delete)

		req := httptest.NewRequest(http.MethodDelete, "/records/999", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("期望 404，got %d", w.Code)
		}
	})
}
