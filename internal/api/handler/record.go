package handler

import (
	"encoding/json"
	"log/slog"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/imysm/db-backup/internal/api/dto"
	"github.com/imysm/db-backup/internal/api/model"
	"github.com/imysm/db-backup/internal/storage"
	"github.com/imysm/db-backup/internal/util"
	"github.com/imysm/db-backup/internal/verify"
	"gorm.io/gorm"
)

// RecordHandler 备份记录处理器
type RecordHandler struct {
	db      *gorm.DB
	storage storage.Storage
	logger  *slog.Logger
}

// NewRecordHandler 创建处理器
func NewRecordHandler(db *gorm.DB) *RecordHandler {
	return &RecordHandler{
		db:      db,
		storage: nil,
		logger:  slog.Default(),
	}
}

// NewRecordHandlerWithStorage 创建带存储的处理器
func NewRecordHandlerWithStorage(db *gorm.DB, s storage.Storage) *RecordHandler {
	return &RecordHandler{
		db:      db,
		storage: s,
		logger:  slog.Default(),
	}
}

// List 获取记录列表
func (h *RecordHandler) List(c *gin.Context) {
	var records []model.BackupRecord

	// 分页
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// 过滤条件
	jobID := c.Query("job_id")

	query := h.db.Model(&model.BackupRecord{})
	if jobID != "" {
		query = query.Where("job_id = ?", jobID)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if startDate := c.Query("start_date"); startDate != "" {
		query = query.Where("started_at >= ?", startDate)
	}
	if endDate := c.Query("end_date"); endDate != "" {
		query = query.Where("started_at < ?", endDate+"T23:59:59")
	}

	// 排序
	sortBy := c.DefaultQuery("sort_by", "created_at")
	sortOrder := c.DefaultQuery("sort_order", "desc")
	allowedSortFields := map[string]bool{"created_at": true, "started_at": true, "file_size": true, "duration": true}
	if !allowedSortFields[sortBy] {
		sortBy = "created_at"
	}
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}
	orderBy := sortBy + " " + sortOrder

	// 统计总数
	var total int64
	query.Count(&total)

	// 分页查询
	if err := query.Offset(offset).Limit(pageSize).Order(orderBy).Find(&records).Error; err != nil {
		c.JSON(500, dto.Error("获取记录列表失败: "+err.Error()))
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"data":      records,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// Get 获取单条记录
func (h *RecordHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(400, dto.Error("无效的记录ID"))
		return
	}

	var record model.BackupRecord
	if err := h.db.First(&record, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(404, dto.Error("记录不存在"))
			return
		}
		c.JSON(500, dto.Error("获取记录失败: "+err.Error()))
		return
	}

	c.JSON(200, dto.Success(record))
}

// Verify 验证备份
func (h *RecordHandler) Verify(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(400, dto.Error("无效的记录ID"))
		return
	}

	var record model.BackupRecord
	if err := h.db.First(&record, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(404, dto.Error("记录不存在"))
			return
		}
		c.JSON(500, dto.Error("获取记录失败: "+err.Error()))
		return
	}

	// 获取任务信息以获取存储类型
	var job model.BackupJob
	if err := h.db.First(&job, record.JobID).Error; err != nil {
		c.JSON(500, dto.Error("获取任务信息失败: "+err.Error()))
		return
	}

	// 创建验证器
	v := verify.NewVerifier(string(job.StorageType), nil)

	// 执行验证
	result, err := v.VerifyFile(c.Request.Context(), record.FilePath, record.Checksum)
	if err != nil {
		c.JSON(500, dto.Error("验证失败: "+err.Error()))
		return
	}

	// 更新验证状态
	now := time.Now()
	record.Verified = result.Passed
	record.VerifiedAt = &now
	if err := h.db.Save(&record).Error; err != nil {
		c.JSON(500, dto.Error("更新记录失败: "+err.Error()))
		return
	}

	c.JSON(200, dto.Success(gin.H{
		"verified":    result.Passed,
		"file_exists": result.FileExists,
		"checksum_ok": result.ChecksumOK,
		"verified_at": now,
		"error":       result.Error,
	}))
}

// Download 下载备份文件
func (h *RecordHandler) Download(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(400, dto.Error("无效的记录ID"))
		return
	}

	var record model.BackupRecord
	if err := h.db.First(&record, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(404, dto.Error("记录不存在"))
			return
		}
		c.JSON(500, dto.Error("获取记录失败: "+err.Error()))
		return
	}

	// 获取任务信息以获取存储路径
	var job model.BackupJob
	if err := h.db.First(&job, record.JobID).Error; err != nil {
		c.JSON(500, dto.Error("获取任务信息失败: "+err.Error()))
		return
	}

	// 获取存储路径用于路径验证
	storagePath := getStoragePath(job.StorageConfig)
	if storagePath == "" {
		c.JSON(500, dto.Error("无法确定存储路径"))
		return
	}

	// 验证文件路径在允许的目录下，防止路径遍历攻击
	if err := util.ValidateFilePath(record.FilePath, storagePath); err != nil {
		c.JSON(403, dto.Error("文件路径非法: "+err.Error()))
		return
	}

	c.File(record.FilePath)
}

// Delete 删除记录
func (h *RecordHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(400, dto.Error("无效的记录ID"))
		return
	}

	// 先查询记录以获取文件路径
	var record model.BackupRecord
	if err := h.db.First(&record, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(404, dto.Error("记录不存在"))
			return
		}
		c.JSON(500, dto.Error("查询记录失败: "+err.Error()))
		return
	}

	// 尝试删除存储中的文件（失败不阻止记录删除）
	if h.storage != nil && record.FilePath != "" {
		if err := h.storage.Delete(c.Request.Context(), record.FilePath); err != nil {
			h.logger.Warn("删除存储文件失败，记录仍将被删除",
				"record_id", id,
				"file_path", record.FilePath,
				"error", err,
			)
		}
	}

	if err := h.db.Delete(&model.BackupRecord{}, id).Error; err != nil {
		c.JSON(500, dto.Error("删除记录失败: "+err.Error()))
		return
	}

	c.JSON(200, dto.Success(nil))
}

// storageConfigJSON 用于解析 BackupJob 的 StorageConfig JSON 字段
type storageConfigJSON struct {
	Path string `json:"path"`
}

// getStoragePath 从 StorageConfig JSON 中提取存储路径
func getStoragePath(storageConfigJSONStr string) string {
	if storageConfigJSONStr == "" {
		return ""
	}
	var sc storageConfigJSON
	if err := json.Unmarshal([]byte(storageConfigJSONStr), &sc); err != nil {
		return ""
	}
	return sc.Path
}
