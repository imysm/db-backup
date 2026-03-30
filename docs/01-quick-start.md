# 🚀 快速入门

本文档帮助你 **5 分钟** 启动并运行 db-backup。

---

## 前置要求

| 要求 | 说明 |
|------|------|
| Go | 1.22+ (仅编译时需要) |
| 数据库客户端 | mysqldump / pg_dump 等 (根据需要) |
| 磁盘空间 | 视备份数据量而定 |

---

## 方式一：Docker 部署（推荐）

### 1. 启动服务

```bash
docker run -d \
  --name db-backup \
  -p 8080:8080 \
  -v /data/backups:/data/backups \
  -v $(pwd)/config.yaml:/app/config.yaml \
  imysm/db-backup:latest
```

### 2. 访问控制台

打开浏览器访问：**http://localhost:8080**

默认账户：`admin` / `admin123`

### 3. 创建第一个任务

1. 点击左侧「任务管理」
2. 点击「新建任务」
3. 填写数据库连接信息
4. 设置调度时间
5. 点击「保存」

---

## 方式二：二进制部署

### 1. 下载二进制

```bash
# 下载最新版本
wget https://github.com/imysm/db-backup/releases/latest/db-backup-linux-amd64.tar.gz
tar -xzf db-backup-linux-amd64.tar.gz
```

### 2. 配置文件

```bash
mkdir -p /etc/db-backup /var/lib/db-backup
cp config.yaml.example /etc/db-backup/config.yaml
```

编辑 `/etc/db-backup/config.yaml`：

```yaml
global:
  work_dir: /var/lib/db-backup

database:
  type: sqlite
  dsn: /var/lib/db-backup/db-backup.db

tasks:
  - id: mysql-backup
    database:
      type: mysql
      host: localhost
      port: 3306
      username: root
      password: your_password
      database: mydb
    schedule:
      cron: "0 2 * * *"
    storage:
      type: local
      path: /data/backups/mysql
```

### 3. 启动服务

```bash
# 前台运行
./db-backup -web -static ./web/dist -port 8080

# 或后台运行
nohup ./db-backup -web -static ./web/dist -port 8080 &
```

### 4. 访问控制台

打开浏览器访问：**http://localhost:8080**

默认账户：`admin` / `admin123`

---

## 方式三：源码编译

### 1. 克隆代码

```bash
git clone https://github.com/imysm/db-backup.git
cd db-backup
```

### 2. 编译前端

```bash
cd web
npm install
npm run build
cd ..
```

### 3. 编译后端

```bash
go build -o db-backup ./cmd/server
```

### 4. 运行

```bash
./db-backup -web -static ./web/dist -port 8080
```

---

## 常见问题

### Q: 端口 8080 被占用？

```bash
# 使用其他端口
./db-backup -web -port 9090
```

### Q: 忘记密码？

编辑 `config.yaml`，临时添加管理员账户：

```yaml
admin:
  username: admin
  password: admin123
```

### Q: 备份任务不执行？

```bash
# 检查配置是否正确
./db-backup -validate

# 查看日志
tail -f /var/log/db-backup/app.log
```

---

## 下一步

- 📖 [Web 控制台使用](05-usage/web-console.md) - 详细控制台操作
- ⚙️ [配置详解](04-configuration.md) - 完整配置项
- 🔒 [安全配置](06-security.md) - 加密和密码保护

---

## 需要帮助？

- 📝 [常见问题](08-faq.md)
- 🐛 [提交 Issue](https://github.com/imysm/db-backup/issues)
