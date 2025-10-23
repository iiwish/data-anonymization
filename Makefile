.PHONY: build run test clean install help

# 默认目标
.DEFAULT_GOAL := help

# 变量定义
BINARY_NAME=data-anonymization
MAIN_PATH=./cmd/server
CONFIG_FILE=config.json

# 帮助信息
help:
	@echo "可用的命令:"
	@echo "  make install    - 安装依赖"
	@echo "  make build      - 编译程序"
	@echo "  make run        - 运行程序"
	@echo "  make test       - 运行测试"
	@echo "  make test-v     - 运行测试（详细输出）"
	@echo "  make clean      - 清理编译文件"
	@echo "  make setup      - 初始化项目（复制配置文件）"

# 安装依赖
install:
	@echo "安装依赖..."
	go get github.com/google/uuid
	go mod tidy
	@echo "依赖安装完成"

# 初始化项目
setup:
	@echo "初始化项目..."
	@if not exist $(CONFIG_FILE) (copy config.example.json $(CONFIG_FILE) && echo 已创建配置文件 config.json) else (echo 配置文件已存在)
	@if not exist logs (mkdir logs && echo 已创建日志目录 logs) else (echo 日志目录已存在)
	@echo "项目初始化完成"

# 编译
build:
	@echo "编译程序..."
	go build -o $(BINARY_NAME).exe $(MAIN_PATH)
	@echo "编译完成: $(BINARY_NAME).exe"

# 运行
run: build
	@echo "启动服务..."
	.\$(BINARY_NAME).exe -config $(CONFIG_FILE)

# 运行测试
test:
	@echo "运行测试..."
	go test ./...

# 运行测试（详细输出）
test-v:
	@echo "运行测试（详细输出）..."
	go test -v ./...

# 运行测试并生成覆盖率报告
test-coverage:
	@echo "生成测试覆盖率报告..."
	go test -coverprofile=coverage.txt ./...
	go tool cover -html=coverage.txt -o coverage.html
	@echo "覆盖率报告已生成: coverage.html"

# 清理
clean:
	@echo "清理编译文件..."
	@if exist $(BINARY_NAME).exe (del $(BINARY_NAME).exe && echo 已删除 $(BINARY_NAME).exe)
	@if exist coverage.txt (del coverage.txt && echo 已删除 coverage.txt)
	@if exist coverage.html (del coverage.html && echo 已删除 coverage.html)
	@echo "清理完成"

# 完整构建流程
all: clean install build test
	@echo "完整构建完成"