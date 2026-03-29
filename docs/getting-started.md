# 完整安装部署

本指南详细介绍 db-backup 在各种环境下的安装和部署方式。

## 系统要求

### 硬件要求

| 组件 | 最低要求 | 推荐配置 |
|------|----------|----------|
| CPU | 1 核 | 2+ 核 |
| 内存 | 512 MB | 1+ GB |
| 磁盘 | 1 GB | 根据备份数据量 |

### 软件要求

| 软件 | 版本 | 必需 |
|------|------|------|
| Go | 1.21+ | 运行源码 |
| MySQL | 5.7+ | 备份 MySQL |
| PostgreSQL | 12+ | 备份 PostgreSQL |
| MongoDB | 4.0+ | 备份 MongoDB |

### 客户端工具

备份数据库需要安装对应的客户端工具：

| 数据库 | 工具 | 安装命令 |
|--------|------|----------|
| MySQL | mysql, mysqldump | `apt install mysql-client` |
| PostgreSQL | psql, pg_dump | `apt install postgresql-client` |
| MongoDB | mongo, mongodump | `apt install mongodb-clients` |

---

## 安装方式

### 方式一：从源码构建

```bash
# 1. 安装 Go 1.21+
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# 2. 克隆项目
git clone https://github.com/imysm/db-backup.git
cd db-backup

# 3. 下载依赖
go mod download

# 4. 构建
go build -o db-backup ./cmd/server

# 5. 验证
./db-backup -version
```

### 方式二：Docker

```bash
# 1. 拉取镜像
docker pull imysm/db-backup:latest

# 2. 创建配置目录
mkdir -p /opt/db-backup/configs
mkdir -p /opt/db-backup/backups

# 3. 创建配置文件
cat > /opt/db-backup/configs/config.yaml << EOF
global:
  work_dir: /backups
tasks:
  - id: mysql-backup
    database:
      type: mysql
      host: host.docker.internal
      port: 3306
      username: root
      password: password
      database: mydb
    schedule: "0 2 * * *"
EOF

# 4. 运行容器
docker run -d \
  --name db-backup \
  --restart unless-stopped \
  -v /opt/db-backup/configs:/app/configs \
  -v /opt/db-backup/backups:/backups \
  -e TZ=Asia/Shanghai \
  imysm/db-backup:latest
```

### 方式三：Docker Compose

创建 `docker-compose.yml`：

```yaml
version: '3.8'

services:
  db-backup:
    image: imysm/db-backup:latest
    container_name: db-backup
    restart: unless-stopped
    volumes:
      - ./configs:/app/configs
      - ./backups:/backups
    environment:
      - TZ=Asia/Shanghai
      - MYSQL_PASSWORD=${MYSQL_PASSWORD}
    networks:
      - backup-network

networks:
  backup-network:
    driver: bridge
```

启动：

```bash
docker-compose up -d
```

### 方式四：Kubernetes

创建 `k8s-deployment.yaml`：

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: db-backup-config
data:
  config.yaml: |
    global:
      work_dir: /backups
    tasks:
      - id: mysql-backup
        database:
          type: mysql
          host: mysql-service
          port: 3306
          username: root
          password: ${MYSQL_PASSWORD}
          database: mydb
        schedule: "0 2 * * *"
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: db-backup-pvc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 100Gi
---
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
        volumeMounts:
        - name: config
          mountPath: /app/configs
        - name: backups
          mountPath: /backups
        env:
        - name: TZ
          value: "Asia/Shanghai"
        - name: MYSQL_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db-backup-secret
              key: mysql-password
      volumes:
      - name: config
        configMap:
          name: db-backup-config
      - name: backups
        persistentVolumeClaim:
          claimName: db-backup-pvc
```

部署：

```bash
kubectl apply -f k8s-deployment.yaml
```

### 方式五：Systemd 服务

```bash
# 1. 复制二进制文件
sudo cp db-backup /usr/local/bin/
sudo chmod +x /usr/local/bin/db-backup

# 2. 创建配置目录
sudo mkdir -p /etc/db-backup
sudo mkdir -p /var/lib/db-backup

# 3. 复制配置文件
sudo cp configs/config.yaml /etc/db-backup/

# 4. 创建 systemd 服务
sudo cat > /etc/systemd/system/db-backup.service << EOF
[Unit]
Description=DB Backup Service
After=network.target

[Service]
Type=simple
User=backup
Group=backup
WorkingDirectory=/var/lib/db-backup
ExecStart=/usr/local/bin/db-backup -config /etc/db-backup/config.yaml
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
EOF

# 5. 创建用户
sudo useradd -r -s /bin/false backup
sudo chown -R backup:backup /var/lib/db-backup

# 6. 启动服务
sudo systemctl daemon-reload
sudo systemctl enable db-backup
sudo systemctl start db-backup

# 7. 查看状态
sudo systemctl status db-backup
```

---

## 目录结构

安装完成后的目录结构：

```
/opt/db-backup/
├── bin/
│   └── db-backup           # 二进制文件
├── configs/
│   └── config.yaml         # 配置文件
├── backups/                # 备份文件存储
├── logs/                   # 日志文件
└── data/
    └── metadata.db         # 元数据库
```

---

## 配置

### 基础配置

编辑 `configs/config.yaml`：

```yaml
global:
  work_dir: /opt/db-backup/backups
  default_tz: Asia/Shanghai
  max_concurrent: 5
  timeout: 2h

log:
  level: info
  format: console
  file: /opt/db-backup/logs/app.log

tasks:
  - id: mysql-prod-backup
    database:
      type: mysql
      host: localhost
      port: 3306
      username: root
      password: ${MYSQL_PASSWORD}
      database: mydb
    schedule: "0 2 * * *"
    compression:
      enabled: true
      level: 6
```

### 环境变量

创建 `.env` 文件：

```bash
# 数据库密码
MYSQL_PASSWORD=your_password
PGPASSWORD=your_password
MONGO_PASSWORD=your_password

# 加密密钥
DB_BACKUP_ENCRYPTION_KEY=your_32_char_encryption_key

# 云存储
AWS_ACCESS_KEY=your_access_key
AWS_SECRET_KEY=your_secret_key
```

---

## 验证安装

```bash
# 1. 检查版本
./db-backup -version

# 2. 验证配置
./db-backup -config configs/config.yaml -validate

# 3. 测试备份
./db-backup -config configs/config.yaml -run mysql-prod-backup
```

---

## 升级

### 从源码升级

```bash
cd db-backup
git pull origin main
go build -o db-backup ./cmd/server
sudo systemctl restart db-backup
```

### Docker 升级

```bash
docker pull imysm/db-backup:latest
docker-compose down
docker-compose up -d
```

---

## 卸载

### Systemd 服务

```bash
sudo systemctl stop db-backup
sudo systemctl disable db-backup
sudo rm /etc/systemd/system/db-backup.service
sudo systemctl daemon-reload
sudo rm -rf /opt/db-backup
```

### Docker

```bash
docker-compose down
docker rmi imysm/db-backup:latest
rm -rf /opt/db-backup
```

---

## 下一步

- ⚙️ [配置详解](ops-guide/configuration.md)
- 📊 [监控告警](ops-guide/monitoring.md)
- 🚀 [高可用部署](ops-guide/deployment.md)

---

*最后更新: 2026-03-20*
