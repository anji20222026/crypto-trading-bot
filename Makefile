.PHONY: build run clean test help query build-web run-web demo test-json

# 默认目标
.DEFAULT_GOAL := help

# 变量定义
BINARY_NAME=crypto-trading-bot
WEB_BINARY=crypto-trading-bot-web
QUERY_BINARY=query
DEMO_BINARY=demo
TEST_JSON_BINARY=test-json-format
BUILD_DIR=bin
CMD_DIR=cmd
MAIN_FILE=$(CMD_DIR)/main.go
WEB_FILE=$(CMD_DIR)/web/main.go
QUERY_FILE=$(CMD_DIR)/query/main.go
DEMO_FILE=$(CMD_DIR)/demo/main.go
TEST_JSON_FILE=$(CMD_DIR)/test_json_format/main.go

## build: 编译项目
build:
	@echo "🔨 编译项目..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_FILE)
	@echo "✅ 编译完成: $(BUILD_DIR)/$(BINARY_NAME)"

## build-all: 编译所有工具
build-all: build
	@echo "🔨 编译查询工具..."
	@go build -o $(BUILD_DIR)/$(QUERY_BINARY) $(QUERY_FILE)
	@echo "✅ 查询工具编译完成: $(BUILD_DIR)/$(QUERY_BINARY)"
	@echo "🔨 编译 Web 监控程序..."
	@go build -o $(BUILD_DIR)/$(WEB_BINARY) $(WEB_FILE)
	@echo "✅ Web 监控程序编译完成: $(BUILD_DIR)/$(WEB_BINARY)"

## build-web: 编译 Web 监控程序
build-web:
	@echo "🔨 编译 Web 监控程序..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(WEB_BINARY) $(WEB_FILE)
	@echo "✅ Web 监控程序编译完成: $(BUILD_DIR)/$(WEB_BINARY)"

## run-web: 编译并运行 Web 监控程序
run-web: build-web
	@echo "🚀 运行 Web 监控程序..."
	@./$(BUILD_DIR)/$(WEB_BINARY)

## run: 编译并运行
run: build
	@echo "🚀 运行程序..."
	@./$(BUILD_DIR)/$(BINARY_NAME)

## query: 编译并运行查询工具
query:
	@go build -o $(BUILD_DIR)/$(QUERY_BINARY) $(QUERY_FILE)
	@./$(BUILD_DIR)/$(QUERY_BINARY) $(ARGS)

## demo: 编译并运行市场数据 Demo
demo:
	@echo "🔨 编译 Demo 程序..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(DEMO_BINARY) $(DEMO_FILE)
	@echo "✅ Demo 编译完成"
	@echo "🚀 运行 Demo..."
	@./$(BUILD_DIR)/$(DEMO_BINARY)

## test-json: 编译并运行 JSON 格式测试
test-json:
	@echo "🔨 编译 JSON 格式测试程序..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(TEST_JSON_BINARY) $(TEST_JSON_FILE)
	@echo "✅ 编译完成"
	@echo "🚀 运行测试..."
	@./$(BUILD_DIR)/$(TEST_JSON_BINARY)

## clean: 清理编译产物
clean:
	@echo "🧹 清理编译产物..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(BINARY_NAME)
	@echo "✅ 清理完成"

## test: 运行测试
test:
	@echo "🧪 运行测试..."
	@go test ./internal/... -v

## test-cover: 运行测试并生成覆盖率报告
test-cover:
	@echo "🧪 运行测试并生成覆盖率报告..."
	@go test ./internal/... -cover
	@go test ./internal/... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ 覆盖率报告已生成: coverage.html"

## test-ollama-simple: 测试 Ollama 极简交易决策（推荐）⭐
test-ollama-simple:
	@echo "🦙 测试 Ollama 极简交易决策..."
	@go test -v -run TestOllamaSimpleTradingDecision ./internal/agents -timeout 60s

## test-ollama-connection: 测试 Ollama 连接和模型列表
test-ollama-connection:
	@echo "🦙 测试 Ollama 连接..."
	@go test -v -run TestOllamaConnection ./internal/agents

## test-ollama-trading: 测试 Ollama DeepSeek 交易决策
test-ollama-trading:
	@echo "🦙 测试 Ollama DeepSeek 交易决策..."
	@go test -v -run TestOllamaDeepSeekTrading ./internal/agents

## test-ollama-all: 运行所有 Ollama 测试
test-ollama-all:
	@echo "🦙 运行所有 Ollama 测试..."
	@go test -v -run TestOllama ./internal/agents -timeout 120s

## deps: 安装依赖
deps:
	@echo "📦 安装依赖..."
	@go mod download
	@go mod tidy
	@echo "✅ 依赖安装完成"

## fmt: 格式化代码
fmt:
	@echo "🎨 格式化代码..."
	@go fmt ./...
	@echo "✅ 格式化完成"

## vet: 代码检查
vet:
	@echo "🔍 代码检查..."
	@go vet ./...
	@echo "✅ 检查完成"

## help: 显示帮助信息
help: Makefile
	@echo " 选择一个命令:"
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo ""
	@echo "💡 提示:"
	@echo "   - 使用 'make test-json' 查看发送给 LLM 的 JSON 格式"
	@echo "   - 使用 'make demo' 查看市场数据结构"
