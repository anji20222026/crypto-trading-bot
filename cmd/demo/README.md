# 市场数据结构化输出 Demo

## 功能说明

这个 Demo 程序演示如何：
1. 从 `.env` 文件加载配置
2. 连接币安 API 获取市场数据
3. 计算技术指标
4. 构建结构化的 JSON 数据
5. 输出到控制台和文件

## 使用方法

### 1. 确保 .env 配置正确

确保项目根目录的 `.env` 文件包含以下配置：

```bash
# 币安 API 配置
BINANCE_API_KEY=your_api_key_here
BINANCE_API_SECRET=your_api_secret_here
BINANCE_TEST_MODE=false

# 交易对配置
CRYPTO_SYMBOLS=BTC/USDT,ETH/USDT
CRYPTO_TIMEFRAME=15m
CRYPTO_LOOKBACK_DAYS=2

# 多时间框架配置
ENABLE_MULTI_TIMEFRAME=true
CRYPTO_LONGER_TIMEFRAME=1h
CRYPTO_LONGER_LOOKBACK_DAYS=3

# 代理配置（可选）
BINANCE_PROXY=
```

### 2. 运行 Demo

使用 Makefile 命令：

```bash
make demo
```

或者直接运行：

```bash
go run cmd/demo/main.go
```

### 3. 查看输出

Demo 会：
- 在控制台显示市场数据摘要
- 将完整的 JSON 数据保存到 `market_data_output.json` 文件

## 输出示例

### 控制台输出

```
🚀 币安市场数据结构化输出 Demo
==================================================
✅ 配置加载成功
   - 交易对: [BTC/USDT ETH/USDT]
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

### JSON 文件输出

`market_data_output.json` 文件包含完整的结构化数据，格式如下：

```json
{
  "data": {
    "BTC/USDT": {
      "current_price": 91685.4,
      "timeframe": "15m",
      "indicators": {
        "EMA": {
          "12": 91234.56,
          "26": 91123.45
        },
        "MACD": 111.11,
        "RSI": {
          "7": 72.34,
          "14": 65.43
        },
        "ADX": 32.10,
        "BB": { ... },
        "ATR": { ... },
        "VWAP": { ... }
      },
      "volume": { ... },
      "price_history": { ... },
      "indicator_history": { ... },
      "multi_tf": { ... },
      "market_stats": { ... },
      "long_term_1h": { ... },
      "positions": { ... }
    }
  }
}
```

## 数据结构说明

详细的数据结构说明请参考：[docs/STRUCTURED_DATA.md](../../docs/STRUCTURED_DATA.md)

## 注意事项

1. **API 密钥**: 确保 `.env` 中配置了有效的币安 API 密钥
2. **网络连接**: 需要能够访问币安 API（可能需要代理）
3. **速率限制**: 币安 API 有速率限制，频繁调用可能被限制
4. **数据延迟**: 市场数据可能有几秒的延迟

## 故障排查

### 连接失败

如果遇到连接失败，可以尝试：
- 检查网络连接
- 配置代理：`BINANCE_PROXY=http://127.0.0.1:7890`
- 检查 API 密钥是否正确

### 数据为空

如果某些数据为空：
- 检查交易对是否正确
- 检查时间周期配置
- 查看日志中的错误信息

## 扩展使用

你可以修改 `cmd/demo/main.go` 来：
- 获取多个交易对的数据
- 自定义输出格式
- 添加更多的数据分析
- 集成到其他系统

