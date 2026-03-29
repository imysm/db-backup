package handler

import (
	"github.com/imysm/db-backup/internal/api/dto"
	"github.com/imysm/db-backup/internal/api/model"
	"github.com/imysm/db-backup/internal/verify"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// VerifyHandler 验证处理器
type VerifyHandler struct {
	db *gorm.DB
}

// NewVerifyHandler 创建处理器
func NewVerifyHandler(db *gorm.DB) *VerifyHandler {
	return &VerifyHandler{db: db}
}

// VerifyBackup 验证备份
func (h *VerifyHandler) VerifyBackup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(400, dto.Error("无效的记录ID"))
		return
	}

	// 获取备份记录
	var record model.BackupRecord
	if err := h.db.First(&record, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(404, dto.Error("备份记录不存在"))
			return
		}
		c.JSON(500, dto.Error("获取备份记录失败: "+err.Error()))
		return
	}

	// 获取任务信息
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

	// 更新记录
	record.Verified = result.Passed
	now := time.Now()
	record.VerifiedAt = &now
	h.db.Save(&record)

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"passed":      result.Passed,
			"file_exists": result.FileExists,
			"checksum_ok": result.ChecksumOK,
			"error":       result.Error,
		},
	})
}

// TestRestore 测试恢复
func (h *VerifyHandler) TestRestore(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(400, dto.Error("无效的记录ID"))
		return
	}

	// 获取备份记录
	var record model.BackupRecord
	if err := h.db.First(&record, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(404, dto.Error("备份记录不存在"))
			return
		}
		c.JSON(500, dto.Error("获取备份记录失败: "+err.Error()))
		return
	}

	// 获取任务信息
	var job model.BackupJob
	if err := h.db.First(&job, record.JobID).Error; err != nil {
		c.JSON(500, dto.Error("获取任务信息失败: "+err.Error()))
		return
	}

	// 创建验证器
	v := verify.NewVerifier(string(job.StorageType), nil)

	// 执行恢复测试
	err = v.TestRestore(c.Request.Context(), record.FilePath, string(job.DatabaseType))
	if err != nil {
		c.JSON(200, gin.H{
			"code":    0,
			"message": "success",
			"data": gin.H{
				"passed": false,
				"error":  err.Error(),
			},
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"passed": true,
		},
	})
}

// BatchVerifyRequest 批量验证请求
type BatchVerifyRequest struct {
	IDs []uint `json:"ids"`
}

// BatchVerify 批量验证备份记录
func (h *VerifyHandler) BatchVerify(c *gin.Context) {
	var req BatchVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.Error("无效的请求参数"))
		return
	}

	if len(req.IDs) == 0 {
		c.JSON(400, dto.Error("ID列表不能为空"))
		return
	}

	var results []gin.H
	for _, id := range req.IDs {
		var record model.BackupRecord
		if err := h.db.First(&record, id).Error; err != nil {
			results = append(results, gin.H{
				"id":     id,
				"passed": false,
				"error":  "记录不存在",
			})
			continue
		}

		var job model.BackupJob
		if err := h.db.First(&job, record.JobID).Error; err != nil {
			results = append(results, gin.H{
				"id":     id,
				"passed": false,
				"error":  "获取任务信息失败",
			})
			continue
		}

		v := verify.NewVerifier(string(job.StorageType), nil)
		result, err := v.VerifyFile(c.Request.Context(), record.FilePath, record.Checksum)
		if err != nil {
			results = append(results, gin.H{
				"id":     id,
				"passed": false,
				"error":  "验证失败: " + err.Error(),
			})
			continue
		}

		now := time.Now()
		record.Verified = result.Passed
		record.VerifiedAt = &now
		h.db.Save(&record)

		results = append(results, gin.H{
			"id":          id,
			"passed":      result.Passed,
			"file_exists": result.FileExists,
			"checksum_ok": result.ChecksumOK,
			"verified_at": now,
			"error":       result.Error,
		})
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"results": results,
			"total":   len(results),
			"passed":  func() int { n := 0; for _, r := range results { if r["passed"].(bool) { n++ } }; return n }(),
		},
	})
}
