.PHONY: all server agent clean deps run-server run-agent release

# 构建目标
all: server agent

server:
	GOOS=$(shell go env GOOS) GOARCH=$(shell go env GOARCH) go build -o bin/vps-monitor-server ./cmd/server

agent:
	GOOS=$(shell go env GOOS) GOARCH=$(shell go env GOARCH) go build -o bin/vps-agent ./cmd/agent

# 清理
clean:
	rm -rf bin/ dist/

# 安装依赖
deps:
	go mod tidy

# 测试运行
run-server:
	./bin/vps-monitor-server -port 8080 -tokens "test-token-123"

run-agent:
	./bin/vps-agent -server http://localhost:8080 -token test-token-123

# 交叉编译 + 打包 release
# 支持的平台
PLATFORMS := linux-amd64 linux-arm64 darwin-arm64

release: clean $(PLATFORMS)

$(PLATFORMS):
	$(eval OS=$(firstword $(subst -, ,$@)))
	$(eval ARCH=$(lastword $(subst -, ,$@)))
	@echo "Building for $(OS)/$(ARCH)..."
	@mkdir -p dist/vps-monitor-$(OS)-$(ARCH)/bin
	@mkdir -p dist/vps-monitor-$(OS)-$(ARCH)/config
	@mkdir -p dist/vps-monitor-$(OS)-$(ARCH)/scripts/systemd
	GOOS=$(OS) GOARCH=$(ARCH) go build -o dist/vps-monitor-$(OS)-$(ARCH)/bin/vps-monitor-server ./cmd/server
	GOOS=$(OS) GOARCH=$(ARCH) go build -o dist/vps-monitor-$(OS)-$(ARCH)/bin/vps-agent ./cmd/agent
	@cp configs/server.yaml dist/vps-monitor-$(OS)-$(ARCH)/config/
	@cp configs/agent.yaml dist/vps-monitor-$(OS)-$(ARCH)/config/
	@cp scripts/install.sh dist/vps-monitor-$(OS)-$(ARCH)/scripts/
	@cp scripts/systemd/*.service dist/vps-monitor-$(OS)-$(ARCH)/scripts/systemd/
	@cp README.md dist/vps-monitor-$(OS)-$(ARCH)/
	@cd dist && tar -czf vps-monitor-$(OS)-$(ARCH).tar.gz vps-monitor-$(OS)-$(ARCH)/
	@echo "Packaged dist/vps-monitor-$(OS)-$(ARCH).tar.gz"
