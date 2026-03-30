package config

import (
	"fmt"
	"os"

	"github.com/imysm/db-backup/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitDB 初始化数据库连接
func InitDB(cfg model.AppDatabaseConfig) (*gorm.DB, error) {
	// 确保目录存在
	if err := os.MkdirAll(getDBDir(cfg.DSN), 0755); err != nil {
		return nil, fmt.Errorf("创建数据库目录失败: %w", err)
	}

	var dialector gorm.Dialector
	var err error

	switch cfg.Type {
	case "sqlite", "sqlite3":
		dialector = sqlite.Open(cfg.DSN)
	case "mysql":
		dialector = mysql.Open(cfg.DSN)
	case "postgres", "postgresql":
		dialector = postgres.Open(cfg.DSN)
	default:
		return nil, fmt.Errorf("不支持的数据库类型: %s", cfg.Type)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 设置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库连接失败: %w", err)
	}

	maxConns := cfg.MaxConns
	if maxConns <= 0 {
		maxConns = 10
	}
	sqlDB.SetMaxOpenConns(maxConns)
	sqlDB.SetMaxIdleConns(maxConns / 2)

	return db, nil
}

// getDBDir 从 DSN 中提取目录
func getDBDir(dsn string) string {
	// SQLite DSN: /path/to/file.db
	if len(dsn) > 0 && dsn[0] == '/' {
		for i := len(dsn) - 1; i >= 0; i-- {
			if dsn[i] == '/' {
				return dsn[:i]
			}
		}
	}
	return "/tmp"
}
