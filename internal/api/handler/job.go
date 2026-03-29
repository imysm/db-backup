package handler

import (
	"net"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"github.com/imysm/db-backup/internal/api/dto"
	"github.com/imysm/db-backup/internal/api/model"
	"github.com/imysm/db-backup/internal/api/service"
	"github.com/imysm/db-backup/internal/crypto"
	"gorm.io/gorm"
)

// JobHandler 备份任务处理器
type JobHandler struct {
	db            *gorm.DB
	backupService *service.BackupService
	encryptor     crypto.Encryptor
}

// NewJobHandler 创建处理器
func NewJobHandler(db *gorm.DB, encryptor crypto.Encryptor) *JobHandler {
	return &JobHandler{
		db:            db,
		backupService: service.NewBackupService(db, encryptor),
		encryptor:     encryptor,
	}
}

// List 获取任务列表
func (h *JobHandler) List(c *gin.Context) {
	var jobs []model.BackupJob

	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("page_size"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	query := h.db.Model(&model.BackupJob{})

	// 筛选参数
	if search := c.Query("search"); search != "" {
		query = query.Where("name LIKE ?", "%"+search+"%")
	}
	if dbType := c.Query("database_type"); dbType != "" {
		query = query.Where("database_type = ?", dbType)
	}
	if enabledStr := c.Query("enabled"); enabledStr != "" {
		enabled := enabledStr == "true"
		query = query.Where("enabled = ?", enabled)
	}

	var total int64
	query.Count(&total)

	if err := query.Offset(offset).Limit(pageSize).Find(&jobs).Error; err != nil {
		c.JSON(500, dto.Error("获取任务列表失败: "+err.Error()))
		return
	}

	c.JSON(200, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"data":      jobs,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// Get 获取单个任务
func (h *JobHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(400, dto.Error("无效的任务ID"))
		return
	}

	var job model.BackupJob
	if err := h.db.First(&job, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(404, dto.Error("任务不存在"))
			return
		}
		c.JSON(500, dto.Error("获取任务失败: "+err.Error()))
		return
	}

	c.JSON(200, dto.Success(job))
}

// Create 创建任务
func (h *JobHandler) Create(c *gin.Context) {
	var req dto.CreateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.Error("请求参数错误: "+err.Error()))
		return
	}

	job := model.BackupJob{
		Name:         req.Name,
		DatabaseType: model.DatabaseType(req.DatabaseType),
		Host:         req.Host,
		Port:         req.Port,
		Database:     req.Database,
		Username:     req.Username,
		Schedule:     req.Schedule,
		StorageType:  model.StorageType(req.StorageType),
		Compress:     req.Compress,
		Encrypt:      req.Encrypt,
		Enabled:      true,
	}

	// 加密密码
	if req.Password != "" {
		encrypted, err := h.encryptor.EncryptString(req.Password)
		if err != nil {
			c.JSON(500, dto.Error("密码加密失败: "+err.Error()))
			return
		}
		job.Password = encrypted
	}

	if req.RetentionDays > 0 {
		job.RetentionDays = req.RetentionDays
	} else {
		job.RetentionDays = 7
	}

	if req.BackupType != "" {
		job.BackupType = model.BackupType(req.BackupType)
	} else {
		job.BackupType = model.BackupTypeFull
	}

	if err := h.db.Create(&job).Error; err != nil {
		c.JSON(500, dto.Error("创建任务失败: "+err.Error()))
		return
	}

	c.JSON(200, dto.Success(job))
}

// Update 更新任务
func (h *JobHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(400, dto.Error("无效的任务ID"))
		return
	}

	var job model.BackupJob
	if err := h.db.First(&job, id).Error; err != nil {
		c.JSON(404, dto.Error("任务不存在"))
		return
	}

	var req dto.UpdateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, dto.Error("请求参数错误: "+err.Error()))
		return
	}

	updates := make(map[string]interface{})

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Schedule != "" {
		updates["schedule"] = req.Schedule
	}
	if req.RetentionDays > 0 {
		updates["retention_days"] = req.RetentionDays
	}
	if req.StorageType != "" {
		updates["storage_type"] = req.StorageType
	}
	updates["compress"] = req.Compress
	updates["encrypt"] = req.Encrypt
	if req.NotifyOnSuccess {
		updates["notify_on_success"] = req.NotifyOnSuccess
	}
	if req.NotifyOnFail {
		updates["notify_on_fail"] = req.NotifyOnFail
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}

	if err := h.db.Model(&job).Updates(updates).Error; err != nil {
		c.JSON(500, dto.Error("更新任务失败: "+err.Error()))
		return
	}

	c.JSON(200, dto.Success(job))
}

// Delete 删除任务
func (h *JobHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(400, dto.Error("无效的任务ID"))
		return
	}

	if err := h.db.Delete(&model.BackupJob{}, id).Error; err != nil {
		c.JSON(500, dto.Error("删除任务失败: "+err.Error()))
		return
	}

	c.JSON(200, dto.Success(nil))
}

// Run 立即运行任务
func (h *JobHandler) Run(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(400, dto.Error("无效的任务ID"))
		return
	}

	// 检查任务是否存在
	var job model.BackupJob
	if err := h.db.First(&job, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(404, dto.Error("任务不存在"))
			return
		}
		c.JSON(500, dto.Error("查询任务失败: "+err.Error()))
		return
	}

	// 检查任务是否已启用
	if !job.Enabled {
		c.JSON(400, dto.Error("任务已禁用，无法执行"))
		return
	}

	// 触发备份服务立即执行
	record, err := h.backupService.RunNow(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(500, dto.Error("触发任务执行失败: "+err.Error()))
		return
	}

	c.JSON(200, dto.Success(gin.H{
		"message":   "任务已触发执行",
		"record_id": record.ID,
		"status":    record.Status,
	}))
}

// TestConnection 测试数据库连接
func (h *JobHandler) TestConnection(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(400, dto.Error("无效的任务ID"))
		return
	}

	var job model.BackupJob
	if err := h.db.First(&job, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(404, dto.Error("任务不存在"))
			return
		}
		c.JSON(500, dto.Error("查询任务失败: "+err.Error()))
		return
	}

	// 使用 net.Dial 测试端口连通性（通用方案，支持所有数据库类型）
	addr := net.JoinHostPort(job.Host, strconv.Itoa(job.Port))
	start := time.Now()

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		c.JSON(200, dto.Success(gin.H{
			"success":    false,
			"message":    "连接失败: " + err.Error(),
			"latency_ms": latency,
		}))
		return
	}
	conn.Close()

	c.JSON(200, dto.Success(gin.H{
		"success":    true,
		"message":    "连接成功",
		"latency_ms": latency,
	}))
}

// NextRuns 计算下次执行时间
func (h *JobHandler) NextRuns(c *gin.Context) {
	cronExpr := c.Query("cron")

	// 如果没有传入 cron 表达式，从任务中获取
	if cronExpr == "" {
		id, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.JSON(400, dto.Error("无效的任务ID"))
			return
		}

		var job model.BackupJob
		if err := h.db.First(&job, id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(404, dto.Error("任务不存在"))
				return
			}
			c.JSON(500, dto.Error("查询任务失败: "+err.Error()))
			return
		}
		cronExpr = job.Schedule
	}

	// 解析 cron 表达式（支持标准 5 字段格式）
	// 添加秒字段前缀以兼容 cron/v3 的 6 字段格式
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(cronExpr)
	if err != nil {
		c.JSON(400, dto.Error("无效的 Cron 表达式: "+err.Error()))
		return
	}

	// 计算接下来 5 次执行时间
	now := time.Now()
	var nextRuns []string
	for i := 0; i < 5; i++ {
		next := schedule.Next(now)
		nextRuns = append(nextRuns, next.Format("2006-01-02 15:04:05"))
		now = next
	}

	c.JSON(200, dto.Success(gin.H{
		"next_runs": nextRuns,
	}))
}
