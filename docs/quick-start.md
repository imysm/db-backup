# 5 分钟快速上手

本指南帮助你在 5 分钟内快速启动 db-backup 并完成第一次备份。

## 前提条件

- Go 1.21+ 已安装
- MySQL 5.7+ / PostgreSQL 12+ / MongoDB 4.0+ （任一数据库）

## 第一步：下载并构建

```bash
# 克隆项目
git clone https://github.com/imysm/db-backup.git
cd db-backup

# 构建
go build -o db-backup ./cmd/server
```

## 第二步：创建配置文件

创建 `configs/config.yaml`：

```yaml
global:
  work_dir: /tmp/db-backup
  default_tz: Asia/Shanghai
  max_concurrent: 5
  timeout: 2h

log:
  level: info
  format: console

tasks:
  - id: quick-start-backup
    database:
      type: mysql
      host: localhost
      port: 3306
      username: root
      password: your_password
      database: test
    schedule: "0 2 * * *"
    compression:
      enabled: true
      level: 6
```

> 💡 **提示**: 将 `your_password` 替换为你的数据库密码

## 第三步：验证连接

```bash
./db-backup -config configs/config.yaml -validate
```

输出示例：

```
数据库备份系统 vdev
配置文件: configs/config.yaml
任务数量: 1

验证数据库连接...
  ✅ quick-start-backup: OK
```

## 第四步：执行第一次备份

```bash
./db-backup -config configs/config.yaml -run quick-start-backup
```

输出示例：

```
立即执行任务: quick-start-backup
[09:30:00] 开始备份...
[09:30:01] 连接数据库...
[09:30:02] 执行 mysqldump...
[09:30:05] 备份完成: /tmp/db-backup/test_20260320_093000.sql
[09:30:05] 压缩文件...
[09:30:06] 压缩完成: /tmp/db-backup/test_20260320_093000.sql.gz
```

## 第五步：查看备份文件

```bash
ls -lh /tmp/db-backup/
```

输出示例：

```
-rw-r--r-- 1 user user 1.2M Mar 20 09:30 test_20260320_093000.sql.gz
```

## 下一步

- 📖 [完整安装部署](getting-started.md) - 了解生产环境部署
- ⚙️ [配置详解](ops-guide/configuration.md) - 完整配置选项
- 🔄 [备份恢复](user-guide/restore.md) - 如何恢复数据

## 常见问题

### Q: 连接数据库失败？

检查：
1. 数据库服务是否启动
2. 用户名密码是否正确
3. 网络是否连通

### Q: 备份目录权限不足？

```bash
mkdir -p /tmp/db-backup
chmod 755 /tmp/db-backup
```

### Q: 如何使用环境变量？

配置文件支持环境变量：

```yaml
database:
  password: ${MYSQL_PASSWORD}
```

然后：

```bash
export MYSQL_PASSWORD=your_password
./db-backup -config configs/config.yaml -validate
```

---

*预计阅读时间: 5 分钟*
