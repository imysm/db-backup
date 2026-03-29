package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/imysm/db-backup/internal/model"
	"gorm.io/gorm"
)

// HealthCheckHandler 健康检查处理器
type HealthCheckHandler struct {
	db *gorm.DB
}

// NewHealthCheckHandler 创建处理器
func NewHealthCheckHandler(db *gorm.DB) *HealthCheckHandler {
	return &HealthCheckHandler{db: db}
}

// HealthReport 健康报告
type HealthReport struct {
	Score     int           `json:"score"`
	Level     string        `json:"level"`
	Timestamp string        `json:"timestamp"`
	Items     []HealthItem  `json:"items"`
}

// HealthItem 健康检查项
type HealthItem struct {
	Category string `json:"category"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	Message  string `json:"message"`
	Detail   string `json:"detail,omitempty"`
}

// GetHealthReport 获取健康报告
func (h *HealthCheckHandler) GetHealthReport(c *gin.Context) {
	report := HealthReport{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Items:    make([]HealthItem, 0),
		Score:    100,
	}

	h.checkDatabase(&report)
	h.checkBackupJobs(&report)
	h.checkBackupRecords(&report)
	h.calculateScore(&report)

	c.JSON(200, gin.H{"code": 0, "data": report})
}

// checkDatabase 检查数据库
func (h *HealthCheckHandler) checkDatabase(report *HealthReport) {
	item := HealthItem{
		Category: "数据库",
		Name:     "数据库连接",
		Status:   "pass",
		Message:  "数据库连接正常",
	}

	sqlDB, err := h.db.DB()
	if err != nil {
		item.Status = "fail"
		item.Message = "无法获取数据库连接"
	} else if err := sqlDB.Ping(); err != nil {
		item.Status = "fail"
		item.Message = "数据库连接失败"
	}

	report.Items = append(report.Items, item)
}

// checkBackupJobs 检查备份任务
func (h *HealthCheckHandler) checkBackupJobs(report *HealthReport) {
	var total, enabled int64
	h.db.Model(&model.BackupTask{}).Count(&total)
	h.db.Model(&model.BackupTask{}).Where("enabled = ?", true).Count(&enabled)

	item := HealthItem{
		Category: "备份任务",
		Name:     "任务配置",
		Detail:   fmt.Sprintf("共 %d 个任务，%d 个启用", total, enabled),
	}

	if total == 0 {
		item.Status = "warning"
		item.Message = "暂无备份任务"
	} else if enabled == 0 {
		item.Status = "fail"
		item.Message = "所有备份任务已禁用"
	} else {
		item.Message = fmt.Sprintf("配置正常 (%d/%d 启用)", enabled, total)
	}

	report.Items = append(report.Items, item)
}

// checkBackupRecords 检查备份记录
func (h *HealthCheckHandler) checkBackupRecords(report *HealthReport) {
	var total, success, failed int64
	h.db.Model(&model.BackupRecord{}).Count(&total)
	h.db.Model(&model.BackupRecord{}).Where("status = ?", "success").Count(&success)
	h.db.Model(&model.BackupRecord{}).Where("status = ?", "failed").Count(&failed)

	item := HealthItem{
		Category: "备份记录",
		Name:     "执行情况",
		Detail:   fmt.Sprintf("总记录 %d 条，成功 %d 条，失败 %d 条", total, success, failed),
	}

	if total == 0 {
		item.Status = "warning"
		item.Message = "暂无备份记录"
	} else if failed > 0 && float64(failed)/float64(success+failed) > 0.3 {
		item.Status = "warning"
		item.Message = fmt.Sprintf("失败率 %.1f%%", float64(failed)/float64(success+failed)*100)
	} else {
		item.Message = "备份执行正常"
	}

	report.Items = append(report.Items, item)
}

// calculateScore 计算分数
func (h *HealthCheckHandler) calculateScore(report *HealthReport) {
	failCount := 0
	warnCount := 0

	for _, item := range report.Items {
		switch item.Status {
		case "fail":
			failCount++
		case "warning":
			warnCount++
		}
	}

	report.Score = 100 - failCount*20 - warnCount*5
	if report.Score < 0 {
		report.Score = 0
	}

	switch {
	case report.Score >= 80:
		report.Level = "healthy"
	case report.Score >= 50:
		report.Level = "warning"
	default:
		report.Level = "critical"
	}
}

// HealthCheckEndpoint 负载均衡器健康检查
func HealthCheckEndpoint(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}
