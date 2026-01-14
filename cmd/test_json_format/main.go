package main

import (
	"fmt"
	"log"
	"os"

	"github.com/bytedance/sonic"
	"github.com/oak/crypto-trading-bot/internal/dataflows"
)

func main() {
	fmt.Println("🧪 测试交易上下文 JSON 格式")
	fmt.Println()

	// Create sample market data
	// 创建示例市场数据
	marketData := map[string]*dataflows.SymbolMarketData{
		"BTC/USDT": {
			CurrentPrice: 90592.1,
			Timeframe:    "15m",
			Indicators: &dataflows.IndicatorsData{
				EMA: map[string]float64{
					"12":  90673.82,
					"26":  90945.0,
					"50":  91200.0,
					"100": 91500.0,
					"200": 92000.0,
				},
				MACD: -271.18,
				RSI: map[string]float64{
					"7":  45.2,
					"14": 48.5,
					"21": 50.1,
				},
				ADX: 25.3,
				BB: &dataflows.BBData{
					Timeframe: map[string]*dataflows.BBBands{
						"15m": {
							Upper: []float64{92000, 91800, 91600},
							Lower: []float64{89000, 89200, 89400},
						},
					},
				},
				ATR: map[string]float64{
					"3":  450.0,
					"7":  520.0,
					"14": 580.0,
				},
				VWAP: &dataflows.VWAPData{
					H24:              90800.0,
					CurrentDeviation: -0.23,
					History15m: &dataflows.VWAPHistoryData{
						Price:        []float64{90500, 90600, 90700, 90592.1},
						DeviationPct: []float64{-0.33, -0.22, -0.11, -0.23},
					},
				},
			},
			Volume: &dataflows.VolumeData{
				Current: 1250000,
				Average: 1100000,
				Period:  "15m",
			},
		},
	}

	// Sample instructions
	// 示例指令
	instructions := `你是专业的趋势交易分析师。根据提供的市场数据，输出严格可解析的 JSON 格式分析结果。

## 核心规则

1. 基于结构、趋势、动量指标完成 decision_gate 判断（TRADE / NO_TRADE）
2. decision_gate = NO_TRADE 时不得引用 ATR
3. 仅当 decision_gate = TRADE 时允许计算 ATR
4. ATR 仅用于计算 STOP_LOSS_DISTANCE 与盈亏比
5. 若 estimated_risk_reward < 3.5，则 decision_gate = NO_TRADE
6. 若 confidence < 0.7，则 trading_signal.action = "HOLD"
7. position_adjustment 仅对已有仓位生效，允许动作：HOLD, CLOSE_LONG, CLOSE_SHORT`

	// Build market JSON data with schema
	// 构建带 Schema 的市场 JSON 数据
	marketJSONData := &dataflows.MarketJSONData{
		Schema: dataflows.GetDefaultSchema(),
		Data:   marketData,
	}

	// Build final JSON: {instructions, schema, data}
	// 构建最终 JSON：{instructions, schema, data}
	context := map[string]interface{}{
		"instructions": instructions,
		"schema":       marketJSONData.Schema,
		"data":         marketJSONData.Data,
	}

	// Convert to JSON
	// 转换为 JSON
	jsonBytes, err := sonic.MarshalIndent(context, "", "  ")
	if err != nil {
		log.Fatalf("❌ JSON 序列化失败: %v", err)
	}

	// Print to console
	// 打印到控制台
	fmt.Println("📋 生成的交易上下文 JSON:")
	fmt.Println()
	fmt.Println(string(jsonBytes))
	fmt.Println()

	// Save to file
	// 保存到文件
	outputFile := "trading_context_example.json"
	err = os.WriteFile(outputFile, jsonBytes, 0644)
	if err != nil {
		log.Fatalf("❌ 文件保存失败: %v", err)
	}

	fmt.Printf("💾 完整 JSON 已保存到: %s\n", outputFile)
	fmt.Println()

	// Print statistics
	// 打印统计信息
	fmt.Println("📊 统计信息:")
	fmt.Printf("   - JSON 大小: %d 字节\n", len(jsonBytes))
	fmt.Printf("   - 交易对数量: %d\n", len(marketData))
	fmt.Printf("   - 指令长度: %d 字符\n", len(instructions))
	fmt.Println()

	fmt.Println("✅ 测试完成！")
}
