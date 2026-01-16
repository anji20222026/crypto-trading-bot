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
	fmt.Println("==================================================")

	// Load configuration
	cfg, err := config.LoadConfig(".env")
	if err != nil {
		log.Fatalf("❌ 配置加载失败: %v", err)
	}

	fmt.Printf("✅ 配置加载成功\n")
	fmt.Printf("   - 交易对: %v\n", cfg.CryptoSymbols)
	fmt.Printf("   - 时间周期: %s\n", cfg.CryptoTimeframe)
	fmt.Printf("   - 测试模式: %v\n", cfg.BinanceTestMode)
	fmt.Println()

	if len(cfg.CryptoSymbols) == 0 {
		log.Fatal("❌ 配置中没有交易对")
	}

	ctx := context.Background()
	marketData := dataflows.NewMarketData(cfg)

	// ===== 统一的输出结构 =====
	marketJSONData := &dataflows.MarketJSONData{
		Schema: dataflows.GetDefaultSchema(),
		Data:   make(map[string]*dataflows.SymbolMarketData),
	}

	// ===== 主循环：每个 symbol 一次 =====
	for _, symbol := range cfg.CryptoSymbols {

		binanceSymbol := cfg.GetBinanceSymbolFor(symbol)

		fmt.Printf("📊 正在处理 %s (%s)\n", symbol, binanceSymbol)

		// ---- 主周期 OHLCV ----
		ohlcvData, err := marketData.GetOHLCV(
			ctx,
			binanceSymbol,
			cfg.CryptoTimeframe,
			cfg.CryptoLookbackDays,
		)
		if err != nil {
			log.Printf("❌ %s OHLCV 获取失败: %v", symbol, err)
			continue
		}

		indicators := dataflows.CalculateIndicators(ohlcvData)

		// ---- 多周期（可选）----
		var longerOHLCV []dataflows.OHLCV
		var longerIndicators *dataflows.TechnicalIndicators

		if cfg.EnableMultiTimeframe {
			longerOHLCV, err = marketData.GetOHLCV(
				ctx,
				binanceSymbol,
				cfg.CryptoLongerTimeframe,
				cfg.CryptoLongerLookbackDays,
			)
			if err != nil {
				log.Printf("⚠️ %s 长周期数据失败: %v", symbol, err)
			} else {
				longerIndicators = dataflows.CalculateIndicators(longerOHLCV)
			}
		}

		// ---- 构建结构化数据 ----
		jsonData, err := dataflows.BuildMarketJSONData(
			ctx,
			marketData,
			binanceSymbol,
			cfg.CryptoTimeframe,
			ohlcvData,
			indicators,
			longerIndicators,
			longerOHLCV,
			nil, // No position info in demo / Demo 中不需要持仓信息
		)
		if err != nil {
			log.Printf("❌ %s 结构化数据构建失败: %v", symbol, err)
			continue
		}

		// ---- 合并进总结构 ----
		marketJSONData.Data[symbol] = jsonData

		fmt.Printf("   ✅ %s 完成 | Price: %.2f\n", symbol, jsonData.CurrentPrice)
		fmt.Println()
	}

	// ===== 输出 JSON =====
	jsonBytes, err := sonic.MarshalIndent(marketJSONData, "", "  ")
	if err != nil {
		log.Fatalf("❌ JSON 序列化失败: %v", err)
	}

	outputFile := "market_data_output.json"
	if err := os.WriteFile(outputFile, jsonBytes, 0644); err != nil {
		log.Fatalf("❌ 文件保存失败: %v", err)
	}

	fmt.Println("📈 市场数据生成完成")
	fmt.Printf("💾 已保存到: %s\n", outputFile)
	fmt.Printf("📦 包含交易对数量: %d\n", len(marketJSONData.Data))
	fmt.Println()

	// ===== Schema 输出 =====
	schema, err := dataflows.GenerateMarketDataSchema()
	if err == nil {
		_ = os.WriteFile("market_data_schema.json", []byte(schema), 0644)
		fmt.Println("📋 JSON Schema 已生成")
	}

	schemaDesc := dataflows.GetMarketDataSchemaWithDescription()
	_ = os.WriteFile("market_data_schema_description.md", []byte(schemaDesc), 0644)
	fmt.Println("📖 Schema 描述文档已生成")

	fmt.Println()
	fmt.Println("✅ Demo 运行完成")
}
