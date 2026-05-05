# VPS Monitor

轻量级多服务器监控平台，Go 全栈实现，单二进制部署。

## 架构

```
┌─────────────────────────────────────────────────────────────┐
│                     Agent Layer (每台 VPS)                   │
│  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐              │
│  │Agent │ │Agent │ │Agent │ │Agent │ │Agent │              │
│  └──────┘ └──────┘ └──────┘ └──────┘ └──────┘              │
└─────────────────────────────────────────────────────────────┘
                          ↓ HTTP POST (每 10-30 秒)
┌─────────────────────────────────────────────────────────────┐
│                     Go Server (监控中心)                     │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  REST API + Web UI + SQLite                        │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
                          ↓ 浏览器访问
┌─────────────────────────────────────────────────────────────┐
│                     Web Dashboard                           │
│  实时图表 + 服务器列表 + 历史查询                            │
└─────────────────────────────────────────────────────────────┘
```

每台 VPS 运行一个轻量 Agent，定期采集系统指标并通过 HTTP POST 推送到中心 Server。Server 接收数据存入 SQLite，通过 Web Dashboard 实时展示。

## 功能

- **CPU**：总使用率、用户态、系统态、IO 等待
- **内存**：物理内存、Swap 使用率
- **磁盘**：多分区自动采集，使用率趋势
- **网络**：发送/接收流量、速率计算
- **系统负载**：Load1 / Load5 / Load15
- **服务可用性**：HTTP 响应检测、TCP 端口连通性、SSL 证书有效期

## 快速开始

### 前提

- Go 1.21+（如需自行编译）
- 已编译的二进制文件可从 [Releases](https://github.com/Uuclear/servers-agent/releases) 下载

### 本地运行（macOS / Linux）

```bash
# 1. 启动 Server
./bin/vps-monitor-server -port 8080 -tokens "your-secret-token"

# 2. 打开浏览器访问 Dashboard
#    http://localhost:8080

# 3. 启动 Agent（在同一台机器或远程 VPS 上）
./bin/vps-agent -server http://your-server-ip:8080 -token your-secret-token
```

### 使用配置文件

```bash
# Server
./bin/vps-monitor-server -config configs/server.yaml

# Agent
./bin/vps-agent -config configs/agent.yaml
```

## Linux 生产部署

### 一键安装

```bash
sudo ./scripts/install.sh
```

安装路径：`/opt/vps-monitor/`

### systemd 管理

```bash
# 编辑配置
sudo vim /opt/vps-monitor/config/server.yaml
sudo vim /opt/vps-monitor/config/agent.yaml

# 启动服务
sudo systemctl start vps-monitor-server
sudo systemctl start vps-agent

# 开机自启
sudo systemctl enable vps-monitor-server
sudo systemctl enable vps-agent

# 查看状态
sudo systemctl status vps-monitor-server
sudo journalctl -u vps-monitor-server -f
```

## 配置说明

### Server 配置 (server.yaml)

```yaml
port: 8080                    # 监听端口
database: "./vps-monitor.db"  # SQLite 数据库路径
tokens:                       # Agent 认证 token 列表
  - "your-secret-token-here"
retention_days: 7             # 数据保留天数
web_static: "./web/static"    # 静态文件路径
log_level: "info"             # 日志级别
```

### Agent 配置 (agent.yaml)

```yaml
server: "http://your-server-ip:8080"  # Server 地址
token: "your-secret-token-here"       # 认证 token
interval: 10                          # 采集间隔（秒）
disk_paths:                           # 监控的磁盘路径
  - "/"
  - "/home"
services:                             # 服务健康检查（可选）
  - name: "nginx"
    type: "http"
    target: "http://localhost"
  - name: "mysql"
    type: "tcp"
    target: "localhost:3306"
  - name: "api-ssl"
    type: "ssl"
    target: "example.com:443"
```

## 命令行参数

**Server**：

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-port` | 监听端口 | `8080` |
| `-db` | 数据库路径 | `./vps-monitor.db` |
| `-tokens` | Token 列表（逗号分隔） | 空 |
| `-config` | 配置文件路径（YAML） | 空 |

**Agent**：

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `-server` | Server 地址 | `http://localhost:8080` |
| `-token` | 认证 token | 空 |
| `-interval` | 采集间隔（秒） | `10` |
| `-config` | 配置文件路径（YAML） | 空 |

## 生成 Token

```bash
openssl rand -hex 16
# 输出: 6d1f7a403432190324393c9a85465080
```

## 自行编译

```bash
# 安装依赖
make deps

# 编译 Server 和 Agent（本地架构）
make all

# 交叉编译所有平台（linux/amd64, linux/arm64, darwin/arm64）
make release
```

## 项目结构

```
vps-monitor/
├── cmd/
│   ├── server/main.go         # Server 入口
│   └── agent/main.go          # Agent 入口
├── internal/
│   ├── collector/             # Agent 采集模块
│   ├── config/                # YAML 配置解析
│   ├── models/                # 数据模型
│   └── server/
│       ├── api/               # REST API handlers
│       └── storage/           # SQLite 存储
├── pkg/protocol/              # 通信协议定义
├── web/static/                # Web Dashboard
├── configs/                   # 配置模板
├── scripts/                   # 安装脚本 + systemd 服务
├── Makefile
└── README.md
```

## Dashboard

访问 `http://your-server-ip:8080` 查看：

- 服务器列表 + 实时状态指示（online / offline / warning）
- 概览卡片：总数、在线、离线、告警
- 实时图表：CPU / 内存 / Swap / 负载趋势
- 多磁盘分区可视化
- 服务检测结果（HTTP 状态码、TCP 延迟、SSL 到期天数）
- 自动刷新（每 10 秒）
- 历史时间范围选择：1h / 6h / 24h / 7d / 30d
