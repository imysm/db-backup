# db-backup

<div align="center">

**企业级多源数据库备份系统**，支持 MySQL、PostgreSQL、MongoDB、SQL Server、Oracle。

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8.svg)](https://golang.org/)
[![Stars](https://img.shields.io/github/stars/imysm/db-backup?style=social)](https://github.com/imysm/db-backup)

</div>

---

## ✨ 特性

- 🔐 **AES-256-GCM 加密** — 密钥强制 32 字节校验
- 🛡️ **安全加固** — SQL 注入防护、密码安全传递、登录限速
- 📦 **多数据库** — MySQL、PostgreSQL、MongoDB、SQL Server、Oracle
- ☁️ **多存储后端** — 本地、S3、OSS、COS
- 🌐 **Web 控制台** — 任务配置、实时日志、历史记录
- 📊 **Prometheus 监控** — 内置监控指标

---

## 🚀 快速开始

### Docker（推荐）

```bash
docker run -d \
  --name db-backup \
  -p 8080:8080 \
  -v /data/backups:/data/backups \
  imysm/db-backup:latest
```

### 二进制

```bash
# 下载
wget https://github.com/imysm/db-backup/releases/latest/db-backup-linux-amd64.tar.gz
tar -xzf db-backup-linux-amd64.tar.gz

# 启动
./db-backup -web -static ./web/dist -port 8080
```

访问 **http://localhost:8080**，默认账户 `admin` / `admin123`。

详细步骤请参阅 [快速入门](docs/01-quick-start.md)。

---

## 📖 文档

| 文档 | 说明 |
|------|------|
| [快速入门](docs/01-quick-start.md) | 5 分钟快速上手 |
| [安装指南](docs/02-installation.md) | Docker/二进制/K8s 安装 |
| [部署指南](docs/03-deployment.md) | Nginx 反代、高可用部署 |
| [配置详解](docs/04-configuration.md) | 完整配置项说明 |
| [Web 控制台](docs/05-usage/web-console.md) | 控制台使用手册 |
| [安全配置](docs/06-security.md) | 加密、密码、登录安全 |
| [API 参考](docs/07-api.md) | RESTful API 文档 |
| [常见问题](docs/08-faq.md) | FAQ 故障排查 |

---

## 🗄️ 支持的数据库

| 数据库 | 全量备份 | 增量备份 | 压缩 | 加密 | 恢复 |
|--------|:--------:|:--------:|:----:|:----:|:----:|
| MySQL | ✅ | ✅ | ✅ | ✅ | ✅ |
| PostgreSQL | ✅ | ✅ | ✅ | ✅ | ✅ |
| MongoDB | ✅ | ✅ | ✅ | ✅ | ✅ |
| SQL Server | ✅ | ✅ | ✅ | ✅ | ✅ |
| Oracle | ✅ | — | ✅ | ✅ | ✅ |

---

## ☁️ 支持的存储

| 存储 | 说明 |
|------|------|
| 本地存储 | 磁盘路径 |
| S3 / MinIO | AWS S3 兼容 |
| 阿里云 OSS | 国内访问快 |
| 腾讯云 COS | 国内访问快 |

---

## 🔧 构建（源码）

```bash
# 克隆
git clone https://github.com/imysm/db-backup.git
cd db-backup

# 编译前端
cd web && npm install && npm run build && cd ..

# 编译后端
go build -o db-backup ./cmd/server

# 运行
./db-backup -web -static ./web/dist
```

---

## 📊 API

```
GET  /health           # 健康检查
GET  /api/jobs        # 任务列表
POST /api/jobs        # 创建任务
GET  /api/jobs/:id    # 任务详情
PUT  /api/jobs/:id    # 更新任务
DELETE /api/jobs/:id  # 删除任务
POST /api/jobs/:id/run  # 立即执行
GET  /api/records     # 备份记录
POST /api/restore     # 执行恢复
GET  /metrics         # Prometheus 指标
GET  /ws              # WebSocket 日志
```

详细请参阅 [API 参考](docs/07-api.md)。

---

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

1. Fork 本仓库
2. 创建特性分支
3. 提交更改
4. 推送分支
5. 创建 Pull Request

详细请参阅 [贡献指南](docs/contributing.md)。

---

## 📄 License

MIT License

---

*有问题？提交 [Issue](https://github.com/imysm/db-backup/issues) 或加入讨论！*
