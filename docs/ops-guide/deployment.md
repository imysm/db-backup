# 部署指南

本文档介绍 db-backup 在各种环境下的部署方式。

## 部署架构

### 单机部署

```
┌─────────────────────────────────────┐
│           单机服务器                  │
│  ┌───────────┐    ┌───────────┐    │
│  │ db-backup │───▶│  Database │    │
│  └─────┬─────┘    └───────────┘    │
│        │                           │
│        ▼                           │
│  ┌───────────┐                     │
│  │  Storage  │ (本地/S3)           │
│  └───────────┘                     │
└─────────────────────────────────────┘
```

### 高可用部署

```
┌──────────────────────────────────────────────────┐
│                   负载均衡器                       │
└────────────────┬─────────────────────────────────┘
                 │
    ┌────────────┼────────────┐
    │            │            │
    ▼            ▼            ▼
┌────────┐  ┌────────┐  ┌────────┐
│ Node 1 │  │ Node 2 │  │ Node 3 │
└───┬────┘  └───┬────┘  └───┬────┘
    │           │           │
    └───────────┼───────────┘
                │
    ┌───────────┼───────────┐
    │           │           │
    ▼           ▼           ▼
┌────────┐ ┌────────┐ ┌────────┐
│ Redis  │ │  MySQL │ │   S3   │
│(分布式锁)│ │ (元数据)│ │ (备份) │
└────────┘ └────────┘ └────────┘
```

---

## 单机部署

### 系统要求

| 组件 | 最低配置 | 推荐配置 |
|------|----------|----------|
| CPU | 1 核 | 2+ 核 |
| 内存 | 512 MB | 1+ GB |
| 磁盘 | 1 GB | 根据备份数据量 |
| OS | Linux/Windows | Ubuntu 20.04+ |

### 安装步骤

```bash
# 1. 创建用户
sudo useradd -r -s /bin/false backup

# 2. 创建目录
sudo mkdir -p /opt/db-backup/{bin,configs,backups,logs}
sudo chown -R backup:backup /opt/db-backup

# 3. 下载或构建
sudo cp db-backup /opt/db-backup/bin/
sudo chmod +x /opt/db-backup/bin/db-backup

# 4. 创建配置文件
sudo cp configs/config.yaml /opt/db-backup/configs/

# 5. 创建 systemd 服务
sudo cat > /etc/systemd/system/db-backup.service << 'EOF'
[Unit]
Description=DB Backup Service
After=network.target

[Service]
Type=simple
User=backup
Group=backup
WorkingDirectory=/opt/db-backup
ExecStart=/opt/db-backup/bin/db-backup -config /opt/db-backup/configs/config.yaml
Restart=on-failure
RestartSec=5s
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

# 6. 启动服务
sudo systemctl daemon-reload
sudo systemctl enable db-backup
sudo systemctl start db-backup

# 7. 查看状态
sudo systemctl status db-backup
```

---

## Docker 部署

### Dockerfile

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o db-backup ./cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/db-backup .
COPY configs/config.yaml /app/configs/

ENV TZ=Asia/Shanghai

EXPOSE 8080

ENTRYPOINT ["./db-backup"]
CMD ["-config", "configs/config.yaml"]
```

### 构建镜像

```bash
docker build -t db-backup:latest .
```

### 运行容器

```bash
docker run -d \
  --name db-backup \
  --restart unless-stopped \
  -v /opt/db-backup/configs:/app/configs \
  -v /opt/db-backup/backups:/backups \
  -e TZ=Asia/Shanghai \
  -p 8080:8080 \
  db-backup:latest
```

---

## Docker Compose 部署

```yaml
version: '3.8'

services:
  db-backup:
    image: db-backup:latest
    container_name: db-backup
    restart: unless-stopped
    volumes:
      - ./configs:/app/configs
      - ./backups:/backups
      - ./logs:/app/logs
    environment:
      - TZ=Asia/Shanghai
      - MYSQL_PASSWORD=${MYSQL_PASSWORD}
    ports:
      - "8080:8080"
    networks:
      - backup-network
    depends_on:
      - redis

  redis:
    image: redis:7-alpine
    container_name: db-backup-redis
    restart: unless-stopped
    volumes:
      - redis-data:/data
    networks:
      - backup-network

networks:
  backup-network:
    driver: bridge

volumes:
  redis-data:
```

启动：

```bash
docker-compose up -d
```

---

## Kubernetes 部署

### ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: db-backup-config
  namespace: backup
data:
  config.yaml: |
    global:
      work_dir: /backups
      max_concurrent: 5
    log:
      level: info
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
```

### Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: db-backup-secret
  namespace: backup
type: Opaque
stringData:
  mysql-password: your_password
```

### PersistentVolumeClaim

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: db-backup-pvc
  namespace: backup
spec:
  accessModes:
    - ReadWriteOnce
  storageClassName: standard
  resources:
    requests:
      storage: 100Gi
```

### Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: db-backup
  namespace: backup
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
        image: db-backup:latest
        ports:
        - containerPort: 8080
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
        resources:
          requests:
            cpu: 100m
            memory: 256Mi
          limits:
            cpu: 500m
            memory: 512Mi
      volumes:
      - name: config
        configMap:
          name: db-backup-config
      - name: backups
        persistentVolumeClaim:
          claimName: db-backup-pvc
```

### Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: db-backup
  namespace: backup
spec:
  selector:
    app: db-backup
  ports:
  - port: 8080
    targetPort: 8080
  type: ClusterIP
```

---

## 高可用部署

### 架构要求

- 多节点部署（至少 3 节点）
- Redis 用于分布式锁
- 共享存储（S3/OSS）

### 配置

```yaml
global:
  work_dir: /backups
  max_concurrent: 5
  ha:
    enabled: true
    redis:
      host: redis-service
      port: 6379
      password: ${REDIS_PASSWORD}
    lock_ttl: 60s
```

### 部署步骤

1. 部署 Redis 集群
2. 配置共享存储
3. 部署多节点 db-backup
4. 配置负载均衡

---

## 监控和日志

### 健康检查

```yaml
# Kubernetes
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

### 日志收集

```yaml
# Docker Compose
services:
  db-backup:
    logging:
      driver: "json-file"
      options:
        max-size: "100m"
        max-file: "3"
```

---

*最后更新: 2026-03-20*
