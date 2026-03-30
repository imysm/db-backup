# 🚢 部署指南

本文档介绍 db-backup 的各种生产环境部署方式。

---

## 部署架构

```
                    ┌─────────────────┐
                    │   Nginx/Caddy   │
                    │   (反向代理)     │
                    └────────┬────────┘
                             │
              ┌─────────────┴─────────────┐
              │                            │
    ┌─────────▼─────────┐    ┌────────────▼────────┐
    │    Web 控制台      │    │     Web 控制台      │
    │   (8080端口)      │    │    (多实例部署)     │
    └─────────┬─────────┘    └────────────┬────────┘
              │                            │
              └─────────────┬──────────────┘
                            │
              ┌─────────────▼─────────────┐
              │       SQLite/MySQL        │
              │      (元数据存储)          │
              └───────────────────────────┘
```

---

## 部署方式

### 方式一：Docker Compose（推荐）

#### 1. 创建目录

```bash
mkdir -p /opt/db-backup/{config,data,logs}
```

#### 2. 创建配置文件

```bash
cat > /opt/db-backup/config/config.yaml << 'EOF'
global:
  work_dir: /data/backups
  default_tz: Asia/Shanghai
  max_concurrent: 5

database:
  type: sqlite
  dsn: /data/db-backup.db

log:
  level: info
  file: /var/log/db-backup/app.log

tasks: []
EOF
```

#### 3. 创建 docker-compose.yml

```yaml
version: '3.8'

services:
  db-backup:
    image: imysm/db-backup:latest
    container_name: db-backup
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ./config:/app/config
      - ./data:/data
      - ./logs:/var/log/db-backup
    environment:
      - TZ=Asia/Shanghai
```

#### 4. 启动

```bash
cd /opt/db-backup
docker-compose up -d
```

---

### 方式二：Nginx 反向代理

#### 1. 安装 Nginx

```bash
apt install -y nginx
```

#### 2. 配置反向代理

```nginx
# /etc/nginx/sites-available/db-backup

upstream db_backup {
    server 127.0.0.1:8080;
    keepalive 32;
}

server {
    listen 80;
    server_name db-backup.yourdomain.com;

    # SSL 配置（推荐）
    # ssl_certificate     /etc/ssl/certs/yourdomain.crt;
    # ssl_certificate_key /etc/ssl/private/yourdomain.key;
    # ssl_protocols       TLSv1.2 TLSv1.3;
    # ssl_ciphers         HIGH:!aNULL:!MD5;

    location / {
        proxy_pass http://db_backup;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket 支持
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";

        # 超时配置
        proxy_read_timeout 300s;
        proxy_connect_timeout 75s;
    }

    # 静态文件缓存
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2)$ {
        proxy_pass http://db_backup;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

#### 3. 启用配置

```bash
ln -s /etc/nginx/sites-available/db-backup /etc/nginx/sites-enabled/
nginx -t
systemctl reload nginx
```

---

### 方式三：Caddy 反向代理

```nginx
# /etc/caddy/Caddyfile

db-backup.yourdomain.com {
    reverse_proxy localhost:8080

    # 可选：配置 SSL 自动
    # tls admin@yourdomain.com
}
```

---

## 高可用部署

### 多实例部署

```yaml
version: '3.8'

services:
  db-backup-1:
    image: imysm/db-backup:latest
    deploy:
      replicas: 2
    volumes:
      - ./config:/app/config
      - ./data:/data
      - ./logs1:/var/log/db-backup

  nginx:
    image: nginx:alpine
    ports:
      - "8080:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
```

### 使用 MySQL 替代 SQLite

```yaml
version: '3.8'

services:
  db-backup:
    image: imysm/db-backup:latest
    environment:
      - DB_HOST=mysql-server
      - DB_PORT=3306
      - DB_NAME=db_backup
      - DB_USER=db_backup
      - DB_PASSWORD=xxx
```

配置：

```yaml
database:
  type: mysql
  dsn: "user:password@tcp(mysql-server:3306)/db_backup?charset=utf8mb4&parseTime=True&loc=Local"
  max_conns: 20
```

---

## 性能优化

### 系统参数调优

```bash
# /etc/sysctl.conf

# 网络优化
net.core.somaxconn = 1024
net.ipv4.tcp_tw_reuse = 1
net.ipv4.tcp_fin_timeout = 30

# 文件描述符
fs.file-max = 65536
```

应用配置：

```bash
sysctl -p
```

### 资源限制

```bash
# /etc/security/limits.conf

db-backup soft nofile 65536
db-backup hard nofile 65536
db-backup soft nproc 4096
db-backup hard nproc 4096
```

---

## 监控配置

### Prometheus 抓取配置

```yaml
# /etc/prometheus/prometheus.yml

scrape_configs:
  - job_name: 'db-backup'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: /metrics
```

### Grafana 仪表盘

导入预定义的仪表盘或创建自定义：

| 指标 | 说明 |
|------|------|
| `db_backup_tasks_total` | 任务总数 |
| `db_backup_task_duration_seconds` | 任务执行时间 |
| `db_backup_storage_used_bytes` | 存储使用量 |
| `db_backup_errors_total` | 错误总数 |

---

## 备份与恢复

### 系统备份

```bash
# 备份配置和数据
tar -czf db-backup-backup.tar.gz \
  /etc/db-backup \
  /var/lib/db-backup \
  /var/log/db-backup
```

### 系统恢复

```bash
# 恢复
tar -xzf db-backup-backup.tar.gz -C /
systemctl restart db-backup
```

---

## 安全加固

### 防火墙配置

```bash
# 只允许 80/443 端口
ufw allow 80/tcp
ufw allow 443/tcp
ufw deny 8080/tcp
ufw enable
```

### SSL/TLS 证书

使用 Let's Encrypt：

```bash
apt install -y certbot python3-certbot-nginx

certbot --nginx -d db-backup.yourdomain.com
```

---

## 故障排查

### 查看日志

```bash
# Docker
docker logs db-backup

# Systemd
journalctl -u db-backup -f

# 文件
tail -f /var/log/db-backup/app.log
```

### 常见问题

| 问题 | 解决方案 |
|------|----------|
| 8080 端口被占用 | `lsof -i:8080` 查找进程 |
| 权限不足 | 检查目录权限 `chown -R db-backup:db-backup /data/backups` |
| 数据库锁定 | 重启服务 `systemctl restart db-backup` |

---

## 下一步

- 📖 [快速入门](01-quick-start.md) - 快速启动
- ⚙️ [配置详解](04-configuration.md) - 完整配置项
- 🔒 [安全配置](06-security.md) - 安全加固

---

*有问题？查看 [常见问题](08-faq.md) 或提交 [Issue](https://github.com/imysm/db-backup/issues)。*
