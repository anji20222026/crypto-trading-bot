package dataflows

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// TestJSONOutputFormat tests the JSON output format
// TestJSONOutputFormat 测试 JSON 输出格式
func TestJSONOutputFormat(t *testing.T) {
	// Create mock OHLCV data
	// 创建模拟 OHLCV 数据
	ohlcvData := make([]OHLCV, 100)
	for i := 0; i < 100; i++ {
		ohlcvData[i] = OHLCV{
			Timestamp: time.Now().Add(-time.Duration(100-i) * 15 * time.Minute),
			Open:      90000.0 + float64(i)*10,
			High:      90100.0 + float64(i)*10,
			Low:       89900.0 + float64(i)*10,
			Close:     90050.0 + float64(i)*10,
			Volume:    1000.0 + float64(i)*5,
		}
	}

	indicators := CalculateIndicators(ohlcvData)

	longerOHLCV := make([]OHLCV, 50)
	for i := 0; i < 50; i++ {
		longerOHLCV[i] = OHLCV{
			Timestamp: time.Now().Add(-time.Duration(50-i) * time.Hour),
			Open:      90000.0 + float64(i)*20,
			High:      90200.0 + float64(i)*20,
			Low:       89800.0 + float64(i)*20,
			Close:     90100.0 + float64(i)*20,
			Volume:    5000.0 + float64(i)*10,
		}
	}

	longerIndicators := CalculateIndicators(longerOHLCV)

	// Build all components
	// 构建所有组件
	indicatorsData := buildIndicatorsData(ohlcvData, indicators, "15m")
	volumeData := buildVolumeData(ohlcvData, "15m")
	priceHistory := buildPriceHistory(ohlcvData, longerOHLCV)
	indicatorHistory := buildIndicatorHistory(ohlcvData, indicators)
	longTermData := buildLongTermData(longerOHLCV, longerIndicators)

	// Create a sample SymbolMarketData
	// 创建示例 SymbolMarketData
	symbolData := &SymbolMarketData{
		CurrentPrice:     91685.4,
		Timeframe:        "15m",
		Indicators:       indicatorsData,
		Volume:           volumeData,
		PriceHistory:     priceHistory,
		IndicatorHistory: indicatorHistory,
		MultiTF: map[string]*MultiTFIndicator{
			"5m": {
				EMA20: 90561.787,
				EMA50: 90479.412,
				MACD:  170.807,
				RSI7:  87.49,
				RSI14: 76.62,
			},
			"15m": {
				EMA20: 90623.219,
				EMA50: 90653.579,
				MACD:  98.492,
				RSI7:  78.68,
				RSI14: 70.17,
			},
		},
		MarketStats: &MarketStatsData{
			FundingRate:       0.000073,
			PriceChange24hPct: 1.271,
			High24h:           91766.7,
			Low24h:            89632.8,
		},
		LongTerm1H: longTermData,
		Positions: &PositionsData{
			OpenInterest: &OpenInterestData{
				TF15m: &OITimeframeData{
					ChangeRate: []float64{0.0, -0.04, -0.03, -0.01, -0.12},
					Volume:     []float64{1200, 1100, 1300, 1250, 1100},
				},
				DailyStats: &DailyStatsData{
					PriceChangePercent: 1.271,
					High:               91766.7,
					Low:                89632.8,
					Volume:             133574.789,
				},
			},
		},
	}

	// Create MarketJSONData with multiple symbols and schema
	// 创建包含多个交易对和 schema 的 MarketJSONData
	marketData := &MarketJSONData{
		Schema: GetDefaultSchema(),
		Data: map[string]*SymbolMarketData{
			"BTC/USDT": symbolData,
		},
	}

	// Marshal to JSON
	// 序列化为 JSON
	jsonBytes, err := json.MarshalIndent(marketData, "", "  ")
	if err != nil {
		t.Fatalf("JSON 序列化失败: %v", err)
	}

	// Print JSON output (for manual inspection)
	// 打印 JSON 输出（用于手动检查）
	fmt.Println("=== JSON 输出示例 ===")
	fmt.Println(string(jsonBytes))

	// Verify JSON can be unmarshaled back
	// 验证 JSON 可以反序列化
	var unmarshaled MarketJSONData
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	if err != nil {
		t.Fatalf("JSON 反序列化失败: %v", err)
	}

	// Verify data integrity
	// 验证数据完整性
	if len(unmarshaled.Data) != 1 {
		t.Errorf("期望 1 个交易对，实际: %d", len(unmarshaled.Data))
	}

	btcData, ok := unmarshaled.Data["BTC/USDT"]
	if !ok {
		t.Fatal("BTC/USDT 数据缺失")
	}

	if btcData.CurrentPrice != 91685.4 {
		t.Errorf("CurrentPrice 不匹配，期望: 91685.4，实际: %f", btcData.CurrentPrice)
	}

	if btcData.Timeframe != "15m" {
		t.Errorf("Timeframe 不匹配，期望: 15m，实际: %s", btcData.Timeframe)
	}

	t.Logf("✅ JSON 格式验证通过")
}

