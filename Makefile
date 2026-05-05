.PHONY: all server agent clean

# 构建
all: server agent

server:
	cd vps-monitor && go build -o bin/vps-monitor-server ./cmd/server

agent:
	cd vps-monitor && go build -o bin/vps-agent ./cmd/agent

# 清理
clean:
	rm -rf vps-monitor/bin/

# 安装依赖
deps:
	cd vps-monitor && go mod tidy

# 测试运行
run-server:
	cd vps-monitor && ./bin/vps-monitor-server -port 8080 -tokens "test-token-123"

run-agent:
	cd vps-monitor && ./bin/vps-agent -server http://localhost:8080 -token test-token-123

# 打包静态资源（可选）
embed:
	cd vps-monitor && go build -tags embed -o bin/vps-monitor-server ./cmd/server