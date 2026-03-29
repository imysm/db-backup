# 概述和核心概念

本文档介绍 db-backup 系统的核心概念和基本使用方法。

## 什么是 db-backup？

db-backup 是一个企业级多源数据库备份系统，提供以下核心功能：

- **多数据库支持**：MySQL、PostgreSQL、MongoDB、SQL Server、Oracle
- **灵活调度**：支持 Cron 表达式定时备份
- **多种存储**：本地、S3、OSS、COS
- **加密压缩**：AES-256 加密 + gzip 压缩
- **Web 管理**：HTTP API + 实时日志

## 核心概念

### 任务 (Job)

任务是备份的基本单位，定义了：

- **源数据库**：要备份的数据库连接信息
- **调度规则**：何时执行备份
- **存储位置**：备份文件保存位置
- **保留策略**：备份文件保留时间

### 备份模式

| 模式 | 说明 | 适用场景 |
|------|------|----------|
| **full** | 全量备份 | 定期完整备份 |
| **incremental** | 增量备份 | 频繁备份场景 |

### 存储后端 (Storage)

| 类型 | 说明 | 特点 |
|------|------|------|
| **local** | 本地磁盘 | 访问快，无异地容灾 |
| **s3** | S3/MinIO | 兼容性好 |
| **oss** | 阿里云 OSS | 国内访问快 |
| **cos** | 腾讯云 COS | 国内访问快 |

### 保留策略 (Retention)

| 策略 | 说明 |
|------|------|
| `days` | 保留最近 N 天的备份 |
| `count` | 保留最近 N 个备份 |

---

## 快速参考

### 常用命令

```bash
# 启动服务
./db-backup -config configs/config.yaml

# 验证配置
./db-backup -config configs/config.yaml -validate

# 立即执行任务
./db-backup -config configs/config.yaml -run <task_id>

# 查看版本
./db-backup -version
```

### 常用 API

```bash
# 获取任务列表
curl http://localhost:8080/api/jobs

# 立即执行任务
curl -X POST http://localhost:8080/api/jobs/<task_id>/run

# 获取备份记录
curl http://localhost:8080/api/records
```

---

## 下一步

- 📋 [任务管理](jobs.md) - 学习如何创建和管理备份任务
- 📁 [备份记录](records.md) - 查看和管理备份历史
- 🔄 [数据恢复](restore.md) - 从备份恢复数据

---

*最后更新: 2026-03-20*
