package dataflows

import (
	"context"
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/oak/crypto-trading-bot/internal/config"
)

// TestMarketJSONDataStructure tests the JSON data structure
// TestMarketJSONDataStructure 测试 JSON 数据结构
func TestMarketJSONDataStructure(t *testing.T) {
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

	// Calculate indicators
	// 计算指标
	indicators := CalculateIndicators(ohlcvData)

	// Create mock longer OHLCV data
	// 创建模拟长期 OHLCV 数据
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

	// Build indicators data
	// 构建指标数据
	indicatorsData := buildIndicatorsData(ohlcvData, indicators, "15m")

	// Verify indicators data
	// 验证指标数据
	if indicatorsData == nil {
		t.Fatal("indicatorsData 不应为 nil")
	}

	if len(indicatorsData.EMA) == 0 {
		t.Error("EMA 数据为空")
	}

	if _, ok := indicatorsData.EMA["12"]; !ok {
		t.Error("EMA 缺少 12 期数据")
	}

	if _, ok := indicatorsData.EMA["26"]; !ok {
		t.Error("EMA 缺少 26 期数据")
	}

	if len(indicatorsData.RSI) == 0 {
		t.Error("RSI 数据为空")
	}

	if _, ok := indicatorsData.RSI["7"]; !ok {
		t.Error("RSI 缺少 7 期数据")
	}

	if _, ok := indicatorsData.RSI["14"]; !ok {
		t.Error("RSI 缺少 14 期数据")
	}

	if math.IsNaN(indicatorsData.MACD) {
		t.Error("MACD 不应为 NaN")
	}

	if math.IsNaN(indicatorsData.ADX) {
		t.Error("ADX 不应为 NaN")
	}

	// Verify BB data
	// 验证布林带数据
	if indicatorsData.BB == nil {
		t.Error("BB 不应为 nil")
	} else {
		if len(indicatorsData.BB) == 0 {
			t.Error("BB 数据为空")
		}
	}

	// Build volume data
	// 构建成交量数据
	volumeData := buildVolumeData(ohlcvData, "15m")
	if volumeData == nil {
		t.Fatal("volumeData 不应为 nil")
	}

	if volumeData.Current <= 0 {
		t.Error("当前成交量应该大于 0")
	}

	if volumeData.Average7 <= 0 {
		t.Error("7 周期平均成交量应该大于 0")
	}

	if volumeData.Average14 <= 0 {
		t.Error("14 周期平均成交量应该大于 0")
	}

	if volumeData.Average20 <= 0 {
		t.Error("20 周期平均成交量应该大于 0")
	}

	// Build price history
	// 构建价格历史
	priceHistory := buildPriceHistory(ohlcvData, longerOHLCV)
	if priceHistory == nil {
		t.Fatal("priceHistory 不应为 nil")
	}

	if len(priceHistory.TF15m.Mid) == 0 {
		t.Error("15m 价格历史为空")
	}

	// Build indicator history
	// 构建指标历史
	indicatorHistory := buildIndicatorHistory(ohlcvData, indicators)
	if indicatorHistory == nil {
		t.Fatal("indicatorHistory 不应为 nil")
	}

	if len(indicatorHistory.TF15m.EMA12) == 0 {
		t.Error("EMA12 历史为空")
	}

	// Build long-term data
	// 构建长期数据
	longTermData := buildLongTermData(longerOHLCV, longerIndicators)
	if longTermData == nil {
		t.Fatal("longTermData 不应为 nil")
	}

	if len(longTermData.PriceMid) == 0 {
		t.Error("长期价格数据为空")
	}

	t.Logf("✅ 所有数据结构验证通过")
}

// TestGenerateMarketDataSchema tests the JSON Schema generation
// TestGenerateMarketDataSchema 测试 JSON Schema 生成
func TestGenerateMarketDataSchema(t *testing.T) {
	// Generate schema
	// 生成 Schema
	schema, err := GenerateMarketDataSchema()
	if err != nil {
		t.Fatalf("Schema 生成失败: %v", err)
	}

	if schema == "" {
		t.Fatal("Schema 不应为空")
	}

	// Verify it's valid JSON
	// 验证是否为有效的 JSON
	var schemaObj map[string]interface{}
	err = json.Unmarshal([]byte(schema), &schemaObj)
	if err != nil {
		t.Fatalf("Schema 不是有效的 JSON: %v", err)
	}

	// Log schema keys for debugging
	// 记录 Schema 的键用于调试
	t.Logf("Schema 顶层键: %v", getKeys(schemaObj))

	// Check for common JSON Schema fields
	// 检查常见的 JSON Schema 字段
	hasValidStructure := false
	if _, ok := schemaObj["type"]; ok {
		hasValidStructure = true
	}
	if _, ok := schemaObj["$schema"]; ok {
		hasValidStructure = true
	}
	if _, ok := schemaObj["properties"]; ok {
		hasValidStructure = true
	}

	if !hasValidStructure {
		t.Error("Schema 缺少有效的结构字段")
	}

	t.Logf("✅ Schema 生成成功，长度: %d 字节", len(schema))
}

// TestGetMarketDataSchemaWithDescription tests the schema with description
// TestGetMarketDataSchemaWithDescription 测试带描述的 Schema
func TestGetMarketDataSchemaWithDescription(t *testing.T) {
	// Get schema with description
	// 获取带描述的 Schema
	description := GetMarketDataSchemaWithDescription()

	if description == "" {
		t.Fatal("Schema 描述不应为空")
	}

	// Check for key sections
	// 检查关键部分
	if !contains(description, "JSON Schema") {
		t.Error("描述中缺少 'JSON Schema' 标题")
	}

	if !contains(description, "主要字段说明") {
		t.Error("描述中缺少 '主要字段说明' 部分")
	}

	if !contains(description, "indicators") {
		t.Error("描述中缺少 'indicators' 说明")
	}

	if !contains(description, "数据使用建议") {
		t.Error("描述中缺少 '数据使用建议' 部分")
	}

	t.Logf("✅ Schema 描述生成成功，长度: %d 字节", len(description))

	// Print first 500 characters for inspection
	// 打印前 500 个字符用于检查
	if len(description) > 500 {
		t.Logf("描述预览:\n%s...", description[:500])
	} else {
		t.Logf("完整描述:\n%s", description)
	}
}

// contains checks if a string contains a substring
// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// getKeys returns all keys from a map
// getKeys 返回 map 的所有键
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// TestBuildMarketJSONData tests the BuildMarketJSONData function
// TestBuildMarketJSONData 测试 BuildMarketJSONData 函数
func TestBuildMarketJSONData(t *testing.T) {
	// Skip if not in integration test mode
	// 如果不在集成测试模式下则跳过
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	cfg := &config.Config{
		BinanceAPIKey:    "",
		BinanceAPISecret: "",
		BinanceTestMode:  false,
		BinanceProxy:     "", // 不使用代理
	}

	marketData := NewMarketData(cfg)
	ctx := context.Background()

	symbol := "BTCUSDT"
	timeframe := "15m"
	lookbackDays := 2

	// Get OHLCV data
	// 获取 OHLCV 数据
	ohlcvData, err := marketData.GetOHLCV(ctx, symbol, timeframe, lookbackDays)
	if err != nil {
		t.Fatalf("获取 OHLCV 数据失败: %v", err)
	}

	if len(ohlcvData) == 0 {
		t.Fatal("OHLCV 数据为空")
	}

	// Calculate indicators
	// 计算指标
	indicators := CalculateIndicators(ohlcvData)

	// Get longer timeframe data
	// 获取长期时间框架数据
	longerOHLCV, err := marketData.GetOHLCV(ctx, symbol, "1h", 3)
	if err != nil {
		t.Fatalf("获取长期 OHLCV 数据失败: %v", err)
	}

	longerIndicators := CalculateIndicators(longerOHLCV)

	// Build JSON data
	// 构建 JSON 数据
	jsonData, err := BuildMarketJSONData(
		ctx,
		marketData,
		symbol,
		timeframe,
		ohlcvData,
		indicators,
		longerIndicators,
		longerOHLCV,
	)

	if err != nil {
		t.Fatalf("构建 JSON 数据失败: %v", err)
	}

	// Verify basic fields
	// 验证基本字段
	if jsonData.CurrentPrice <= 0 {
		t.Errorf("CurrentPrice 应该大于 0，实际值: %f", jsonData.CurrentPrice)
	}

	if jsonData.Timeframe != timeframe {
		t.Errorf("Timeframe 不匹配，期望: %s，实际: %s", timeframe, jsonData.Timeframe)
	}

	// Verify indicators
	// 验证指标
	if jsonData.Indicators == nil {
		t.Fatal("Indicators 不应为 nil")
	}

	if len(jsonData.Indicators.EMA) == 0 {
		t.Error("EMA 数据为空")
	}

	if len(jsonData.Indicators.RSI) == 0 {
		t.Error("RSI 数据为空")
	}

	// Verify VWAP
	// 验证 VWAP
	if jsonData.Indicators.VWAP == nil {
		t.Error("VWAP 不应为 nil")
	} else {
		if jsonData.Indicators.VWAP.History15m != nil {
			if len(jsonData.Indicators.VWAP.History15m.Price) == 0 {
				t.Error("VWAP 价格历史为空")
			}
			if len(jsonData.Indicators.VWAP.History15m.DeviationPct) == 0 {
				t.Error("VWAP 偏离历史为空")
			}
			t.Logf("VWAP 历史数据点数: %d", len(jsonData.Indicators.VWAP.History15m.Price))
		}
	}

	// Verify volume
	// 验证成交量
	if jsonData.Volume == nil {
		t.Fatal("Volume 不应为 nil")
	}

	if jsonData.Volume.Current <= 0 {
		t.Error("当前成交量应该大于 0")
	}

	// Verify multi-timeframe indicators
	// 验证多时间框架指标
	if len(jsonData.MultiTF) == 0 {
		t.Error("多时间框架指标为空")
	} else {
		t.Logf("多时间框架指标数量: %d", len(jsonData.MultiTF))
	}

	// Verify positions data
	// 验证持仓数据
	if jsonData.Positions == nil {
		t.Error("Positions 不应为 nil")
	} else if jsonData.Positions.OpenInterest != nil {
		if jsonData.Positions.OpenInterest.TF15m != nil {
			changeRates := jsonData.Positions.OpenInterest.TF15m.ChangeRate
			volumes := jsonData.Positions.OpenInterest.TF15m.Volume

			t.Logf("持仓量变化率数据点数: %d", len(changeRates))
			t.Logf("持仓量数据点数: %d", len(volumes))

			if len(changeRates) != len(volumes) {
				t.Errorf("变化率和持仓量数据点数不匹配: %d vs %d", len(changeRates), len(volumes))
			}
		}
	}

	t.Logf("✅ 所有数据验证通过")
}

// TestGetDefaultSchema tests the GetDefaultSchema function
// TestGetDefaultSchema 测试 GetDefaultSchema 函数
func TestGetDefaultSchema(t *testing.T) {
	schema := GetDefaultSchema()

	if schema == nil {
		t.Fatal("Schema 不应为 nil")
	}

	// Verify all required fields are present
	// 验证所有必需字段都存在
	requiredFields := []string{
		"VWAP",
		"multi_tf",
		"long_term_1h",
		"price_history",
		"BB",
		"current_position",
	}

	for _, field := range requiredFields {
		if _, ok := schema[field]; !ok {
			t.Errorf("Schema 缺少必需字段: %s", field)
		}
	}

	// Verify field descriptions
	// 验证字段描述
	if schema["VWAP"].Type != "float" {
		t.Errorf("VWAP type 应该是 'float'，实际: %s", schema["VWAP"].Type)
	}

	if schema["VWAP"].Description == "" {
		t.Error("VWAP description 不应为空")
	}

	if schema["multi_tf"].Type != "object" {
		t.Errorf("multi_tf type 应该是 'object'，实际: %s", schema["multi_tf"].Type)
	}

	if schema["current_position"].Type != "object|null" {
		t.Errorf("current_position type 应该是 'object|null'，实际: %s", schema["current_position"].Type)
	}

	t.Logf("✅ Schema 包含 %d 个字段", len(schema))
	for field, schemaField := range schema {
		t.Logf("  - %s: %s - %s", field, schemaField.Type, schemaField.Description)
	}
}

// TestSchemaJSONSerialization tests that schema can be serialized to JSON
// TestSchemaJSONSerialization 测试 Schema 可以序列化为 JSON
func TestSchemaJSONSerialization(t *testing.T) {
	schema := GetDefaultSchema()

	// Serialize to JSON
	// 序列化为 JSON
	jsonBytes, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		t.Fatalf("Schema 序列化失败: %v", err)
	}

	// Verify it's valid JSON
	// 验证是否为有效的 JSON
	var schemaMap map[string]*SchemaField
	err = json.Unmarshal(jsonBytes, &schemaMap)
	if err != nil {
		t.Fatalf("Schema 反序列化失败: %v", err)
	}

	// Verify all fields are preserved
	// 验证所有字段都被保留
	if len(schemaMap) != len(schema) {
		t.Errorf("序列化后字段数量不匹配，期望: %d，实际: %d", len(schema), len(schemaMap))
	}

	t.Logf("✅ Schema JSON 序列化成功，大小: %d 字节", len(jsonBytes))
	t.Logf("Schema JSON:\n%s", string(jsonBytes))
}

// TestCompleteJSONFormat tests the complete JSON format with instructions, schema, and data
// TestCompleteJSONFormat 测试包含 instructions、schema 和 data 的完整 JSON 格式
func TestCompleteJSONFormat(t *testing.T) {
	// Create sample market data
	// 创建示例市场数据
	marketData := map[string]*SymbolMarketData{
		"BTC/USDT": {
			CurrentPrice: 90592.1,
			Timeframe:    "15m",
			Indicators: &IndicatorsData{
				EMA: map[string]float64{
					"12": 90673.82,
					"26": 90945.0,
				},
				MACD: -271.18,
				RSI: map[string]float64{
					"7":  45.2,
					"14": 48.5,
				},
				ADX: 25.3,
			},
			Volume: &VolumeData{
				Current:   1250000,
				Average7:  1150000,
				Average14: 1100000,
				Average20: 1080000,
			},
		},
	}

	// Sample instructions
	// 示例指令
	instructions := "你是专业的趋势交易分析师。根据提供的市场数据，输出严格可解析的 JSON 格式分析结果。"

	// Build complete JSON
	// 构建完整 JSON
	completeJSON := map[string]interface{}{
		"instructions": instructions,
		"schema":       GetDefaultSchema(),
		"data":         marketData,
	}

	// Serialize to JSON
	// 序列化为 JSON
	jsonBytes, err := json.MarshalIndent(completeJSON, "", "  ")
	if err != nil {
		t.Fatalf("JSON 序列化失败: %v", err)
	}

	// Verify it's valid JSON
	// 验证是否为有效的 JSON
	var jsonMap map[string]interface{}
	err = json.Unmarshal(jsonBytes, &jsonMap)
	if err != nil {
		t.Fatalf("JSON 反序列化失败: %v", err)
	}

	// Verify top-level keys
	// 验证顶层键
	requiredKeys := []string{"instructions", "schema", "data"}
	for _, key := range requiredKeys {
		if _, ok := jsonMap[key]; !ok {
			t.Errorf("JSON 缺少必需的顶层键: %s", key)
		}
	}

	// Verify instructions
	// 验证 instructions
	if instructionsVal, ok := jsonMap["instructions"].(string); !ok {
		t.Error("instructions 应该是字符串类型")
	} else if instructionsVal == "" {
		t.Error("instructions 不应为空")
	}

	// Verify schema
	// 验证 schema
	if schemaVal, ok := jsonMap["schema"].(map[string]interface{}); !ok {
		t.Error("schema 应该是对象类型")
	} else if len(schemaVal) == 0 {
		t.Error("schema 不应为空")
	}

	// Verify data
	// 验证 data
	if dataVal, ok := jsonMap["data"].(map[string]interface{}); !ok {
		t.Error("data 应该是对象类型")
	} else if len(dataVal) == 0 {
		t.Error("data 不应为空")
	}

	t.Logf("✅ 完整 JSON 格式验证通过")
	t.Logf("JSON 大小: %d 字节", len(jsonBytes))
	t.Logf("包含交易对数量: %d", len(marketData))
}
