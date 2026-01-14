package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/bytedance/sonic"
	"github.com/oak/crypto-trading-bot/internal/config"
	"github.com/oak/crypto-trading-bot/internal/dataflows"
)

func main() {
	fmt.Println("🚀 币安市场数据结构化输出 Demo")
	fmt.Println("=" + string(make([]byte, 50)) + "=")

	// Load configuration from .env
	// 从 .env 加载配置
	cfg, err := config.LoadConfig(".env")
	if err != nil {
		log.Fatalf("❌ 配置加载失败: %v", err)
	}

	fmt.Printf("✅ 配置加载成功\n")
	fmt.Printf("   - 交易对: %v\n", cfg.CryptoSymbols)
	fmt.Printf("   - 时间周期: %s\n", cfg.CryptoTimeframe)
	fmt.Printf("   - 测试模式: %v\n", cfg.BinanceTestMode)
	fmt.Println()

	// Create market data instance
	// 创建市场数据实例
	marketData := dataflows.NewMarketData(cfg)
	ctx := context.Background()

	// Get the first symbol from config
	// 从配置中获取第一个交易对
	if len(cfg.CryptoSymbols) == 0 {
		log.Fatal("❌ 配置中没有交易对")
	}
	for _, symbol := range cfg.CryptoSymbols {

		binanceSymbol := cfg.GetBinanceSymbolFor(symbol)

		fmt.Printf("📊 正在获取 %s 的市场数据...\n", symbol)
		fmt.Println()
		// Fetch OHLCV data for primary timeframe
		// 获取主时间周期的 OHLCV 数据
		fmt.Printf("   🔄 获取 %s OHLCV 数据...\n", cfg.CryptoTimeframe)
		ohlcvData, err := marketData.GetOHLCV(ctx, binanceSymbol, cfg.CryptoTimeframe, cfg.CryptoLookbackDays)
		if err != nil {
			log.Fatalf("❌ OHLCV 数据获取失败: %v", err)
		}
		fmt.Printf("   ✅ 获取到 %d 条 K 线数据\n", len(ohlcvData))

		// Calculate indicators
		// 计算技术指标
		fmt.Printf("   🔄 计算技术指标...\n")
		indicators := dataflows.CalculateIndicators(ohlcvData)
		fmt.Printf("   ✅ 技术指标计算完成\n")

		// Fetch longer timeframe data if enabled
		// 如果启用，获取更长期时间周期数据
		var longerOHLCV []dataflows.OHLCV
		var longerIndicators *dataflows.TechnicalIndicators

		if cfg.EnableMultiTimeframe {
			fmt.Printf("   🔄 获取 %s OHLCV 数据...\n", cfg.CryptoLongerTimeframe)
			longerOHLCV, err = marketData.GetOHLCV(ctx, binanceSymbol, cfg.CryptoLongerTimeframe, cfg.CryptoLongerLookbackDays)
			if err != nil {
				log.Printf("   ⚠️  长期数据获取失败: %v", err)
			} else {
				fmt.Printf("   ✅ 获取到 %d 条长期 K 线数据\n", len(longerOHLCV))
				longerIndicators = dataflows.CalculateIndicators(longerOHLCV)
			}
		}
		// Build structured JSON data
		// 构建结构化 JSON 数据
		fmt.Println()
		fmt.Printf("🔧 正在构建结构化市场数据...\n")
		jsonData, err := dataflows.BuildMarketJSONData(
			ctx,
			marketData,
			binanceSymbol,
			cfg.CryptoTimeframe,
			ohlcvData,
			indicators,
			longerIndicators,
			longerOHLCV,
		)
		if err != nil {
			log.Fatalf("❌ 结构化数据构建失败: %v", err)
		}
		fmt.Printf("✅ 结构化数据构建完成\n")
		fmt.Println()
		fmt.Println(jsonData)

	}
	symbol := cfg.CryptoSymbols[0]
	binanceSymbol := cfg.GetBinanceSymbolFor(symbol)

	fmt.Printf("📊 正在获取 %s 的市场数据...\n", symbol)
	fmt.Println()

	// Fetch OHLCV data for primary timeframe
	// 获取主时间周期的 OHLCV 数据
	fmt.Printf("   🔄 获取 %s OHLCV 数据...\n", cfg.CryptoTimeframe)
	ohlcvData, err := marketData.GetOHLCV(ctx, binanceSymbol, cfg.CryptoTimeframe, cfg.CryptoLookbackDays)
	if err != nil {
		log.Fatalf("❌ OHLCV 数据获取失败: %v", err)
	}
	fmt.Printf("   ✅ 获取到 %d 条 K 线数据\n", len(ohlcvData))

	// Calculate indicators
	// 计算技术指标
	fmt.Printf("   🔄 计算技术指标...\n")
	indicators := dataflows.CalculateIndicators(ohlcvData)
	fmt.Printf("   ✅ 技术指标计算完成\n")

	// Fetch longer timeframe data if enabled
	// 如果启用，获取更长期时间周期数据
	var longerOHLCV []dataflows.OHLCV
	var longerIndicators *dataflows.TechnicalIndicators

	if cfg.EnableMultiTimeframe {
		fmt.Printf("   🔄 获取 %s OHLCV 数据...\n", cfg.CryptoLongerTimeframe)
		longerOHLCV, err = marketData.GetOHLCV(ctx, binanceSymbol, cfg.CryptoLongerTimeframe, cfg.CryptoLongerLookbackDays)
		if err != nil {
			log.Printf("   ⚠️  长期数据获取失败: %v", err)
		} else {
			fmt.Printf("   ✅ 获取到 %d 条长期 K 线数据\n", len(longerOHLCV))
			longerIndicators = dataflows.CalculateIndicators(longerOHLCV)
		}
	}

	// Build structured JSON data
	// 构建结构化 JSON 数据
	fmt.Println()
	fmt.Printf("🔧 正在构建结构化市场数据...\n")
	jsonData, err := dataflows.BuildMarketJSONData(
		ctx,
		marketData,
		binanceSymbol,
		cfg.CryptoTimeframe,
		ohlcvData,
		indicators,
		longerIndicators,
		longerOHLCV,
	)
	if err != nil {
		log.Fatalf("❌ 结构化数据构建失败: %v", err)
	}
	fmt.Printf("✅ 结构化数据构建完成\n")
	fmt.Println()

	// Create MarketJSONData with the symbol and schema
	// 创建包含交易对和 schema 的 MarketJSONData
	marketJSONData := &dataflows.MarketJSONData{
		Schema: dataflows.GetDefaultSchema(),
		Data: map[string]*dataflows.SymbolMarketData{
			symbol: jsonData,
		},
	}

	// Marshal to JSON
	// 序列化为 JSON
	jsonBytes, err := sonic.MarshalIndent(marketJSONData, "", "  ")
	if err != nil {
		log.Fatalf("❌ JSON 序列化失败: %v", err)
	}

	// Print summary
	// 打印摘要
	fmt.Println("📈 市场数据摘要")
	fmt.Println("=" + string(make([]byte, 50)) + "=")
	fmt.Printf("交易对: %s\n", symbol)
	fmt.Printf("当前价格: %.2f USDT\n", jsonData.CurrentPrice)
	fmt.Printf("时间周期: %s\n", jsonData.Timeframe)
	fmt.Println()

	if jsonData.Indicators != nil {
		fmt.Println("📊 技术指标:")
		if ema12, ok := jsonData.Indicators.EMA["12"]; ok {
			fmt.Printf("   EMA12: %.2f\n", ema12)
		}
		if ema26, ok := jsonData.Indicators.EMA["26"]; ok {
			fmt.Printf("   EMA26: %.2f\n", ema26)
		}
		fmt.Printf("   MACD: %.2f\n", jsonData.Indicators.MACD)
		if rsi14, ok := jsonData.Indicators.RSI["14"]; ok {
			fmt.Printf("   RSI14: %.2f\n", rsi14)
		}
		fmt.Printf("   ADX: %.2f\n", jsonData.Indicators.ADX)
		fmt.Println()
	}

	// Save to file
	// 保存到文件
	outputFile := "market_data_output.json"
	err = os.WriteFile(outputFile, jsonBytes, 0644)
	if err != nil {
		log.Fatalf("❌ 文件保存失败: %v", err)
	}

	fmt.Printf("💾 完整 JSON 数据已保存到: %s\n", outputFile)

	// Generate and save JSON Schema
	// 生成并保存 JSON Schema
	schema, err := dataflows.GenerateMarketDataSchema()
	if err != nil {
		log.Printf("⚠️  Schema 生成失败: %v", err)
	} else {
		schemaFile := "market_data_schema.json"
		err = os.WriteFile(schemaFile, []byte(schema), 0644)
		if err != nil {
			log.Printf("⚠️  Schema 文件保存失败: %v", err)
		} else {
			fmt.Printf("📋 JSON Schema 已保存到: %s\n", schemaFile)
		}
	}

	// Save schema with description
	// 保存带描述的 Schema
	schemaDesc := dataflows.GetMarketDataSchemaWithDescription()
	schemaDescFile := "market_data_schema_description.md"
	err = os.WriteFile(schemaDescFile, []byte(schemaDesc), 0644)
	if err != nil {
		log.Printf("⚠️  Schema 描述文件保存失败: %v", err)
	} else {
		fmt.Printf("📖 Schema 描述已保存到: %s\n", schemaDescFile)
	}

	fmt.Println()
	fmt.Println("✅ Demo 运行完成！")
	fmt.Println()
	fmt.Println("📁 生成的文件:")
	fmt.Println("   1. market_data_output.json - 完整的市场数据")
	fmt.Println("   2. market_data_schema.json - JSON Schema 定义")
	fmt.Println("   3. market_data_schema_description.md - Schema 说明文档")
}
