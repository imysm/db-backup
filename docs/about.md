# About db-backup

<div align="center">

# 🔐 企业级多源数据库备份系统

**db-backup** 是一款开源的企业级数据库备份解决方案，支持 MySQL、PostgreSQL、MongoDB、SQL Server、Oracle 等主流数据库。

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8.svg)](https://golang.org/)
[![Stars](https://img.shields.io/github/stars/imysm/db-backup?style=social)](https://github.com/imysm/db-backup)

</div>

---

## ✨ 核心特性

### 🔒 安全加固
- **AES-256-GCM 加密** — 备份文件端到端加密，密钥强制 32 字节校验
- **密码安全传递** — CNF 文件 / 环境变量，避免密码泄露
- **SQL 注入防护** — 参数消毒 + 白名单验证
- **路径遍历防护** — 严格路径校验
- **登录限速** — 防暴力破解
- **WebSocket Origin 白名单** — 防止跨站请求

### 📦 多数据库支持
| 数据库 | 全量备份 | 增量备份 | 压缩 | 加密 | 恢复 |
|--------|:--------:|:--------:|:----:|:----:|:----:|
| MySQL | ✅ | ✅ | ✅ | ✅ | ✅ |
| PostgreSQL | ✅ | ✅ | ✅ | ✅ | ✅ |
| MongoDB | ✅ | ✅ | ✅ | ✅ | ✅ |
| SQL Server | ✅ | ✅ | ✅ | ✅ | ✅ |
| Oracle | ✅ | — | ✅ | ✅ | ✅ |

### ☁️ 多种存储后端
- 本地存储
- S3 / MinIO
- 阿里云 OSS
- 腾讯云 COS

### 🚀 运维友好
- **Web 管理控制台** — 任务配置、日志、记录一站式管理
- **WebSocket 实时日志** — 备份进度实时推送
- **Prometheus 监控** — 指标暴露，Grafana 可视化
- **Cron 调度** — 灵活配置备份时间
- **Docker / K8s 支持** — 快速部署

---

## 🏗️ 架构

```
┌─────────────────────────────────────────────────────────┐
│                      Web UI                              │
│              (Vue 3 + Element Plus)                      │
├─────────────────────────────────────────────────────────┤
│                      HTTP API                             │
│              (Gin + GORM)                                │
├──────────┬──────────┬──────────┬──────────┬──────────┤
│  MySQL   │ PostgreSQL │ MongoDB  │ SQLServer │ Oracle  │
│ Executor │ Executor  │ Executor  │ Executor  │ Executor │
├──────────┴──────────┴──────────┴──────────┴──────────┤
│                  Scheduler (Cron)                        │
├─────────────────────────────────────────────────────────┤
│           Storage (Local / S3 / OSS / COS)              │
└─────────────────────────────────────────────────────────┘
```

---

## 🔐 安全设计

### 密钥验证流程
```
用户配置密钥 → ValidateEncryptionKey(必须32字节hex)
                        ↓
              加密: EncryptFile
                        ↓
              解密: DecryptFile → VerifyEncryptedFile
```

### 密码传递安全
| 数据库 | 方式 | 说明 |
|--------|------|------|
| MySQL | CNF 文件 (0600) | 替代 MYSQL_PWD |
| PostgreSQL | 环境变量 | PGPASSWORD |
| MongoDB | 环境变量 | MONGOPASSWORD |
| SQL Server | 环境变量 | SQLCMDPASSWORD |

---

## 📊 项目统计

| 指标 | 数值 |
|------|------|
| 代码行数 | 30,000+ |
| 单元测试 | 29 个包覆盖 |
| 支持数据库 | 5 种 |
| 存储后端 | 4 种 |
| 加密算法 | AES-256-GCM |

---

## 🚀 快速开始

### Docker 部署
```bash
docker run -d \
  --name db-backup \
  -p 8080:8080 \
  -v /data/backups:/data/backups \
  -v ./config.yaml:/app/config.yaml \
  imysm/db-backup
```

### 二进制部署
```bash
# 下载 releases
wget https://github.com/imysm/db-backup/releases/latest

# 配置
cp config.yaml.example config.yaml

# 启动
./db-backup -config config.yaml
```

### Web 控制台
访问 `http://localhost:8080`

---

## 📖 文档

- [快速入门](docs/quick-start.md) — 5 分钟上手
- [部署指南](docs/deployment.md) — 生产环境部署
- [安全文档](docs/security.md) — 安全功能详解
- [配置详解](docs/configuration.md) — 完整配置参考
- [API 文档](docs/dev-guide/api.md) — RESTful API

---

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add AmazingFeature'`)
4. 推送分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

---

## 📄 License

MIT License - 详见 [LICENSE](LICENSE) 文件

---

## 🙏 致谢

- [Gin](https://github.com/gin-gonic/gin) - HTTP 框架
- [GORM](https://gorm.io/) - ORM 库
- [Vue 3](https://vuejs.org/) - 前端框架
- [Element Plus](https://element-plus.org/) - UI 组件库
- [robfig/cron](https://github.com/robfig/cron) - Cron 调度器
- [Tencent COS SDK](https://github.com/tencentyun/cos-go-sdk-v5) - 腾讯云 COS SDK
- [aliyun/aliyun-oss-go-sdk](https://github.com/aliyun/aliyun-oss-go-sdk) - 阿里云 OSS SDK
- [aws/aws-sdk-go](https://github.com/aws/aws-sdk-go) - AWS SDK

---

<div align="center">

**如果这个项目对你有帮助，请给个 ⭐**

</div>
