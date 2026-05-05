#!/bin/bash

# VPS Monitor 安装脚本

set -e

INSTALL_DIR="/opt/vps-monitor"
BIN_DIR="$INSTALL_DIR/bin"
CONFIG_DIR="$INSTALL_DIR/config"
DATA_DIR="$INSTALL_DIR/data"

echo "=== VPS Monitor 安装脚本 ==="

# 创建目录
mkdir -p "$BIN_DIR" "$CONFIG_DIR" "$DATA_DIR"

# 复制二进制文件（假设已编译）
if [ -f "./bin/vps-monitor-server" ]; then
    cp ./bin/vps-monitor-server "$BIN_DIR/"
    echo "已安装 server"
fi

if [ -f "./bin/vps-agent" ]; then
    cp ./bin/vps-agent "$BIN_DIR/"
    echo "已安装 agent"
fi

# 复制配置示例
if [ -f "./configs/server.yaml" ]; then
    cp ./configs/server.yaml "$CONFIG_DIR/"
fi

if [ -f "./configs/agent.yaml" ]; then
    cp ./configs/agent.yaml "$CONFIG_DIR/"
fi

# 安装 systemd 服务
if [ -d "./scripts/systemd" ]; then
    cp ./scripts/systemd/*.service /etc/systemd/system/
    systemctl daemon-reload
    echo "已安装 systemd 服务"
fi

# 设置权限
chmod +x "$BIN_DIR"/*

echo ""
echo "安装完成！"
echo ""
echo "下一步："
echo "1. 编辑 $CONFIG_DIR/server.yaml 配置 server"
echo "2. 编辑 $CONFIG_DIR/agent.yaml 配置 agent"
echo "3. 启动服务: systemctl start vps-monitor-server"
echo "4. 启动 agent: systemctl start vps-agent"
echo ""