package dataflows

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/oak/crypto-trading-bot/internal/config"
	"github.com/oak/crypto-trading-bot/internal/constant"
	"github.com/oak/crypto-trading-bot/internal/logger"
)

// TestBinanceFetchKlines 测试从币安获取 K 线数据
// 注意：K线数据是公开接口，不需要 API key
// 运行方式：go test -v ./internal/dataflows -run TestBinanceFetchKlines
func TestBinanceFetchKlines(t *testing.T) {

	// Load configuration
	cfg, err := config.LoadConfig(constant.BlankStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger.Init(cfg.DebugMode)

	marketData := NewMarketData(cfg)
	ctx := context.Background()
	symbol := "BTCUSDT"
	timeframe := "1h"

	t.Logf("获取 %s K 线，时间周期 %s，回看天数 %d", symbol, timeframe, cfg.CryptoLookbackDays)
	ohlcvData, err := marketData.GetOHLCV(ctx, symbol, timeframe, cfg.CryptoLookbackDays)
	if err != nil {
		t.Fatalf("获取 %s K 线失败: %v", timeframe, err)
	}

	if len(ohlcvData) == 0 {
		t.Fatalf("%s 返回的数据为空", symbol)
	}

	t.Logf("✅ %s: 成功获取 %d 根 K 线（无需 API key）", timeframe, len(ohlcvData))

	// 打印最新价格
	latestPrice := ohlcvData[len(ohlcvData)-1].Close
	t.Logf("最新价格: $%.2f", latestPrice)
}

// TestBinanceFetchMultipleTimeframes 测试获取多个时间周期的数据
func TestBinanceFetchMultipleTimeframes(t *testing.T) {
	cfg := &config.Config{
		BinanceAPIKey:    "", // 公开数据不需要 API key
		BinanceAPISecret: "",
		BinanceTestMode:  false,
	}

	marketData := NewMarketData(cfg)
	ctx := context.Background()
	symbol := "BTCUSDT"

	// 测试不同的时间周期
	timeframes := []struct {
		tf           string
		lookbackDays int
		minCandles   int // 期望的最少 K 线数量
	}{
		{"15m", 1, 80}, // 1 天约 96 根 15 分钟 K 线
		{"1h", 1, 20},  // 1 天约 24 根 1 小时 K 线
		{"4h", 2, 10},  // 2 天约 12 根 4 小时 K 线
		{"1d", 7, 5},   // 7 天约 7 根 1 日 K 线
	}

	for _, tc := range timeframes {
		t.Run(tc.tf, func(t *testing.T) {
			ohlcvData, err := marketData.GetOHLCV(ctx, symbol, tc.tf, tc.lookbackDays)
			if err != nil {
				t.Fatalf("获取 %s K 线失败: %v", tc.tf, err)
			}

			if len(ohlcvData) < tc.minCandles {
				t.Errorf("期望至少 %d 根 K 线，实际获取: %d", tc.minCandles, len(ohlcvData))
			}

			t.Logf("✅ %s: 成功获取 %d 根 K 线（无需 API key）", tc.tf, len(ohlcvData))
		})
	}
}

// TestBinanceFetchWithIndicators 测试获取数据后计算技术指标
func TestBinanceFetchWithIndicators(t *testing.T) {
	cfg := &config.Config{
		BinanceAPIKey:    "", // 公开数据
		BinanceAPISecret: "",
		BinanceTestMode:  false,
	}

	marketData := NewMarketData(cfg)
	ctx := context.Background()

	// 获取足够的数据来计算 200 日均线
	ohlcvData, err := marketData.GetOHLCV(ctx, "BTCUSDT", "1d", 250)
	if err != nil {
		t.Fatalf("获取 K 线数据失败: %v", err)
	}

	t.Logf("获取了 %d 根日线数据", len(ohlcvData))

	// 计算技术指标
	indicators := CalculateIndicators(ohlcvData)

	// 验证指标长度
	if len(indicators.RSI) != len(ohlcvData) {
		t.Errorf("RSI 长度不匹配: 期望 %d, 实际 %d", len(ohlcvData), len(indicators.RSI))
	}

	if len(indicators.MACD) != len(ohlcvData) {
		t.Errorf("MACD 长度不匹配: 期望 %d, 实际 %d", len(ohlcvData), len(indicators.MACD))
	}

	// 获取最新的指标值
	lastIdx := len(ohlcvData) - 1
	latestRSI := indicators.RSI[lastIdx]
	latestMACD := indicators.MACD[lastIdx]
	latestSMA20 := indicators.SMA_20[lastIdx]

	t.Logf("最新指标值：")
	t.Logf("  RSI(14): %.2f", latestRSI)
	t.Logf("  MACD: %.2f", latestMACD)
	t.Logf("  SMA(20): %.2f", latestSMA20)
	t.Logf("  当前价格: %.2f", ohlcvData[lastIdx].Close)

	// 验证 RSI 在合理范围内
	if latestRSI < 0 || latestRSI > 100 {
		t.Errorf("RSI 应该在 0-100 之间，实际: %.2f", latestRSI)
	}
}

// TestBinanceAPIRateLimit 测试 API 速率限制处理
func TestBinanceAPIRateLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过速率限制测试（运行时间较长）")
	}

	cfg := &config.Config{
		BinanceAPIKey:    "", // 公开数据
		BinanceAPISecret: "",
		BinanceTestMode:  false,
	}

	marketData := NewMarketData(cfg)
	ctx := context.Background()

	// 连续请求多次，测试是否会触发速率限制
	requestCount := 5
	successCount := 0

	for i := 0; i < requestCount; i++ {
		_, err := marketData.GetOHLCV(ctx, "BTCUSDT", "1h", 1)
		if err != nil {
			t.Logf("请求 %d 失败: %v", i+1, err)
		} else {
			successCount++
		}

		// 请求之间稍微延迟，避免过快
		time.Sleep(200 * time.Millisecond)
	}

	t.Logf("完成 %d 次请求，成功 %d 次（无需 API key）", requestCount, successCount)

	if successCount == 0 {
		t.Error("所有请求都失败了，可能是网络问题")
	}
}

// TestBinanceDataQuality 测试数据质量
func TestBinanceDataQuality(t *testing.T) {
	cfg := &config.Config{
		BinanceAPIKey:    "", // 公开数据
		BinanceAPISecret: "",
		BinanceTestMode:  false,
	}

	marketData := NewMarketData(cfg)
	ctx := context.Background()

	ohlcvData, err := marketData.GetOHLCV(ctx, "BTCUSDT", "1h", 7)
	if err != nil {
		t.Fatalf("获取 K 线数据失败: %v", err)
	}

	// 检查数据质量
	issues := 0

	for i, candle := range ohlcvData {
		// 检查是否有异常的价格关系
		if candle.High < candle.Low {
			t.Errorf("K线 %d: 最高价 (%.2f) < 最低价 (%.2f)", i, candle.High, candle.Low)
			issues++
		}

		if candle.High < candle.Open || candle.High < candle.Close {
			t.Errorf("K线 %d: 最高价 (%.2f) 不是真正的最高", i, candle.High)
			issues++
		}

		if candle.Low > candle.Open || candle.Low > candle.Close {
			t.Errorf("K线 %d: 最低价 (%.2f) 不是真正的最低", i, candle.Low)
			issues++
		}

		// 检查价格是否合理（BTC 通常在 10k-200k 范围）
		if candle.Close < 1000 || candle.Close > 200000 {
			t.Logf("警告: K线 %d 的收盘价 (%.2f) 看起来不太正常", i, candle.Close)
		}
	}

	if issues > 0 {
		t.Errorf("发现 %d 个数据质量问题", issues)
	} else {
		t.Log("✅ 数据质量检查通过")
	}
}

// TestBinanceDifferentSymbols 测试不同的交易对
func TestBinanceDifferentSymbols(t *testing.T) {
	cfg := &config.Config{
		BinanceAPIKey:    "",
		BinanceAPISecret: "",
		BinanceTestMode:  false,
	}

	marketData := NewMarketData(cfg)
	ctx := context.Background()

	symbols := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT"}

	for _, symbol := range symbols {
		t.Run(symbol, func(t *testing.T) {
			ohlcvData, err := marketData.GetOHLCV(ctx, symbol, "1h", 1)
			if err != nil {
				t.Fatalf("获取 %s K 线失败: %v", symbol, err)
			}

			if len(ohlcvData) == 0 {
				t.Fatalf("%s 返回的数据为空", symbol)
			}

			latestPrice := ohlcvData[len(ohlcvData)-1].Close
			t.Logf("✅ %s 最新价格: $%.2f", symbol, latestPrice)
		})
	}
}

// TestBinanceGenerateCompleteJSON 测试从币安 API 获取数据并生成完整的 JSON 格式
// 这个测试会调用真实的币安 API，生成包含 instructions + schema + data 的完整 JSON
// 运行方式：go test -v ./internal/dataflows -run TestBinanceGenerateCompleteJSON
func TestBinanceGenerateCompleteJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试（需要调用币安 API）")
	}

	// Load configuration from project root
	// 从项目根目录加载配置
	cfg, err := config.LoadConfig("../../.env")
	if err != nil {
		// Try loading from current directory
		// 尝试从当前目录加载
		cfg, err = config.LoadConfig(".env")
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}
	}

	marketData := NewMarketData(cfg)
	ctx := context.Background()

	// 从配置读取交易对和时间周期
	symbols := cfg.CryptoSymbols
	timeframe := cfg.CryptoTimeframe

	t.Logf("🚀 开始从币安获取数据")
	t.Logf("   - 交易对: %v", symbols)
	t.Logf("   - 时间周期: %s", timeframe)
	t.Logf("   - 回看天数: %d", cfg.CryptoLookbackDays)

	// 构建市场数据映射
	marketDataMap := make(map[string]*SymbolMarketData)

	// 遍历所有交易对
	for _, symbol := range symbols {
		t.Logf("\n📊 正在处理 %s...", symbol)

		// 转换为币安格式（去掉斜杠）
		binanceSymbol := strings.ReplaceAll(symbol, "/", "")

		// 1. 获取主时间框架数据
		ohlcvData, err := marketData.GetOHLCV(ctx, binanceSymbol, timeframe, cfg.CryptoLookbackDays)
		if err != nil {
			t.Logf("⚠️  获取 %s 的 %s K 线数据失败: %v，跳过", symbol, timeframe, err)
			continue
		}
		t.Logf("   ✅ 获取了 %d 根 %s K 线", len(ohlcvData), timeframe)

		// 2. 获取长期时间框架数据（1h）
		longerOHLCV, err := marketData.GetOHLCV(ctx, binanceSymbol, "1h", 10)
		if err != nil {
			t.Logf("⚠️  获取 %s 的 1h K 线数据失败: %v，跳过", symbol, err)
			continue
		}
		t.Logf("   ✅ 获取了 %d 根 1h K 线", len(longerOHLCV))

		// 3. 计算技术指标
		indicators := CalculateIndicators(ohlcvData)
		longerIndicators := CalculateIndicators(longerOHLCV)
		t.Logf("   ✅ 计算技术指标完成")

		// 4. 构建市场 JSON 数据
		symbolMarketData, err := BuildMarketJSONData(
			ctx,
			marketData,
			binanceSymbol,
			timeframe,
			ohlcvData,
			indicators,
			longerIndicators,
			longerOHLCV,
		)
		if err != nil {
			t.Logf("⚠️  构建 %s 市场 JSON 数据失败: %v，跳过", symbol, err)
			continue
		}

		// 添加到映射（使用带斜杠的格式作为 key）
		marketDataMap[symbol] = symbolMarketData
		t.Logf("   ✅ %s 数据构建完成（当前价格: $%.2f）", symbol, symbolMarketData.CurrentPrice)
	}

	if len(marketDataMap) == 0 {
		t.Fatal("❌ 没有成功获取任何交易对的数据")
	}

	t.Logf("\n✅ 成功获取 %d 个交易对的数据", len(marketDataMap))

	// 构建完整 JSON（schema + data）
	completeJSON := map[string]interface{}{
		"schema": GetDefaultSchema(),
		"data":   marketDataMap,
	}

	// 6. 序列化为 JSON
	jsonBytes, err := sonic.MarshalIndent(completeJSON, "", "  ")
	if err != nil {
		t.Fatalf("JSON 序列化失败: %v", err)
	}

	// 7. 输出统计信息
	t.Logf("\n📊 生成的完整 JSON 统计:")
	t.Logf("   - JSON 大小: %d 字节", len(jsonBytes))
	t.Logf("   - 交易对数量: %d", len(marketDataMap))
	t.Logf("   - 时间周期: %s", timeframe)

	// 输出每个交易对的价格
	for symbol, data := range marketDataMap {
		t.Logf("   - %s 当前价格: $%.2f", symbol, data.CurrentPrice)
	}

	// 8. 验证 JSON 结构
	var jsonMap map[string]interface{}
	err = sonic.Unmarshal(jsonBytes, &jsonMap)
	if err != nil {
		t.Fatalf("JSON 反序列化失败: %v", err)
	}

	// 验证顶层键
	requiredKeys := []string{"schema", "data"}
	for _, key := range requiredKeys {
		if _, ok := jsonMap[key]; !ok {
			t.Errorf("JSON 缺少必需的顶层键: %s", key)
		}
	}

	// 验证 schema 字段
	if schemaVal, ok := jsonMap["schema"].(map[string]interface{}); ok {
		t.Logf("   - Schema 字段数量: %d", len(schemaVal))
		for field := range schemaVal {
			t.Logf("     • %s", field)
		}
	}

	// 验证数据字段
	if dataVal, ok := jsonMap["data"].(map[string]interface{}); ok {
		if btcData, ok := dataVal["BTC/USDT"].(map[string]interface{}); ok {
			if indicators, ok := btcData["indicators"].(map[string]interface{}); ok {
				if ema, ok := indicators["EMA"].(map[string]interface{}); ok {
					t.Logf("   - EMA 周期数量: %d", len(ema))
				}
				if rsi, ok := indicators["RSI"].(map[string]interface{}); ok {
					t.Logf("   - RSI 周期数量: %d", len(rsi))
				}
			}
		}
	}

	// 9. 保存到文件（可选）
	filename := "binance_market_data_output.json"
	err = os.WriteFile(filename, jsonBytes, 0644)
	if err != nil {
		t.Logf("⚠️  保存文件失败: %v", err)
	} else {
		t.Logf("💾 完整 JSON 已保存到: %s", filename)
	}

	t.Logf("\n✅ 测试完成！成功从币安 API 生成完整的交易上下文 JSON")
}
