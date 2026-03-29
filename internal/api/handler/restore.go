package handler

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/imysm/db-backup/internal/api/dto"
	"github.com/imysm/db-backup/internal/api/model"
	"github.com/imysm/db-backup/internal/restore"
	"github.com/imysm/db-backup/internal/util"
	"gorm.io/gorm"
)

// RestoreHandler 恢复处理器
type RestoreHandler struct {
	db *gorm.DB
}

// NewRestoreHandler 创建处理器
func NewRestoreHandler(db *gorm.DB) *RestoreHandler {
	return &RestoreHandler{db: db}
}

// RestoreRequest 恢复请求
type RestoreAPIRequest struct {
	RecordID   uint   `json:"record_id"`   // 备份记录ID
	TargetHost string `json:"target_host"` // 目标主机
	TargetPort int    `json:"target_port"` // 目标端口
	TargetDB   string `json:"target_db"`   // 目标数据库
	TargetUser string `json:"target_user"` // 目标用户
	TargetPass string `json:"target_pass"` // 目标密码
	NoRestore  bool   `json:"no_restore"`  // 只校验不恢复
}

// ValidateRequest 预检查请求
type ValidateRequest struct {
	TargetHost string `json:"target_host"` // 目标主机
	TargetPort int    `json:"target_port"` // 目标端口
	TargetDB   string `json:"target_db"`   // 目标数据库
}

// ValidateCheck 单项检查结果
type ValidateCheck struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

// ValidateResult 预检查结果
type ValidateResult struct {
	Checks    []ValidateCheck `json:"checks"`
	AllPassed bool            `json:"all_passed"`
}

// Restore 恢复备份
func (h *RestoreHandler) Restore(c *gin.Context) {
	var req RestoreAPIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.Error("请求参数错误: "+err.Error()))
		return
	}

	// 获取备份记录
	var record model.BackupRecord
	if err := h.db.First(&record, req.RecordID).Error; err != nil {
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

	// 获取恢复器
	restorer, err := restore.GetRestorer(string(job.DatabaseType))
	if err != nil {
		c.JSON(400, dto.Error("不支持的数据库类型: "+string(job.DatabaseType)))
		return
	}

	// 构建恢复请求
	restoreReq := &restore.RestoreRequest{
		BackupFile: record.FilePath,
		TargetHost: req.TargetHost,
		TargetPort: req.TargetPort,
		TargetDB:   req.TargetDB,
		TargetUser: req.TargetUser,
		TargetPass: req.TargetPass,
		DBType:     string(job.DatabaseType),
		NoRestore:  req.NoRestore,
	}

	// 验证备份文件路径在允许的目录下，防止路径遍历攻击
	storagePath := getStoragePath(job.StorageConfig)
	if storagePath == "" {
		c.JSON(500, dto.Error("无法确定存储路径"))
		return
	}
	if err := util.ValidateFilePath(record.FilePath, storagePath); err != nil {
		c.JSON(403, dto.Error("文件路径非法: "+err.Error()))
		return
	}

	// 设置默认值
	if restoreReq.TargetHost == "" {
		restoreReq.TargetHost = "localhost"
	}
	if restoreReq.TargetPort == 0 {
		switch job.DatabaseType {
		case model.DatabaseTypePostgres:
			restoreReq.TargetPort = 5432
		case model.DatabaseTypeMySQL:
			restoreReq.TargetPort = 3306
		case model.DatabaseTypeMongoDB:
			restoreReq.TargetPort = 27017
		}
	}

	// 执行恢复
	result, err := restorer.Restore(c.Request.Context(), restoreReq)
	if err != nil {
		c.JSON(200, gin.H{
			"code":    0,
			"message": "恢复完成",
			"data": gin.H{
				"success": false,
				"error":   err.Error(),
			},
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"success":       result.Success,
			"duration":      result.Duration,
			"rows_affected": result.RowsAffected,
			"error":         result.Error,
		},
	})

	// 恢复成功后记录历史
	if result.Success && !req.NoRestore {
		detailMsg := fmt.Sprintf("恢复到 %s:%d/%s，耗时 %dms",
			restoreReq.TargetHost, restoreReq.TargetPort, restoreReq.TargetDB, result.Duration)
		history := model.BackupHistory{
			JobID:    job.ID,
			RecordID: record.ID,
			Action:   "restored",
			Details:  detailMsg,
		}
		h.db.Create(&history)
	}
}

// Validate 验证备份文件
func (h *RestoreHandler) Validate(c *gin.Context) {
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

	// 检查文件是否存在
	if _, err := os.Stat(record.FilePath); os.IsNotExist(err) {
		c.JSON(200, gin.H{
			"code":    0,
			"message": "success",
			"data": gin.H{
				"valid":       false,
				"file_exists": false,
				"error":       "备份文件不存在",
			},
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"valid":       true,
			"file_exists": true,
			"file_size":   record.FileSize,
			"checksum":    record.Checksum,
		},
	})
}

// List 获取可恢复的备份列表
func (h *RestoreHandler) List(c *gin.Context) {
	var records []model.BackupRecord

	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	// 只查询成功的备份
	query := h.db.Model(&model.BackupRecord{}).Where("status = ?", model.BackupStatusSuccess)

	var total int64
	query.Count(&total)

	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&records).Error; err != nil {
		c.JSON(500, dto.Error("获取列表失败: "+err.Error()))
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

// ValidatePOST POST - 增强预检查
func (h *RestoreHandler) ValidatePOST(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(400, dto.Error("无效的记录ID"))
		return
	}

	var record model.BackupRecord
	if err := h.db.First(&record, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(404, dto.Error("备份记录不存在"))
			return
		}
		c.JSON(500, dto.Error("获取备份记录失败: "+err.Error()))
		return
	}

	// 解析可选的请求体
	var req ValidateRequest
	_ = c.ShouldBindJSON(&req) // 允许空 body

	var checks []ValidateCheck
	allPassed := true

	// 检查1: 文件完整性
	fileCheck := ValidateCheck{Name: "备份文件"}
	info, statErr := os.Stat(record.FilePath)
	if os.IsNotExist(statErr) {
		fileCheck.Passed = false
		fileCheck.Message = "备份文件不存在"
		allPassed = false
	} else if info.Size() == 0 {
		fileCheck.Passed = false
		fileCheck.Message = "文件大小为 0，文件可能已损坏"
		allPassed = false
	} else {
		fileCheck.Passed = true
		fileCheck.Message = fmt.Sprintf("文件完整 (%s)", formatFileSize(info.Size()))
	}
	checks = append(checks, fileCheck)

	// 检查2: 目标数据库连通性（仅当提供了目标配置时）
	dbCheck := ValidateCheck{Name: "目标数据库"}
	if req.TargetHost != "" && req.TargetPort > 0 {
		start := time.Now()
		conn, dialErr := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", req.TargetHost, req.TargetPort), 3*time.Second)
		elapsed := time.Since(start)
		if dialErr != nil {
			dbCheck.Passed = false
			dbCheck.Message = fmt.Sprintf("连接失败: %s", dialErr.Error())
			allPassed = false
		} else {
			conn.Close()
			dbCheck.Passed = true
			dbCheck.Message = fmt.Sprintf("连接成功 (%dms)", elapsed.Milliseconds())
		}
	} else {
		dbCheck.Passed = true
		dbCheck.Message = "未提供目标配置，跳过检查"
	}
	checks = append(checks, dbCheck)

	// 检查3: 磁盘空间（仅本地文件）
	diskCheck := ValidateCheck{Name: "磁盘空间"}
	if info != nil && info.Size() > 0 {
		var statFS syscall.Statfs_t
		dir := filepath.Dir(record.FilePath)
		if err := syscall.Statfs(dir, &statFS); err != nil {
			diskCheck.Passed = false
			diskCheck.Message = "无法获取磁盘信息: " + err.Error()
			allPassed = false
		} else {
			available := statFS.Bavail * uint64(statFS.Bsize)
			needed := uint64(info.Size())
			diskCheck.Passed = available >= needed
			diskCheck.Message = fmt.Sprintf("需要 %s，可用 %s", formatFileSize(int64(needed)), formatFileSize(int64(available)))
			if !diskCheck.Passed {
				allPassed = false
			}
		}
	} else {
		diskCheck.Passed = true
		diskCheck.Message = "跳过（文件不可用）"
	}
	checks = append(checks, diskCheck)

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"checks":     checks,
			"all_passed": allPassed,
		},
	})
}

// GetDetail 获取恢复详情
func (h *RestoreHandler) GetDetail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(400, dto.Error("无效的记录ID"))
		return
	}

	var history model.BackupHistory
	if err := h.db.Where("id = ? AND action = ?", id, "restored").First(&history).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(404, dto.Error("恢复记录不存在"))
			return
		}
		c.JSON(500, dto.Error("查询失败: "+err.Error()))
		return
	}

	// 获取关联的任务名
	var jobName string
	var record model.BackupRecord
	if err := h.db.First(&record, history.RecordID).Error; err == nil {
		jobName = record.JobName
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"id":         history.ID,
			"record_id":  history.RecordID,
			"job_name":   jobName,
			"action":     history.Action,
			"details":    history.Details,
			"created_at": history.CreatedAt.Format("2006-01-02 15:04:05"),
		},
	})
}

// formatFileSize 格式化文件大小
func formatFileSize(bytes int64) string {
	if bytes <= 0 {
		return "0 B"
	}
	const unit = 1024
	sizes := []string{"B", "KB", "MB", "GB", "TB"}
	i := 0
	val := float64(bytes)
	for val >= unit && i < len(sizes)-1 {
		val /= unit
		i++
	}
	return fmt.Sprintf("%.1f %s", val, sizes[i])
}
