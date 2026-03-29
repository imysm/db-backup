package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/imysm/db-backup/internal/api/model"
	"gorm.io/gorm"
)

// StatsHandler 统计处理器
type StatsHandler struct {
	db *gorm.DB
}

// NewStatsHandler 创建统计处理器
func NewStatsHandler(db *gorm.DB) *StatsHandler {
	return &StatsHandler{db: db}
}

// DailyStat 每日统计
type DailyStat struct {
	Date    string  `json:"date"`
	Total   int64   `json:"total"`
	Success int64   `json:"success"`
	Failed  int64   `json:"failed"`
}

// StatsResponse 统计响应
type StatsResponse struct {
	TotalTasks        int64        `json:"total_tasks"`
	EnabledTasks      int64        `json:"enabled_tasks"`
	TodayTotal        int64        `json:"today_total"`
	TodaySuccess      int64        `json:"today_success"`
	TodayFailed       int64        `json:"today_failed"`
	TodayRunning      int64        `json:"today_running"`
	WeekSuccessRate   float64      `json:"week_success_rate"`
	TotalStorageBytes int64        `json:"total_storage_bytes"`
	StorageLimitBytes int64        `json:"storage_limit_bytes"`
	Last24hFailed     int64        `json:"last_24h_failed"`
	DailyStats        []DailyStat  `json:"daily_stats"`
}

// GetStats 获取统计信息
func (h *StatsHandler) GetStats(c *gin.Context) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrow := today.Add(24 * time.Hour)
	hoursAgo24 := now.Add(-24 * time.Hour)

	// 任务统计
	var totalTasks, enabledTasks int64
	h.db.Model(&model.BackupJob{}).Count(&totalTasks)
	h.db.Model(&model.BackupJob{}).Where("enabled = ?", true).Count(&enabledTasks)

	// 今日备份记录统计
	var todaySuccess, todayFailed, todayRunning int64
	h.db.Model(&model.BackupRecord{}).Where("started_at >= ? AND started_at < ? AND status = ?", today, tomorrow, model.BackupStatusSuccess).Count(&todaySuccess)
	h.db.Model(&model.BackupRecord{}).Where("started_at >= ? AND started_at < ? AND status = ?", today, tomorrow, model.BackupStatusFailed).Count(&todayFailed)
	h.db.Model(&model.BackupRecord{}).Where("started_at >= ? AND started_at < ? AND status = ?", today, tomorrow, model.BackupStatusRunning).Count(&todayRunning)

	todayTotal := enabledTasks // 今日计划任务数 = 启用的任务数

	// 最近7天成功率
	weekAgo := today.Add(-7 * 24 * time.Hour)
	var weekTotal, weekSuccess int64
	h.db.Model(&model.BackupRecord{}).Where("started_at >= ?", weekAgo).Count(&weekTotal)
	h.db.Model(&model.BackupRecord{}).Where("started_at >= ? AND status = ?", weekAgo, model.BackupStatusSuccess).Count(&weekSuccess)

	var weekSuccessRate float64
	if weekTotal > 0 {
		weekSuccessRate = float64(weekSuccess) / float64(weekTotal) * 100
		// 保留一位小数
		weekSuccessRate = float64(int(weekSuccessRate*10)) / 10
	}

	// 总存储量（成功记录的文件大小之和）
	var totalStorageBytes int64
	h.db.Model(&model.BackupRecord{}).
		Where("status = ?", model.BackupStatusSuccess).
		Select("COALESCE(SUM(file_size), 0)").
		Scan(&totalStorageBytes)

	// 最近24小时失败数
	var last24hFailed int64
	h.db.Model(&model.BackupRecord{}).
		Where("started_at >= ? AND status = ?", hoursAgo24, model.BackupStatusFailed).
		Count(&last24hFailed)

	// 最近7天每日统计
	var dailyStats []DailyStat
	for i := 6; i >= 0; i-- {
		dayStart := today.Add(-time.Duration(i) * 24 * time.Hour)
		dayEnd := dayStart.Add(24 * time.Hour)

		var dayTotal, daySuccess, dayFailed int64
		h.db.Model(&model.BackupRecord{}).Where("started_at >= ? AND started_at < ?", dayStart, dayEnd).Count(&dayTotal)
		h.db.Model(&model.BackupRecord{}).Where("started_at >= ? AND started_at < ? AND status = ?", dayStart, dayEnd, model.BackupStatusSuccess).Count(&daySuccess)
		h.db.Model(&model.BackupRecord{}).Where("started_at >= ? AND started_at < ? AND status = ?", dayStart, dayEnd, model.BackupStatusFailed).Count(&dayFailed)

		dailyStats = append(dailyStats, DailyStat{
			Date:    dayStart.Format("2006-01-02"),
			Total:   dayTotal,
			Success: daySuccess,
			Failed:  dayFailed,
		})
	}

	c.JSON(200, gin.H{
		"code": 0,
		"data": StatsResponse{
			TotalTasks:        totalTasks,
			EnabledTasks:      enabledTasks,
			TodayTotal:        todayTotal,
			TodaySuccess:      todaySuccess,
			TodayFailed:       todayFailed,
			TodayRunning:      todayRunning,
			WeekSuccessRate:   weekSuccessRate,
			TotalStorageBytes: totalStorageBytes,
			StorageLimitBytes: 0,
			Last24hFailed:     last24hFailed,
			DailyStats:        dailyStats,
		},
	})
}
