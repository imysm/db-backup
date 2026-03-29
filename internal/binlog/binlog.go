package binlog

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// BinlogRecord binlog 记录
type BinlogRecord struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	JobID     uint      `json:"job_id" gorm:"not null;index"`

	// binlog 信息
	Filename   string    `json:"filename" gorm:"size:255;not null"`
	Position   int64     `json:"position" gorm:"not null"`
	ServerID  uint      `json:"server_id" gorm:"not null"`

	// 时间范围
	FirstEventTime time.Time `json:"first_event_time"`
	LastEventTime  time.Time `json:"last_event_time"`

	// 文件信息
	FileSize   int64  `json:"file_size" gorm:"not null"`
	EventCount int64  `json:"event_count" gorm:"default:0"`
	Checksum   string `json:"checksum" gorm:"size:100"`

	// 状态
	Status    string `json:"status" gorm:"size:20;default:archived"` // archived / deleted / error
	ArchivedAt time.Time `json:"archived_at"`
	DeletedAt *time.Time `json:"deleted_at"`

	// 审计
	CreatedAt time.Time `json:"created_at"`
}

func (BinlogRecord) TableName() string {
	return "binlog_records"
}

// PITRConfig PITR 配置
type PITRConfig struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	JobID       uint      `json:"job_id" gorm:"not null;uniqueIndex"`

	// binlog 归档配置
	BinlogRetentionDays int `json:"binlog_retention_days" gorm:"default:7"` // 保留天数
	ArchiveEnabled     bool `json:"archive_enabled" gorm:"default:false"`

	// PITR 配置
	PITREnabled       bool   `json:"pitr_enabled" gorm:"default:false"`
	EarliestRecoveryTime *time.Time `json:"earliest_recovery_time"`

	// 审计
	CreatedBy string    `json:"created_by" gorm:"size:100"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (PITRConfig) TableName() string {
	return "pitr_configurations"
}

// BinlogInfo binlog 信息
type BinlogInfo struct {
	Filename   string
	Position   int64
	ServerID  uint
	FileSize  int64
	FirstTime time.Time
	LastTime  time.Time
}

// ArchiveConfig 归档配置
type ArchiveConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	// 归档模式: local / cos
	Mode     string
	LocalDir string
	COSPath  string
}

// Storage binlog 存储访问
type Storage struct {
	db *gorm.DB
}

// NewStorage 创建存储访问
func NewStorage(db *gorm.DB) *Storage {
	return &Storage{db: db}
}

// CreateBinlogRecord 创建 binlog 记录
func (s *Storage) CreateBinlogRecord(record *BinlogRecord) error {
	record.CreatedAt = time.Now()
	record.ArchivedAt = time.Now()
	return s.db.Create(record).Error
}

// GetBinlogByFilename 根据文件名获取记录
func (s *Storage) GetBinlogByFilename(filename string) (*BinlogRecord, error) {
	var record BinlogRecord
	err := s.db.Where("filename = ? AND status != ?", filename, "deleted").
		First(&record).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &record, nil
}

// ListBinlogsByJob 列出任务的 binlog 记录
func (s *Storage) ListBinlogsByJob(jobID uint, status string, limit int) ([]BinlogRecord, error) {
	var records []BinlogRecord
	query := s.db.Where("job_id = ?", jobID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Order("first_event_time ASC").Find(&records).Error
	return records, err
}

// GetLatestBinlog 获取最新的 binlog
func (s *Storage) GetLatestBinlog(jobID uint) (*BinlogRecord, error) {
	var record BinlogRecord
	err := s.db.Where("job_id = ? AND status = ?", jobID, "archived").
		Order("last_event_time DESC").
		First(&record).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &record, nil
}

// UpdateBinlogStatus 更新 binlog 状态
func (s *Storage) UpdateBinlogStatus(id uint, status string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if status == "deleted" {
		now := time.Now()
		updates["deleted_at"] = &now
	}
	return s.db.Model(&BinlogRecord{}).Where("id = ?", id).Updates(updates).Error
}

// GetPITRConfig 获取 PITR 配置
func (s *Storage) GetPITRConfig(jobID uint) (*PITRConfig, error) {
	var config PITRConfig
	err := s.db.Where("job_id = ?", jobID).First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &config, nil
}

// CreateOrUpdatePITRConfig 创建或更新 PITR 配置
func (s *Storage) CreateOrUpdatePITRConfig(config *PITRConfig) error {
	existing, err := s.GetPITRConfig(config.JobID)
	if err != nil {
		return err
	}

	if existing != nil {
		config.ID = existing.ID
		config.CreatedAt = existing.CreatedAt
		config.UpdatedAt = time.Now()
		return s.db.Save(config).Error
	}

	config.CreatedAt = time.Now()
	config.UpdatedAt = time.Now()
	return s.db.Create(config).Error
}

// BinlogFetcher binlog 拉取器
type BinlogFetcher struct {
	config *ArchiveConfig
}

// NewBinlogFetcher 创建 binlog 拉取器
func NewBinlogFetcher(config *ArchiveConfig) *BinlogFetcher {
	return &BinlogFetcher{config: config}
}

// FetchBinlogs 拉取 binlog 文件
func (f *BinlogFetcher) FetchBinlogs(startFile string) ([]string, error) {
	// 构建 mysqlbinlog 命令
	args := []string{
		"--raw",
		"--read-from-remote-server",
		"--to-last-log",
		"--host=" + f.config.Host,
		"--port=" + strconv.Itoa(f.config.Port),
		"--user=" + f.config.User,
		"--password=" + f.config.Password,
		"--result-dir=" + f.config.LocalDir,
	}

	if startFile != "" {
		args = append(args, startFile)
	}

	cmd := exec.Command("mysqlbinlog", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("mysqlbinlog 执行失败: %w, output: %s", err, string(output))
	}

	// 列出新拉取的 binlog 文件
	files, err := f.listFetchedFiles(startFile)
	if err != nil {
		return nil, err
	}

	return files, nil
}

// listFetchedFiles 列出新拉取的文件
func (f *BinlogFetcher) listFetchedFiles(sinceFile string) ([]string, error) {
	var files []string
	entries, err := os.ReadDir(f.config.LocalDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "mysql-bin.") {
			if sinceFile == "" || name > sinceFile {
				files = append(files, filepath.Join(f.config.LocalDir, name))
			}
		}
	}

	return files, nil
}

// ParseBinlogIndex 解析 binlog 索引
func ParseBinlogIndex(output string) ([]BinlogInfo, error) {
	var binlogs []BinlogInfo
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 解析格式: mysql-bin.000001
		if strings.HasPrefix(line, "mysql-bin.") {
			binlogs = append(binlogs, BinlogInfo{
				Filename: line,
			})
		}
	}

	return binlogs, scanner.Err()
}

// GetBinlogInfoFromFile 从 binlog 文件获取信息
func GetBinlogInfoFromFile(path string) (*BinlogInfo, error) {
	cmd := exec.Command("mysqlbinlog", "--verbose", "--base64-output=DECODE-ROWS", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("解析 binlog 失败: %w", err)
	}

	info := &BinlogInfo{
		Filename: filepath.Base(path),
	}

	// 解析文件大小
	if stat, err := os.Stat(path); err == nil {
		info.FileSize = stat.Size()
	}

	// 解析第一行时间戳
	re := regexp.MustCompile(`#(\d{6}\s+\d{1,2}:\d{2}:\d{2})\s+server\s+id\s+(\d+)`)
	matches := re.FindStringSubmatch(string(output))
	if len(matches) >= 3 {
		timeStr := matches[1]
		serverID, _ := strconv.ParseUint(matches[2], 10, 64)
		info.ServerID = uint(serverID)
		info.FirstTime, _ = time.Parse("060102 15:04:05", timeStr)
	}

	// 解析最后位置
	posRe := regexp.MustCompile(`#Position: (\d+)`)
	posMatch := posRe.FindStringSubmatch(string(output))
	if len(posMatch) >= 2 {
		info.Position, _ = strconv.ParseInt(posMatch[1], 10, 64)
	}

	return info, nil
}
