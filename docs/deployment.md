# db-backup 部署文档

## 系统要求

| 项目 | 最低要求 |
|------|---------|
| 操作系统 | Linux (amd64/arm64) |
| Go 版本 | 1.22+（仅编译时需要） |
| 磁盘空间 | 视备份数据量而定 |
| 内存 | 256MB+ |

**客户端工具**（按需安装，用于执行备份命令）：

| 数据库 | 所需工具 |
|--------|---------|
| MySQL | `mysql-client` 或 `mysqldump` |
| PostgreSQL | `postgresql-client` 或 `pg_dump` |
| MongoDB | `mongodump` |
| SQL Server | `sqlcmd` 或 `mssql-tools` |
| Oracle | `expdp`（Oracle Instant Client） |

## 方式一：二进制部署（推荐）

### 1. 编译

```bash
cd db-backup
CGO_ENABLED=0 go build -ldflags="-s -w" -o db-backup ./cmd/server
```

### 2. 安装

```bash
# 创建用户
sudo useradd -r -s /sbin/nologin -d /var/lib/db-backup db-backup

# 创建目录
sudo mkdir -p /var/lib/db-backup /var/log/db-backup /etc/db-backup
sudo chown -R db-backup:db-backup /var/lib/db-backup /var/log/db-backup

# 安装二进制
sudo cp db-backup /usr/local/bin/
sudo chmod +x /usr/local/bin/db-backup
```

### 3. 配置

```bash
# 复制配置模板
sudo cp configs/config.yaml /etc/db-backup/config.yaml
sudo cp .env.example /etc/db-backup/.env

# 编辑配置（必须修改数据库连接信息）
sudo vim /etc/db-backup/config.yaml
```

### 4. 验证配置

```bash
db-backup -config /etc/db-backup/config.yaml -validate
```

### 5. 使用 systemd 管理

```bash
# 安装 service 文件
sudo cp deploy/db-backup.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable db-backup
sudo systemctl start db-backup

# 查看状态
sudo systemctl status db-backup

# 查看日志
sudo journalctl -u db-backup -f
```

### 配置文件说明

主配置文件为 YAML 格式，主要结构：

```yaml
global:
  work_dir: "/var/lib/db-backup"    # 备份临时目录
  default_tz: "Asia/Shanghai"       # 默认时区
  max_concurrent: 5                 # 最大并发数
  timeout: 2h                       # 单次备份超时

log:
  level: "info"                     # debug/info/warn/error
  format: "console"                 # console/json
  file: "/var/log/db-backup/app.log"

tasks:
  - id: "mysql-prod"
    name: "生产环境 MySQL 备份"
    enabled: true
    database:
      type: "mysql"
      host: "127.0.0.1"
      port: 3306
      username: "backup_user"
      password: "your_password"
      databases: ["app_db"]
    schedule:
      cron: "0 2 * * *"             # 每天凌晨2点
    retention:
      keep_days: 30                 # 保留30天
    compression:
      enabled: true
      type: "gzip"
      level: 6
    storage:
      type: "local"
      path: "/var/lib/db-backup"
```

## 方式二：Docker 部署

### 1. 构建镜像

```bash
docker build -t db-backup:latest .
```

### 2. 运行容器

```bash
docker run -d \
  --name db-backup \
  --restart unless-stopped \
  -v /etc/db-backup/config.yaml:/etc/db-backup/config.yaml:ro \
  -v /var/lib/db-backup:/var/lib/db-backup \
  -v /var/log/db-backup:/var/log/db-backup \
  db-backup:latest
```

### 3. Docker Compose

```yaml
version: "3.8"
services:
  db-backup:
    image: db-backup:latest
    container_name: db-backup
    restart: unless-stopped
    volumes:
      - ./config.yaml:/etc/db-backup/config.yaml:ro
      - ./backups:/var/lib/db-backup
      - ./logs:/var/log/db-backup
```

## 使用部署脚本

项目提供了自动化部署脚本：

```bash
# 部署最新版本
./deploy/deploy.sh

# 部署指定版本
./deploy/deploy.sh v1.2.0
```

脚本会自动完成：创建用户 → 创建目录 → 编译 → 安装 → 配置 → 启动 → 健康检查。

## 升级步骤

```bash
# 1. 拉取最新代码
git pull origin main

# 2. 使用部署脚本重新部署
./deploy/deploy.sh

# 或者手动升级：
# 编译新版本
CGO_ENABLED=0 go build -ldflags="-s -w" -o db-backup ./cmd/server
sudo cp db-backup /usr/local/bin/
sudo systemctl restart db-backup

# 3. 验证
sudo systemctl status db-backup
db-backup -config /etc/db-backup/config.yaml -version
```

## 常见问题

### Q: 服务启动失败

```bash
# 查看详细日志
sudo journalctl -u db-backup -n 100 --no-pager

# 常见原因：
# 1. 配置文件路径错误 → 检查 -config 参数
# 2. 数据库连接失败 → 使用 -validate 验证
# 3. 目录权限不足 → 检查 /var/lib/db-backup 权限
```

### Q: 备份文件磁盘空间不足

```bash
# 检查磁盘使用
df -h /var/lib/db-backup

# 调整保留天数（config.yaml 中 retention.keep_days）
# 或配置远程存储（S3/OSS/COS）
```

### Q: 如何手动触发备份

```bash
db-backup -config /etc/db-backup/config.yaml -run <任务ID>
```

### Q: IPv6 地址连接失败

确保数据库主机使用方括号格式：`[::1]:3306`。v1.x+ 已自动处理 IPv6 地址格式。

### Q: Docker 中无法连接宿主机数据库

使用 `host.docker.internal` 或宿主机内网 IP 作为数据库地址，不要使用 `127.0.0.1`。
