# ❓ 常见问题

本文档收录 db-backup 的常见问题和解决方案。

---

## 安装问题

### Q: Docker 镜像拉取失败？

```bash
# 检查 Docker 状态
systemctl status docker

# 手动拉取
docker pull imysm/db-backup:latest

# 使用代理（如果需要）
docker pull --registry-mirror=https://mirror.gcr.io imysm/db-backup:latest
```

### Q: 编译失败，提示缺少依赖？

```bash
# 确保 Go 版本正确
go version  # 需要 1.22+

# 更新依赖
go mod tidy

# 重新编译
go build -o db-backup ./cmd/server
```

### Q: 前端构建失败？

```bash
cd web

# 清除缓存
rm -rf node_modules package-lock.json

# 重新安装
npm install

# 再次构建
npm run build
```

---

## 运行问题

### Q: 启动后访问不了控制台？

```bash
# 检查端口是否被占用
lsof -i:8080

# 检查防火墙
ufw status
# 如果需要
ufw allow 8080/tcp

# 检查服务是否启动
curl http://localhost:8080/health
```

### Q: 任务不执行？

```bash
# 1. 检查配置
./db-backup -validate

# 2. 检查 Cron 表达式是否正确
# 正确: "0 2 * * *"
# 错误: "2:00" (不正确)

# 3. 检查系统时区
timedatectl

# 4. 查看日志
tail -f /var/log/db-backup/app.log
```

### Q: 备份失败，提示权限不足？

```bash
# 检查目录权限
ls -la /data/backups

# 修改权限
chown -R db-backup:db-backup /data/backups
chmod 755 /data/backups
```

---

## 数据库问题

### Q: SQLite 数据库锁定？

```bash
# 重启服务
systemctl restart db-backup

# 或删除锁文件
rm -f /var/lib/db-backup/db-backup.db-journal
```

### Q: MySQL 连接失败？

```bash
# 检查 MySQL 服务
systemctl status mysql

# 测试连接
mysql -h localhost -u root -p -e "SELECT 1"

# 检查防火墙
ufw allow 3306/tcp
```

### Q: 备份超时？

配置文件增加超时时间：

```yaml
global:
  timeout: 4h  # 默认 2h，增加到 4h
```

---

## 存储问题

### Q: 云存储上传失败？

```bash
# 1. 检查凭证是否正确
# AWS: access_key / secret_key
# 阿里云: access_key / secret_key
# 腾讯云: secret_id / secret_key

# 2. 检查权限
# S3: s3:PutObject
# OSS: oss:PutObject
# COS: cos:PutObject

# 3. 检查网络
curl -I https://s3.amazonaws.com
```

### Q: 存储空间不足？

```bash
# 查看使用情况
df -h

# 清理旧备份
find /data/backups -name "*.bak" -mtime +30 -delete

# 或配置自动清理
retention:
  keep_days: 7
```

---

## 安全问题

### Q: 忘记 Web 控制台密码？

编辑配置文件，添加管理员：

```yaml
admin:
  username: admin
  password: admin123
```

然后重启服务，用新密码登录后修改。

### Q: API 认证失败？

```bash
# 检查 API Key 配置
grep api_keys /etc/db-backup/config.yaml

# 使用正确的 Key
curl -H "Authorization: Bearer YOUR-API-KEY" http://localhost:8080/api/jobs
```

### Q: 加密备份无法解密？

```bash
# 确保密钥一致
# 加密时用的密钥必须和恢复时一样

# 检查密钥格式（必须是 32 字节 hex 字符串）
openssl rand -hex 32
```

---

## 性能问题

### Q: 备份太慢？

```yaml
# 1. 启用压缩（减少传输量）
compression:
  enabled: true
  type: gzip
  level: 1  # 降低级别加速

# 2. 增加并发
global:
  max_concurrent: 10

# 3. 使用增量备份
mode: incremental
```

### Q: 内存占用过高？

```yaml
# 限制并发
global:
  max_concurrent: 3
```

---

## Docker 问题

### Q: 容器内无法访问宿主机数据库？

```bash
# 使用宿主机网络
docker run --network host ...

# 或指定 DNS
docker run --dns 8.8.8.8 ...
```

### Q: 数据卷权限问题？

```bash
# 创建命名卷（Docker 自动管理权限）
docker volume create db-backup-data

# 或预先创建目录并授权
mkdir -p /data/db-backup
chmod 777 /data/db-backup
```

---

## 日志问题

### Q: 日志文件太大？

```yaml
# 使用日志轮转
log:
  level: warn  # 减少日志量
  format: json

# 或配置 logrotate
# /etc/logrotate.d/db-backup
/var/log/db-backup/*.log {
    daily
    rotate 7
    compress
    delaycompress
    notifempty
    create 0644 root root
}
```

---

## 其他问题

### Q: 如何查看版本？

```bash
./db-backup -version
# 输出: db-backup version v0.2.0
```

### Q: 如何查看所有配置？

```bash
./db-backup -validate
# 会打印所有加载的配置
```

### Q: 如何完全卸载？

```bash
# 停止服务
docker stop db-backup && docker rm db-backup

# 删除数据
rm -rf /var/lib/db-backup /etc/db-backup /var/log/db-backup

# 删除镜像
docker rmi imysm/db-backup:latest
```

---

## 获取帮助

- 📖 查看 [文档](README.md)
- 🐛 提交 [Issue](https://github.com/imysm/db-backup/issues)
- 💬 加入讨论

---

*有更多问题？欢迎提交 PR 或 Issue！*
