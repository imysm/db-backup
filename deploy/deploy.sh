#!/usr/bin/env bash
# ============================================
# db-backup 部署脚本
# 用法: ./deploy.sh [版本号]
# ============================================

set -euo pipefail

VERSION="${1:-latest}"
APP_NAME="db-backup"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/db-backup"
DATA_DIR="/var/lib/db-backup"
LOG_DIR="/var/log/db-backup"
SERVICE_FILE="/etc/systemd/system/${APP_NAME}.service"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BINARY="${INSTALL_DIR}/${APP_NAME}"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC}  $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }

# 检查是否 root
if [ "$(id -u)" -ne 0 ]; then
    error "请使用 root 用户运行此脚本"
    exit 1
fi

echo "============================================"
echo " db-backup 部署脚本 v${VERSION}"
echo "============================================"
echo

# 1. 创建系统用户
info "创建系统用户..."
if ! id -u "${APP_NAME}" &>/dev/null; then
    useradd -r -s /sbin/nologin -d "${DATA_DIR}" "${APP_NAME}"
    info "已创建用户: ${APP_NAME}"
else
    info "用户已存在: ${APP_NAME}"
fi

# 2. 创建目录
info "创建目录..."
mkdir -p "${DATA_DIR}" "${LOG_DIR}" "${CONFIG_DIR}"
chown -R "${APP_NAME}:${APP_NAME}" "${DATA_DIR}" "${LOG_DIR}"

# 3. 构建二进制
info "构建二进制..."
cd "${PROJECT_DIR}"
CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o "${BINARY}" ./cmd/server
chmod +x "${BINARY}"
info "二进制已安装到: ${BINARY}"

# 4. 复制配置文件（仅在不存在时）
if [ ! -f "${CONFIG_DIR}/config.yaml" ]; then
    if [ -f "${PROJECT_DIR}/configs/config.yaml" ]; then
        cp "${PROJECT_DIR}/configs/config.yaml" "${CONFIG_DIR}/config.yaml"
        warn "已复制默认配置，请修改 ${CONFIG_DIR}/config.yaml 后重启服务"
    fi
fi

if [ ! -f "${CONFIG_DIR}/.env" ]; then
    if [ -f "${PROJECT_DIR}/.env.example" ]; then
        cp "${PROJECT_DIR}/.env.example" "${CONFIG_DIR}/.env"
        warn "已复制 .env 模板，请按需修改 ${CONFIG_DIR}/.env"
    fi
fi

# 5. 安装 systemd service
info "安装 systemd service..."
cp "${SCRIPT_DIR}/${APP_NAME}.service" "${SERVICE_FILE}"
systemctl daemon-reload
systemctl enable "${APP_NAME}"

# 6. 重启服务
info "重启服务..."
systemctl restart "${APP_NAME}"

# 7. 健康检查
info "等待服务启动..."
sleep 2

if systemctl is-active --quiet "${APP_NAME}"; then
    info "服务状态: ✅ 运行中"
else
    error "服务启动失败，请检查日志: journalctl -u ${APP_NAME} -n 50"
    exit 1
fi

echo
echo "============================================"
info "部署完成！"
echo
echo "  配置文件: ${CONFIG_DIR}/config.yaml"
echo "  环境变量: ${CONFIG_DIR}/.env"
echo "  数据目录: ${DATA_DIR}"
echo "  日志目录: ${LOG_DIR}"
echo
echo "  常用命令:"
echo "    systemctl status ${APP_NAME}"
echo "    journalctl -u ${APP_NAME} -f"
echo "    ${APP_NAME} -config ${CONFIG_DIR}/config.yaml -version"
echo "============================================"
