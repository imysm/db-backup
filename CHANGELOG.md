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

## [0.2.0] - 2026-03-30

### Security (P0/P1 高危修复)

- **AES-256-GCM 加密增强**: 强制密钥长度必须为 32 字节，防止密钥验证形同虚设
- **VerifyEncryptedFile 修复**: 统一密钥处理逻辑（不再偷偷填充/截断）
- **LoginRateLimiter goroutine leak**: 修复 Stop() 无法等待清理协程退出的问题
- **COS GetSignedURL 错误处理**: 空密钥时返回明确错误而非静默失败
- **SQL Server Restore REPLACE**: 由用户明确控制，不再硬编码
- **MySQL 密码传递**: CNF 文件权限 0600，使用 CreateTemp
- **MongoDB 密码传递**: 通过 MONGOPASSWORD 环境变量传递
- **SQL 注入防护**: sanitizeDBName + ValidateDatabaseName
- **路径遍历防护**: ValidateBackupPath + filepath.Join

### Features

- **备份加密**: MySQL/PostgreSQL/SQLServer/MongoDB 全类型支持
- **Throttle 限速**: CPU/IO 限速 + 并发控制
- **PreScript/PostScript**: 备份前后脚本执行
- **SQL Server Restorer**: 完整恢复器，支持链式恢复

### Tests

- 单元测试覆盖 29 个包全部通过
- logger_test.go 修复 root 权限测试问题

### Docs

- 新增 `docs/security.md`: 安全功能完整文档
- 新增 `docs/user-guide/security.md`: 用户安全配置指南
- README.md 更新安全特性列表

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

[Unreleased]: https://github.com/imysm/db-backup/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/imysm/db-backup/releases/tag/v0.2.0
[0.1.0]: https://github.com/imysm/db-backup/releases/tag/v0.1.0
