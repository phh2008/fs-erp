# fs-erp 商品管理系统 构建脚本
# 常用命令: make build / make run / make release / make clean / make help

APP       := fs-erp
BUILD_DIR := dist
VERSION   := $(shell date +%Y%m%d-%H%M%S)
LDFLAGS   := -s -w -H=windowsgui     # 去符号减小体积；windowsgui 隐藏控制台窗口
# 纯 Go sqlite 驱动(glebarez/sqlite)，关闭 CGO 可交叉编译
GOFLAGS   := CGO_ENABLED=0

.DEFAULT_GOAL := help

## build            编译当前平台单文件 fs-erp.exe（GUI 模式，无控制台黑窗口）
.PHONY: build
build:
	$(GOFLAGS) go build -trimpath -ldflags "$(LDFLAGS)" -o $(APP).exe .
	go run ./tools/pefix $(APP).exe

## run              开发模式运行（go run）
.PHONY: run
run:
	go run .

## check            静态检查
.PHONY: check
check:
	go vet ./...

## release          交叉编译 Windows 双架构产物到 dist/（文件名带版本时间戳）
# 说明: linux/macOS 下 systray 依赖 CGO，无法静态交叉编译；如需可设 CGO_ENABLED=1 本机构建
.PHONY: release
release: clean
	mkdir -p $(BUILD_DIR)
	$(GOFLAGS) GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP)-$(VERSION)-windows-amd64.exe .
	$(GOFLAGS) GOOS=windows GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP)-$(VERSION)-windows-arm64.exe .
	go run ./tools/pefix $(BUILD_DIR)/$(APP)-$(VERSION)-windows-amd64.exe
	go run ./tools/pefix $(BUILD_DIR)/$(APP)-$(VERSION)-windows-arm64.exe
	@echo "产物已输出到 $(BUILD_DIR)/"

## autostart-install    编译并设置开机自启
.PHONY: autostart-install
autostart-install: build
	./$(APP).exe --autostart install

## autostart-uninstall  取消开机自启
.PHONY: autostart-uninstall
autostart-uninstall:
	./$(APP).exe --autostart uninstall

## autostart-status     查看自启状态
.PHONY: autostart-status
autostart-status:
	./$(APP).exe --autostart status

## clean            清理构建产物
.PHONY: clean
clean:
	rm -f $(APP).exe
	rm -rf $(BUILD_DIR)

## help             显示帮助
.PHONY: help
help:
	@grep -E '^## ' Makefile | sed 's/## //'
