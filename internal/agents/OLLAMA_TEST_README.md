# Ollama 集成测试指南

本文档介绍如何使用本地 Ollama 运行 DeepSeek 模型进行交易决策测试。

## 📋 前置要求

### 1. 安装 Ollama

**macOS / Linux:**
```bash
curl -fsSL https://ollama.com/install.sh | sh
```

**Windows:**
下载并安装：https://ollama.com/download

### 2. 启动 Ollama 服务

```bash
ollama serve
```

默认监听端口：`11434`

### 3. 拉取 DeepSeek 模型

```bash
# 推荐：8B 参数版本（约 4.7 GB）
ollama pull deepseek-r1:8b

# 或者其他版本：
# ollama pull deepseek-r1:1.5b   # 更小，速度更快
# ollama pull deepseek-r1:14b    # 更大，效果更好
# ollama pull deepseek-r1:32b    # 最大，需要更多内存
```

### 4. 验证模型安装

```bash
ollama list
```

输出示例：
```
NAME                    ID              SIZE      MODIFIED
deepseek-r1:8b          abc123def456    4.7 GB    2 minutes ago
```

---

## 🧪 运行测试

### 测试 1: Ollama 连接和模型列表

这个测试会：
- ✅ 检查 Ollama 服务是否运行
- ✅ 获取所有已安装的模型
- ✅ 查找 deepseek-r1:8b 模型
- ✅ 发送简单测试请求

```bash
# 运行测试
go test -v -run TestOllamaConnection ./internal/agents

# 或使用 make（如果配置了）
make test-ollama-connection
```

**预期输出：**
```
=== RUN   TestOllamaConnection
========================================
🦙 Ollama 集成测试：连接检查 + 模型列表
========================================

📡 步骤 1/4: 检查 Ollama 服务...
✅ Ollama 服务运行正常

📋 步骤 2/4: 获取 Ollama 模型列表...
✅ 找到 2 个已安装的模型:
   1. deepseek-r1:8b (4.70 GB)
   2. llama2:7b (3.83 GB)

🔍 步骤 3/4: 查找 deepseek-r1:8b 模型...
✅ 找到 DeepSeek 模型: deepseek-r1:8b

🤖 步骤 4/4: 测试模型推理...
   模型: deepseek-r1:8b
   API: http://localhost:11434/v1

📤 发送测试请求...
✅ 模型响应成功！

========================================
📝 模型响应内容:
========================================
我是 DeepSeek，一个由深度求索公司开发的 AI 助手。
========================================

🎉 Ollama 集成测试全部通过！
--- PASS: TestOllamaConnection (3.45s)
```

---

### 测试 2: DeepSeek 交易决策

这个测试会：
- ✅ 连接到本地 Ollama
- ✅ 使用 deepseek-r1:8b 模型
- ✅ 发送简化的市场数据
- ✅ 获取 JSON 格式的交易决策
- ✅ 验证决策字段

```bash
# 运行测试
go test -v -run TestOllamaDeepSeekTrading ./internal/agents

# 或使用 make（如果配置了）
make test-ollama-trading
```

**预期输出：**
```
=== RUN   TestOllamaDeepSeekTrading
========================================
🦙 Ollama DeepSeek 交易决策测试
========================================

📡 检查 Ollama 服务...
🔍 检查 deepseek-r1:8b 模型...
✅ 找到模型: deepseek-r1:8b

🤖 初始化 ChatModel...
📊 准备市场数据...
📤 发送交易决策请求...
   (这可能需要 10-30 秒，取决于模型大小和硬件性能)

✅ 模型响应成功！耗时: 12.34 秒

========================================
📝 原始 LLM 响应:
========================================
{
  "symbol": "BTC/USDT",
  "action": "BUY",
  "confidence": 0.75,
  "leverage": 10,
  "position_size": 30.0,
  "stop_loss": 93000.00,
  "reasoning": "RSI 处于中性区域，MACD 金叉，24h 涨幅 2.34%，趋势向上",
  "risk_reward_ratio": 2.5,
  "summary": "技术面偏多，建议小仓位做多"
}
========================================

🔍 解析 JSON 响应...
✅ JSON 解析成功！

========================================
📊 解析后的交易决策:
========================================
🎯 交易对:       BTC/USDT
📈 交易动作:     BUY
💯 置信度:       0.75 (75%)
🔢 杠杆倍数:     10x
💰 建议仓位:     30.0%
🛑 止损价格:     $93000.00
⚖️  盈亏比:       2.5:1
📝 交易理由:     RSI 处于中性区域，MACD 金叉，24h 涨幅 2.34%，趋势向上
📄 决策总结:     技术面偏多，建议小仓位做多
========================================

🔍 验证决策字段...
✅ 所有字段验证通过！

========================================
📊 性能统计:
========================================
⏱️  响应时间:     12.34 秒
📝 输入 Token:   245
📝 输出 Token:   128
📝 总计 Token:   373
⚡ 生成速度:     10.4 tokens/秒

🎉 Ollama DeepSeek 交易决策测试完成！
--- PASS: TestOllamaDeepSeekTrading (12.34s)
```

---

## 🔧 配置 .env 使用 Ollama

如果要在实际交易中使用 Ollama，请修改 `.env` 文件：

```env
# LLM 配置
LLM_PROVIDER=openai
LLM_BACKEND_URL=http://localhost:11434/v1
OPENAI_API_KEY=ollama

# 模型选择
QUICK_THINK_LLM=deepseek-r1:8b
DEEP_THINK_LLM=deepseek-r1:8b

# 其他配置保持不变...
```

---

## ⚠️ 注意事项

### 性能考虑

1. **响应时间**：本地模型比云端 API 慢，8B 模型通常需要 10-30 秒
2. **硬件要求**：
   - deepseek-r1:8b 需要至少 8GB RAM
   - 推荐使用 GPU 加速（NVIDIA/AMD）
3. **并发限制**：Ollama 默认单线程，不适合高频交易

### 模型选择

| 模型 | 大小 | 内存需求 | 速度 | 质量 |
|------|------|----------|------|------|
| deepseek-r1:1.5b | ~1 GB | 2 GB | ⚡⚡⚡ | ⭐⭐ |
| deepseek-r1:8b | ~5 GB | 8 GB | ⚡⚡ | ⭐⭐⭐ |
| deepseek-r1:14b | ~8 GB | 16 GB | ⚡ | ⭐⭐⭐⭐ |
| deepseek-r1:32b | ~18 GB | 32 GB | 🐌 | ⭐⭐⭐⭐⭐ |

### 常见问题

**Q: 测试失败，提示 "Ollama 服务未运行"**
```bash
# 启动 Ollama
ollama serve
```

**Q: 找不到 deepseek-r1:8b 模型**
```bash
# 拉取模型
ollama pull deepseek-r1:8b
```

**Q: 响应速度太慢**
- 使用更小的模型（1.5b）
- 启用 GPU 加速
- 增加系统内存

**Q: JSON 解析失败**
- 本地模型可能不完全遵循 JSON 格式
- 尝试调整 system prompt
- 使用更大的模型（14b/32b）

---

## 📚 相关资源

- [Ollama 官方文档](https://github.com/ollama/ollama)
- [DeepSeek 模型介绍](https://github.com/deepseek-ai/DeepSeek-R1)
- [OpenAI 兼容 API 文档](https://github.com/ollama/ollama/blob/main/docs/openai.md)

---

**贡献者**: AI Assistant  
**最后更新**: 2026-01-12

