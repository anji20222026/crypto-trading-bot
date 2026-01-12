#!/bin/bash

# Ollama 测试脚本
# Quick test script for Ollama

echo "=========================================="
echo "🦙 Ollama 快速测试"
echo "=========================================="
echo ""

# 检查 Ollama 服务
echo "📡 检查 Ollama 服务..."
if curl -s http://localhost:11434 > /dev/null 2>&1; then
    echo "✅ Ollama 服务运行正常"
else
    echo "❌ Ollama 服务未运行"
    echo "请先启动: ollama serve"
    exit 1
fi
echo ""

# 列出已安装的模型
echo "📋 已安装的模型:"
ollama list
echo ""

# 检查 DeepSeek 模型
if ollama list | grep -q "deepseek-r1"; then
    echo "✅ 找到 DeepSeek 模型"
else
    echo "❌ 未找到 DeepSeek 模型"
    echo "请先安装: ollama pull deepseek-r1:8b"
    exit 1
fi
echo ""

# 运行测试
echo "🧪 运行 Ollama 集成测试..."
echo ""

cd "$(dirname "$0")/.."

echo "测试 1: 连接检查"
make test-ollama-connection

echo ""
echo "测试 2: 交易决策"
make test-ollama-trading

echo ""
echo "=========================================="
echo "✅ 所有测试完成！"
echo "=========================================="

