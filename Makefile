.PHONY: build clean generate dist dist-win7

APP_NAME := jpy
DIST_DIR := dist
GO119 := $(HOME)/go/bin/go1.19.13
WIN7_BUILD_DIR := /tmp/jpy-win7-build

# 版本信息
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d %H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -s -w -X 'jpy-cli/internal/cmd.Version=$(VERSION)' -X 'jpy-cli/internal/cmd.BuildTime=$(BUILD_TIME)' -X 'jpy-cli/internal/cmd.GitCommit=$(GIT_COMMIT)'

build:
	go build -ldflags="$(LDFLAGS)" -o bin/$(APP_NAME) ./cmd/jpy-cli

generate:
	./tools/generate_schema.sh

clean:
	rm -rf bin/
	rm -rf $(DIST_DIR)/

dist:
	mkdir -p $(DIST_DIR)
	@echo "Building for Linux (amd64)..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(APP_NAME)-linux-amd64 ./cmd/jpy-cli
	@echo "Building for Linux (arm64)..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(APP_NAME)-linux-arm64 ./cmd/jpy-cli
	@echo "Building for macOS (amd64)..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(APP_NAME)-darwin-amd64 ./cmd/jpy-cli
	@echo "Building for macOS (arm64)..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(APP_NAME)-darwin-arm64 ./cmd/jpy-cli
	@echo "Building for Windows (amd64)..."
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(APP_NAME)-windows-amd64.exe ./cmd/jpy-cli
	@echo "Done! Artifacts are in $(DIST_DIR)/"

# Win7 兼容版：使用 Go 1.19.13 + 降级依赖编译
# 在临时目录操作，不影响主项目 go.mod
# 注意：cloud 命令依赖 goTool v1.0.51（需要 Go 1.24），Win7 版不包含 cloud 功能
# 注意：stress 命令依赖 adminApi，Win7 版不包含压测功能
dist-win7:
	@echo "Building Win7 compatible version..."
	@rm -rf $(WIN7_BUILD_DIR)
	@mkdir -p $(WIN7_BUILD_DIR)
	@cp -r cmd pkg internal sdk third_party go.mod go.sum $(WIN7_BUILD_DIR)/
	@cp build/win7/logger.go $(WIN7_BUILD_DIR)/pkg/logger/logger.go
	@# 替换 main.go 为简化版（移除 goTool/logs 依赖）
	@echo 'package main\n\nimport "jpy-cli/internal/cmd"\n\nfunc main() {\n\tcmd.Execute()\n}' > $(WIN7_BUILD_DIR)/cmd/jpy-cli/main.go
	@# 移除 cloud 命令（依赖 Go 1.24+ 的 goTool），Win7 版不需要云端功能
	@rm -rf $(WIN7_BUILD_DIR)/internal/cmd/cloud $(WIN7_BUILD_DIR)/pkg/cloud
	@sed -i '' '/cloudCmd "jpy-cli\/internal\/cmd\/cloud"/d' $(WIN7_BUILD_DIR)/internal/cmd/root.go
	@sed -i '' '/rootCmd.AddCommand(cloudCmd.NewCloudCmd())/d' $(WIN7_BUILD_DIR)/internal/cmd/root.go
	@# 移除 stress 命令（依赖 adminApi），Win7 版不需要压测功能
	@rm -rf $(WIN7_BUILD_DIR)/internal/cmd/stress
	@sed -i '' '/"jpy-cli\/internal\/cmd\/stress"/d' $(WIN7_BUILD_DIR)/internal/cmd/root.go
	@sed -i '' '/stress.NewStressCmd()/d' $(WIN7_BUILD_DIR)/internal/cmd/root.go
	@# 移除 adminApi 和 goTool 相关依赖
	@sed -i '' '/adminApi/d' $(WIN7_BUILD_DIR)/go.mod
	@sed -i '' '/cnb.cool\/accbot\/goTool/d' $(WIN7_BUILD_DIR)/go.mod
	@cd $(WIN7_BUILD_DIR) && \
		sed -i '' 's/^go [0-9].*/go 1.19/' go.mod && \
		$(GO119) get golang.org/x/crypto@v0.14.0 golang.org/x/sys@v0.13.0 golang.org/x/term@v0.13.0 golang.org/x/text@v0.13.0 && \
		$(GO119) mod tidy && \
		CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO119) build -ldflags="-s -w -X 'jpy-cli/internal/cmd.Version=$(VERSION)-win7' -X 'jpy-cli/internal/cmd.BuildTime=$(BUILD_TIME)' -X 'jpy-cli/internal/cmd.GitCommit=$(GIT_COMMIT)'" -o $(APP_NAME)-windows-amd64-win7.exe ./cmd/jpy-cli
	@mkdir -p $(DIST_DIR)
	@cp $(WIN7_BUILD_DIR)/$(APP_NAME)-windows-amd64-win7.exe $(DIST_DIR)/
	@echo "Done! Win7 artifact: $(DIST_DIR)/$(APP_NAME)-windows-amd64-win7.exe"
