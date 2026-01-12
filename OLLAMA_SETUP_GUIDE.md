# 🦙 Ollama 本地 LLM 配置指南

本指南介绍如何配置和使用本地 Ollama 运行 DeepSeek 模型进行加密货币交易。

---

## 📋 目录

1. [为什么使用 Ollama？](#为什么使用-ollama)
2. [安装步骤](#安装步骤)
3. [配置 .env](#配置-env)
4. [运行测试](#运行测试)
5. [性能优化](#性能优化)
6. [常见问题](#常见问题)

---

## 为什么使用 Ollama？

### ✅ 优势

1. **完全免费**：无需 API Key，无使用限制
2. **数据隐私**：所有数据在本地处理，不会上传到云端
3. **离线运行**：无需网络连接即可使用
4. **可定制**：可以微调模型以适应特定交易策略
5. **无速率限制**：不受 API 调用次数限制

### ⚠️ 劣势

1. **响应较慢**：本地模型比云端 API 慢（8B 模型约 10-30 秒）
2. **硬件要求**：需要足够的 RAM 和 CPU/GPU
3. **质量差异**：小模型（1.5B/8B）质量可能不如云端大模型
4. **维护成本**：需要自己管理模型更新和硬件

---

## 安装步骤

### 1️⃣ 安装 Ollama

#### macOS
```bash
curl -fsSL https://ollama.com/install.sh | sh
```

#### Linux
```bash
curl -fsSL https://ollama.com/install.sh | sh
```

#### Windows
下载并安装：https://ollama.com/download

### 2️⃣ 启动 Ollama 服务

```bash
ollama serve
```

**验证服务运行：**
```bash
curl http://localhost:11434
# 应该返回: Ollama is running
```

### 3️⃣ 拉取 DeepSeek 模型

```bash
# 推荐：8B 参数版本（约 4.7 GB）
ollama pull deepseek-r1:8b
```

**其他可选版本：**
```bash
ollama pull deepseek-r1:1.5b   # 更小，速度更快（~1 GB）
ollama pull deepseek-r1:14b    # 更大，效果更好（~8 GB）
ollama pull deepseek-r1:32b    # 最大，需要更多内存（~18 GB）
```

### 4️⃣ 验证模型安装

```bash
ollama list
```

**预期输出：**
```
NAME                    ID              SIZE      MODIFIED
deepseek-r1:8b          abc123def456    4.7 GB    2 minutes ago
```

---

## 配置 .env

### 方式 1: 修改现有 .env 文件

编辑项目根目录的 `.env` 文件：

```env
# ==================== Ollama 本地配置 ====================
LLM_PROVIDER=openai
LLM_BACKEND_URL=http://localhost:11434/v1
OPENAI_API_KEY=ollama

# 模型选择
QUICK_THINK_LLM=deepseek-r1:8b
DEEP_THINK_LLM=deepseek-r1:8b

# 其他配置保持不变...
CRYPTO_SYMBOLS=BTC/USDT
CRYPTO_TIMEFRAME=15m
BINANCE_TEST_MODE=true
AUTO_EXECUTE=false
```

### 方式 2: 创建专用 Ollama 配置文件

```bash
cp .env .env.ollama
```

编辑 `.env.ollama`，然后运行时指定：
```bash
ENV_FILE=.env.ollama make run-web
```

---

## 运行测试

### 测试 1: 连接检查

```bash
make test-ollama-connection
```

**这个测试会：**
- ✅ 检查 Ollama 服务是否运行
- ✅ 获取所有已安装的模型
- ✅ 查找 deepseek-r1:8b 模型
- ✅ 发送简单测试请求

### 测试 2: 交易决策测试

```bash
make test-ollama-trading
```

**这个测试会：**
- ✅ 使用 deepseek-r1:8b 模型
- ✅ 发送简化的市场数据
- ✅ 获取 JSON 格式的交易决策
- ✅ 验证决策字段

### 测试 3: 运行所有 Ollama 测试

```bash
make test-ollama-all
```

---

## 性能优化

### 硬件要求

| 模型 | 最小 RAM | 推荐 RAM | GPU | 响应时间 |
|------|----------|----------|-----|----------|
| deepseek-r1:1.5b | 2 GB | 4 GB | 可选 | 5-10 秒 |
| deepseek-r1:8b | 8 GB | 16 GB | 推荐 | 10-30 秒 |
| deepseek-r1:14b | 16 GB | 32 GB | 推荐 | 20-60 秒 |
| deepseek-r1:32b | 32 GB | 64 GB | 必需 | 60-120 秒 |

### GPU 加速

#### NVIDIA GPU (CUDA)
```bash
# Ollama 会自动检测并使用 NVIDIA GPU
ollama serve
```

#### Apple Silicon (M1/M2/M3)
```bash
# Ollama 会自动使用 Metal 加速
ollama serve
```

#### AMD GPU (ROCm)
```bash
# 需要安装 ROCm 驱动
ollama serve
```

### 调整并发设置

编辑 Ollama 配置（如果需要）：
```bash
# 设置最大并发请求数
export OLLAMA_MAX_LOADED_MODELS=1
export OLLAMA_NUM_PARALLEL=1

ollama serve
```

---

## 常见问题

### Q1: Ollama 服务启动失败

**问题：** `ollama serve` 报错

**解决方案：**
```bash
# 检查端口是否被占用
lsof -i :11434

# 杀死占用进程
kill -9 <PID>

# 重新启动
ollama serve
```

### Q2: 模型下载速度慢

**问题：** `ollama pull` 下载很慢

**解决方案：**
```bash
# 使用代理（如果在国内）
export HTTP_PROXY=http://127.0.0.1:7890
export HTTPS_PROXY=http://127.0.0.1:7890

ollama pull deepseek-r1:8b
```

### Q3: 响应速度太慢

**问题：** 模型推理时间超过 1 分钟

**解决方案：**
1. 使用更小的模型（1.5b）
2. 启用 GPU 加速
3. 增加系统内存
4. 减少输入 token 数量

### Q4: JSON 解析失败

**问题：** 模型返回的 JSON 格式不正确

**解决方案：**
1. 使用更大的模型（14b/32b）
2. 调整 system prompt，强调 JSON 格式
3. 在代码中添加 JSON 清理逻辑

### Q5: 内存不足

**问题：** 系统内存不足，模型加载失败

**解决方案：**
```bash
# 使用量化版本（更小）
ollama pull deepseek-r1:8b-q4_0

# 或使用更小的模型
ollama pull deepseek-r1:1.5b
```

---

## 📚 相关资源

- [Ollama 官方文档](https://github.com/ollama/ollama)
- [DeepSeek 模型介绍](https://github.com/deepseek-ai/DeepSeek-R1)
- [OpenAI 兼容 API 文档](https://github.com/ollama/ollama/blob/main/docs/openai.md)
- [Ollama 模型库](https://ollama.com/library)

---

## 🎯 下一步

1. ✅ 完成 Ollama 安装和配置
2. ✅ 运行测试验证连接
3. ✅ 修改 `.env` 配置
4. ✅ 运行 `make run-web` 启动交易系统
5. ✅ 观察 LLM 决策质量
6. ✅ 根据需要调整模型大小

---

**贡献者**: AI Assistant  
**最后更新**: 2026-01-12

