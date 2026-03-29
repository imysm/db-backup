# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added
- MySQL 压缩配置支持 (--compress)
- AES 加密功能 (AES-256-GCM)
- Prometheus 监控指标
- WebSocket 实时日志
- Context 传播支持取消操作
- 统一日志库 (internal/logger)

### Fixed
- 密码暴露安全问题 (使用 MYSQL_PWD 环境变量)
- 硬编码临时目录问题 (可配置 DB_BACKUP_TEMP_DIR)
- 循环逻辑错误
- crypto 测试与实现签名不匹配
- cmd/server 测试默认值判断

### Changed
- 替换 fmt.Print 为统一日志库
- 完善 Context 传播
- 优化资源清理逻辑

### Tests
- 补充 backup 模块测试用例 (覆盖率 23.3% → 61.7%)
- 修复 cmd/server 测试用例
- 修复 crypto 测试用例

---

## [0.1.0] - 2026-03-18

### Added
- 项目初始化
- 基础架构设计
- MySQL 备份执行器
- PostgreSQL 备份执行器
- MongoDB 备份执行器
- SQL Server 备份执行器
- Oracle 备份执行器
- 本地存储后端
- S3/MinIO 存储后端
- 阿里云 OSS 存储后端
- 腾讯云 COS 存储后端
- Cron 调度器
- HTTP API
- 备份恢复功能
- 备份验证功能

[Unreleased]: https://github.com/imysm/db-backup/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/imysm/db-backup/releases/tag/v0.1.0
