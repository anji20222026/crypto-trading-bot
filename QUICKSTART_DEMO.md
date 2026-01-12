# 快速开始 - 市场数据 Demo

如果你想快速了解系统如何获取和处理币安市场数据，可以先运行这个 demo。

## 前置要求

1. **Go 1.21+** 已安装
2. **币安账户**（可选，demo 可以不需要 API 密钥）
3. **网络连接**（能访问币安 API）

## 快速运行（3 步）

### 1. 克隆项目

```bash
git clone https://github.com/yourusername/crypto-trading-bot.git
cd crypto-trading-bot
```

### 2. 配置 .env

复制示例配置：

```bash
cp .env.example .env
```

编辑 `.env`，最少只需要配置这几项：

```env
# 币安配置（可以留空，使用公开 API）
BINANCE_API_KEY=
BINANCE_API_SECRET=
BINANCE_TEST_MODE=false

# 交易对
CRYPTO_SYMBOLS=BTC/USDT

# 时间周期
CRYPTO_TIMEFRAME=15m
CRYPTO_LOOKBACK_DAYS=2

# 多时间框架
ENABLE_MULTI_TIMEFRAME=true
CRYPTO_LONGER_TIMEFRAME=1h
CRYPTO_LONGER_LOOKBACKDAYS=3

# 如果无法直接访问币安，配置代理
# BINANCE_PROXY=http://127.0.0.1:7890
```

### 3. 运行 Demo

```bash
make demo
```

## 预期输出

### 控制台输出

```
🚀 币安市场数据结构化输出 Demo
==================================================
✅ 配置加载成功
   - 交易对: [BTC/USDT]
   - 时间周期: 15m
   - 测试模式: false

📊 正在获取 BTC/USDT 的市场数据...

   🔄 获取 15m OHLCV 数据...
   ✅ 获取到 192 条 K 线数据
   🔄 计算技术指标...
   ✅ 技术指标计算完成
   🔄 获取 1h OHLCV 数据...
   ✅ 获取到 72 条长期 K 线数据

🔧 正在构建结构化市场数据...
✅ 结构化数据构建完成

📈 市场数据摘要
==================================================
交易对: BTC/USDT
当前价格: 91685.40 USDT
时间周期: 15m

📊 技术指标:
   EMA12: 91234.56
   EMA26: 91123.45
   MACD: 111.11
   RSI14: 65.43
   ADX: 32.10

💾 完整 JSON 数据已保存到: market_data_output.json

✅ Demo 运行完成！
```

### JSON 文件

会在项目根目录生成 `market_data_output.json`，包含完整的结构化市场数据。

## 查看 JSON 数据

```bash
# 使用 jq 格式化查看（如果已安装）
cat market_data_output.json | jq .

# 或者直接用文本编辑器打开
code market_data_output.json  # VS Code
vim market_data_output.json   # Vim
```

## 数据结构说明

JSON 数据包含以下部分：

- **current_price**: 当前价格
- **indicators**: 技术指标（EMA, MACD, RSI, ADX, BB, ATR, VWAP）
- **volume**: 成交量数据
- **price_history**: 价格历史（15m 和 1h）
- **indicator_history**: 指标历史序列
- **multi_tf**: 多时间框架指标（5m, 15m, 30m, 1h, 4h）
- **market_stats**: 市场统计（资金费率、24h 涨跌等）
- **long_term_1h**: 长期数据（1h 时间框架）
- **positions**: 持仓量数据

详细说明请查看：
- [cmd/demo/README.md](cmd/demo/README.md) - Demo 使用说明
- [cmd/demo/EXAMPLE_OUTPUT.md](cmd/demo/EXAMPLE_OUTPUT.md) - 输出示例
- [docs/STRUCTURED_DATA.md](docs/STRUCTURED_DATA.md) - 数据结构详解

## 常见问题

### Q: 连接失败怎么办？

A: 检查以下几点：
1. 网络是否能访问币安 API
2. 如果需要代理，配置 `BINANCE_PROXY`
3. 检查防火墙设置

### Q: 可以不配置 API 密钥吗？

A: 可以！Demo 主要使用公开 API 获取市场数据，不需要 API 密钥。但某些功能（如持仓量数据）可能需要认证。

### Q: 如何获取多个交易对的数据？

A: 修改 `cmd/demo/main.go`，循环处理 `cfg.CryptoSymbols` 中的所有交易对。

### Q: 数据更新频率是多少？

A: Demo 是一次性运行，获取当前时刻的数据。如果需要持续监控，请使用主程序 `make run-web`。

## 下一步

了解了数据结构后，你可以：

1. **运行完整系统**: 查看 [README.md](README.md) 了解如何配置和运行交易机器人
2. **自定义策略**: 编辑 `prompts/` 目录下的 Prompt 文件
3. **查看历史数据**: 使用 `make query ARGS="latest 10"` 查询历史交易记录

## 技术细节

如果你想了解代码实现：

- **数据获取**: `internal/dataflows/market_data.go`
- **指标计算**: `internal/dataflows/indicators.go`
- **JSON 构建**: `internal/dataflows/market_json_data.go`
- **Demo 入口**: `cmd/demo/main.go`

---

**祝你使用愉快！** 🚀

如有问题，请提交 Issue 或查看完整文档。

