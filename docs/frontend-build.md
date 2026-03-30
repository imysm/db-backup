# 前端打包说明

## 打包方式

db-backup 采用 **前端独立构建 + 运行时指定路径** 的方式：

```
┌─────────────────┐     ┌─────────────────┐
│   Vue 3 前端    │     │   Go 后端       │
│   web/dist/     │ ──→ │   通过参数传入   │
│   (npm build)  │     │   staticPath    │
└─────────────────┘     └─────────────────┘
```

**前端不是嵌入到二进制中**，而是在运行时通过 `-static` 参数指定静态文件目录。

---

## 构建步骤

### 1. 构建前端

```bash
cd web
npm install
npm run build
```

构建产物在 `web/dist/` 目录：

```
dist/
├── index.html
└── assets/
    ├── index-xxxx.js
    ├── Dashboard-xxxx.css
    └── ...
```

### 2. 启动后端并指定静态文件目录

```bash
# 方式一：命令行参数
./db-backup -config config.yaml -static ./web/dist

# 方式二：配置文件
# config.yaml 中配置 static_path

# 方式三：环境变量
DB_BACKUP_STATIC_PATH=./web/dist ./db-backup
```

### 3. 访问

打开浏览器访问 `http://localhost:8080`

---

## Docker 部署

### Dockerfile (已包含前后端)

```dockerfile
# 前端构建
FROM node:20 AS frontend
WORKDIR /app/web
COPY web/package*.json ./
RUN npm install
COPY web/ ./
RUN npm run build

# 后端构建
FROM golang:1.26-alpine AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -o db-backup ./cmd/server

# 运行
FROM alpine:3.19
COPY --from=backend /app/db-backup /usr/local/bin/
COPY --from=frontend /app/web/dist /app/web/dist
EXPOSE 8080
CMD ["db-backup", "-config", "/etc/db-backup/config.yaml"]
```

### 使用已有构建

```bash
# 1. 构建前端
cd web && npm install && npm run build && cd ..

# 2. 构建后端（包含前端）
CGO_ENABLED=0 go build -ldflags="-s -w" -o db-backup ./cmd/server

# 3. 打包 Docker
docker build -t db-backup:latest .
```

---

## 部署脚本

使用 `deploy/deploy.sh` 部署：

```bash
# 完整部署（包含前端构建）
./deploy/deploy.sh v0.2.0
```

脚本会自动：
1. 构建前端 `npm run build`
2. 构建后端 `go build`
3. 复制到系统目录
4. 配置 systemd 服务

---

## 目录结构

```
db-backup/
├── web/                    # Vue 3 前端源码
│   ├── src/
│   ├── dist/              # 前端构建产物
│   ├── package.json
│   └── vite.config.ts
├── cmd/
│   └── server/           # Go 后端入口
├── internal/              # 后端代码
├── configs/              # 配置文件
├── deploy/               # 部署脚本
│   ├── deploy.sh
│   └── db-backup.service
└── db-backup            # 编译后的二进制（需指定 -static）
```

---

## 常见问题

### Q: 前端构建失败

```bash
# 检查 Node 版本
node -v  # 需要 18+

# 重新安装依赖
cd web && rm -rf node_modules && npm install
```

### Q: 静态文件 404

```bash
# 检查路径是否正确
ls -la web/dist/

# 启动时指定正确路径
./db-backup -static ./web/dist
```

### Q: 前端和后端在不同端口

前端可以配置 API 地址：

```bash
# 前端环境变量
VITE_API_BASE_URL=http://localhost:8080
```

---

## 开发模式

前后端分离开发：

```bash
# 终端 1: 后端（不加载静态文件）
./db-backup -config config.yaml

# 终端 2: 前端（开发服务器）
cd web && npm run dev
```

前端开发服务器会代理 API 请求到后端。

---

*最后更新: 2026-03-30*
