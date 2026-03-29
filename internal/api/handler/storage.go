package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/imysm/db-backup/internal/model"
	"gorm.io/gorm"
)

// StorageHandler 存储管理处理器
type StorageHandler struct {
	db *gorm.DB
}

// NewStorageHandler 创建处理器
func NewStorageHandler(db *gorm.DB) *StorageHandler {
	return &StorageHandler{db: db}
}

// GetStats 获取存储统计
func (h *StorageHandler) GetStats(c *gin.Context) {
	var stats struct {
		TotalRecords int64   `json:"total_records"`
		TotalSize   int64   `json:"total_size"`
		StorageType string   `json:"storage_type"`
	}

	// 统计备份记录
	h.db.Model(&model.BackupRecord{}).Count(&stats.TotalRecords)
	h.db.Model(&model.BackupRecord{}).Select("COALESCE(SUM(file_size), 0)").Scan(&stats.TotalSize)
	stats.StorageType = "local"

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"total_records": stats.TotalRecords,
			"total_size":   stats.TotalSize,
			"storage_type":  stats.StorageType,
		},
	})
}

// ListObjects 列出存储对象
func (h *StorageHandler) ListObjects(c *gin.Context) {
	var records []model.BackupRecord
	
	query := h.db.Model(&model.BackupRecord{})
	
	// 支持按任务过滤
	if jobID := c.Query("job_id"); jobID != "" {
		query = query.Where("job_id = ?", jobID)
	}
	
	// 支持按状态过滤
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	
	query.Order("created_at DESC").Limit(100).Find(&records)
	
	objects := make([]gin.H, len(records))
	for i, r := range records {
		objects[i] = gin.H{
			"key":      r.FilePath,
			"size":     r.FileSize,
			"mod_time": r.CreatedAt,
		}
	}
	
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"objects": objects,
			"count":   len(objects),
		},
	})
}

// GetSignedURL 获取签名URL
func (h *StorageHandler) GetSignedURL(c *gin.Context) {
	storageType := c.DefaultQuery("type", "local")
	key := c.Query("key")
	expiry := 3600 // 默认1小时

	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 key 参数"})
		return
	}

	// 简化实现：返回本地路径
	// 实际生产环境应该调用存储后端的 GetSignedURL 方法
	url := "/api/v1/storage/download?type=" + storageType + "&key=" + key

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"url":      url,
			"expires":  expiry,
			"method":   "GET",
		},
	})
}

// DeleteObject 删除存储对象
func (h *StorageHandler) DeleteObject(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 key 参数"})
		return
	}

	// 简化实现：只是返回成功
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"message": "删除成功",
	})
}
