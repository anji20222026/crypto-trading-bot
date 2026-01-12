# 市场数据 JSON Schema

以下是市场数据的完整 JSON Schema 定义，描述了所有字段的类型和结构：

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://github.com/oak/crypto-trading-bot/internal/dataflows/market-json-data",
  "$ref": "#/$defs/MarketJSONData",
  "$defs": {
    "BBBands": {
      "properties": {
        "upper": {
          "items": {
            "type": "number"
          },
          "type": "array"
        },
        "lower": {
          "items": {
            "type": "number"
          },
          "type": "array"
        }
      },
      "additionalProperties": false,
      "required": [
        "upper",
        "lower"
      ],
      "type": "object"
    },
    "BBData": {
      "properties": {
        "15m": {
          "additionalProperties": {
            "$ref": "#/$defs/BBBands",
            "type": ""
          },
          "type": "object"
        }
      },
      "additionalProperties": false,
      "required": [
        "15m"
      ],
      "type": "object"
    },
    "DailyStatsData": {
      "properties": {
        "price_change_percent": {
          "type": "number"
        },
        "high": {
          "type": "number"
        },
        "low": {
          "type": "number"
        },
        "volume": {
          "type": "number"
        }
      },
      "additionalProperties": false,
      "required": [
        "price_change_percent",
        "high",
        "low",
        "volume"
      ],
      "type": "object"
    },
    "IndicatorHistoryData": {
      "properties": {
        "15m": {
          "$ref": "#/$defs/IndicatorSeriesData",
          "type": ""
        }
      },
      "additionalProperties": false,
      "required": [
        "15m"
      ],
      "type": "object"
    },
    "IndicatorSeriesData": {
      "properties": {
        "EMA12": {
          "items": {
            "type": "number"
          },
          "type": "array"
        },
        "MACD": {
          "items": {
            "type": "number"
          },
          "type": "array"
        },
        "RSI7": {
          "items": {
            "type": "number"
          },
          "type": "array"
        },
        "RSI14": {
          "items": {
            "type": "number"
          },
          "type": "array"
        },
        "ADX": {
          "items": {
            "type": "number"
          },
          "type": "array"
        }
      },
      "additionalProperties": false,
      "required": [
        "EMA12",
        "MACD",
        "RSI7",
        "RSI14",
        "ADX"
      ],
      "type": "object"
    },
    "IndicatorsData": {
      "properties": {
        "EMA": {
          "additionalProperties": {
            "type": "number"
          },
          "type": "object"
        },
        "MACD": {
          "type": "number"
        },
        "RSI": {
          "additionalProperties": {
            "type": "number"
          },
          "type": "object"
        },
        "ADX": {
          "type": "number"
        },
        "BB": {
          "$ref": "#/$defs/BBData",
          "type": ""
        },
        "ATR": {
          "additionalProperties": {
            "type": "number"
          },
          "type": "object"
        },
        "VWAP": {
          "$ref": "#/$defs/VWAPData",
          "type": ""
        }
      },
      "additionalProperties": false,
      "required": [
        "EMA",
        "MACD",
        "RSI",
        "ADX",
        "BB",
        "ATR",
        "VWAP"
      ],
      "type": "object"
    },
    "LongTermData": {
      "properties": {
        "price_mid": {
          "items": {
            "type": "number"
          },
          "type": "array"
        },
        "EMA": {
          "additionalProperties": {
            "type": "number"
          },
          "type": "object"
        },
        "ATR": {
          "additionalProperties": {
            "type": "number"
          },
          "type": "object"
        },
        "MACD": {
          "items": {
            "type": "number"
          },
          "type": "array"
        },
        "RSI14": {
          "items": {
            "type": "number"
          },
          "type": "array"
        }
      },
      "additionalProperties": false,
      "required": [
        "price_mid",
        "EMA",
        "ATR",
        "MACD",
        "RSI14"
      ],
      "type": "object"
    },
    "MarketJSONData": {
      "properties": {
        "schema": {
          "additionalProperties": {
            "$ref": "#/$defs/SchemaField",
            "type": ""
          },
          "type": "object"
        },
        "data": {
          "additionalProperties": {
            "$ref": "#/$defs/SymbolMarketData",
            "type": ""
          },
          "type": "object"
        }
      },
      "additionalProperties": false,
      "required": [
        "schema",
        "data"
      ],
      "type": "object"
    },
    "MarketStatsData": {
      "properties": {
        "funding_rate": {
          "type": "number"
        },
        "price_change_24h_pct": {
          "type": "number"
        },
        "high_24h": {
          "type": "number"
        },
        "low_24h": {
          "type": "number"
        }
      },
      "additionalProperties": false,
      "required": [
        "funding_rate",
        "price_change_24h_pct",
        "high_24h",
        "low_24h"
      ],
      "type": "object"
    },
    "MultiTFIndicator": {
      "properties": {
        "EMA20": {
          "type": "number"
        },
        "EMA50": {
          "type": "number"
        },
        "MACD": {
          "type": "number"
        },
        "RSI7": {
          "type": "number"
        },
        "RSI14": {
          "type": "number"
        }
      },
      "additionalProperties": false,
      "required": [
        "EMA20",
        "EMA50",
        "MACD",
        "RSI7",
        "RSI14"
      ],
      "type": "object"
    },
    "OITimeframeData": {
      "properties": {
        "change_rate": {
          "items": {
            "type": "number"
          },
          "type": "array"
        },
        "volume": {
          "items": {
            "type": "number"
          },
          "type": "array"
        }
      },
      "additionalProperties": false,
      "required": [
        "change_rate",
        "volume"
      ],
      "type": "object"
    },
    "OpenInterestData": {
      "properties": {
        "15m": {
          "$ref": "#/$defs/OITimeframeData",
          "type": ""
        },
        "daily_stats": {
          "$ref": "#/$defs/DailyStatsData",
          "type": ""
        }
      },
      "additionalProperties": false,
      "required": [
        "15m",
        "daily_stats"
      ],
      "type": "object"
    },
    "PositionsData": {
      "properties": {
        "open_interest": {
          "$ref": "#/$defs/OpenInterestData",
          "type": ""
        }
      },
      "additionalProperties": false,
      "required": [
        "open_interest"
      ],
      "type": "object"
    },
    "PriceHistoryData": {
      "properties": {
        "15m": {
          "$ref": "#/$defs/PriceSeriesData",
          "type": ""
        },
        "1h": {
          "$ref": "#/$defs/PriceSeriesData",
          "type": ""
        }
      },
      "additionalProperties": false,
      "required": [
        "15m",
        "1h"
      ],
      "type": "object"
    },
    "PriceSeriesData": {
      "properties": {
        "mid": {
          "items": {
            "type": "number"
          },
          "type": "array"
        }
      },
      "additionalProperties": false,
      "required": [
        "mid"
      ],
      "type": "object"
    },
    "SchemaField": {
      "properties": {
        "type": {
          "type": "string"
        },
        "description": {
          "type": "string"
        }
      },
      "additionalProperties": false,
      "required": [
        "type",
        "description"
      ],
      "type": "object"
    },
    "SymbolMarketData": {
      "properties": {
        "current_price": {
          "type": "number"
        },
        "timeframe": {
          "type": "string"
        },
        "indicators": {
          "$ref": "#/$defs/IndicatorsData",
          "type": ""
        },
        "volume": {
          "$ref": "#/$defs/VolumeData",
          "type": ""
        },
        "price_history": {
          "$ref": "#/$defs/PriceHistoryData",
          "type": ""
        },
        "indicator_history": {
          "$ref": "#/$defs/IndicatorHistoryData",
          "type": ""
        },
        "multi_tf": {
          "additionalProperties": {
            "$ref": "#/$defs/MultiTFIndicator",
            "type": ""
          },
          "type": "object"
        },
        "market_stats": {
          "$ref": "#/$defs/MarketStatsData",
          "type": ""
        },
        "long_term_1h": {
          "$ref": "#/$defs/LongTermData",
          "type": ""
        },
        "positions": {
          "$ref": "#/$defs/PositionsData",
          "type": ""
        }
      },
      "additionalProperties": false,
      "required": [
        "current_price",
        "timeframe",
        "indicators",
        "volume",
        "price_history",
        "indicator_history",
        "multi_tf",
        "market_stats",
        "long_term_1h",
        "positions"
      ],
      "type": "object"
    },
    "VWAPData": {
      "properties": {
        "24h": {
          "type": "number"
        },
        "current_deviation": {
          "type": "number"
        },
        "history_15m": {
          "$ref": "#/$defs/VWAPHistoryData",
          "type": ""
        }
      },
      "additionalProperties": false,
      "required": [
        "24h",
        "current_deviation",
        "history_15m"
      ],
      "type": "object"
    },
    "VWAPHistoryData": {
      "properties": {
        "price": {
          "items": {
            "type": "number"
          },
          "type": "array"
        },
        "deviation_pct": {
          "items": {
            "type": "number"
          },
          "type": "array"
        }
      },
      "additionalProperties": false,
      "required": [
        "price",
        "deviation_pct"
      ],
      "type": "object"
    },
    "VolumeData": {
      "properties": {
        "current": {
          "type": "number"
        },
        "average": {
          "type": "number"
        },
        "period": {
          "type": "string"
        }
      },
      "additionalProperties": false,
      "required": [
        "current",
        "average",
        "period"
      ],
      "type": "object"
    }
  },
  "type": ""
}
```

## 主要字段说明

### data (object)
包含所有交易对的市场数据，key 为交易对名称（如 "BTC/USDT"）

### SymbolMarketData (每个交易对的数据)

#### current_price (number)
当前价格（USDT）

#### timeframe (string)
主时间周期（如 "15m", "1h"）

#### indicators (object)
技术指标数据：
- **EMA**: 指数移动平均线，包含多个周期（12, 26, 50, 100, 200）
- **MACD**: MACD 指标当前值
- **RSI**: 相对强弱指标，包含多个周期（7, 14, 21）
- **ADX**: 平均趋向指标，衡量趋势强度
- **BB**: 布林带，包含上轨和下轨的历史数据
- **ATR**: 平均真实波幅，包含多个周期（3, 7, 14）
- **VWAP**: 成交量加权平均价格
  - 24h: 24小时 VWAP
  - current_deviation: 当前价格相对 VWAP 的偏离百分比
  - history_15m: 15分钟间隔的价格和偏离历史

#### volume (object)
成交量数据：
- current: 当前周期成交量
- average: 平均成交量
- period: 统计周期

#### price_history (object)
价格历史数据：
- 15m: 15分钟时间框架的最近10个价格点（中间价）
- 1h: 1小时时间框架的最近10个价格点（中间价）

#### indicator_history (object)
指标历史序列（最近10个数据点）：
- 15m: 包含 EMA12, MACD, RSI7, RSI14, ADX 的历史数组

#### multi_tf (object)
多时间框架指标快照：
- 包含 5m, 15m, 30m, 1h, 4h 等时间框架
- 每个时间框架包含: EMA20, EMA50, MACD, RSI7, RSI14

#### market_stats (object)
市场统计数据：
- funding_rate: 资金费率
- price_change_24h_pct: 24小时价格变化百分比
- high_24h: 24小时最高价
- low_24h: 24小时最低价

#### long_term_1h (object)
长期（1小时）时间框架数据：
- price_mid: 价格历史（最近10个数据点）
- EMA: 长期 EMA（20, 50）
- ATR: 长期 ATR（3, 7, 14）
- MACD: MACD 历史数组
- RSI14: RSI14 历史数组

#### positions (object)
持仓量数据：
- open_interest: 持仓量信息
  - 15m: 15分钟间隔的变化率和持仓量序列
  - daily_stats: 每日统计数据

## 数据使用建议

1. **趋势判断**: 结合 EMA, MACD, ADX 判断趋势方向和强度
2. **超买超卖**: 使用 RSI 判断是否超买或超卖
3. **波动性**: 使用 ATR 和 BB 评估市场波动性
4. **成交量确认**: 使用 volume 和 VWAP 确认价格走势
5. **多时间框架**: 结合 multi_tf 进行多周期分析
6. **持仓量**: 使用 positions 数据判断市场情绪
