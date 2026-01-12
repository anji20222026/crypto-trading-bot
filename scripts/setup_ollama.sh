#!/bin/bash

# Ollama 快速设置脚本
# Quick setup script for Ollama

set -e

echo "=========================================="
echo "🦙 Ollama 快速设置脚本"
echo "=========================================="
echo ""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 检查操作系统
OS="$(uname -s)"
case "${OS}" in
    Linux*)     MACHINE=Linux;;
    Darwin*)    MACHINE=Mac;;
    *)          MACHINE="UNKNOWN:${OS}"
esac

echo -e "${BLUE}检测到操作系统: ${MACHINE}${NC}"
echo ""

# 步骤 1: 检查 Ollama 是否已安装
echo "📋 步骤 1/5: 检查 Ollama 安装状态..."
if command -v ollama &> /dev/null; then
    echo -e "${GREEN}✅ Ollama 已安装${NC}"
    ollama --version
else
    echo -e "${YELLOW}⚠️  Ollama 未安装${NC}"
    echo ""
    echo "请选择安装方式："
    echo "  1) 自动安装（推荐）"
    echo "  2) 手动安装"
    echo "  3) 跳过安装"
    read -p "请输入选项 (1-3): " install_choice
    
    case $install_choice in
        1)
            echo ""
            echo "🔨 开始自动安装 Ollama..."
            if [ "$MACHINE" = "Mac" ] || [ "$MACHINE" = "Linux" ]; then
                curl -fsSL https://ollama.com/install.sh | sh
                echo -e "${GREEN}✅ Ollama 安装完成${NC}"
            else
                echo -e "${RED}❌ 不支持的操作系统，请手动安装${NC}"
                echo "访问: https://ollama.com/download"
                exit 1
            fi
            ;;
        2)
            echo ""
            echo "请访问 https://ollama.com/download 下载并安装 Ollama"
            echo "安装完成后重新运行此脚本"
            exit 0
            ;;
        3)
            echo -e "${YELLOW}⚠️  跳过 Ollama 安装${NC}"
            ;;
        *)
            echo -e "${RED}❌ 无效选项${NC}"
            exit 1
            ;;
    esac
fi
echo ""

# 步骤 2: 检查 Ollama 服务是否运行
echo "📡 步骤 2/5: 检查 Ollama 服务..."
if curl -s http://localhost:11434 > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Ollama 服务运行正常${NC}"
else
    echo -e "${YELLOW}⚠️  Ollama 服务未运行${NC}"
    echo ""
    echo "请选择操作："
    echo "  1) 启动 Ollama 服务（后台运行）"
    echo "  2) 手动启动（在新终端运行 'ollama serve'）"
    echo "  3) 跳过"
    read -p "请输入选项 (1-3): " serve_choice
    
    case $serve_choice in
        1)
            echo ""
            echo "🚀 启动 Ollama 服务..."
            nohup ollama serve > /tmp/ollama.log 2>&1 &
            sleep 2
            if curl -s http://localhost:11434 > /dev/null 2>&1; then
                echo -e "${GREEN}✅ Ollama 服务启动成功${NC}"
            else
                echo -e "${RED}❌ Ollama 服务启动失败，请查看日志: /tmp/ollama.log${NC}"
                exit 1
            fi
            ;;
        2)
            echo ""
            echo "请在新终端运行: ollama serve"
            echo "然后重新运行此脚本"
            exit 0
            ;;
        3)
            echo -e "${YELLOW}⚠️  跳过服务启动${NC}"
            ;;
        *)
            echo -e "${RED}❌ 无效选项${NC}"
            exit 1
            ;;
    esac
fi
echo ""

# 步骤 3: 检查 DeepSeek 模型
echo "🔍 步骤 3/5: 检查 DeepSeek 模型..."
if ollama list | grep -q "deepseek-r1"; then
    echo -e "${GREEN}✅ DeepSeek 模型已安装${NC}"
    ollama list | grep "deepseek-r1"
else
    echo -e "${YELLOW}⚠️  DeepSeek 模型未安装${NC}"
    echo ""
    echo "请选择要安装的模型："
    echo "  1) deepseek-r1:1.5b  (~1 GB,  速度快，质量一般)"
    echo "  2) deepseek-r1:8b    (~5 GB,  平衡选择，推荐) ⭐"
    echo "  3) deepseek-r1:14b   (~8 GB,  质量好，速度慢)"
    echo "  4) deepseek-r1:32b   (~18 GB, 质量最好，需要大内存)"
    echo "  5) 跳过"
    read -p "请输入选项 (1-5): " model_choice
    
    case $model_choice in
        1)
            echo ""
            echo "📥 下载 deepseek-r1:1.5b..."
            ollama pull deepseek-r1:1.5b
            echo -e "${GREEN}✅ 模型下载完成${NC}"
            ;;
        2)
            echo ""
            echo "📥 下载 deepseek-r1:8b..."
            ollama pull deepseek-r1:8b
            echo -e "${GREEN}✅ 模型下载完成${NC}"
            ;;
        3)
            echo ""
            echo "📥 下载 deepseek-r1:14b..."
            ollama pull deepseek-r1:14b
            echo -e "${GREEN}✅ 模型下载完成${NC}"
            ;;
        4)
            echo ""
            echo "📥 下载 deepseek-r1:32b..."
            ollama pull deepseek-r1:32b
            echo -e "${GREEN}✅ 模型下载完成${NC}"
            ;;
        5)
            echo -e "${YELLOW}⚠️  跳过模型下载${NC}"
            ;;
        *)
            echo -e "${RED}❌ 无效选项${NC}"
            exit 1
            ;;
    esac
fi
echo ""

# 步骤 4: 运行测试
echo "🧪 步骤 4/5: 运行连接测试..."
echo ""
read -p "是否运行 Ollama 连接测试？(y/n): " test_choice

if [ "$test_choice" = "y" ] || [ "$test_choice" = "Y" ]; then
    echo ""
    echo "🚀 运行测试..."
    cd "$(dirname "$0")/.."
    make test-ollama-connection
else
    echo -e "${YELLOW}⚠️  跳过测试${NC}"
fi
echo ""

# 步骤 5: 配置 .env
echo "⚙️  步骤 5/5: 配置 .env 文件..."
echo ""
read -p "是否自动配置 .env 使用 Ollama？(y/n): " config_choice

if [ "$config_choice" = "y" ] || [ "$config_choice" = "Y" ]; then
    cd "$(dirname "$0")/.."
    
    # 备份现有 .env
    if [ -f .env ]; then
        cp .env .env.backup.$(date +%Y%m%d_%H%M%S)
        echo -e "${GREEN}✅ 已备份现有 .env 文件${NC}"
    fi
    
    # 创建或更新 .env
    if [ ! -f .env ]; then
        cp .env.example .env
        echo -e "${GREEN}✅ 已创建 .env 文件${NC}"
    fi
    
    # 获取已安装的 DeepSeek 模型
    INSTALLED_MODEL=$(ollama list | grep "deepseek-r1" | head -1 | awk '{print $1}')
    
    if [ -n "$INSTALLED_MODEL" ]; then
        # 更新 .env 配置
        sed -i.bak "s|^LLM_BACKEND_URL=.*|LLM_BACKEND_URL=http://localhost:11434/v1|" .env
        sed -i.bak "s|^OPENAI_API_KEY=.*|OPENAI_API_KEY=ollama|" .env
        sed -i.bak "s|^QUICK_THINK_LLM=.*|QUICK_THINK_LLM=${INSTALLED_MODEL}|" .env
        sed -i.bak "s|^DEEP_THINK_LLM=.*|DEEP_THINK_LLM=${INSTALLED_MODEL}|" .env
        rm .env.bak
        
        echo -e "${GREEN}✅ .env 配置已更新${NC}"
        echo ""
        echo "配置详情："
        echo "  LLM_BACKEND_URL: http://localhost:11434/v1"
        echo "  OPENAI_API_KEY: ollama"
        echo "  QUICK_THINK_LLM: ${INSTALLED_MODEL}"
        echo "  DEEP_THINK_LLM: ${INSTALLED_MODEL}"
    else
        echo -e "${YELLOW}⚠️  未找到已安装的 DeepSeek 模型，请手动配置 .env${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  跳过 .env 配置${NC}"
fi
echo ""

# 完成
echo "=========================================="
echo -e "${GREEN}✅ Ollama 设置完成！${NC}"
echo "=========================================="
echo ""
echo "下一步："
echo "  1. 运行测试: make test-ollama-all"
echo "  2. 启动交易系统: make run-web"
echo "  3. 查看文档: cat OLLAMA_SETUP_GUIDE.md"
echo ""
echo "=========================================="

