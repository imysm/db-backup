# 📦 安装指南

本文档详细介绍 db-backup 的各种安装方式。

---

## 系统要求

| 项目 | 最低要求 | 推荐 |
|------|---------|------|
| 操作系统 | Linux (amd64/arm64) | Linux (amd64) |
| Go 版本 | 1.22+ | 1.26 |
| 内存 | 256MB | 1GB+ |
| 磁盘空间 | 视备份数据量 | 100GB+ |

---

## 客户端工具（可选）

根据要备份的数据库类型安装相应工具：

| 数据库 | 必需工具 | 安装命令 |
|--------|---------|----------|
| MySQL | mysqldump | `apt install mysql-client` |
| PostgreSQL | pg_dump | `apt install postgresql-client` |
| MongoDB | mongodump | `apt install mongodb-clients` |
| SQL Server | sqlcmd | [微软文档](https://docs.microsoft.com/en-us/sql/linux/sql-server-linux-setup-tools) |
| Oracle | expdp | Oracle Instant Client |

---

## 安装方式

### 方式一：Docker（推荐）

#### 1. 安装 Docker

```bash
# Ubuntu/Debian
curl -fsSL https://get.docker.com | sh

# 启动 Docker
systemctl enable --now docker
```

#### 2. 拉取镜像

```bash
docker pull imysm/db-backup:latest
```

#### 3. 创建配置文件

```bash
mkdir -p /etc/db-backup /data/backups
```

#### 4. 启动容器

```bash
docker run -d \
  --name db-backup \
  --restart unless-stopped \
  -p 8080:8080 \
  -v /etc/db-backup:/app/config \
  -v /data/backups:/data/backups \
  imysm/db-backup:latest
```

#### 5. 配置持久化

创建 `/etc/db-backup/config.yaml`：

```yaml
global:
  work_dir: /data/backups

database:
  type: sqlite
  dsn: /data/db-backup.db

log:
  level: info
  file: /var/log/db-backup/app.log

tasks:
  # 添加你的备份任务
```

---

### 方式二：二进制安装

#### 1. 下载二进制

```bash
# 下载最新版本
VERSION=$(curl -s https://api.github.com/repos/imysm/db-backup/releases/latest | grep tag_name | cut -d'"' -f4)
wget https://github.com/imysm/db-backup/releases/download/${VERSION}/db-backup-linux-amd64.tar.gz
tar -xzf db-backup-linux-amd64.tar.gz
```

#### 2. 安装

```bash
# 安装到系统目录
cp db-backup /usr/local/bin/
chmod +x /usr/local/bin/db-backup

# 创建必要目录
mkdir -p /etc/db-backup /var/lib/db-backup /var/log/db-backup
```

#### 3. 配置 systemd 服务

创建 `/etc/systemd/system/db-backup.service`：

```ini
[Unit]
Description=db-backup
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/db-backup -web -config /etc/db-backup/config.yaml -static /opt/db-backup/web/dist
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
```

启动服务：

```bash
systemctl enable db-backup
systemctl start db-backup
systemctl status db-backup
```

---

### 方式三：源码编译

#### 1. 安装 Go

```bash
# Ubuntu/Debian
apt update
apt install -y golang-go

# 或使用官方安装脚本
wget https://go.dev/dl/go1.26.0.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.26.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

#### 2. 克隆代码

```bash
git clone https://github.com/imysm/db-backup.git
cd db-backup
```

#### 3. 编译前端

```bash
cd web
npm install
npm run build
cd ..
```

#### 4. 编译后端

```bash
CGO_ENABLED=0 go build -ldflags="-s -w" -o db-backup ./cmd/server
```

#### 5. 安装

```bash
cp db-backup /usr/local/bin/
cp -r web/dist /opt/db-backup/web
```

---

### 方式四：Kubernetes 部署

#### 1. 创建配置 Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: db-backup-config
type: Opaque
stringData:
  config.yaml: |
    global:
      work_dir: /data/backups
    database:
      type: sqlite
      dsn: /data/db-backup.db
```

#### 2. 创建 Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: db-backup
spec:
  replicas: 1
  selector:
    matchLabels:
      app: db-backup
  template:
    metadata:
      labels:
        app: db-backup
    spec:
      containers:
      - name: db-backup
        image: imysm/db-backup:latest
        ports:
        - containerPort: 8080
        volumeMounts:
        - name: config
          mountPath: /app/config
        - name: data
          mountPath: /data
      volumes:
      - name: config
        secret:
          secretName: db-backup-config
      - name: data
        persistentVolumeClaim:
          claimName: db-backup-data
```

---

## 验证安装

### 检查版本

```bash
db-backup -version
```

输出类似：
```
db-backup version v0.2.0 (built at 2026-03-30)
```

### 检查健康状态

```bash
curl http://localhost:8080/health
```

输出：
```json
{"status":"ok","version":"v0.2.0"}
```

### 访问控制台

打开浏览器访问：**http://localhost:8080**

---

## 卸载

### Docker 卸载

```bash
docker stop db-backup
docker rm db-backup
docker rmi imysm/db-backup:latest
```

### 二进制卸载

```bash
rm /usr/local/bin/db-backup
rm -rf /etc/db-backup /var/lib/db-backup /var/log/db-backup
systemctl stop db-backup
systemctl disable db-backup
rm /etc/systemd/system/db-backup.service
```

---

## 下一步

- 📖 [快速入门](01-quick-start.md) - 启动第一个备份
- 🚢 [部署指南](03-deployment.md) - 生产环境部署
- ⚙️ [配置详解](04-configuration.md) - 完整配置项

---

*有问题？查看 [常见问题](08-faq.md) 或提交 [Issue](https://github.com/imysm/db-backup/issues)。*
