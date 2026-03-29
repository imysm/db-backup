// Package retention 提供备份文件保留策略管理
package retention

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/imysm/db-backup/internal/model"
	"github.com/imysm/db-backup/internal/storage"
)

// Policy 保留策略
type Policy struct {
	KeepLast    int
	KeepDays    int
	KeepWeekly  int
	KeepMonthly int
}

// NewPolicy 创建保留策略
func NewPolicy(cfg model.RetentionConfig) *Policy {
	return &Policy{
		KeepLast:    cfg.KeepLast,
		KeepDays:    cfg.KeepDays,
		KeepWeekly:  cfg.KeepWeekly,
		KeepMonthly: cfg.KeepMonthly,
	}
}

// Apply 应用保留策略，返回需要删除的文件列表
func (p *Policy) Apply(ctx context.Context, store storage.Storage, taskID string) ([]string, error) {
	// 获取所有备份记录
	records, err := store.List(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("列出备份文件失败: %w", err)
	}

	if len(records) == 0 {
		return nil, nil
	}

	// 按时间降序排列
	sort.Slice(records, func(i, j int) bool {
		return records[i].CreatedAt.After(records[j].CreatedAt)
	})

	// 需要保留的文件集合
	keepSet := make(map[string]bool)
	now := time.Now()

	// 1. 保留最近 N 个
	if p.KeepLast > 0 {
		for i := 0; i < min(p.KeepLast, len(records)); i++ {
			keepSet[records[i].FilePath] = true
		}
	}

	// 2. 保留最近 N 天的备份
	if p.KeepDays > 0 {
		cutoff := now.AddDate(0, 0, -p.KeepDays)
		for _, rec := range records {
			if rec.CreatedAt.After(cutoff) {
				keepSet[rec.FilePath] = true
			}
		}
	}

	// 3. 保留每周备份（每周保留一个，最近的N周）
	if p.KeepWeekly > 0 {
		weeklyMap := make(map[string]model.BackupRecord) // year-week -> record
		for _, rec := range records {
			y, w := rec.CreatedAt.ISOWeek()
			key := fmt.Sprintf("%d-W%02d", y, w)
			if _, exists := weeklyMap[key]; !exists {
				weeklyMap[key] = rec
			}
		}

		// 取最近的 N 周
		weeks := make([]string, 0, len(weeklyMap))
		for k := range weeklyMap {
			weeks = append(weeks, k)
		}
		sort.Sort(sort.Reverse(sort.StringSlice(weeks)))

		for i := 0; i < min(p.KeepWeekly, len(weeks)); i++ {
			keepSet[weeklyMap[weeks[i]].FilePath] = true
		}
	}

	// 4. 保留每月备份（每月保留一个，最近的N月）
	if p.KeepMonthly > 0 {
		monthlyMap := make(map[string]model.BackupRecord) // year-month -> record
		for _, rec := range records {
			key := rec.CreatedAt.Format("2006-01")
			if _, exists := monthlyMap[key]; !exists {
				monthlyMap[key] = rec
			}
		}

		// 取最近的 N 月
		months := make([]string, 0, len(monthlyMap))
		for k := range monthlyMap {
			months = append(months, k)
		}
		sort.Sort(sort.Reverse(sort.StringSlice(months)))

		for i := 0; i < min(p.KeepMonthly, len(months)); i++ {
			keepSet[monthlyMap[months[i]].FilePath] = true
		}
	}

	// 计算需要删除的文件
	var toDelete []string
	for _, rec := range records {
		if !keepSet[rec.FilePath] {
			toDelete = append(toDelete, rec.FilePath)
		}
	}

	return toDelete, nil
}

// Cleanup 执行清理
func (p *Policy) Cleanup(ctx context.Context, store storage.Storage, taskID string) (int, int, error) {
	toDelete, err := p.Apply(ctx, store, taskID)
	if err != nil {
		return 0, 0, err
	}

	success := 0
	failed := 0

	for _, filePath := range toDelete {
		if err := store.Delete(ctx, filePath); err != nil {
			failed++
		} else {
			success++
		}
	}

	return success, failed, nil
}
