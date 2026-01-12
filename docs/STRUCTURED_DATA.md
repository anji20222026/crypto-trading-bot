# 结构化市场数据 (Structured Market Data)

## 概述

本项目已从基于文本的市场报告迁移到结构化的 JSON 数据格式，以提供更精确、更易于 LLM 处理的市场信息。

## 主要改进

### 1. 数据格式
- **之前**: 使用文本格式的市场报告（`allReports`）
- **现在**: 使用结构化的 JSON 数据（`MarketJSONData`）

### 2. 数据内容

每个交易对的数据包含以下部分：

#### 基础信息
- `current_price`: 当前价格
- `timeframe`: 时间周期（如 "15m"）

#### 技术指标 (`indicators`)
- **EMA**: 多周期指数移动平均线（12, 26 等）
- **MACD**: MACD 指标值
- **RSI**: 多周期相对强弱指标（7, 14 等）
- **ADX**: 平均趋向指标
- **BB**: 布林带（上轨、下轨历史数据）
- **ATR**: 平均真实波幅（3, 7, 14 等）
- **VWAP**: 成交量加权平均价格
  - `24h`: 24小时 VWAP
  - `current_deviation`: 当前价格相对 VWAP 的偏离百分比
  - `history_15m`: 15分钟间隔的价格和偏离历史

#### 成交量 (`volume`)
- `current`: 当前成交量
- `average`: 平均成交量
- `period`: 统计周期

#### 价格历史 (`price_history`)
- `15m`: 15分钟时间框架的价格历史（最近10个数据点）
- `1h`: 1小时时间框架的价格历史（最近10个数据点）

#### 指标历史 (`indicator_history`)
- `15m`: 15分钟时间框架的指标历史
  - `EMA12`: EMA12 历史
  - `MACD`: MACD 历史
  - `RSI7`: RSI7 历史
  - `RSI14`: RSI14 历史
  - `ADX`: ADX 历史

#### 多时间框架指标 (`multi_tf`)
- 包含 5m, 15m, 30m, 1h, 4h 等多个时间框架的指标快照
- 每个时间框架包含: EMA20, EMA50, MACD, RSI7, RSI14

#### 市场统计 (`market_stats`)
- `funding_rate`: 资金费率
- `price_change_24h_pct`: 24小时价格变化百分比
- `high_24h`: 24小时最高价
- `low_24h`: 24小时最低价

#### 长期数据 (`long_term_1h`)
- `price_mid`: 1小时价格历史
- `EMA`: 长期 EMA（20, 50）
- `ATR`: 长期 ATR（3, 7, 14）
- `MACD`: 长期 MACD 历史
- `RSI14`: 长期 RSI14 历史

#### 持仓数据 (`positions`)
- `open_interest`: 持仓量数据
  - `15m`: 15分钟间隔的持仓量变化率和持仓量序列
  - `daily_stats`: 每日统计数据

## 代码结构

### 核心文件
- `internal/dataflows/market_json_data.go`: JSON 数据结构定义和构建逻辑
- `internal/agents/graph.go`: 数据收集和 LLM 决策逻辑

### 关键函数
- `BuildMarketJSONData()`: 构建单个交易对的结构化数据
- `buildIndicatorsData()`: 构建技术指标数据
- `buildVolumeData()`: 构建成交量数据
- `buildPriceHistory()`: 构建价格历史
- `buildIndicatorHistory()`: 构建指标历史
- `buildLongTermData()`: 构建长期数据

## 测试

### 单元测试
```bash
# 测试数据结构
go test -v -run TestMarketJSONDataStructure ./internal/dataflows/

# 测试 JSON 输出格式
go test -v -run TestJSONOutputFormat ./internal/dataflows/
```

### 集成测试
```bash
# 需要币安 API 访问
go test -v -run TestBuildMarketJSONData ./internal/dataflows/
```

## 使用示例

在 `makeLLMDecision` 函数中，系统会：

1. 从所有交易对的 `SymbolReports` 中收集 `MarketJSONData`
2. 将数据序列化为 JSON 字符串
3. 作为 Prompt 的一部分发送给 LLM
4. LLM 基于结构化数据做出交易决策

## 优势

1. **精确性**: 数值数据不会因文本格式化而丢失精度
2. **可解析性**: LLM 可以更容易地理解和处理结构化数据
3. **可扩展性**: 添加新字段更容易，不会破坏现有格式
4. **可测试性**: 更容易验证数据的正确性
5. **可维护性**: 代码结构更清晰，逻辑更分离

## 注意事项

- 所有历史数据默认保留最近 10 个数据点
- VWAP 历史数据可能为 `null`（如果获取失败）
- 多时间框架指标会根据配置动态生成
- 持仓量数据依赖币安 API 的可用性

