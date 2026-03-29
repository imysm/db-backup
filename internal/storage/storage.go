// Package storage 提供备份文件存储管理
package storage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	cos "github.com/tencentyun/cos-go-sdk-v5"

	"github.com/imysm/db-backup/internal/model"
)

// Storage 存储接口
type Storage interface {
	// Save 保存文件
	Save(ctx context.Context, localPath, remotePath string) error

	// Download 下载文件
	Download(ctx context.Context, remotePath, localPath string) error

	// List 列出文件
	List(ctx context.Context, prefix string) ([]model.BackupRecord, error)

	// Delete 删除文件
	Delete(ctx context.Context, filePath string) error

	// Exists 检查文件是否存在
	Exists(ctx context.Context, filePath string) (bool, error)

	// Size 获取文件大小
	Size(ctx context.Context, filePath string) (int64, error)

	// GetUsage 获取存储使用量（P0功能）
	GetUsage(ctx context.Context, prefix string) (int64, error)

	// Type 存储类型
	Type() string
}

// NewStorage 创建存储实例
func NewStorage(cfg model.StorageConfig) (Storage, error) {
	switch cfg.Type {
	case "local", "":
		return NewLocalStorage(cfg.Path), nil
	case "s3":
		return NewS3Storage(cfg)
	case "oss":
		return NewOSSStorage(cfg)
	case "cos":
		return NewCOSStorage(cfg)
	default:
		return nil, fmt.Errorf("不支持的存储类型: %s", cfg.Type)
	}
}

// ========== LocalStorage ==========

// LocalStorage 本地存储
type LocalStorage struct {
	BasePath string
}

// NewLocalStorage 创建本地存储
func NewLocalStorage(basePath string) *LocalStorage {
	return &LocalStorage{BasePath: basePath}
}

func (s *LocalStorage) Type() string { return "local" }

func (s *LocalStorage) Save(ctx context.Context, localPath, remotePath string) error {
	destPath := filepath.Join(s.BasePath, remotePath)
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}
	if localPath == destPath {
		return nil
	}
	src, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer src.Close()
	dst, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("复制文件失败: %w", err)
	}
	if err := os.Remove(localPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("复制文件成功但删除源文件失败: %w", err)
	}
	return nil
}

func (s *LocalStorage) Download(ctx context.Context, remotePath, localPath string) error {
	srcPath := filepath.Join(s.BasePath, remotePath)
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer src.Close()
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}
	dst, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("复制文件失败: %w", err)
	}
	return nil
}

func (s *LocalStorage) List(ctx context.Context, prefix string) ([]model.BackupRecord, error) {
	dir := filepath.Join(s.BasePath, prefix)
	var records []model.BackupRecord
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return records, nil
		}
		return nil, fmt.Errorf("读取目录失败: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if !isBackupFile(entry.Name()) {
			continue
		}
		records = append(records, model.BackupRecord{
			FilePath:  filepath.Join(prefix, entry.Name()),
			FileSize:  info.Size(),
			CreatedAt: info.ModTime(),
			Status:    model.TaskStatusSuccess,
		})
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].CreatedAt.After(records[j].CreatedAt)
	})
	return records, nil
}

func (s *LocalStorage) Delete(ctx context.Context, filePath string) error {
	fullPath := filepath.Join(s.BasePath, filePath)
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除文件失败: %w", err)
	}
	return nil
}

func (s *LocalStorage) Exists(ctx context.Context, filePath string) (bool, error) {
	_, err := os.Stat(filepath.Join(s.BasePath, filePath))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (s *LocalStorage) Size(ctx context.Context, filePath string) (int64, error) {
	info, err := os.Stat(filepath.Join(s.BasePath, filePath))
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func (s *LocalStorage) GetUsage(ctx context.Context, prefix string) (int64, error) {
	var totalSize int64
	err := filepath.Walk(filepath.Join(s.BasePath, prefix), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	return totalSize, err
}

// ========== S3Storage (AWS SDK v2) ==========

// S3Storage S3/MinIO 存储
type S3Storage struct {
	client *s3.Client
	bucket string
}

// NewS3Storage 创建 S3 存储
func NewS3Storage(cfg model.StorageConfig) (*S3Storage, error) {
	endpoint := cfg.Endpoint
	region := cfg.Region
	if region == "" {
		region = "us-east-1"
	}

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:           "https://" + endpoint,
			SigningRegion: region,
		}, nil
	})

	cfgOpts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithEndpointResolverWithOptions(customResolver),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
		awsconfig.WithRegion(region),
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), cfgOpts...)
	if err != nil {
		return nil, fmt.Errorf("加载 AWS 配置失败: %w", err)
	}

	clientOpts := []func(*s3.Options){}
	if cfg.ForcePathStyle {
		clientOpts = append(clientOpts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String("https://" + endpoint)
		})
	}

	client := s3.NewFromConfig(awsCfg, clientOpts...)
	return &S3Storage{client: client, bucket: cfg.Bucket}, nil
}

func (s *S3Storage) Type() string { return "s3" }

func (s *S3Storage) Save(ctx context.Context, localPath, remotePath string) error {
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("打开本地文件失败: %w", err)
	}
	defer file.Close()

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(remotePath),
		Body:   file,
	})
	if err != nil {
		return fmt.Errorf("上传失败: %w", err)
	}
	return nil
}

func (s *S3Storage) Download(ctx context.Context, remotePath, localPath string) error {
	resp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(remotePath),
	})
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()

	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}
	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()
	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}
	return nil
}

func (s *S3Storage) List(ctx context.Context, prefix string) ([]model.BackupRecord, error) {
	var records []model.BackupRecord
	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("列出文件失败: %w", err)
		}
		for _, obj := range page.Contents {
			name := filepath.Base(aws.ToString(obj.Key))
			if !isBackupFile(name) {
				continue
			}
			records = append(records, model.BackupRecord{
				FilePath:  aws.ToString(obj.Key),
				FileSize:  derefInt64(obj.Size),
				CreatedAt: derefTime(obj.LastModified),
				Status:    model.TaskStatusSuccess,
			})
		}
	}
	return records, nil
}

func (s *S3Storage) Delete(ctx context.Context, filePath string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(filePath),
	})
	if err != nil {
		return fmt.Errorf("删除失败: %w", err)
	}
	return nil
}

func (s *S3Storage) Exists(ctx context.Context, filePath string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(filePath),
	})
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (s *S3Storage) Size(ctx context.Context, filePath string) (int64, error) {
	resp, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(filePath),
	})
	if err != nil {
		return 0, fmt.Errorf("获取文件大小失败: %w", err)
	}
	return derefInt64(resp.ContentLength), nil
}

func (s *S3Storage) GetUsage(ctx context.Context, prefix string) (int64, error) {
	records, err := s.List(ctx, prefix)
	if err != nil {
		return 0, err
	}
	var totalSize int64
	for _, record := range records {
		totalSize += record.FileSize
	}
	return totalSize, nil
}

// ========== OSSStorage (Aliyun OSS SDK) ==========

// OSSStorage 阿里云 OSS 存储
type OSSStorage struct {
	client *oss.Client
	bucket string
}

// NewOSSStorage 创建 OSS 存储
func NewOSSStorage(cfg model.StorageConfig) (*OSSStorage, error) {
	client, err := oss.New(cfg.OSSEndpoint, cfg.AccessKey, cfg.SecretKey)
	if err != nil {
		return nil, fmt.Errorf("创建 OSS 客户端失败: %w", err)
	}
	return &OSSStorage{client: client, bucket: cfg.OSSBucket}, nil
}

func (s *OSSStorage) Type() string { return "oss" }

func (s *OSSStorage) Save(ctx context.Context, localPath, remotePath string) error {
	bucket, err := s.client.Bucket(s.bucket)
	if err != nil {
		return fmt.Errorf("获取 Bucket 失败: %w", err)
	}
	err = bucket.PutObjectFromFile(remotePath, localPath)
	if err != nil {
		return fmt.Errorf("上传失败: %w", err)
	}
	return nil
}

func (s *OSSStorage) Download(ctx context.Context, remotePath, localPath string) error {
	bucket, err := s.client.Bucket(s.bucket)
	if err != nil {
		return fmt.Errorf("获取 Bucket 失败: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}
	err = bucket.GetObjectToFile(remotePath, localPath)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	return nil
}

func (s *OSSStorage) List(ctx context.Context, prefix string) ([]model.BackupRecord, error) {
	bucket, err := s.client.Bucket(s.bucket)
	if err != nil {
		return nil, fmt.Errorf("获取 Bucket 失败: %w", err)
	}
	var records []model.BackupRecord
	marker := ""
	for {
		lsRes, err := bucket.ListObjects(oss.Prefix(prefix), oss.Marker(marker))
		if err != nil {
			return nil, fmt.Errorf("列出文件失败: %w", err)
		}
		for _, obj := range lsRes.Objects {
			name := filepath.Base(obj.Key)
			if !isBackupFile(name) {
				continue
			}
			records = append(records, model.BackupRecord{
				FilePath:  obj.Key,
				FileSize:  obj.Size,
				CreatedAt: obj.LastModified,
				Status:    model.TaskStatusSuccess,
			})
		}
		if !lsRes.IsTruncated {
			break
		}
		marker = lsRes.NextMarker
	}
	return records, nil
}

func (s *OSSStorage) Delete(ctx context.Context, filePath string) error {
	bucket, err := s.client.Bucket(s.bucket)
	if err != nil {
		return fmt.Errorf("获取 Bucket 失败: %w", err)
	}
	return bucket.DeleteObject(filePath)
}

func (s *OSSStorage) Exists(ctx context.Context, filePath string) (bool, error) {
	bucket, err := s.client.Bucket(s.bucket)
	if err != nil {
		return false, fmt.Errorf("获取 Bucket 失败: %w", err)
	}
	return bucket.IsObjectExist(filePath)
}

func (s *OSSStorage) Size(ctx context.Context, filePath string) (int64, error) {
	bucket, err := s.client.Bucket(s.bucket)
	if err != nil {
		return 0, fmt.Errorf("获取 Bucket 失败: %w", err)
	}
	info, err := bucket.GetObjectDetailedMeta(filePath)
	if err != nil {
		return 0, fmt.Errorf("获取文件信息失败: %w", err)
	}
	cl := info.Get("Content-Length")
	if cl == "" {
		return 0, nil
	}
	return strconv.ParseInt(cl, 10, 64)
}

func (s *OSSStorage) GetUsage(ctx context.Context, prefix string) (int64, error) {
	records, err := s.List(ctx, prefix)
	if err != nil {
		return 0, err
	}
	var totalSize int64
	for _, record := range records {
		totalSize += record.FileSize
	}
	return totalSize, nil
}

// ========== COSStorage (Tencent COS SDK) ==========

// COSStorage 腾讯云 COS 存储
type COSStorage struct {
	client *cos.Client
	bucket string
}

// NewCOSStorage 创建 COS 存储
func NewCOSStorage(cfg model.StorageConfig) (*COSStorage, error) {
	bucketURL, err := url.Parse(fmt.Sprintf("https://%s.cos.%s.myqcloud.com", cfg.COSBucket, cfg.COSRegion))
	if err != nil {
		return nil, fmt.Errorf("解析 Bucket URL 失败: %w", err)
	}
	serviceURL, _ := url.Parse(fmt.Sprintf("https://cos.%s.myqcloud.com", cfg.COSRegion))

	client := cos.NewClient(&cos.BaseURL{BucketURL: bucketURL, ServiceURL: serviceURL}, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  cfg.AccessKey,
			SecretKey: cfg.SecretKey,
		},
	})
	return &COSStorage{client: client, bucket: cfg.COSBucket}, nil
}

func (s *COSStorage) Type() string { return "cos" }

func (s *COSStorage) Save(ctx context.Context, localPath, remotePath string) error {
	_, err := s.client.Object.PutFromFile(ctx, remotePath, localPath, nil)
	if err != nil {
		return fmt.Errorf("上传失败: %w", err)
	}
	return nil
}

func (s *COSStorage) Download(ctx context.Context, remotePath, localPath string) error {
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}
	_, err := s.client.Object.GetToFile(ctx, remotePath, localPath, nil)
	if err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	return nil
}

func (s *COSStorage) List(ctx context.Context, prefix string) ([]model.BackupRecord, error) {
	var records []model.BackupRecord
	var marker string
	for {
		opt := &cos.BucketGetOptions{Prefix: prefix}
		if marker != "" {
			opt.Marker = marker
		}
		res, _, err := s.client.Bucket.Get(ctx, opt)
		if err != nil {
			return nil, fmt.Errorf("列出文件失败: %w", err)
		}
		for _, obj := range res.Contents {
			name := filepath.Base(obj.Key)
			if !isBackupFile(name) {
				continue
			}
			records = append(records, model.BackupRecord{
				FilePath:  obj.Key,
				FileSize:  obj.Size,
				CreatedAt: parseTime(obj.LastModified),
				Status:    model.TaskStatusSuccess,
			})
		}
		if !res.IsTruncated {
			break
		}
		marker = res.NextMarker
	}
	return records, nil
}

func (s *COSStorage) Delete(ctx context.Context, filePath string) error {
	_, err := s.client.Object.Delete(ctx, filePath)
	if err != nil {
		return fmt.Errorf("删除失败: %w", err)
	}
	return nil
}

func (s *COSStorage) Exists(ctx context.Context, filePath string) (bool, error) {
	_, err := s.client.Object.Head(ctx, filePath, nil)
	if err != nil {
		// cos returns error for non-existent objects
		return false, nil
	}
	return true, nil
}

func (s *COSStorage) Size(ctx context.Context, filePath string) (int64, error) {
	resp, err := s.client.Object.Head(ctx, filePath, nil)
	if err != nil {
		return 0, fmt.Errorf("获取文件信息失败: %w", err)
	}
	return resp.ContentLength, nil
}

func (s *COSStorage) GetUsage(ctx context.Context, prefix string) (int64, error) {
	records, err := s.List(ctx, prefix)
	if err != nil {
		return 0, err
	}
	var totalSize int64
	for _, record := range records {
		totalSize += record.FileSize
	}
	return totalSize, nil
}

// ========== Helpers ==========

func derefInt64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

func derefTime(p *time.Time) time.Time {
	if p == nil {
		return time.Time{}
	}
	return *p
}

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	if t.IsZero() {
		t, _ = time.Parse("2006-01-02T15:04:05.000Z", s)
	}
	return t
}

func isBackupFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".sql", ".gz", ".zip", ".dump", ".bak", ".dmp", ".tar", ".archive":
		return true
	default:
		return false
	}
}

// GetFileAge 获取文件年龄（天数）
func GetFileAge(filePath string) (int, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}
	age := time.Since(info.ModTime())
	return int(age.Hours() / 24), nil
}
